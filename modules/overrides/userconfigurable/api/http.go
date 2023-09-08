package api

import (
	"context"
	"io"
	"net/http"

	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/tracing"
	"github.com/grafana/dskit/user"
	jsoniter "github.com/json-iterator/go"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	ot_log "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"

	"github.com/grafana/tempo/modules/overrides/userconfigurable/client"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/tempodb/backend"
)

const (
	headerEtag    = "ETag"
	headerIfMatch = "If-Match"

	errNoIfMatchHeader = "must specify If-Match header"
)

// GetHandler retrieves the user-configured overrides from the backend.
func (a *UserConfigOverridesAPI) GetHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	ctx, f := a.logRequest(r.Context(), "UserConfigOverridesAPI.GetHandler", r)
	defer f(&err)

	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	limits, version, err := a.get(ctx, userID)
	if err != nil {
		writeError(w, err)
		return
	}

	err = writeLimits(w, limits, version)
}

// PostHandler accepts post requests with json payload and writes it to config backend.
func (a *UserConfigOverridesAPI) PostHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	ctx, f := a.logRequest(r.Context(), "UserConfigOverridesAPI.PostHandler", r)
	defer f(&err)

	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ifMatchVersion := r.Header.Get(headerIfMatch)
	if ifMatchVersion == "" {
		http.Error(w, errNoIfMatchHeader, http.StatusPreconditionRequired)
		return
	}

	limits, err := a.parseLimits(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	version, err := a.set(ctx, userID, limits, backend.Version(ifMatchVersion))
	if err != nil {
		writeError(w, err)
	}

	w.Header().Set(headerEtag, string(version))
}

func (a *UserConfigOverridesAPI) PatchHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	ctx, f := a.logRequest(r.Context(), "UserConfigOverridesAPI.PatchHandler", r)
	defer f(&err)

	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	patch, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	patchedLimits, version, err := a.update(ctx, userID, patch)
	if err != nil {
		writeError(w, err)
		return
	}

	err = writeLimits(w, patchedLimits, version)
}

func (a *UserConfigOverridesAPI) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	ctx, f := a.logRequest(r.Context(), "UserConfigOverridesAPI.DeleteHandler", r)
	defer f(&err)

	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ifMatchVersion := r.Header.Get(headerIfMatch)
	if ifMatchVersion == "" {
		http.Error(w, errNoIfMatchHeader, http.StatusPreconditionRequired)
		return
	}

	err = a.delete(ctx, userID, backend.Version(ifMatchVersion))
	if err != nil {
		writeError(w, err)
	}
}

func writeError(w http.ResponseWriter, err error) {
	if errors.Is(err, backend.ErrDoesNotExist) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if errors.Is(err, backend.ErrVersionDoesNotMatch) || errors.Is(err, backend.ErrVersionInvalid) {
		http.Error(w, err.Error(), http.StatusPreconditionFailed)
		return
	}
	var valErr validationError
	if ok := errors.As(err, &valErr); ok {
		http.Error(w, valErr.Error(), http.StatusBadRequest)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func writeLimits(w http.ResponseWriter, limits *client.Limits, version backend.Version) error {
	data, err := jsoniter.Marshal(limits)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set(headerEtag, string(version))
	w.Header().Set(api.HeaderContentType, api.HeaderAcceptJSON)
	_, _ = w.Write(data)
	return nil
}

func (a *UserConfigOverridesAPI) logRequest(ctx context.Context, handler string, r *http.Request) (context.Context, func(*error)) {
	span, ctx := opentracing.StartSpanFromContext(ctx, handler)
	traceID, _ := tracing.ExtractTraceID(ctx)

	level.Info(a.logger).Log("traceID", traceID, "method", r.Method, "url", r.URL.RequestURI(), "user-agent", r.UserAgent())

	return ctx, func(errPtr *error) {
		err := *errPtr

		if err != nil && !errors.Is(err, backend.ErrDoesNotExist) {
			span.LogFields(ot_log.Error(err))
			ext.Error.Set(span, true)
			level.Error(a.logger).Log("traceID", traceID, "method", r.Method, "url", r.URL.RequestURI(), "user-agent", r.UserAgent(), "err", err)
		}

		span.Finish()
	}
}
