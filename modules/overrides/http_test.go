package overrides

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/util"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weaveworks/common/user"
	"golang.org/x/net/context"
)

func Test_runtimeConfigOverridesManager_OverridesHandler(t *testing.T) {
	o, err := newRuntimeConfigOverrides(Limits{})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	o.OverridesHandler(w, req)

	data := w.Body.String()
	require.Equal(t, "user configured overrides are not enabled\n", data)

	res := w.Result()
	require.Equal(t, "text/plain; charset=utf-8", w.Header().Get(api.HeaderContentType))
	require.Equal(t, 400, res.StatusCode)

}

func Test_userConfigOverridesManager_OverridesHandler(t *testing.T) {
	tempDir := t.TempDir()
	bl := Limits{Forwarders: []string{"my-forwarder"}}
	configurableOverrides := localUserConfigOverrides(t, tempDir, bl)

	err := configurableOverrides.setLimits(context.Background(), "single-tenant", &UserConfigurableLimits{
		Version:    "v1",
		Forwarders: &[]string{"my-other-forwarder"},
	})
	require.NoError(t, err)

	postJson, err := jsoniter.Marshal(&UserConfigurableLimits{
		Version:    "v1",
		Forwarders: &[]string{"my-updated-forwarder"},
	})
	require.NoError(t, err)

	tests := []struct {
		name           string
		overrides      *userConfigOverridesManager
		req            *http.Request
		expResp        string
		expContentType string
		expStatusCode  int
	}{
		{
			name:           "test GET",
			overrides:      configurableOverrides,
			req:            httptest.NewRequest("GET", "/", nil),
			expResp:        "{\"version\":\"v1\",\"forwarders\":[\"my-other-forwarder\"]}",
			expContentType: api.HeaderAcceptJSON,
			expStatusCode:  200,
		},
		{
			name:           "test POST",
			overrides:      configurableOverrides,
			req:            httptest.NewRequest("POST", "/", bytes.NewReader(postJson)),
			expResp:        "ok",
			expContentType: "text/plain; charset=utf-8",
			expStatusCode:  200,
		},
		{
			name:           "test PUT",
			overrides:      configurableOverrides,
			req:            httptest.NewRequest("PUT", "/", nil),
			expResp:        "Only GET and POST is allowed\n",
			expContentType: "text/plain; charset=utf-8",
			expStatusCode:  400,
		},
		{
			name:           "test DELETE",
			overrides:      configurableOverrides,
			req:            httptest.NewRequest("DELETE", "/", nil),
			expResp:        "Only GET and POST is allowed\n",
			expContentType: "text/plain; charset=utf-8",
			expStatusCode:  400,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// inject "single-tenant" in tc.req
			ctx := user.InjectOrgID(tc.req.Context(), util.FakeTenantID)
			tc.req = tc.req.WithContext(ctx)

			w := httptest.NewRecorder()
			tc.overrides.OverridesHandler(w, tc.req)

			data := w.Body.String()
			require.Equal(t, tc.expResp, data)

			res := w.Result()
			require.Equal(t, tc.expContentType, w.Header().Get(api.HeaderContentType))
			require.Equal(t, tc.expStatusCode, res.StatusCode)

			if tc.req.Method == http.MethodPost {
				require.Contains(t, tc.overrides.Forwarders("single-tenant"), "my-updated-forwarder")
			}
		})
	}
}
