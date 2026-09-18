package overrides

import (
	_ "embed" // Used to embed html templates
	"fmt"
	"html/template"
	"net/http"
	"slices"
	"time"

	"github.com/gorilla/mux"
	"go.yaml.in/yaml/v2"

	"github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	"github.com/grafana/tempo/pkg/secrets"
	"github.com/grafana/tempo/pkg/util"
)

//go:embed tenant_status.gohtml
var tenantStatusPageHTML string
var tenantStatusTemplate = template.Must(template.New("webpage").Parse(tenantStatusPageHTML))

type tenantStatusPageContents struct {
	Now                       time.Time `json:"now"`
	Tenant                    string    `json:"tenant"`
	UserConfigurableOverrides string    `json:"user_configurable_overrides"`
	RuntimeOverrides          string    `json:"runtime_overrides"`
	RuntimeOverridesSource    string    `json:"using_default_or_wildcard_runtime_overrides"`
}

func TenantStatusHandler(o Interface) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		page := tenantStatusPageContents{
			Now: time.Now(),
		}

		vars := mux.Vars(req)

		page.Tenant = vars["tenant"]
		if page.Tenant == "" {
			util.WriteTextResponse(w, "Tenant ID can't be empty")
			return
		}

		// runtime overrides
		overrides := statusOverrides(o.GetRuntimeOverridesFor(page.Tenant))
		runtimeOverrides, err := yaml.Marshal(overrides)
		if err != nil {
			util.WriteTextResponse(w, fmt.Sprintf("Marshalling runtime overrides failed: %s", err))
			return
		}
		page.RuntimeOverrides = string(runtimeOverrides)

		var runtimeTenants []string
		switch o := o.(type) {
		case *runtimeConfigOverridesManager:
			runtimeTenants = o.GetTenantIDs()
		case *userConfigurableOverridesManager:
			runtimeTenants = o.Interface.GetTenantIDs()
		default:
			util.WriteTextResponse(w, "Internal error happened when retrieving runtime overrides")
		}
		if slices.Contains(runtimeTenants, page.Tenant) {
			page.RuntimeOverridesSource = page.Tenant
		} else if slices.Contains(runtimeTenants, wildcardTenant) {
			page.RuntimeOverridesSource = wildcardTenant
		} else {
			page.RuntimeOverridesSource = "default overrides"
		}

		// user-configurable overrides
		if userConfigOverridesManager, ok := o.(*userConfigurableOverridesManager); ok {
			overrides := RedactUserLimits(userConfigOverridesManager.getTenantLimits(page.Tenant))
			if overrides != nil {
				marshalledOverrides, err := yaml.Marshal(overrides)
				if err != nil {
					util.WriteTextResponse(w, fmt.Sprintf("Marshalling user-configurable overrides failed: %s", err))
					return
				}
				page.UserConfigurableOverrides = string(marshalledOverrides)
			} else {
				page.UserConfigurableOverrides = "No user-configurable overrides set"
			}
		} else {
			page.UserConfigurableOverrides = "User-configurable overrides are not enabled"
		}

		util.RenderHTTPResponse(w, page, tenantStatusTemplate, req)
	}
}

// Status is a diagnostic surface, not a policy read API. Preserve configured
// presence but never expose tenant-authored policy text or mutate live limits.
func statusOverrides(limits *Overrides) *Overrides {
	if limits == nil {
		return nil
	}
	out := *limits
	if out.MetricsGenerator.Processor.SecretDetection != nil {
		out.MetricsGenerator.Processor.SecretDetection = &secrets.Policy{}
	}
	return &out
}

// RedactUserLimits copies limits for diagnostics without exposing policy text.
func RedactUserLimits(limits *client.Limits) *client.Limits {
	if limits == nil {
		return nil
	}
	out := *limits
	if out.MetricsGenerator.Processor.SecretDetection != nil {
		out.MetricsGenerator.Processor.SecretDetection = &secrets.Policy{}
	}
	return &out
}
