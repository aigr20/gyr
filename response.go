package gyr

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	w       http.ResponseWriter
	status  int
	toWrite []byte
	wasSent bool
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

func (r *Response) Json(object any) *Response {
	jsonBytes, err := json.Marshal(object)
	if err != nil {
		r.Error("Internal Server Error", http.StatusInternalServerError)
		return r
	}
	r.w.Header().Set("Content-Type", "application/json")
	r.toWrite = append(r.toWrite, jsonBytes...)
	return r
}

func (r *Response) Error(err string, statusCode int) *Response {
	r.Status(statusCode).Text(err)
	return r
}

func (r *Response) Header(name string, value string) *Response {
	r.w.Header().Set(name, value)
	return r
}

func (r *Response) Send() {
	r.w.WriteHeader(r.status)
	r.w.Write(r.toWrite)
	r.wasSent = true
}
