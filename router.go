package gyr

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type Router struct {
	routes      []*Route
	middlewares []func(*Context)
}

func DefaultRouter() *Router {
	return &Router{
		routes:      make([]*Route, 0),
		middlewares: make([]func(*Context), 0),
	}
}

func (router *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	context := CreateContext(w, req)
	route := router.FindRoute(req.URL.Path)
	if route == nil {
		context.Response.Error("404 - Not Found", http.StatusNotFound).Send()
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
				return
			}
		}

		handler(context)
		if !context.Response.wasSent {
			context.Response.Send()
		}
		return
	}
	context.Response.Error("405 - Method Not Allowed", http.StatusMethodNotAllowed).Send()
}

func (router *Router) Middleware(middleware ...func(*Context)) {
	router.middlewares = append(router.middlewares, middleware...)
}

func (router *Router) Path(path string) *Route {
	route := createRoute(path)
	router.routes = append(router.routes, route)
	return route
}

func (router *Router) FindRoute(path string) *Route {
	var route *Route = nil
	for _, potentialRoute := range router.routes {
		if potentialRoute.pattern.MatchString(path) {
			route = potentialRoute
			break
		}
	}

	return route
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

func (route *Route) Middleware(middleware ...func(*Context)) *Route {
	route.middlewares = append(route.middlewares, middleware...)
	return route
}

func (route *Route) method(method string, handler func(*Context)) *Route {
	route.handlers[method] = handler
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
