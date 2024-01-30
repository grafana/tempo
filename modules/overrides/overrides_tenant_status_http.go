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

func TenantStatusHandler(o Interface) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)

		tenantID := vars["tenant"]
		if tenantID == "" {
			util.WriteTextResponse(w, "Tenant ID can't be empty")
			return
		}

		// runtime overrides
		overrides := o.GetRuntimeOverridesFor(tenantID)
		runtimeOverrides, err := yaml.Marshal(overrides)
		if err != nil {
			util.WriteTextResponse(w, fmt.Sprintf("Marshalling runtime overrides failed: %s", err))
			return
		}

		// user-configurable overrides
		var userConfigurableOverrides string

		if userConfigOverridesManager, ok := o.(*userConfigurableOverridesManager); ok {
			overrides := userConfigOverridesManager.getTenantLimits(tenantID)
			if overrides != nil {
				marshalledOverrides, err := yaml.Marshal(overrides)
				if err != nil {
					util.WriteTextResponse(w, fmt.Sprintf("Marshalling user-configurable overrides failed: %s", err))
					return
				}
				userConfigurableOverrides = string(marshalledOverrides)
			} else {
				userConfigurableOverrides = "No user-configurable overrides set"
			}
		} else {
			userConfigurableOverrides = "User-configurable overrides are not enabled"
		}

		util.RenderHTTPResponse(w, tenantStatusPageContents{
			Now:                       time.Now(),
			Tenant:                    tenantID,
			RuntimeOverrides:          string(runtimeOverrides),
			UserConfigurableOverrides: userConfigurableOverrides,
		}, tenantStatusTemplate, req)
	}
}
