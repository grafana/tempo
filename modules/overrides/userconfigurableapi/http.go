package userconfigurableapi

import (
	"context"
	"net/http"

	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/services"
	jsoniter "github.com/json-iterator/go"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	ot_log "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/weaveworks/common/user"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/util/log"
)

type Validator func(limits *UserConfigurableLimits) error

// UserConfigOverridesAPI manages the API to retrieve, update and delete user-configurable overrides
// from the backend.
type UserConfigOverridesAPI struct {
	services.Service

	client    Client
	validator Validator
}

func NewUserConfigOverridesAPI(config *UserConfigurableOverridesClientConfig, validator Validator) (*UserConfigOverridesAPI, error) {
	client, err := NewUserConfigOverridesClient(config)
	if err != nil {
		return nil, err
	}

	api := &UserConfigOverridesAPI{
		client:    client,
		validator: validator,
	}

	api.Service = services.NewIdleService(api.starting, api.stopping)
	return api, nil
}

func (a *UserConfigOverridesAPI) starting(_ context.Context) error {
	return nil
}

func (a *UserConfigOverridesAPI) stopping(_ error) error {
	a.client.Shutdown()
	return nil
}

// GetOverridesHandler retrieves the user-configured overrides from the backend.
func (a *UserConfigOverridesAPI) GetOverridesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span, ctx := opentracing.StartSpanFromContext(ctx, "UserConfigOverridesAPI.GetOverridesHandler")
	defer span.Finish()

	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		handleError(span, userID, r, w, http.StatusBadRequest, err)
		return
	}
	logRequest(userID, r)

	limits, err := a.client.Get(ctx, userID)
	if err != nil {
		handleError(span, userID, r, w, http.StatusInternalServerError, err)
		return
	}

	if limits == nil {
		w.WriteHeader(http.StatusNotFound)
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
func (a *UserConfigOverridesAPI) PostOverridesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span, ctx := opentracing.StartSpanFromContext(ctx, "UserConfigOverridesAPI.PostOverridesHandler")
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

	limits := &UserConfigurableLimits{}

	err = d.Decode(&limits)
	if err != nil {
		// bad JSON or unrecognized json field
		handleError(span, userID, r, w, http.StatusBadRequest, err)
		return
	}

	err = a.validator(limits)
	if err != nil {
		handleError(span, userID, r, w, http.StatusBadRequest, err)
		return
	}

	err = a.client.Set(ctx, userID, limits)
	if err != nil {
		handleError(span, userID, r, w, http.StatusInternalServerError, errors.Wrap(err, "failed to store user-configurable limits"))
	}

	w.WriteHeader(http.StatusOK)
}

func (a *UserConfigOverridesAPI) DeleteOverridesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span, ctx := opentracing.StartSpanFromContext(ctx, "UserConfigOverridesAPI.DeleteOverridesHandler")
	defer span.Finish()

	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		handleError(span, userID, r, w, http.StatusBadRequest, err)
		return
	}
	logRequest(userID, r)

	err = a.client.Delete(ctx, userID)
	if err != nil {
		handleError(span, userID, nil, w, http.StatusInternalServerError, errors.Wrap(err, "failed to delete user-configurable limits"))
	}

	w.WriteHeader(http.StatusOK)
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
