package search

import (
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/traceql"
)

const (
	ErrorTag        = "error"
	StatusCodeTag   = "status.code"
	StatusCodeUnset = "unset"
	StatusCodeOK    = "ok"
	StatusCodeError = "error"

	RootSpanNotYetReceivedText = "<root span not yet received>"
)

func GetVirtualTagValues(tagName string) []string {
	switch tagName {

	case StatusCodeTag:
		return []string{StatusCodeUnset, StatusCodeOK, StatusCodeError}

	case ErrorTag:
		return []string{"true"}
	}

	return nil
}

func GetVirtualTagValuesV2(tagName string) []tempopb.TagValue {
	switch tagName {
	case traceql.IntrinsicStatus.String(), traceql.IntrinsicSpanStatus.String():
		return []tempopb.TagValue{
			{Type: "keyword", Value: traceql.StatusOk.String()},
			{Type: "keyword", Value: traceql.StatusError.String()},
			{Type: "keyword", Value: traceql.StatusUnset.String()},
		}
	case traceql.IntrinsicKind.String(), traceql.IntrinsicSpanKind.String():
		return []tempopb.TagValue{
			{Type: "keyword", Value: traceql.KindClient.String()},
			{Type: "keyword", Value: traceql.KindServer.String()},
			{Type: "keyword", Value: traceql.KindProducer.String()},
			{Type: "keyword", Value: traceql.KindConsumer.String()},
			{Type: "keyword", Value: traceql.KindInternal.String()},
			{Type: "keyword", Value: traceql.KindUnspecified.String()},
		}
	case traceql.IntrinsicDuration.String(), traceql.IntrinsicSpanDuration.String():
		return []tempopb.TagValue{}
	case traceql.IntrinsicTraceDuration.String():
		return []tempopb.TagValue{}
	}

	return nil
}

func GetVirtualIntrinsicValues() []string {
	return []string{
		traceql.IntrinsicDuration.String(),
		traceql.IntrinsicKind.String(),
		traceql.IntrinsicName.String(),
		traceql.IntrinsicStatus.String(),
		traceql.IntrinsicStatusMessage.String(),
		traceql.IntrinsicTraceDuration.String(),
		traceql.IntrinsicTraceRootService.String(),
		traceql.IntrinsicTraceRootSpan.String(),
		traceql.IntrinsicSpanStatus.String(),
		traceql.IntrinsicSpanStatusMessage.String(),
		traceql.IntrinsicSpanDuration.String(),
		traceql.IntrinsicSpanName.String(),
		traceql.IntrinsicSpanKind.String(),
		traceql.IntrinsicTraceRootName.String(),
		traceql.IntrinsicTraceRootServiceName.String(),
		/* these are technically intrinsics that can be requested, but they are not generally of interest to a user
		   typing a query. for simplicity and clarity we are leaving them out of autocomplete
			IntrinsicNestedSetLeft
			IntrinsicNestedSetRight
			IntrinsicNestedSetParent
		*/
	}
}
