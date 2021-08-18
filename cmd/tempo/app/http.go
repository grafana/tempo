package app

import (
	"net/http"

	"github.com/NYTimes/gziphandler"
	"github.com/weaveworks/common/middleware"
)

func httpGzipMiddleware() middleware.Interface {
	return middleware.Func(func(handler http.Handler) http.Handler {
		gzipHandler := gziphandler.GzipHandler(handler)

		return http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
			// do not gzip the response when requesting protobuf as this will mess up the encoding
			if r.Header.Get("Accept") == "application/protobuf" {
				handler.ServeHTTP(writer, r)
			} else {
				gzipHandler.ServeHTTP(writer, r)
			}
		})
	})
}
