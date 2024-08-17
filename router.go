package gyr

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type RouterMatchable interface {
	MatchesPath(string) bool
}

type Handler func(*Context) *Response

type Router struct {
	routes      []RouterMatchable
	middlewares []Handler
	logger      *slog.Logger
}

func DefaultRouter() *Router {
	var logLevel slog.Level
	if isGyrDebug() {
		logLevel = slog.LevelDebug
	} else {
		logLevel = slog.LevelInfo
	}
	return &Router{
		routes:      make([]RouterMatchable, 0),
		middlewares: make([]Handler, 0),
		logger:      slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})),
	}
}

func (router *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	router.logger.Info("Incoming request", "method", req.Method, "path", req.URL.Path)

	context := CreateContext(w, req)
	route := router.FindRoute(req.URL.Path)

	var response *Response
	defer func() {
		response.send()
		router.logger.Info("Response sent", "status", response.status, "length", len(response.toWrite))
	}()

	if route == nil {
		response = context.Response().Error("404 - Not Found", http.StatusNotFound)
		return
	}

	if handler := route.handlers[req.Method]; handler != nil {
		if len(route.variables) > 0 {
			extractVariablesIntoContext(route, context)
		}

		if len(route.middlewares) > 0 || len(router.middlewares) > 0 {
			middlewares := make([]Handler, len(router.middlewares), len(router.middlewares)+len(route.middlewares))
			copy(middlewares, router.middlewares)
			middlewares = append(middlewares, route.middlewares...)

			response = runMiddlewares(middlewares, context)
			if response != nil {
				return
			}
		}

		response = handler(context)
		if response == nil {
			router.logger.Warn("Handler returned no response, creating a default response", "path", req.URL.Path)
			response = NewResponse(context)
		}
		return
	}
	response = context.Response().Error("405 - Method Not Allowed", http.StatusMethodNotAllowed)
}

func (router *Router) Middleware(middleware ...Handler) {
	router.middlewares = append(router.middlewares, middleware...)
}

func (router *Router) Path(path string) *Route {
	route := createRoute(path)
	router.routes = append(router.routes, route)
	return route
}

func (router *Router) Group(prefix string) *RouteGroup {
	group := createGroup(prefix)
	router.routes = append(router.routes, group)
	return group
}

func (router *Router) StaticDir(directory string) {
	group := router.Group(directory)
	filepath.WalkDir(directory, func(path string, file fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if file.IsDir() {
			return nil
		}

		cleaned := strings.ReplaceAll(path, "\\", "/")
		cleaned = strings.TrimPrefix(cleaned, directory)
		group.Path(cleaned).Get(staticFileHandler(router, path))
		return nil
	})
}

func (router *Router) FindRoute(path string) *Route {
	return searchRoute(router.routes, path)
}

func staticFileHandler(router *Router, fpath string) Handler {
	return func(ctx *Context) *Response {
		file, err := os.Open(fpath)
		if errors.Is(err, os.ErrNotExist) {
			return ctx.Response().Error(fmt.Sprintf("404 %s not found", fpath), http.StatusNotFound)
		} else if err != nil {
			router.logger.Error("failed reading static file", "err", err)
			return ctx.Response().Error("Internal Server Error", http.StatusInternalServerError)
		}

		content, err := io.ReadAll(file)
		if err != nil {
			router.logger.Error("failed reading static file", "err", err)
			return ctx.Response().Error("Internal Server Error", http.StatusInternalServerError)
		}
		return responseBasedOnFileExtension(ctx, fpath, string(content))
	}
}

// Create a [Response]-object based on the extension of the file.
func responseBasedOnFileExtension(ctx *Context, fpath string, content string) *Response {
	response := ctx.Response()
	lastPeriod := strings.LastIndex(fpath, ".")
	if lastPeriod == -1 {
		return response.Raw(content)
	}

	extension := fpath[lastPeriod:]
	switch extension {
	case ".html":
		return response.Html(content)
	case ".css":
		return response.Raw(content).Header("Content-Type", "text/css")
	case ".js":
		return response.Raw(content).Header("Content-Type", "text/javascript")
	case ".txt":
		return response.Text(content)
	default:
		return response.Raw(content)
	}
}

func extractVariablesIntoContext(route *Route, ctx *Context) {
	urlParts := strings.Split(ctx.Request.URL.Path, "/")
	for variableName, variableIndex := range route.variables {
		value := urlParts[variableIndex]

		valueInt, err := strconv.Atoi(value)
		if err == nil {
			ctx.SetVariable(variableName, valueInt)
			continue
		}

		valueFloat, err := strconv.ParseFloat(value, 64)
		if err == nil {
			ctx.SetVariable(variableName, valueFloat)
			continue
		}

		if value == "true" || value == "false" {
			valueBool, _ := strconv.ParseBool(value)
			ctx.SetVariable(variableName, valueBool)
			continue
		}

		ctx.SetVariable(variableName, value)
	}
}

type Route struct {
	Path        string
	pattern     *regexp.Regexp
	handlers    map[string]Handler
	middlewares []Handler
	variables   map[string]int
}

func createRoute(path string) *Route {
	route := &Route{
		Path:        path,
		handlers:    make(map[string]Handler),
		middlewares: make([]Handler, 0),
		variables:   make(map[string]int),
	}
	createPathRegex(route)
	return route
}

func (route *Route) MatchesPath(path string) bool {
	return route.pattern.MatchString(path)
}

func createPathRegex(route *Route) {
	if route.Path == "/" {
		route.pattern = regexp.MustCompile("^/$")
		return
	}

	parts := strings.Split(route.Path, "/")
	sb := strings.Builder{}
	sb.WriteRune('^')
	for i, part := range parts {
		if len(part) == 0 {
			continue
		}
		if strings.HasPrefix(part, ":") {
			sb.WriteString("/[a-zA-Z0-9-.]+")
			route.variables[strings.TrimPrefix(part, ":")] = i
		} else {
			sb.WriteRune('/')
			sb.WriteString(part)
		}
	}
	sb.WriteRune('$')
	pathPattern := regexp.MustCompile(sb.String())
	route.pattern = pathPattern
}

func (route *Route) Get(handler Handler) *Route {
	return route.method(http.MethodGet, handler)
}

func (route *Route) Post(handler Handler) *Route {
	return route.method(http.MethodPost, handler)
}

func (route *Route) Put(handler Handler) *Route {
	return route.method(http.MethodPut, handler)
}

func (route *Route) Delete(handler Handler) *Route {
	return route.method(http.MethodDelete, handler)
}

func (route *Route) Patch(handler Handler) *Route {
	return route.method(http.MethodPatch, handler)
}

func (route *Route) Middleware(middleware ...Handler) *Route {
	route.middlewares = append(route.middlewares, middleware...)
	return route
}

func (route *Route) method(method string, handler Handler) *Route {
	route.handlers[method] = handler
	return route
}

type RouteGroup struct {
	Prefix      string
	middlewares []Handler
	routes      []RouterMatchable
}

func createGroup(prefix string) *RouteGroup {
	return &RouteGroup{
		Prefix:      prefix,
		routes:      make([]RouterMatchable, 0),
		middlewares: make([]Handler, 0),
	}
}

func (group *RouteGroup) MatchesPath(path string) bool {
	// Should use Regex if we want groups to be able to contain path variables
	return strings.HasPrefix(path, group.Prefix)
}

func (group *RouteGroup) Path(path string) *Route {
	route := createRoute(path)
	route.middlewares = append(route.middlewares, group.middlewares...)
	group.routes = append(group.routes, route)
	return route
}

func (group *RouteGroup) Group(prefix string) *RouteGroup {
	nestedGroup := createGroup(prefix)
	group.routes = append(group.routes, nestedGroup)
	return nestedGroup
}

// Must be called before any routes are added to the group or the routes added before the call won't have the middlewares.
func (group *RouteGroup) Middleware(middleware ...Handler) *RouteGroup {
	group.middlewares = append(group.middlewares, middleware...)
	return group
}

func (group *RouteGroup) findInGroup(path string) *Route {
	return searchRoute(group.routes, path)
}

func searchRoute(haystack []RouterMatchable, path string) *Route {
	var route *Route = nil
	for _, routeOrGroup := range haystack {
		if routeOrGroup.MatchesPath(path) {
			switch routeOrGroup := routeOrGroup.(type) {
			case *Route:
				route = routeOrGroup
			case *RouteGroup:
				strippedPath := strings.TrimPrefix(path, routeOrGroup.Prefix)
				route = routeOrGroup.findInGroup(strippedPath)
				if route == nil {
					continue
				}
			}
			break
		}
	}
	return route
}

// Non-nil return value means execution should halt and response be sent.
func runMiddlewares(middlewares []Handler, ctx *Context) *Response {
	for _, middleware := range middlewares {
		response := middleware(ctx)
		if response != nil {
			return response
		}
	}

	return nil
}
