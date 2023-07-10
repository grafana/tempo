package overrides

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/require"
	"github.com/weaveworks/common/user"
	"golang.org/x/net/context"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/util"
)

func Test_runtimeConfigOverridesManager_returnsOverridesHandlerNotEnabledError(t *testing.T) {
	o, err := newRuntimeConfigOverrides(Limits{})
	require.NoError(t, err)

	tests := []struct {
		method  string
		handler http.HandlerFunc
	}{
		{http.MethodGet, o.GetOverridesHandler},
		{http.MethodPost, o.PostOverridesHandler},
		{http.MethodDelete, o.DeleteOverridesHandler},
	}
	for _, tc := range tests {
		t.Run(tc.method, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/", nil)

			tc.handler(w, req)

			data := w.Body.String()
			require.Equal(t, "user configured overrides are not enabled\n", data)

			res := w.Result()
			require.Equal(t, "text/plain; charset=utf-8", w.Header().Get(api.HeaderContentType))
			require.Equal(t, http.StatusMethodNotAllowed, res.StatusCode)
		})
	}
}

func Test_userConfigOverridesManager_overridesHandlers(t *testing.T) {
	bl := Limits{Forwarders: []string{"my-forwarder"}}
	_, configurableOverrides := localUserConfigOverrides(t, bl)

	err := configurableOverrides.setTenantLimits(context.Background(), "single-tenant", &UserConfigurableLimits{
		Version:    "v1",
		Forwarders: &[]string{"my-other-forwarder"},
	})
	require.NoError(t, err)

	postJSON, err := jsoniter.Marshal(&UserConfigurableLimits{
		Version:    "v1",
		Forwarders: &[]string{"my-updated-forwarder"},
	})
	require.NoError(t, err)

	tests := []struct {
		name           string
		handler        http.HandlerFunc
		req            *http.Request
		expResp        string
		expContentType string
		expStatusCode  int
	}{
		{
			name:           "test GET",
			handler:        configurableOverrides.GetOverridesHandler,
			req:            httptest.NewRequest("GET", "/", nil),
			expResp:        "{\"version\":\"v1\",\"forwarders\":[\"my-other-forwarder\"]}",
			expContentType: api.HeaderAcceptJSON,
			expStatusCode:  200,
		},
		{
			name:           "test POST",
			handler:        configurableOverrides.PostOverridesHandler,
			req:            httptest.NewRequest("POST", "/", bytes.NewReader(postJSON)),
			expResp:        "ok",
			expContentType: "text/plain; charset=utf-8",
			expStatusCode:  200,
		},
		{
			name:           "test DELETE",
			handler:        configurableOverrides.DeleteOverridesHandler,
			req:            httptest.NewRequest("DELETE", "/", nil),
			expResp:        "ok",
			expContentType: "text/plain; charset=utf-8",
			expStatusCode:  200,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// inject "single-tenant" in tc.req
			ctx := user.InjectOrgID(tc.req.Context(), util.FakeTenantID)
			tc.req = tc.req.WithContext(ctx)

			w := httptest.NewRecorder()
			tc.handler(w, tc.req)

			data := w.Body.String()
			require.Equal(t, tc.expResp, data)

			res := w.Result()
			require.Equal(t, tc.expContentType, w.Header().Get(api.HeaderContentType))
			require.Equal(t, tc.expStatusCode, res.StatusCode)

			if tc.req.Method == http.MethodPost {
				require.Contains(t, configurableOverrides.Forwarders("single-tenant"), "my-updated-forwarder")
			}
		})
	}
}
