package overrides

import (
	"context"
	"net/http"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"github.com/weaveworks/common/user"
)

// Maybe move this handler method into Interface

func (o *UserConfigOverridesManager) OverridesHandler(w http.ResponseWriter, r *http.Request) {
	// FIXME: log the request??
	// FIXME: maybe write a RoundTripper?? and follow the example of what we have done in other components??
	// We implement RoundTripper in other components??

	ctx := r.Context()
	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		http.Error(w, errors.Wrap(err, "failed to find org id in request").Error(), http.StatusBadRequest)
	}

	switch r.Method {
	case "GET":
		o.handleGet(w, r, ctx, userID)
	case "POST":
		o.handlePost(w, r, ctx, userID)
	default:
		http.Error(w, "Only GET and POST methods are supported", http.StatusBadRequest)
	}
}

func (o *UserConfigOverridesManager) handleGet(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	// FIXME: return json instead of yaml??
	// only return UserConfigurableLimits??
	// using WriteStatusRuntimeConfig at the moment??
	err := o.WriteStatusRuntimeConfig(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")

	return
}

// handlePost accepts post requests with json payload and writes it to config backend
func (o *UserConfigOverridesManager) handlePost(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	d := jsoniter.NewDecoder(r.Body)
	// TODO: do we want to this strict?? maybe we throw away extra fields??
	// error in case of unwanted fields
	d.DisallowUnknownFields()

	ucl := &UserConfigurableLimits{}

	err := d.Decode(&ucl)
	if err != nil {
		// bad JSON or unrecognized json field
		http.Error(w, errors.Wrap(err, "bad json or missing required fields in payload").Error(), http.StatusBadRequest)
		return
	}

	// check for extra data
	if d.More() {
		http.Error(w, errors.Wrap(err, "extraneous data in payload").Error(), http.StatusBadRequest)
		return
	}

	err = o.setLimits(ctx, userID, ucl)
	if err != nil {
		http.Error(w, "failed to set user config limits", http.StatusBadRequest)
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("ok"))
	if err != nil {
		http.Error(w, "something went wrong", http.StatusBadRequest)
		return
	}

	return
}

func (o *runtimeConfigOverridesManager) OverridesHandler(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "user configured overrides are not enabled", http.StatusBadRequest)
}
