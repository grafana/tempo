package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/grafana/tempo/pkg/util"
)

const (
	urlParamTraceID = "traceID"

	HeaderAccept         = "Accept"
	HeaderAcceptProtobuf = "application/protobuf"
	HeaderAcceptJSON     = "application/json"

	PathTraces          = "/api/traces/{traceID}"
	PathSearch          = "/api/search"
	PathSearchTags      = "/api/search/tags"
	PathSearchTagValues = "/api/search/tag/{tagName}/values"
	PathEcho            = "/api/echo"
	PathBackendSearch   = "/api/backend_search" // todo(search): integrate with real search
)

func ParseTraceID(r *http.Request) ([]byte, error) {
	vars := mux.Vars(r)
	traceID, ok := vars[urlParamTraceID]
	if !ok {
		return nil, fmt.Errorf("please provide a traceID")
	}

	byteID, err := util.HexStringToTraceID(traceID)
	if err != nil {
		return nil, err
	}

	return byteID, nil
}
