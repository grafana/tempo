package main

import (
	"testing"
	"time"

	v1common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSeed = 1632146180

func TestVerifyTraceContent(t *testing.T) {
	seed := time.Unix(0, testSeed)
	info := util.NewTraceInfo(seed, "")

	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)
	require.NotEmpty(t, expected.ResourceSpans)

	// Same trace should verify successfully
	err = VerifyTraceContent(expected, expected)
	require.NoError(t, err)

	// Modified trace should fail: change one attribute value (use one we know exists on resource)
	retrieved := cloneTrace(expected)
	attrs := retrieved.ResourceSpans[0].Resource.Attributes
	var found bool
	const alterKey = "vulture-process-0"
	for i := range attrs {
		if attrs[i].Key == alterKey {
			attrs[i].Value = &v1common.AnyValue{Value: &v1common.AnyValue_StringValue{StringValue: "wrong"}}
			found = true
			break
		}
	}
	require.True(t, found, "first resource should have attribute %q", alterKey)
	err = VerifyTraceContent(expected, retrieved)
	require.Error(t, err, "VerifyTraceContent should fail when an attribute value differs")
	assert.Contains(t, err.Error(), alterKey)
}
