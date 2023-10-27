package combiner

import (
	"net/http"

	"github.com/grafana/tempo/pkg/tempopb"
)

// Combiner is used to merge multiple responses into a single response.
//
// Implementations must be thread-safe.
type Combiner interface {
	// TODO: The callback is a hacky way of injecting the tenant label in tenant federation.
	//  We should figure out a better way to do this.
	//	FIXME: remove cb and just inject tenant label in Combiner impl.
	AddRequest(r *http.Response, cb func(t *tempopb.Trace)) error
	Complete() (*http.Response, error)
	StatusCode() int

	ShouldQuit() bool
}
