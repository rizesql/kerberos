package server

import "net/http"

type Route interface {
	Method() string
	Path() string
	Handle() http.HandlerFunc
}
