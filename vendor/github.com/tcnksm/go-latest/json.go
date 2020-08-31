package latest

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/go-version"
)

var (
	defaultDialTimeout = 5 * time.Second
)

// JSON is used to get version information as json format from remote host.
type JSON struct {
	// URL is URL which return json response with version information.
	URL string

	// Response is used to decode json as Struct and extract version information.
	// See JSONResponse interface. By Default, it is used defaultJSONResponse.
	Response JSONResponse
}

// JSONResponse is used to decode json as Struct and extract information.
type JSONResponse interface {
	// VersionInfo is called from Fetch to extract version info.
	// It must return Semantic Version format version string list.
	VersionInfo() ([]string, error)

	// MetaInfo is called from Fetch to extract meta info.
	MetaInfo() (*Meta, error)
}

type defaultJSONResponse struct {
	Version string `json:"version"`
	Message string `json:"message"`
	URL     string `json:"url"`
}

func (res *defaultJSONResponse) VersionInfo() ([]string, error) {
	return []string{res.Version}, nil
}

func (res *defaultJSONResponse) MetaInfo() (*Meta, error) {
	return &Meta{
		Message: res.Message,
		URL:     res.URL,
	}, nil
}

func (j *JSON) response() JSONResponse {
	if j.Response == nil {
		return &defaultJSONResponse{}
	}

	return j.Response
}

func (j *JSON) Validate() error {

	if len(j.URL) == 0 {
		return fmt.Errorf("URL must be set")
	}

	// Check URL can be parsed by net.URL
	if _, err := url.Parse(j.URL); err != nil {
		return fmt.Errorf("%s is invalid URL: %s", j.URL, err.Error())
	}

	return nil
}

func (j *JSON) Fetch() (*FetchResponse, error) {

	fr := newFetchResponse()

	// URL is validated before call
	u, _ := url.Parse(j.URL)

	// Create a new request
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return fr, err
	}
	req.Header.Add("Accept", "application/json")

	// Create client
	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: func(n, a string) (net.Conn, error) {
			return net.DialTimeout(n, a, defaultDialTimeout)
		},
	}

	client := &http.Client{
		Transport: t,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fr, err
	}

	if resp.StatusCode != 200 {
		return fr, fmt.Errorf("unknown status: %d", resp.StatusCode)
	}

	result := j.response()
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&result); err != nil {
		return fr, err
	}

	verStrs, err := result.VersionInfo()
	if err != nil {
		return fr, err
	}

	if len(verStrs) == 0 {
		return fr, fmt.Errorf("version info is not found on %s", j.URL)
	}

	for _, verStr := range verStrs {
		v, err := version.NewVersion(verStr)
		if err != nil {
			fr.Malformeds = append(fr.Malformeds, verStr)
		}
		fr.Versions = append(fr.Versions, v)
	}

	fr.Meta, err = result.MetaInfo()
	if err != nil {
		return fr, err
	}

	return fr, nil
}
