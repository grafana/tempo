package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/grafana/e2e"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/grafana/tempo/cmd/tempo/app"
	util "github.com/grafana/tempo/integration"
	"github.com/grafana/tempo/integration/e2e/backend"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/pkg/api"
	tempoUtil "github.com/grafana/tempo/pkg/util"
)

func TestOverrides(t *testing.T) {
	testBackends := []struct {
		name       string
		configFile string
	}{
		{
			name:       "local",
			configFile: configAllInOneLocal,
		},
		{
			name:       "s3",
			configFile: configAllInOneS3,
		},
		{
			name:       "azure",
			configFile: configAllInOneAzurite,
		},
		{
			name:       "gcs",
			configFile: configAllInOneGCS,
		},
	}
	for _, tc := range testBackends {
		t.Run(tc.name, func(t *testing.T) {
			s, err := e2e.NewScenario("tempo_e2e")
			require.NoError(t, err)
			defer s.Close()

			// set up the backend
			cfg := app.Config{}
			buff, err := os.ReadFile(tc.configFile)
			require.NoError(t, err)
			err = yaml.UnmarshalStrict(buff, &cfg)
			require.NoError(t, err)
			_, err = backend.New(s, cfg)
			require.NoError(t, err)

			require.NoError(t, util.CopyFileToSharedDir(s, tc.configFile, "config.yaml"))
			tempo := util.NewTempoAllInOne()
			require.NoError(t, s.StartAndWaitReady(tempo))

			orgID := ""
			apiClient := Client{tempoUtil.NewClient("http://"+tempo.Endpoint(3200), orgID)}

			// Get default overrides
			limits, err := apiClient.GetOverrides()
			fmt.Printf("* Overrides: %+v\n", limits)
			if limits.Forwarders != nil {
				fmt.Printf("*   Fowarders: %+v\n", *limits.Forwarders)
			}
			require.NoError(t, err)

			require.NotNil(t, limits)
			assert.Empty(t, nil, limits.Forwarders)

			// Modify overrides
			fmt.Println("* Setting overrides.forwarders")
			err = apiClient.SetOverrides(&overrides.UserConfigurableLimits{
				Forwarders: &[]string{"my-forwarder"},
			})
			require.NoError(t, err)

			limits, err = apiClient.GetOverrides()
			fmt.Printf("* Overrides: %+v\n", limits)
			if limits.Forwarders != nil {
				fmt.Printf("*   Fowarders: %+v\n", *limits.Forwarders)
			}
			require.NoError(t, err)

			require.NotNil(t, limits)
			require.NotNil(t, limits.Forwarders)
			assert.ElementsMatch(t, *limits.Forwarders, []string{"my-forwarder"})

			// Clear overrides
			fmt.Println("* Deleting overrides")
			err = apiClient.DeleteOverrides()
			require.NoError(t, err)

			limits, err = apiClient.GetOverrides()
			fmt.Printf("* Overrides: %+v\n", limits)
			if limits.Forwarders != nil {
				fmt.Printf("*   Fowarders: %+v\n", *limits.Forwarders)
			}
			require.NoError(t, err)

			require.NotNil(t, limits)
			assert.Empty(t, nil, limits.Forwarders)
		})
	}
}

// Client wraps tempoUtil.Client and adds functions to work with the overrides files.
// Adding this functions to tempoUtil would result in an import cycle.
type Client struct {
	*tempoUtil.Client
}

func (c *Client) GetOverrides() (*overrides.UserConfigurableLimits, error) {
	req, err := http.NewRequest("GET", c.BaseURL+api.PathOverrides, nil)
	if err != nil {
		return nil, err
	}

	if len(c.OrgID) > 0 {
		req.Header.Set(tempoUtil.OrgIDHeader, c.OrgID)
	}
	req.Header.Set(tempoUtil.AcceptHeader, tempoUtil.ApplicationJSON)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error querying Tempo %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 && resp.StatusCode <= 599 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GET request to %s failed with response: %d body: %s", req.URL.String(), resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	limits := &overrides.UserConfigurableLimits{}
	if err = json.Unmarshal(body, limits); err != nil {
		return nil, fmt.Errorf("error decoding overrides, err: %v body: %s", err, string(body))
	}
	return limits, err
}

func (c *Client) SetOverrides(limits *overrides.UserConfigurableLimits) error {
	b, err := json.Marshal(limits)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.BaseURL+api.PathOverrides, bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	if len(c.OrgID) > 0 {
		req.Header.Set(tempoUtil.OrgIDHeader, c.OrgID)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error querying Tempo %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 && resp.StatusCode <= 599 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GET request to %s failed with response: %d body: %s", req.URL.String(), resp.StatusCode, string(body))
	}
	return nil
}

func (c *Client) DeleteOverrides() error {
	req, err := http.NewRequest("DELETE", c.BaseURL+api.PathOverrides, nil)
	if err != nil {
		return err
	}

	if len(c.OrgID) > 0 {
		req.Header.Set(tempoUtil.OrgIDHeader, c.OrgID)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error querying Tempo %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 && resp.StatusCode <= 599 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GET request to %s failed with response: %d body: %s", req.URL.String(), resp.StatusCode, string(body))
	}
	return nil
}
