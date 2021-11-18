package app

import (
	"net/http"

	"github.com/klauspost/compress/gzhttp"
	"github.com/weaveworks/common/middleware"
)

func httpGzipMiddleware() middleware.Interface {
	return middleware.Func(func(handler http.Handler) http.Handler {
		return gzhttp.GzipHandler(handler)
	})
}
