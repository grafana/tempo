package frontend

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/grafana/dskit/tenant"
	"github.com/grafana/dskit/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grafana/tempo/pkg/tempopb"
)

const (
	maxRedactionTraceIDs     = 1000
	maxRedactionRequestBytes = 64 << 10
)

type RedactionHandler struct {
	client tempopb.BackendSchedulerClient
}

type redactionSubmitRequest struct {
	TraceIDs []string `json:"trace_ids"`
}

type redactionSubmitResponse struct {
	BatchID     string `json:"batch_id"`
	JobsCreated int32  `json:"jobs_created"`
}

type redactionErrorResponse struct {
	Error string `json:"error"`
}

func NewRedactionHandler(client tempopb.BackendSchedulerClient) *RedactionHandler {
	return &RedactionHandler{client: client}
}

func (h *RedactionHandler) Submit(w http.ResponseWriter, r *http.Request) {
	ctx, err := redactionGRPCContext(r)
	if err != nil {
		writeRedactionError(w, http.StatusBadRequest, err.Error())
		return
	}

	var body redactionSubmitRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRedactionRequestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		writeRedactionError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := ensureJSONEOF(decoder); err != nil {
		writeRedactionError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(body.TraceIDs) == 0 {
		writeRedactionError(w, http.StatusBadRequest, "trace_ids must contain at least one trace ID")
		return
	}
	if len(body.TraceIDs) > maxRedactionTraceIDs {
		writeRedactionError(w, http.StatusBadRequest, "trace_ids exceeds the limit of 1000")
		return
	}

	traceIDs := make([][]byte, 0, len(body.TraceIDs))
	seen := make(map[[16]byte]struct{}, len(body.TraceIDs))
	for _, encoded := range body.TraceIDs {
		if len(encoded) != 32 {
			writeRedactionError(w, http.StatusBadRequest, "trace_ids must contain canonical 32-character hexadecimal trace IDs")
			return
		}
		decoded, err := hex.DecodeString(encoded)
		if err != nil {
			writeRedactionError(w, http.StatusBadRequest, "trace_ids must contain canonical 32-character hexadecimal trace IDs")
			return
		}
		var key [16]byte
		copy(key[:], decoded)
		if key == ([16]byte{}) {
			writeRedactionError(w, http.StatusBadRequest, "trace_ids must not contain the zero trace ID")
			return
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		traceIDs = append(traceIDs, decoded)
	}

	response, err := h.client.SubmitRedaction(ctx, &tempopb.SubmitRedactionRequest{
		TraceIds: traceIDs,
		Mode:     tempopb.RedactionMode_REDACTION_MODE_APPLY,
	})
	if err != nil {
		writeRedactionGRPCError(w, err)
		return
	}
	writeRedactionJSON(w, http.StatusAccepted, redactionSubmitResponse{
		BatchID:     response.BatchId,
		JobsCreated: response.JobsCreated,
	})
}

func redactionGRPCContext(r *http.Request) (context.Context, error) {
	tenantIDs, err := tenant.TenantIDs(r.Context())
	if err != nil {
		return nil, err
	}
	if len(tenantIDs) != 1 {
		return nil, errors.New("redaction requires exactly one tenant")
	}
	ctx := user.InjectOrgID(r.Context(), tenantIDs[0])
	return user.InjectIntoGRPCRequest(ctx)
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func writeRedactionGRPCError(w http.ResponseWriter, err error) {
	grpcStatus, ok := status.FromError(err)
	if !ok {
		writeRedactionError(w, http.StatusInternalServerError, "internal redaction service error")
		return
	}

	httpStatus := http.StatusInternalServerError
	message := grpcStatus.Message()
	switch grpcStatus.Code() {
	case codes.InvalidArgument:
		httpStatus = http.StatusBadRequest
	case codes.Unauthenticated:
		httpStatus = http.StatusUnauthorized
	case codes.PermissionDenied:
		httpStatus = http.StatusForbidden
	case codes.NotFound:
		httpStatus = http.StatusNotFound
	case codes.AlreadyExists, codes.FailedPrecondition, codes.Aborted:
		httpStatus = http.StatusConflict
	case codes.ResourceExhausted:
		httpStatus = http.StatusTooManyRequests
	case codes.Canceled:
		httpStatus = 499
	case codes.DeadlineExceeded:
		httpStatus = http.StatusGatewayTimeout
	case codes.Unavailable:
		httpStatus = http.StatusServiceUnavailable
		message = "redaction service unavailable"
	case codes.Internal, codes.Unknown, codes.DataLoss:
		message = "internal redaction service error"
	}
	writeRedactionError(w, httpStatus, message)
}

func writeRedactionError(w http.ResponseWriter, statusCode int, message string) {
	writeRedactionJSON(w, statusCode, redactionErrorResponse{Error: message})
}

func writeRedactionJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}
