package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	api "github.com/grafana/tempo/modules/overrides/user_configurable_api"
	tempo_api "github.com/grafana/tempo/pkg/api"
	tempoUtil "github.com/grafana/tempo/pkg/util"
)

// Client wraps tempoUtil.Client and adds functions to work with the user-configurable overrides API.
// We can't add this functions to tempoUtil directly because this would result in an import cycle.
type Client struct {
	*tempoUtil.Client
}

func (c *Client) GetOverrides() (*api.UserConfigurableLimits, error) {
	req, err := http.NewRequest("GET", c.BaseURL+tempo_api.PathOverrides, nil)
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

	limits := &api.UserConfigurableLimits{}
	if err = json.Unmarshal(body, limits); err != nil {
		return nil, fmt.Errorf("error decoding overrides, err: %v body: %s", err, string(body))
	}
	return limits, err
}

func (c *Client) SetOverrides(limits *api.UserConfigurableLimits) error {
	b, err := json.Marshal(limits)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.BaseURL+tempo_api.PathOverrides, bytes.NewBuffer(b))
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
	req, err := http.NewRequest("DELETE", c.BaseURL+tempo_api.PathOverrides, nil)
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
