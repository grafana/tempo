package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/services"
	jsoniter "github.com/json-iterator/go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/grafana/tempo/modules/overrides"
	"github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	tempo_log "github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/pkg/util/tracing"
	"github.com/grafana/tempo/tempodb/backend"
)

var errConflictingRuntimeOverrides = errors.New("tenant has conflicting overrides set in runtime config, contact your system administrator to perform changes through the API")

var tracer = otel.Tracer("modules/overrides/userconfigurable/api")

type Validator interface {
	Validate(limits *client.Limits) error
}

// UserConfigOverridesAPI manages the API to retrieve, update and delete user-configurable overrides
// from the backend.
type UserConfigOverridesAPI struct {
	services.Service

	cfg       *overrides.UserConfigurableOverridesAPIConfig
	client    client.Client
	overrides overrides.Interface
	validator Validator

	logger log.Logger
}

func New(cfg *overrides.UserConfigurableOverridesAPIConfig, clientCfg *client.Config, overrides overrides.Interface, validator Validator) (*UserConfigOverridesAPI, error) {
	client, err := client.New(clientCfg)
	if err != nil {
		return nil, err
	}

	api := &UserConfigOverridesAPI{
		cfg:       cfg,
		client:    client,
		overrides: overrides,
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
	ctx, span := tracer.Start(ctx, "UserConfigOverridesAPI.get", trace.WithAttributes(
		attribute.String("userID", userID),
	))
	defer span.End()

	return a.client.Get(ctx, userID)
}

// set the Limits. Can return backend.ErrVersionDoesNotMatch, validationError
func (a *UserConfigOverridesAPI) set(ctx context.Context, userID string, limits *client.Limits, version backend.Version, skipConflictingOverridesCheck bool) (backend.Version, error) {
	ctx, span := tracer.Start(ctx, "UserConfigOverridesAPI.set", trace.WithAttributes(
		attribute.String("userID", userID),
		attribute.String("version", string(version)),
		attribute.String("limits", logLimits(limits)),
	))
	defer span.End()
	traceID, _ := tracing.ExtractTraceID(ctx)

	err := a.validator.Validate(limits)
	if err != nil {
		return "", newValidationError(err)
	}

	if a.cfg.CheckForConflictingRuntimeOverrides && !skipConflictingOverridesCheck {
		err = a.assertNoConflictingRuntimeOverrides(ctx, userID)
		if err != nil {
			return "", err
		}
	}

	level.Info(a.logger).Log("traceID", traceID, "msg", "storing user-configurable overrides", "userID", userID, "limits", logLimits(limits), "version", version)

	newVersion, err := a.client.Set(ctx, userID, limits, version)

	level.Info(a.logger).Log("traceID", traceID, "msg", "stored user-configurable overrides", "userID", userID, "limits", logLimits(limits), "version", version, "newVersion", newVersion, "err", err)
	return newVersion, err
}

func (a *UserConfigOverridesAPI) update(ctx context.Context, userID string, patch []byte, skipConflictingOverridesCheck bool) (*client.Limits, backend.Version, error) {
	ctx, span := tracer.Start(ctx, "UserConfigOverridesAPI.update", trace.WithAttributes(
		attribute.String("userID", userID),
	))
	defer span.End()
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

	version, err := a.set(ctx, userID, patchedLimits, currVersion, skipConflictingOverridesCheck)
	if errors.Is(err, backend.ErrVersionDoesNotMatch) {
		return nil, "", errors.New("overrides have been modified during request processing, try again")
	}
	if err != nil {
		return nil, "", err
	}

	return patchedLimits, version, err
}

func (a *UserConfigOverridesAPI) delete(ctx context.Context, userID string, version backend.Version) error {
	ctx, span := tracer.Start(ctx, "UserConfigOverridesAPI.delete", trace.WithAttributes(
		attribute.String("userID", userID),
		attribute.String("version", string(version)),
	))
	defer span.End()
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

func (a *UserConfigOverridesAPI) assertNoConflictingRuntimeOverrides(ctx context.Context, userID string) error {
	limits, _, err := a.client.Get(ctx, userID)
	if err != nil && !errors.Is(err, backend.ErrDoesNotExist) {
		return err
	}
	// we already store limits for this user, don't check further
	if limits != nil {
		return nil
	}

	runtimeOverrides := a.overrides.GetRuntimeOverridesFor(userID)

	// convert overrides.Overrides to client.Limits through marshalling and unmarshalling it from json
	// this is the easiest way to convert the optional fields
	marshalledOverrides, err := jsoniter.Marshal(runtimeOverrides)
	if err != nil {
		return err
	}
	var runtimeLimits client.Limits
	err = jsoniter.Unmarshal(marshalledOverrides, &runtimeLimits)
	if err != nil {
		return err
	}

	// clear out processors since we merge this field
	runtimeLimits.MetricsGenerator.Processors = nil

	emptyLimits := client.Limits{}
	if reflect.DeepEqual(runtimeLimits, emptyLimits) {
		return nil
	}

	return errConflictingRuntimeOverrides
}

// validationError is returned when the request can not be accepted because of a client error
type validationError struct {
	err error
}

func newValidationError(err error) *validationError {
	return &validationError{err: err}
}

func (e *validationError) Error() string {
	return e.err.Error()
}

func (e *validationError) Unwrap() error {
	return errors.Unwrap(e.err)
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
