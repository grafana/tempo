package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/go-kit/log/level"
	"github.com/gorilla/mux"
	"github.com/grafana/tempo/cmd/tempo-federated-querier/build"
	"github.com/grafana/tempo/pkg/util"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/prometheus/common/version"
	"gopkg.in/yaml.v3"
)

const apiDocs = "https://grafana.com/docs/tempo/latest/api_docs/"

// ReadyHandler returns ready status.
func (h *Handler) ReadyHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, "ready")
}

// StatusHandler returns status information about the service.
func (h *Handler) StatusHandler(w http.ResponseWriter, r *http.Request) {
	var errs []error
	msg := bytes.Buffer{}

	simpleEndpoints := map[string]func(io.Writer) error{
		"version":   h.writeStatusVersion,
		"endpoints": h.writeStatusEndpoints,
	}

	wrapStatus := func(endpoint string) {
		msg.WriteString("GET /status/" + endpoint + "\n")

		switch endpoint {
		case "config":
			err := h.writeStatusConfig(&msg, r)
			if err != nil {
				errs = append(errs, err)
			}
		default:
			err := simpleEndpoints[endpoint](&msg)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	vars := mux.Vars(r)

	if endpoint, ok := vars["endpoint"]; ok {
		wrapStatus(endpoint)
	} else {
		wrapStatus("version")
		wrapStatus("endpoints")
		wrapStatus("config")
	}

	w.Header().Set("Content-Type", "text/plain")

	joinErrors := func(errs []error) error {
		if len(errs) == 0 {
			return nil
		}
		var err error

		for _, e := range errs {
			if e != nil {
				if err == nil {
					err = e
				} else {
					err = fmt.Errorf("%s: %w", e.Error(), err)
				}
			}
		}
		return err
	}

	err := joinErrors(errs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	if _, err := w.Write(msg.Bytes()); err != nil {
		level.Error(h.logger).Log("msg", "error writing response", "err", err)
	}
}

func (h *Handler) writeStatusVersion(w io.Writer) error {
	_, err := w.Write([]byte(version.Print("tempo-federated-querier") + "\n"))
	return err
}

func (h *Handler) writeStatusConfig(w io.Writer, r *http.Request) error {
	var output interface{}

	mode := r.URL.Query().Get("mode")
	switch mode {
	case "diff":
		defaultCfg := h.defaultCfg

		defaultCfgYaml, err := util.YAMLMarshalUnmarshal(defaultCfg)
		if err != nil {
			return err
		}

		cfgYaml, err := util.YAMLMarshalUnmarshal(h.cfg)
		if err != nil {
			return err
		}

		output, err = util.DiffConfig(defaultCfgYaml, cfgYaml)
		if err != nil {
			return err
		}
	case "defaults":
		output = h.defaultCfg
	case "":
		output = h.cfg
	default:
		return fmt.Errorf("unknown value for mode query parameter: %v", mode)
	}

	out, err := yaml.Marshal(output)
	if err != nil {
		return err
	}

	_, err = w.Write([]byte("---\n"))
	if err != nil {
		return err
	}

	_, err = w.Write(out)
	return err
}

func (h *Handler) writeStatusEndpoints(w io.Writer) error {
	type endpoint struct {
		name  string
		regex string
	}

	endpoints := []endpoint{}

	err := h.router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		e := endpoint{}

		pathTemplate, err := route.GetPathTemplate()
		if err == nil {
			e.name = pathTemplate
		}

		pathRegexp, err := route.GetPathRegexp()
		if err == nil {
			e.regex = pathRegexp
		}

		endpoints = append(endpoints, e)

		return nil
	})
	if err != nil {
		return fmt.Errorf("error walking routes: %w", err)
	}

	sort.Slice(endpoints[:], func(i, j int) bool {
		return endpoints[i].name < endpoints[j].name
	})

	x := table.NewWriter()
	x.SetOutputMirror(w)
	x.AppendHeader(table.Row{"name", "regex"})

	for _, e := range endpoints {
		x.AppendRows([]table.Row{
			{e.name, e.regex},
		})
	}

	x.AppendSeparator()
	x.Render()

	_, err = w.Write([]byte(fmt.Sprintf("\nAPI documentation: %s\n\n", apiDocs)))
	if err != nil {
		return fmt.Errorf("error writing status endpoints: %w", err)
	}

	return nil
}

// BuildInfoHandler returns build information.
func (h *Handler) BuildInfoHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(build.GetVersion())

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		level.Error(h.logger).Log("msg", "error writing response", "err", err)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// InstancesHandler returns the list of configured Tempo instances.
func (h *Handler) InstancesHandler(w http.ResponseWriter, _ *http.Request) {
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
