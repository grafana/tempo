package overrides

import (
	"context"
	"net/http"

	"github.com/go-kit/log/level"
	jsoniter "github.com/json-iterator/go"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/weaveworks/common/user"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/util/log"
)

// OverridesHandler is a http.HandlerFunc to return user configured overrides
func (o *userConfigOverridesManager) OverridesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span, ctx := opentracing.StartSpanFromContext(ctx, "userConfigOverridesManager.OverridesHandler")
	defer span.Finish()

	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		handleError(r, w, userID, errors.Wrap(err, "failed to find org id in request"))
		return
	}
	level.Info(log.Logger).Log("tenant", userID, "method", r.Method, "url", r.URL.RequestURI())

	switch r.Method {
	case http.MethodGet:
		o.handleGet(ctx, r, w, userID)
	case http.MethodPost:
		o.handlePost(ctx, r, w, userID)
	case http.MethodDelete:
		o.handleDelete(ctx, r, w, userID)
	default:
		handleError(r, w, userID, errors.New("Only GET, POST and DELETE is allowed"))
	}
}

func (o *userConfigOverridesManager) handleGet(ctx context.Context, r *http.Request, w http.ResponseWriter, userID string) {
	ucl, err := o.getTenantLimits(ctx, userID)
	if err != nil {
		handleError(r, w, userID, err)
		return
	}

	// TODO when not set, should we return 404 or just an empty json?

	data, err := jsoniter.Marshal(ucl)
	if err != nil {
		handleError(r, w, userID, err)
		return
	}

	w.Header().Set(api.HeaderContentType, api.HeaderAcceptJSON)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// handlePost accepts post requests with json payload and writes it to config backend
func (o *userConfigOverridesManager) handlePost(ctx context.Context, r *http.Request, w http.ResponseWriter, userID string) {
	d := jsoniter.NewDecoder(r.Body)
	// error in case of unwanted fields
	d.DisallowUnknownFields()

	ucl := &UserConfigurableLimits{}

	err := d.Decode(&ucl)
	if err != nil {
		// bad JSON or unrecognized json field
		handleError(r, w, userID, errors.Wrap(err, "bad json or missing required fields in payload"))
		return
	}

	// check for extra data
	if d.More() {
		handleError(r, w, userID, errors.Wrap(err, "extraneous data in payload"))
		return
	}

	err = o.setTenantLimits(ctx, userID, ucl)
	if err != nil {
		handleError(r, w, userID, errors.Wrap(err, "failed to set user config limits"))
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set(api.HeaderContentType, "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok"))
}

func (o *userConfigOverridesManager) handleDelete(ctx context.Context, _ *http.Request, w http.ResponseWriter, userID string) {
	err := o.deleteTenantLimits(ctx, userID)
	if err != nil {
		handleError(nil, w, userID, errors.Wrap(err, "failed to set user config limits"))
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set(api.HeaderContentType, "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok"))
}

func (o *runtimeConfigOverridesManager) OverridesHandler(w http.ResponseWriter, r *http.Request) {
	span, _ := opentracing.StartSpanFromContext(r.Context(), "runtimeConfigOverridesManager.OverridesHandler")
	defer span.Finish()

	http.Error(w, "user configured overrides are not enabled", http.StatusBadRequest)
}

func handleError(r *http.Request, w http.ResponseWriter, userID string, err error) {
	status := http.StatusBadRequest
	level.Error(log.Logger).Log("tenant", userID, "method", r.Method, "status", status, "url", r.URL.RequestURI(), "err", err.Error())
	http.Error(w, err.Error(), status)
}
