package overrides

import (
	_ "embed" // Used to embed html templates
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"

	"golang.org/x/exp/maps"

	"github.com/grafana/tempo/pkg/util"
)

//go:embed tenants.gohtml
var tenantsPageHTML string
var tenantsTemplate = template.Must(template.New("webpage").Parse(tenantsPageHTML))

type tenantsPageContents struct {
	Now     time.Time            `json:"now"`
	Tenants []*tenantsPageTenant `json:"tenants,omitempty"`
}

type tenantsPageTenant struct {
	Name                         string `json:"name"`
	HasRuntimeOverrides          bool   `json:"has_runtime_overrides"`
	HasUserConfigurableOverrides bool   `json:"has_user_configurable_overrides"`
}

func TenantsHandler(o Interface) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		tenants := make(map[string]*tenantsPageTenant)

		// runtime overrides
		var runtimeTenants []string
		switch o := o.(type) {
		case *runtimeConfigOverridesManager:
			runtimeTenants = o.GetTenantIDs()
		case *userConfigurableOverridesManager:
			runtimeTenants = o.Interface.GetTenantIDs()
		default:
			util.WriteTextResponse(w, "Internal error happened when retrieving runtime overrides")
		}
		for _, tenant := range runtimeTenants {
			tenants[tenant] = &tenantsPageTenant{
				Name:                tenant,
				HasRuntimeOverrides: true,
			}
		}

		// user-configurable overrides
		userConfigurableOverridesManager, ok := o.(*userConfigurableOverridesManager)
		if ok {
			for _, tenant := range userConfigurableOverridesManager.GetTenantIDs() {
				tenantsPage := tenants[tenant]
				if tenantsPage == nil {
					tenantsPage = &tenantsPageTenant{Name: tenant}
					tenants[tenant] = tenantsPage
				}

				tenantsPage.HasUserConfigurableOverrides = true
			}
		}

		tenantsList := maps.Values(tenants)
		sortTenantsPageTenant(tenantsList)

		util.RenderHTTPResponse(w, tenantsPageContents{
			Now:     time.Now(),
			Tenants: tenantsList,
		}, tenantsTemplate, req)
	}
}

func sortTenantsPageTenant(list []*tenantsPageTenant) {
	sort.Slice(list, func(i, j int) bool {
		return strings.Compare(list[i].Name, list[j].Name) < 0
	})
}
