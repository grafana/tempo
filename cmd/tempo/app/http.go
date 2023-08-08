package app

import (
	"net/http"

	"github.com/grafana/dskit/middleware"
	"github.com/klauspost/compress/gzhttp"
)

func httpGzipMiddleware() middleware.Interface {
	return middleware.Func(func(handler http.Handler) http.Handler {
		return gzhttp.GzipHandler(handler)
	})
}
