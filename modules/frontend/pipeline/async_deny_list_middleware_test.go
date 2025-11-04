package pipeline

import (
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
