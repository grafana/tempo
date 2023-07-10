package overrides

import (
	"net/http"

	"github.com/go-kit/log/level"
	jsoniter "github.com/json-iterator/go"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	ot_log "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/weaveworks/common/user"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/util/log"
)

// GetOverridesHandler is a http.HandlerFunc that returns the user-configured overrides.
func (o *userConfigOverridesManager) GetOverridesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span, ctx := opentracing.StartSpanFromContext(ctx, "userConfigOverridesManager.GetOverridesHandler")
	defer span.Finish()

	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		handleError(span, userID, r, w, http.StatusBadRequest, err)
		return
	}
	logRequest(userID, r)

	limits, err := o.getTenantLimits(ctx, userID)
	if err != nil {
		handleError(span, userID, r, w, http.StatusInternalServerError, err)
		return
	}

	data, err := jsoniter.Marshal(limits)
	if err != nil {
		handleError(span, userID, r, w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set(api.HeaderContentType, api.HeaderAcceptJSON)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// PostOverridesHandler accepts post requests with json payload and writes it to config backend.
func (o *userConfigOverridesManager) PostOverridesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span, ctx := opentracing.StartSpanFromContext(ctx, "userConfigOverridesManager.PostOverridesHandler")
	defer span.Finish()

	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		handleError(span, userID, r, w, http.StatusBadRequest, err)
		return
	}
	logRequest(userID, r)

	d := jsoniter.NewDecoder(r.Body)
	// error in case of unwanted fields
	d.DisallowUnknownFields()

	ucl := &UserConfigurableLimits{}

	err = d.Decode(&ucl)
	if err != nil {
		// bad JSON or unrecognized json field
		handleError(span, userID, r, w, http.StatusInternalServerError, errors.Wrap(err, "bad json or missing required fields in payload"))
		return
	}

	// check for extra data
	if d.More() {
		handleError(span, userID, r, w, http.StatusInternalServerError, errors.Wrap(err, "extraneous data in payload"))
		return
	}

	err = o.setTenantLimits(ctx, userID, ucl)
	if err != nil {
		handleError(span, userID, r, w, http.StatusInternalServerError, errors.Wrap(err, "failed to set user config limits"))
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set(api.HeaderContentType, "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok"))
}

func (o *userConfigOverridesManager) DeleteOverridesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span, ctx := opentracing.StartSpanFromContext(ctx, "userConfigOverridesManager.DeleteOverridesHandler")
	defer span.Finish()

	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		handleError(span, userID, r, w, http.StatusBadRequest, err)
		return
	}
	logRequest(userID, r)

	err = o.deleteTenantLimits(ctx, userID)
	if err != nil {
		handleError(span, userID, nil, w, http.StatusInternalServerError, errors.Wrap(err, "failed to set user config limits"))
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set(api.HeaderContentType, "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok"))
}

func (o *runtimeConfigOverridesManager) GetOverridesHandler(w http.ResponseWriter, r *http.Request) {
	span, _ := opentracing.StartSpanFromContext(r.Context(), "runtimeConfigOverridesManager.GetOverridesHandler")
	defer span.Finish()

	http.Error(w, "user configured overrides are not enabled", http.StatusMethodNotAllowed)
}

func (o *runtimeConfigOverridesManager) PostOverridesHandler(w http.ResponseWriter, r *http.Request) {
	span, _ := opentracing.StartSpanFromContext(r.Context(), "runtimeConfigOverridesManager.PostOverridesHandler")
	defer span.Finish()

	http.Error(w, "user configured overrides are not enabled", http.StatusMethodNotAllowed)
}

func (o *runtimeConfigOverridesManager) DeleteOverridesHandler(w http.ResponseWriter, r *http.Request) {
	span, _ := opentracing.StartSpanFromContext(r.Context(), "runtimeConfigOverridesManager.DeleteOverridesHandler")
	defer span.Finish()

	http.Error(w, "user configured overrides are not enabled", http.StatusMethodNotAllowed)
}

func logRequest(userID string, r *http.Request) {
	level.Info(log.Logger).Log("tenant", userID, "method", r.Method, "url", r.URL.RequestURI())
}

func handleError(span opentracing.Span, userID string, r *http.Request, w http.ResponseWriter, statusCode int, err error) {
	level.Error(log.Logger).Log("tenant", userID, "method", r.Method, "status", statusCode, "url", r.URL.RequestURI(), "err", err.Error())

	span.LogFields(ot_log.Error(err))
	ext.Error.Set(span, true)
	span.LogKV("status", statusCode)

	http.Error(w, err.Error(), statusCode)
}
