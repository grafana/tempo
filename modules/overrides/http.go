package overrides

import (
	"context"
	"net/http"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/util/log"
	jsoniter "github.com/json-iterator/go"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/weaveworks/common/user"
)

// OverridesHandler is a http.HandlerFunc to return user configured overrides
func (o *userConfigOverridesManager) OverridesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span, ctx := opentracing.StartSpanFromContext(ctx, "userConfigOverridesManager.OverridesHandler")
	defer span.Finish()

	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		handleError(r, w, userID, http.StatusBadRequest, errors.Wrap(err, "failed to find org id in request"))
		return
	}
	level.Info(log.Logger).Log("tenant", userID, "method", r.Method, "url", r.URL.RequestURI())

	switch r.Method {
	case http.MethodGet:
		o.handleGet(w, r, ctx, userID)
	case http.MethodPost:
		o.handlePost(w, r, ctx, userID)
	default:
		handleError(r, w, userID, http.StatusBadRequest, errors.New("Only GET and POST is allowed"))
		return
	}
}

func (o *userConfigOverridesManager) handleGet(w http.ResponseWriter, r *http.Request, _ context.Context, userID string) {
	ucl, err := o.getLimits(userID)
	if err != nil {
		handleError(r, w, userID, http.StatusBadRequest, err)
		return
	}

	data, err := jsoniter.Marshal(ucl)
	if err != nil {
		handleError(r, w, userID, http.StatusBadRequest, err)
		return
	}

	w.Header().Set(api.HeaderContentType, api.HeaderAcceptJSON)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
	return
}

// handlePost accepts post requests with json payload and writes it to config backend
func (o *userConfigOverridesManager) handlePost(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	d := jsoniter.NewDecoder(r.Body)
	// error in case of unwanted fields
	d.DisallowUnknownFields()

	ucl := &UserConfigurableLimits{}

	err := d.Decode(&ucl)
	if err != nil {
		// bad JSON or unrecognized json field
		handleError(r, w, userID, http.StatusBadRequest, errors.Wrap(err, "bad json or missing required fields in payload"))
		return
	}

	// check for extra data
	if d.More() {
		handleError(r, w, userID, http.StatusBadRequest, errors.Wrap(err, "extraneous data in payload"))
		return
	}

	err = o.setLimits(ctx, userID, ucl)
	if err != nil {
		handleError(r, w, userID, http.StatusBadRequest, errors.Wrap(err, "failed to set user config limits"))
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set(api.HeaderContentType, "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok"))
	return
}

func (o *runtimeConfigOverridesManager) OverridesHandler(w http.ResponseWriter, r *http.Request) {
	span, _ := opentracing.StartSpanFromContext(r.Context(), "runtimeConfigOverridesManager.OverridesHandler")
	defer span.Finish()

	http.Error(w, "user configured overrides are not enabled", http.StatusBadRequest)
}

func handleError(r *http.Request, w http.ResponseWriter, userID string, status int, err error) {
	level.Error(log.Logger).Log("tenant", userID, "method", r.Method, "status", status, "url", r.URL.RequestURI(), "err", err.Error())
	http.Error(w, err.Error(), status)
}
