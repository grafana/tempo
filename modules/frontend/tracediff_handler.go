package frontend

import (
	"io"
	"net/http"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level" //nolint:all //deprecated
	"github.com/grafana/tempo/pkg/api"
)

const traceDiffNotImplementedMessage = "trace diff endpoint is not implemented"

// newTraceDiffHandler creates an HTTP handler skeleton for trace diff requests.
//
// EXPERIMENTAL: this endpoint is not yet a stable API contract. Diff execution
// will be added in a follow-up change.
func newTraceDiffHandler(logger log.Logger) http.RoundTripper {
	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			return &http.Response{
				StatusCode: http.StatusMethodNotAllowed,
				Status:     http.StatusText(http.StatusMethodNotAllowed),
				Body:       io.NopCloser(strings.NewReader(http.StatusText(http.StatusMethodNotAllowed))),
			}, nil
		}

		tenant, errResp := extractTenant(req, logger)
		if errResp != nil {
			return errResp, nil
		}

		if _, err := api.ParseTraceDiffRequest(req); err != nil {
			return httpInvalidRequest(err), nil
		}

		level.Info(logger).Log(
			"msg", "trace diff request",
			"tenant", tenant,
			"path", req.URL.Path)

		return &http.Response{
			StatusCode: http.StatusNotImplemented,
			Status:     http.StatusText(http.StatusNotImplemented),
			Body:       io.NopCloser(strings.NewReader(traceDiffNotImplementedMessage)),
		}, nil
	})
}
