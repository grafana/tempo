package userconfigurableapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-kit/log/level"
	jsoniter "github.com/json-iterator/go"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	ot_log "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/weaveworks/common/user"

	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/util/log"
	"github.com/grafana/tempo/tempodb/backend"
)

const (
	headerEtag    = "ETag"
	headerIfMatch = "If-Match"
)

// UserConfigOverridesAPI manages the API to retrieve, update and delete user-configurable overrides
// from the backend.
type UserConfigOverridesAPI struct {
	client Client
}

func NewUserConfigOverridesAPI(config *UserConfigurableOverridesClientConfig) (*UserConfigOverridesAPI, error) {
	client, err := NewUserConfigOverridesClient(config)
	if err != nil {
		return nil, err
	}
	return &UserConfigOverridesAPI{client}, nil
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

	limits, version, err := a.client.Get(ctx, userID)
	if err == backend.ErrDoesNotExist {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		handleError(span, userID, r, w, http.StatusInternalServerError, err)
		return
	}

	data, err := jsoniter.Marshal(limits)
	if err != nil {
		handleError(span, userID, r, w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set(headerEtag, string(version))
	w.Header().Set(api.HeaderContentType, api.HeaderAcceptJSON)
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

	ifMatchVersion := r.Header.Get(headerIfMatch)
	if ifMatchVersion == "" {
		handleError(span, userID, r, w, http.StatusPreconditionRequired, errors.New("must specify If-Match header"))
		return
	}

	d := jsoniter.NewDecoder(r.Body)
	// error in case of unwanted fields
	d.DisallowUnknownFields()

	limits := &UserConfigurableLimits{}

	err = d.Decode(&limits)
	if err != nil {
		handleError(span, userID, r, w, http.StatusBadRequest, err)
		return
	}

	// TODO validate the received data

	version, err := a.client.Set(ctx, userID, limits, backend.Version(ifMatchVersion))
	if err == backend.ErrVersionDoesNotMatch {
		handleError(span, userID, r, w, http.StatusPreconditionFailed, err)
		return
	}
	if err != nil {
		handleError(span, userID, r, w, http.StatusInternalServerError, errors.Wrap(err, "failed to store user-configurable limits"))
		return
	}

	w.Header().Set(headerEtag, string(version))
}

func (a *UserConfigOverridesAPI) PatchOverridesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span, ctx := opentracing.StartSpanFromContext(ctx, "UserConfigOverridesAPI.PatchOverridesHandler")
	defer span.Finish()

	userID, err := user.ExtractOrgID(ctx)
	if err != nil {
		handleError(span, userID, r, w, http.StatusBadRequest, err)
		return
	}
	logRequest(userID, r)

	patch, err := io.ReadAll(r.Body)
	if err != nil {
		handleError(span, userID, r, w, http.StatusInternalServerError, err)
		return
	}

	currLimits, currVersion, err := a.client.Get(ctx, userID)
	if err != nil && err != backend.ErrDoesNotExist {
		handleError(span, userID, r, w, http.StatusInternalServerError, err)
		return
	}

	patchedBytes := patch
	if err != backend.ErrDoesNotExist {
		currBytes, err := json.Marshal(currLimits)
		if err != nil {
			handleError(span, userID, r, w, http.StatusInternalServerError, err)
			return
		}

		patchedBytes, err = jsonpatch.MergePatch(currBytes, patch)
		if err != nil {
			handleError(span, userID, r, w, http.StatusBadRequest, errors.Wrap(err, "applying patch failed"))
			return
		}
	} else {
		currVersion = backend.VersionNew
	}

	var patchedLimits UserConfigurableLimits
	d := jsoniter.NewDecoder(bytes.NewReader(patchedBytes))
	// error in case of unwanted fields
	d.DisallowUnknownFields()

	err = d.Decode(&patchedLimits)
	if err != nil {
		handleError(span, userID, r, w, http.StatusBadRequest, err)
		return
	}

	updatedVersion, err := a.client.Set(ctx, userID, &patchedLimits, currVersion)
	if err == backend.ErrVersionDoesNotMatch {
		handleError(span, userID, r, w, http.StatusInternalServerError, errors.New("overrides have been modified during request processing, try again"))
		return
	}
	if err != nil {
		handleError(span, userID, r, w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set(headerEtag, string(updatedVersion))
	w.Header().Set(api.HeaderContentType, api.HeaderAcceptJSON)
	_, _ = w.Write(patchedBytes)
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

	ifMatchVersion := r.Header.Get(headerIfMatch)
	if ifMatchVersion == "" {
		handleError(span, userID, r, w, http.StatusPreconditionRequired, errors.New("must specify If-Match header"))
		return
	}

	err = a.client.Delete(ctx, userID, backend.Version(ifMatchVersion))
	if err != nil {
		handleError(span, userID, r, w, http.StatusInternalServerError, errors.Wrap(err, "failed to delete user-configurable limits"))
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
