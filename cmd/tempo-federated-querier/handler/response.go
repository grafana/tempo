package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
)

const (
	contentTypeJSON     = "application/json"
	contentTypeProtobuf = "application/protobuf"
	acceptHeader        = "Accept"
)

// writeProtoResponse writes a protobuf message in the appropriate format based on Accept header
func (h *Handler) writeProtoResponse(w http.ResponseWriter, r *http.Request, msg proto.Message) {
	accept := r.Header.Get(acceptHeader)

	// Check if protobuf is requested
	if strings.Contains(accept, contentTypeProtobuf) {
		h.writeProtobuf(w, msg)
		return
	}

	// Default to JSON
	h.writeProtobufAsJSON(w, msg)
}

// writeProtobuf marshals and writes a protobuf message as binary protobuf
func (h *Handler) writeProtobuf(w http.ResponseWriter, msg proto.Message) {
	w.Header().Set("Content-Type", contentTypeProtobuf)
	data, err := proto.Marshal(msg)
	if err != nil {
		level.Error(h.logger).Log("msg", "failed to marshal response to protobuf", "err", err)
		http.Error(w, fmt.Sprintf("failed to marshal response: %v", err), http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

// writeProtobufAsJSON marshals and writes a protobuf message as JSON using jsonpb
func (h *Handler) writeProtobufAsJSON(w http.ResponseWriter, msg proto.Message) {
	w.Header().Set("Content-Type", contentTypeJSON)
	marshaler := &jsonpb.Marshaler{}
	if err := marshaler.Marshal(w, msg); err != nil {
		level.Error(h.logger).Log("msg", "failed to marshal response to JSON", "err", err)
		http.Error(w, fmt.Sprintf("failed to marshal response: %v", err), http.StatusInternalServerError)
		return
	}
}

// writeJSONResponse writes a generic JSON response
func (h *Handler) writeJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", contentTypeJSON)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		level.Error(h.logger).Log("msg", "failed to encode JSON response", "err", err)
	}
}
