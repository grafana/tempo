package app

import (
	"net/http"

	"github.com/gorilla/mux"
)

type handler interface {
	Handle(pattern string, handler http.Handler)
}

type muxWrapper struct {
	*mux.Router
}

func (m muxWrapper) Handle(pattern string, handler http.Handler) {
	m.Router.Handle(pattern, handler)
}
