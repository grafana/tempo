package trace

import v1trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"

func StatusToString(s v1trace.Status_StatusCode) string {
	var status string
	switch s {
	case v1trace.Status_STATUS_CODE_UNSET:
		status = "unset"
	case v1trace.Status_STATUS_CODE_OK:
		status = "ok"
	case v1trace.Status_STATUS_CODE_ERROR:
		status = "error"
	}
	return status
}
