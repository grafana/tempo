package userconfigurableapi

import (
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

	// TODO validate the received data

	err = a.client.Set(ctx, userID, limits)
	if err != nil {
		handleError(span, userID, r, w, http.StatusInternalServerError, errors.Wrap(err, "failed to store user-configurable limits"))
	}

	w.WriteHeader(http.StatusOK)
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
		handleError(span, userID, r, w, http.StatusBadRequest, err)
		return
	}

	var patchedBytes []byte
	err = a.client.Update(ctx, userID, func(current io.ReadCloser) ([]byte, error) {
		if current != nil {
			currBytes, err := io.ReadAll(current)
			if err != nil {
				return nil, errors.Wrap(err, "failed to fetch current user-configurable limits")
			}

			patchedBytes, err = jsonpatch.MergePatch(currBytes, patch)
			if err != nil {
				return nil, err
			}
		} else {
			// No limits have been stored yet, the patch is the new limits
			patchedBytes = patch
		}

		// TODO validate the received data

		return patchedBytes, nil
	})
	if err != nil {
		// TODO if the patch was invalid we should return http.StatusBadRequest
		handleError(span, userID, r, w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set(api.HeaderContentType, api.HeaderAcceptJSON)
	w.WriteHeader(http.StatusOK)
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
