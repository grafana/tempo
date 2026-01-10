package handler

import (
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/api"
)

// writeFormattedContentForRequest writes a protobuf message in the appropriate format based on Accept header
func (h *Handler) writeFormattedContentForRequest(w http.ResponseWriter, r *http.Request, m proto.Message) {
	// Check for both explicit nil and typed nil pointers (e.g., (*tempopb.SearchResponse)(nil))
	// A typed nil pointer is not equal to nil when passed as an interface, so we need reflection
	if m == nil || (reflect.ValueOf(m).Kind() == reflect.Ptr && reflect.ValueOf(m).IsNil()) {
		http.Error(w, "internal error: nil response", http.StatusInternalServerError)
		return
	}

	switch r.Header.Get(api.HeaderAccept) {
	case api.HeaderAcceptProtobuf:
		b, err := proto.Marshal(m)
		if err != nil {
			level.Error(h.logger).Log("msg", "failed to marshal response to protobuf", "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set(api.HeaderContentType, api.HeaderAcceptProtobuf)
		_, err = w.Write(b)
		if err != nil {
			level.Error(h.logger).Log("msg", "failed to write protobuf response", "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	default:
		w.Header().Set(api.HeaderContentType, api.HeaderAcceptJSON)
		err := new(jsonpb.Marshaler).Marshal(w, m)
		if err != nil {
			level.Error(h.logger).Log("msg", "failed to marshal response to JSON", "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// writeJSONResponse writes a generic JSON response
func (h *Handler) writeJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set(api.HeaderContentType, api.HeaderAcceptJSON)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		level.Error(h.logger).Log("msg", "failed to encode JSON response", "err", err)
	}
}
