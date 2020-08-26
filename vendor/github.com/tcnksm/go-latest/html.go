package latest

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"

	"github.com/hashicorp/go-version"
)

// HTML is used to fetch version information from a single HTML page.
type HTML struct {
	// URL is HTML page URL which include version information.
	URL string

	// Scrap is used to scrap a single HTML page and extract version information.
	// See more about HTMLScrap interface.
	// By default, it does nothing, just return HTML contents.
	Scrap HTMLScrap
}

// HTMLScrap is used to scrap a single HTML page and extract version information.
type HTMLScrap interface {
	// Exec is called from Fetch after fetching a HTMl page from source.
	// It must return version information as string list format.
	Exec(r io.Reader) ([]string, *Meta, error)
}

type defaultHTMLScrap struct{}

func (s *defaultHTMLScrap) Exec(r io.Reader) ([]string, *Meta, error) {
	meta := &Meta{}
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return []string{}, meta, err
	}

	b = bytes.Replace(b, []byte("\n"), []byte(""), -1)
	return []string{string(b[:])}, meta, nil
}

func (h *HTML) scrap() HTMLScrap {
	if h.Scrap == nil {
		return &defaultHTMLScrap{}
	}

	return h.Scrap
}

func (h *HTML) Validate() error {

	if len(h.URL) == 0 {
		return fmt.Errorf("URL must be set")
	}

	// Check URL can be parsed
	if _, err := url.Parse(h.URL); err != nil {
		return fmt.Errorf("%s is invalid URL: %s", h.URL, err.Error())
	}

	return nil
}

func (h *HTML) Fetch() (*FetchResponse, error) {

	fr := newFetchResponse()

	// URL is validated before call
	u, _ := url.Parse(h.URL)

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

	scrap := h.scrap()
	verStrs, meta, err := scrap.Exec(resp.Body)
	if err != nil {
		return fr, err
	}

	if len(verStrs) == 0 {
		return fr, fmt.Errorf("version info is not found on %s", h.URL)
	}

	for _, verStr := range verStrs {
		v, err := version.NewVersion(verStr)
		if err != nil {
			fr.Malformeds = append(fr.Malformeds, verStr)
			continue
		}
		fr.Versions = append(fr.Versions, v)
	}

	fr.Meta = meta

	return fr, nil
}
