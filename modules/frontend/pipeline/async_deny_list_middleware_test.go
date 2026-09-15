package pipeline

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestURLBlackListMiddlewareForEmptyBlackList(t *testing.T) {
	regexes := []string{}
	roundTrip := NewURLDenyListWare(regexes).Wrap(GetRoundTripperFunc())
	statusCode := DoRequest(t, "http://localhost:8080/api/v2/traces/123345", roundTrip)
	assert.Equal(t, 200, statusCode)
}

func TestURLBlackListMiddlewarePanicsOnSyntacticallyIncorrectRegex(t *testing.T) {
	regexes := []string{"qr/^(.*\\.traces\\/[a-f0-9]{32}$/"}
	assert.Panics(t, func() {
		NewURLDenyListWare(regexes).Wrap(GetRoundTripperFunc())
	})
}

func TestURLBlackListMiddleware(t *testing.T) {
	regexes := []string{
		"^.*v2.*",
	}
	roundTrip := NewURLDenyListWare(regexes).Wrap(GetRoundTripperFunc())
	statusCode := DoRequest(t, "http://localhost:9000?param1=a&param2=b", roundTrip)
	assert.Equal(t, 200, statusCode)

	// Blacklisted url
	statusCode = DoRequest(t, "http://localhost:8080/api/v2/traces/123345", roundTrip)
	assert.Equal(t, 400, statusCode)
}

func TestURLDenyListMiddlewareEncodedQueries(t *testing.T) {
	denyList := []string{
		`[^(!|%21)](=|%3D)(%20|%2B|\s+|\+)*nil\b`,
	}
	queries := []struct {
		name       string
		query      string
		statusCode int
	}{
		{"denied selector", `{.a = nil}`, http.StatusBadRequest},
		{"denied without spaces", `{.a=nil}`, http.StatusBadRequest},
		{"denied with extra spaces", `{  .a  =  nil  }`, http.StatusBadRequest},
		{"denied other attribute", `{.b = nil}`, http.StatusBadRequest},
		{"allowed inequality", `{.a != nil}`, http.StatusOK},
		{"allowed inequality without spaces", `{.a!=nil}`, http.StatusOK},
		{"allowed inequality with extra spaces", `{  .a  !=  nil  }`, http.StatusOK},
		{"allowed string value", `{.a = "nil"}`, http.StatusOK},
		{"allowed other attribute inequality", `{.b != nil}`, http.StatusOK},
	}
	encodings := []struct {
		name   string
		encode func(string) string
	}{
		{
			name: "percent spaces",
			encode: func(query string) string {
				// Preserve the unescaped '!' in the requested URL form.
				return strings.NewReplacer("+", "%20", "%21", "!").Replace(url.QueryEscape(query))
			},
		},
		{
			name: "form spaces",
			encode: func(query string) string {
				return strings.ReplaceAll(url.QueryEscape(query), "%21", "!")
			},
		},
		{
			name: "encoded literal plus",
			encode: func(query string) string {
				// %2B is a literal plus, unlike '+' used to encode a space.
				query = strings.ReplaceAll(query, " ", "+")
				return strings.ReplaceAll(url.QueryEscape(query), "%21", "!")
			},
		},
		{
			name:   "Go query escape",
			encode: url.QueryEscape,
		},
	}

	for _, query := range queries {
		t.Run(query.name, func(t *testing.T) {
			for _, encoding := range encodings {
				t.Run(encoding.name, func(t *testing.T) {
					forwarded := false
					next := GetRoundTripperFuncWithAsserts(t, func(_ *testing.T, _ Request) {
						forwarded = true
					})
					roundTrip := NewURLDenyListWare(denyList).Wrap(next)
					requestURL := "http://localhost:8080/api/search?q=" + encoding.encode(query.query) + "&limit=20"
					assert.Equal(t, query.statusCode, DoRequest(t, requestURL, roundTrip), requestURL)
					assert.Equal(t, query.statusCode == http.StatusOK, forwarded, "denied queries must not reach the backend")
				})
			}
		})
	}
}
