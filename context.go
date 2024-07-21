package gyr

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"net/http"
)

type Context struct {
	Request         *http.Request
	FallbackDecoder BodyDecoder
	writer          http.ResponseWriter
	variables       map[string]any
}

type BodyDecoder interface {
	Decode(any) error
}

func CreateContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Request:   req,
		writer:    w,
		variables: make(map[string]any),
	}
}

func (ctx *Context) Response() *Response {
	return NewResponse(ctx)
}

func (ctx *Context) SetVariable(key string, value any) {
	ctx.variables[key] = value
}

func (ctx *Context) Variable(key string) any {
	return ctx.variables[key]
}

func (ctx *Context) IntVariable(key string) int {
	return ctx.Variable(key).(int)
}

func (ctx *Context) FloatVariable(key string) float64 {
	return ctx.Variable(key).(float64)
}

func (ctx *Context) BoolVariable(key string) bool {
	return ctx.Variable(key).(bool)
}

func (ctx *Context) StringVariable(key string) string {
	return ctx.Variable(key).(string)
}

func ReadBody[T any](ctx *Context) (T, error) {
	var target T
	var decoder BodyDecoder
	switch ctx.Request.Header.Get("Content-Type") {
	case "application/json":
		decoder = json.NewDecoder(ctx.Request.Body)
	case "application/xml":
	case "text/xml":
		decoder = xml.NewDecoder(ctx.Request.Body)
	default:
		if ctx.FallbackDecoder != nil {
			decoder = ctx.FallbackDecoder
		} else {
			return target, errors.New("can not determine decoder to use from Content-Type header and no fallback set")
		}
	}
	err := decoder.Decode(&target)
	return target, err
}
