package handler

import (
	"fmt"
	"net/http"
)

// ReadyHandler returns ready status
func (h *Handler) ReadyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ready")
}

// EchoHandler echoes the request
func (h *Handler) EchoHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "echo")
}

// BuildInfoHandler returns build information
func (h *Handler) BuildInfoHandler(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"version":   h.buildInfo.Version,
		"revision":  h.buildInfo.Revision,
		"branch":    h.buildInfo.Branch,
		"buildDate": h.buildInfo.BuildDate,
		"goVersion": h.buildInfo.GoVersion,
	}
	h.writeJSONResponse(w, info)
}

// InstancesHandler returns the list of configured Tempo instances
func (h *Handler) InstancesHandler(w http.ResponseWriter, r *http.Request) {
	instances := make([]map[string]interface{}, len(h.cfg.Instances))
	for i, inst := range h.cfg.Instances {
		instances[i] = map[string]interface{}{
			"name":     inst.Name,
			"endpoint": inst.Endpoint,
		}
	}
	h.writeJSONResponse(w, map[string]interface{}{
		"instances": instances,
	})
}
