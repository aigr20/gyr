package gyr

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	w       http.ResponseWriter
	status  int
	toWrite []byte
}

func NewResponse(ctx *Context) *Response {
	return &Response{
		w:       ctx.writer,
		status:  http.StatusOK,
		toWrite: make([]byte, 0),
	}
}

func (r *Response) Status(statusCode int) *Response {
	r.status = statusCode
	return r
}

func (r *Response) Text(text string) *Response {
	r.toWrite = append(r.toWrite, []byte(text)...)
	r.w.Header().Set("Content-Type", "text/plain")
	return r
}

func (r *Response) Html(html string) *Response {
	r.toWrite = append(r.toWrite, []byte(html)...)
	r.w.Header().Set("Content-Type", "text/html")
	return r
}

func (r *Response) Json(object any) *Response {
	jsonBytes, err := json.Marshal(object)
	if err != nil {
		r.InternalError().Text("Internal Server Error")
		return r
	}
	r.w.Header().Set("Content-Type", "application/json")
	r.toWrite = append(r.toWrite, jsonBytes...)
	return r
}

// Set the response content without setting a Content-Type header.
func (r *Response) Raw(text string) *Response {
	r.toWrite = append(r.toWrite, []byte(text)...)
	return r
}

func (r *Response) InternalError() *Response {
	r.Status(http.StatusInternalServerError)
	return r
}

func (r *Response) NoContent() *Response {
	return r.Status(http.StatusNoContent)
}

func (r *Response) Header(name string, value string) *Response {
	r.w.Header().Set(name, value)
	return r
}

func (r *Response) send() {
	r.w.WriteHeader(r.status)
	r.w.Write(r.toWrite)
}
