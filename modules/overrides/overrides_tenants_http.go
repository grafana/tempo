package overrides

import (
	_ "embed" // Used to embed html templates
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/grafana/tempo/pkg/util"
)

//go:embed tenants.gohtml
var tenantsPageHTML string
var tenantsTemplate = template.Must(template.New("webpage").Parse(tenantsPageHTML))

type tenantsPageContents struct {
	Now     time.Time           `json:"now"`
	Tenants []tenantsPageTenant `json:"tenants,omitempty"`
}

type tenantsPageTenant struct {
	Name                         string `json:"name"`
	HasRuntimeOverrides          bool   `json:"has_runtime_overrides"`
	HasUserConfigurableOverrides bool   `json:"has_user_configurable_overrides"`
}

func (o *runtimeConfigOverridesManager) TenantsHandler(w http.ResponseWriter, req *http.Request) {
	var tenants []tenantsPageTenant
	for _, tenant := range o.GetTenantIDs() {
		tenants = append(tenants, tenantsPageTenant{
			Name:                         tenant,
			HasRuntimeOverrides:          true,
			HasUserConfigurableOverrides: false,
		})
	}

	sortTenantsPageTenant(tenants)

	util.RenderHTTPResponse(w, tenantsPageContents{
		Now:     time.Now(),
		Tenants: tenants,
	}, tenantsTemplate, req)
}

func (o *userConfigurableOverridesManager) TenantsHandler(w http.ResponseWriter, req *http.Request) {
	tenants := make(map[string]tenantsPageTenant)

	// runtime overrides
	for _, tenant := range o.Interface.GetTenantIDs() {
		tenants[tenant] = tenantsPageTenant{
			Name:                         tenant,
			HasRuntimeOverrides:          true,
			HasUserConfigurableOverrides: false,
		}
	}

	// user-configurable overrides
	for _, tenant := range o.GetTenantIDs() {
		_, hasRuntimeOverrides := tenants[tenant]

		tenants[tenant] = tenantsPageTenant{
			Name:                         tenant,
			HasRuntimeOverrides:          hasRuntimeOverrides,
			HasUserConfigurableOverrides: true,
		}
	}

	var tenantsList []tenantsPageTenant
	for _, tenant := range tenants {
		tenantsList = append(tenantsList, tenant)
	}

	sortTenantsPageTenant(tenantsList)

	util.RenderHTTPResponse(w, tenantsPageContents{
		Now:     time.Now(),
		Tenants: tenantsList,
	}, tenantsTemplate, req)
}

func sortTenantsPageTenant(list []tenantsPageTenant) {
	sort.Slice(list, func(i, j int) bool {
		return strings.Compare(list[i].Name, list[j].Name) < 0
	})
}
