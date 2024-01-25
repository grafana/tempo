package overrides

import (
	_ "embed" // Used to embed html templates
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"

	"github.com/grafana/tempo/pkg/util"
)

//go:embed tenant_status.gohtml
var tenantStatusPageHTML string
var tenantStatusTemplate = template.Must(template.New("webpage").Parse(tenantStatusPageHTML))

type tenantStatusPageContents struct {
	Now                       time.Time `json:"now"`
	Tenant                    string    `json:"tenant"`
	RuntimeOverrides          string    `json:"runtime_overrides"`
	UserConfigurableOverrides string    `json:"user_configurable_overrides"`
}

func (o *runtimeConfigOverridesManager) TenantStatusHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	tenantID := vars["tenant"]
	if tenantID == "" {
		util.WriteTextResponse(w, "Tenant ID can't be empty")
		return
	}

	runtimeOverrides, err := marshalRuntimeOverrides(o, tenantID)
	if err != nil {
		util.WriteTextResponse(w, err.Error())
		return
	}

	util.RenderHTTPResponse(w, tenantStatusPageContents{
		Now:                       time.Now(),
		Tenant:                    tenantID,
		RuntimeOverrides:          runtimeOverrides,
		UserConfigurableOverrides: "User-configurable overrides not enabled",
	}, tenantStatusTemplate, req)
}

func (o *userConfigurableOverridesManager) TenantStatusHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	tenantID := vars["tenant"]
	if tenantID == "" {
		util.WriteTextResponse(w, "Tenant ID can't be empty")
		return
	}

	runtimeOverrides, err := marshalRuntimeOverrides(o, tenantID)
	if err != nil {
		util.WriteTextResponse(w, err.Error())
		return
	}

	var userConfigurableOverrides string

	overrides := o.getTenantLimits(tenantID)
	if overrides != nil {
		marshalledOverrides, err := yaml.Marshal(overrides)
		if err != nil {
			util.WriteTextResponse(w, fmt.Sprintf("Marshalling runtime overrides failed: %s", err))
		}
		userConfigurableOverrides = string(marshalledOverrides)
	} else {
		userConfigurableOverrides = "No user-configurable overrides set"
	}

	util.RenderHTTPResponse(w, tenantStatusPageContents{
		Now:                       time.Now(),
		Tenant:                    tenantID,
		RuntimeOverrides:          runtimeOverrides,
		UserConfigurableOverrides: userConfigurableOverrides,
	}, tenantStatusTemplate, req)
}

func marshalRuntimeOverrides(o Interface, tenantID string) (string, error) {
	overrides := o.GetRuntimeOverridesFor(tenantID)

	runtimeOverrides, err := yaml.Marshal(overrides)
	if err != nil {
		return "", fmt.Errorf("Marshalling runtime overrides failed: %w", err)
	}
	return string(runtimeOverrides), nil
}
