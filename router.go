package gyr

import (
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type RouterMatchable interface {
	MatchesPath(string) bool
}

type Router struct {
	routes      []RouterMatchable
	middlewares []func(*Context)
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
		middlewares: make([]func(*Context), 0),
		logger:      slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})),
	}
}

func (router *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	router.logger.Info("Incoming request", "method", req.Method, "path", req.URL.Path)

	context := CreateContext(w, req)
	route := router.FindRoute(req.URL.Path)
	if route == nil {
		context.Response.Error("404 - Not Found", http.StatusNotFound).Send()
		router.logger.Info("Response sent", "method", req.Method, "path", req.URL.Path, "status", context.Response.status)
		return
	}

	if handler := route.handlers[req.Method]; handler != nil {
		if len(route.variables) > 0 {
			extractVariablesIntoContext(route, context)
		}

		if len(route.middlewares) > 0 || len(router.middlewares) > 0 {
			middlewares := make([]func(*Context), len(router.middlewares), len(router.middlewares)+len(route.middlewares))
			copy(middlewares, router.middlewares)
			middlewares = append(middlewares, route.middlewares...)

			ok := runMiddlewares(middlewares, context)
			if !ok {
				context.Response.Send()
				router.logger.Info("Response sent", "method", req.Method, "path", req.URL.Path, "status", context.Response.status)
				return
			}
		}

		handler(context)
		if !context.Response.wasSent {
			context.Response.Send()
		}
		router.logger.Info("Response sent", "method", req.Method, "path", req.URL.Path, "status", context.Response.status)
		return
	}
	context.Response.Error("405 - Method Not Allowed", http.StatusMethodNotAllowed).Send()
	router.logger.Info("Response sent", "method", req.Method, "path", req.URL.Path, "status", context.Response.status)
}

func (router *Router) Middleware(middleware ...func(*Context)) {
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

func (router *Router) FindRoute(path string) *Route {
	return searchRoute(router.routes, path)
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
	handlers    map[string]func(*Context)
	middlewares []func(*Context)
	variables   map[string]int
}

func createRoute(path string) *Route {
	route := &Route{
		Path:        path,
		handlers:    make(map[string]func(*Context)),
		middlewares: make([]func(*Context), 0),
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

func (route *Route) Get(handler func(*Context)) *Route {
	return route.method(http.MethodGet, handler)
}

func (route *Route) Post(handler func(*Context)) *Route {
	return route.method(http.MethodPost, handler)
}

func (route *Route) Put(handler func(*Context)) *Route {
	return route.method(http.MethodPut, handler)
}

func (route *Route) Delete(handler func(*Context)) *Route {
	return route.method(http.MethodDelete, handler)
}

func (route *Route) Patch(handler func(*Context)) *Route {
	return route.method(http.MethodPatch, handler)
}

func (route *Route) Middleware(middleware ...func(*Context)) *Route {
	route.middlewares = append(route.middlewares, middleware...)
	return route
}

func (route *Route) method(method string, handler func(*Context)) *Route {
	route.handlers[method] = handler
	return route
}

type RouteGroup struct {
	Prefix      string
	middlewares []func(*Context)
	routes      []RouterMatchable
}

func createGroup(prefix string) *RouteGroup {
	return &RouteGroup{
		Prefix:      prefix,
		routes:      make([]RouterMatchable, 0),
		middlewares: make([]func(*Context), 0),
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
func (group *RouteGroup) Middleware(middleware ...func(*Context)) *RouteGroup {
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

func runMiddlewares(middlewares []func(*Context), ctx *Context) bool {
	if ctx.aborted {
		return false
	}

	for _, middleware := range middlewares {
		middleware(ctx)
		if ctx.aborted {
			return false
		}
	}

	return true
}
