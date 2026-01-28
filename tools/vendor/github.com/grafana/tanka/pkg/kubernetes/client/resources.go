package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/grafana/tanka/pkg/kubernetes/manifest"
)

// Resources the Kubernetes API knows
type Resources []Resource

// Namespaced returns whether a resource is namespace-specific or cluster-wide
func (r Resources) Namespaced(m manifest.Manifest) bool {
	for _, res := range r {
		if m.Kind() == res.Kind {
			return res.Namespaced
		}
	}

	return false
}

// Resource is a Kubernetes API Resource
type Resource struct {
	APIGroup   string `json:"APIGROUP"`
	APIVersion string `json:"APIVERSION"`
	Kind       string `json:"KIND"`
	Name       string `json:"NAME"`
	Namespaced bool   `json:"NAMESPACED,string"`
	Shortnames string `json:"SHORTNAMES"`
	Verbs      string `json:"VERBS"`
	Categories string `json:"CATEGORIES"`
}

func (r Resource) FQN() string {
	apiGroup := ""
	if r.APIGroup != "" {
		// this is only set in kubectl v1.18 and earlier
		apiGroup = r.APIGroup
	} else if pos := strings.Index(r.APIVersion, "/"); pos > 0 {
		apiGroup = r.APIVersion[0:pos]
	}
	return strings.TrimSuffix(r.Name+"."+apiGroup, ".")
}

// Resources returns all API resources known to the server
func (k Kubectl) Resources() (Resources, error) {
	cmd := k.ctl("api-resources", "--cached", "--output=wide")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	var res Resources
	if err := UnmarshalTable(out.String(), &res); err != nil {
		return nil, errors.Wrap(err, "parsing table")
	}

	return res, nil
}

// UnmarshalTable unmarshals a raw CLI table into ptr. `json` package is used
// for getting the dat into the ptr, `json:` struct tags can be used.
func UnmarshalTable(raw string, ptr interface{}) error {
	raw = strings.TrimSpace(raw)

	lines := strings.Split(raw, "\n")
	if len(lines) == 0 {
		return ErrorNoHeader
	}

	headerStr := lines[0]
	// headers are ALL-CAPS
	if !regexp.MustCompile(`^[A-Z\s]+$`).MatchString(headerStr) {
		return ErrorNoHeader
	}

	lines = lines[1:]

	spc := regexp.MustCompile(`[A-Z]+\s*`)
	header := spc.FindAllString(headerStr, -1)

	tbl := make([]map[string]string, 0, len(lines))
	for _, l := range lines {
		elems := splitRow(l, header)
		if len(elems) != len(header) {
			return ErrorElementsMismatch{Header: len(header), Row: len(elems)}
		}

		row := make(map[string]string)
		for i, e := range elems {
			key := strings.TrimSpace(header[i])
			row[key] = strings.TrimSpace(e)
		}
		tbl = append(tbl, row)
	}

	j, err := json.Marshal(tbl)
	if err != nil {
		return err
	}

	return json.Unmarshal(j, ptr)
}

// ErrorNoHeader occurs when the table lacks an ALL-CAPS header
var ErrorNoHeader = fmt.Errorf("table has no header")

// ErrorElementsMismatch occurs when a row has a different count of elements
// than it's header
type ErrorElementsMismatch struct {
	Header, Row int
}

func (e ErrorElementsMismatch) Error() string {
	return fmt.Sprintf("header and row have different element count: %v != %v", e.Header, e.Row)
}

func splitRow(s string, header []string) (elems []string) {
	pos := 0
	for i, h := range header {
		if i == len(header)-1 {
			elems = append(elems, s[pos:])
			continue
		}

		lim := len(h)
		endPos := pos + lim
		if endPos >= len(s) {
			endPos = len(s)
		}

		elems = append(elems, s[pos:endPos])
		pos = endPos
	}
	return elems
}
