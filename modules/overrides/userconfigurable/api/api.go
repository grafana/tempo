package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/services"
	"github.com/grafana/dskit/tracing"
	jsoniter "github.com/json-iterator/go"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"

	"github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	tempo_log "github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
)

type Validator interface {
	Validate(limits *client.Limits) error
}

// UserConfigOverridesAPI manages the API to retrieve, update and delete user-configurable overrides
// from the backend.
type UserConfigOverridesAPI struct {
	services.Service

	client    client.Client
	validator Validator

	logger log.Logger
}

func New(config *client.Config, validator Validator) (*UserConfigOverridesAPI, error) {
	client, err := client.New(config)
	if err != nil {
		return nil, err
	}

	api := &UserConfigOverridesAPI{
		client:    client,
		validator: validator,

		logger: log.With(tempo_log.Logger, "component", "overrides-api"),
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

// get the Limits. Can return backend.ErrDoesNotExist
func (a *UserConfigOverridesAPI) get(ctx context.Context, userID string) (*client.Limits, backend.Version, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "UserConfigOverridesAPI.get", opentracing.Tags{
		"userID": userID,
	})
	defer span.Finish()

	return a.client.Get(ctx, userID)
}

// set the Limits. Can return backend.ErrVersionDoesNotMatch, validationError
func (a *UserConfigOverridesAPI) set(ctx context.Context, userID string, limits *client.Limits, version backend.Version) (backend.Version, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "UserConfigOverridesAPI.set", opentracing.Tags{
		"userID":  userID,
		"version": version,
		"limits":  logLimits(limits),
	})
	defer span.Finish()
	traceID, _ := tracing.ExtractTraceID(ctx)

	err := a.validator.Validate(limits)
	if err != nil {
		return "", newValidationError(err)
	}

	level.Info(a.logger).Log("traceID", traceID, "msg", "storing user-configurable overrides", "userID", userID, "limits", logLimits(limits), "version", version)

	newVersion, err := a.client.Set(ctx, userID, limits, version)

	level.Info(a.logger).Log("traceID", traceID, "msg", "stored user-configurable overrides", "userID", userID, "limits", logLimits(limits), "version", version, "newVersion", newVersion, "err", err)
	return newVersion, err
}

func (a *UserConfigOverridesAPI) update(ctx context.Context, userID string, patch []byte) (*client.Limits, backend.Version, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "UserConfigOverridesAPI.update", opentracing.Tags{
		"userID": userID,
	})
	defer span.Finish()
	traceID, _ := tracing.ExtractTraceID(ctx)

	currLimits, currVersion, err := a.client.Get(ctx, userID)
	if err != nil && !errors.Is(err, backend.ErrDoesNotExist) {
		return nil, "", err
	}

	level.Info(a.logger).Log("traceID", traceID, "msg", "patching user-configurable overrides", "userID", userID, "patch", patch, "currLimits", logLimits(currLimits), "currVersion", currVersion)

	if errors.Is(err, backend.ErrDoesNotExist) {
		currVersion = backend.VersionNew
	}

	patchedBytes := patch
	if !errors.Is(err, backend.ErrDoesNotExist) {
		currBytes, err := json.Marshal(currLimits)
		if err != nil {
			return nil, "", err
		}

		patchedBytes, err = jsonpatch.MergePatch(currBytes, patch)
		if err != nil {
			return nil, "", err
		}
	}

	patchedLimits, err := a.parseLimits(bytes.NewReader(patchedBytes))
	if err != nil {
		return nil, "", newValidationError(err)
	}

	version, err := a.set(ctx, userID, patchedLimits, currVersion)
	if errors.Is(err, backend.ErrVersionDoesNotMatch) {
		return nil, "", errors.New("overrides have been modified during request processing, try again")
	}
	if err != nil {
		return nil, "", err
	}

	return patchedLimits, version, err
}

func (a *UserConfigOverridesAPI) delete(ctx context.Context, userID string, version backend.Version) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "UserConfigOverridesAPI.delete", opentracing.Tags{
		"userID":  userID,
		"version": version,
	})
	defer span.Finish()
	traceID, _ := tracing.ExtractTraceID(ctx)

	level.Info(a.logger).Log("traceID", traceID, "msg", "deleting user-configurable overrides", "userID", userID, "version", version)

	return a.client.Delete(ctx, userID, version)
}

func (a *UserConfigOverridesAPI) parseLimits(body io.Reader) (*client.Limits, error) {
	d := jsoniter.NewDecoder(body)

	// error in case of unwanted fields
	d.DisallowUnknownFields()

	var limits client.Limits

	err := d.Decode(&limits)
	return &limits, err
}

// validationError is returned when the request can not be accepted because of a client error
type validationError struct {
	error
}

func newValidationError(err error) validationError {
	return validationError{err}
}

func logLimits(limits *client.Limits) string {
	if limits == nil {
		return ""
	}
	bytes, err := jsoniter.Marshal(limits)
	if err != nil {
		return ""
	}
	return string(bytes)
}
