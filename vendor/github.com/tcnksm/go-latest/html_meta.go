package latest

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// MetaTagName is common HTML meta tag name which is defined on https://github.com/tcnksm/go-latest/blob/master/doc/html_meta.md
const MetaTagName = "go-latest"

// HTMLMeta is used to fetch a single HTML page and extract version information from
// specific meta tag. See meta tag specification that HTMLMeta tries to extract on https://github.com/tcnksm/go-latest/blob/master/doc/html_meta.md
type HTMLMeta struct {
	// URL is HTML page URL which include version information.
	URL string

	// Name is tool name which you want to check. This name must be
	// written in HTML meta tag content field. HTMLMeta use this to
	// extract version information.
	Name string
}

func (hm *HTMLMeta) newHTML() *HTML {
	return &HTML{
		URL:   hm.URL,
		Scrap: &metaTagScrap{Name: hm.Name},
	}
}

func (hm *HTMLMeta) Validate() error {
	return hm.newHTML().Validate()
}

func (hm *HTMLMeta) Fetch() (*FetchResponse, error) {
	return hm.newHTML().Fetch()
}

type metaTagScrap struct {
	Name string
}

type tagInside struct {
	name    string
	prefix  string
	version string
	meta    *Meta
}

func (mt *metaTagScrap) Exec(r io.Reader) ([]string, *Meta, error) {

	z := html.NewTokenizer(r)

	for {
		switch z.Next() {
		case html.ErrorToken:
			return []string{}, &Meta{}, fmt.Errorf("meta tag for %s is not found", mt.Name)

		case html.StartTagToken, html.SelfClosingTagToken:
			tok := z.Token()
			if tok.DataAtom == atom.Meta {
				product, version, message := attrAnalizer(tok.Attr)
				// Return first founded version.
				// Assumes that mata tag exist only one for each product
				if product == mt.Name {
					return []string{version}, &Meta{Message: message}, nil
				}
			}
		}
	}
}

func attrAnalizer(attrs []html.Attribute) (product, version, message string) {

	for _, a := range attrs {

		if a.Namespace != "" {
			continue
		}

		switch a.Key {
		case "name":
			if a.Val != MetaTagName {
				break
			}

		case "content":
			parts := strings.SplitN(strings.TrimSpace(a.Val), " ", 3)
			if len(parts) < 2 {
				break
			}

			product = parts[0]
			version = parts[1]

			// message is optional
			if len(parts) == 3 {
				message = parts[2]
			}
		}
	}

	return
}
