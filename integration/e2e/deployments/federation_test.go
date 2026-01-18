package deployments

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/httpclient"
	thrift "github.com/jaegertracing/jaeger-idl/thrift-gen/jaeger"
	"github.com/stretchr/testify/require"
)

const (
	configFederatedFrontend = "config-federated-frontend.tmpl.yaml"
	configAllInOneLocal     = "config-all-in-one-local.yaml"
)

// createSpansWithName creates a Jaeger batch with spans that have a specific operation name prefix
// seed is used to generate unique span IDs - different seeds generate different span IDs
func createSpansWithName(traceIDHigh, traceIDLow int64, spanNamePrefix string, numSpans int, timestamp time.Time, seed int64) *thrift.Batch {
	r := rand.New(rand.NewSource(seed))
	var spans []*thrift.Span
	lastSpanID := int64(0)

	for i := 0; i < numSpans; i++ {
		spanID := r.Int63()
		spans = append(spans, &thrift.Span{
			TraceIdLow:    traceIDLow,
			TraceIdHigh:   traceIDHigh,
			SpanId:        spanID,
			ParentSpanId:  lastSpanID,
			OperationName: fmt.Sprintf("%s-span-%d", spanNamePrefix, i),
			StartTime:     timestamp.UnixMicro(),
			Duration:      int64(r.Intn(100) + 1),
		})
		lastSpanID = spanID
	}

	process := &thrift.Process{
		ServiceName: fmt.Sprintf("service-%s", spanNamePrefix),
	}

	return &thrift.Batch{Process: process, Spans: spans}
}

// TestFederation tests the federated query frontend by:
// 1. Starting two independent Tempo all-in-one instances
// 2. Ingesting different traces into each instance
// 3. Starting a federated query frontend that queries both instances
// 4. Querying the federated frontend and verifying it can find traces from both instances
func TestFederation(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e_federation")
	require.NoError(t, err)
	defer s.Close()

	// Copy config for both Tempo instances (using local storage)
	_, err = util.CopyTemplateToSharedDir(s, configAllInOneLocal, "config-tempo1.yaml", nil)
	require.NoError(t, err)

	_, err = util.CopyTemplateToSharedDir(s, configAllInOneLocal, "config-tempo2.yaml", nil)
	require.NoError(t, err)

	// Create first Tempo instance with a unique name
	tempo1 := util.NewTempoAllInOneWithName("tempo1", "-config.file=/shared/config-tempo1.yaml")
	require.NoError(t, s.StartAndWaitReady(tempo1))

	// Create second Tempo instance with a unique name
	tempo2 := util.NewTempoAllInOneWithName("tempo2", "-config.file=/shared/config-tempo2.yaml")
	require.NoError(t, s.StartAndWaitReady(tempo2))

	// Create exporters for both Tempo instances
	c1, err := util.NewJaegerToOTLPExporter(tempo1.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, c1)

	c2, err := util.NewJaegerToOTLPExporter(tempo2.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, c2)

	// Generate a shared trace ID for the split trace test
	timestamp := time.Now()
	r := rand.New(rand.NewSource(timestamp.Unix()))
	traceIDHigh := r.Int63()
	traceIDLow := r.Int63()
	traceID := fmt.Sprintf("%016x%016x", traceIDHigh, traceIDLow)

	// Send spans with "tempo1" prefix to tempo1 (use seed 1 for unique span IDs)
	batch1 := createSpansWithName(traceIDHigh, traceIDLow, "tempo1", 1, timestamp, 1)
	err = c1.EmitBatch(context.Background(), batch1)
	require.NoError(t, err)

	// Send spans with "tempo2" prefix to tempo2 (use seed 2 for different unique span IDs)
	batch2 := createSpansWithName(traceIDHigh, traceIDLow, "tempo2", 1, timestamp, 2)
	err = c2.EmitBatch(context.Background(), batch2)
	require.NoError(t, err)

	// Verify tempo1 has only its spans
	apiClient1 := httpclient.New("http://"+tempo1.Endpoint(3200), "")
	trace1, err := apiClient1.QueryTrace(traceID)
	require.NoError(t, err)
	require.NotNil(t, trace1)

	// Create federated frontend config pointing to both instances
	federatedInstances := []map[string]string{
		{
			"Name":     "tempo1",
			"Endpoint": "http://" + tempo1.NetworkEndpoint(3200),
		},
		{
			"Name":     "tempo2",
			"Endpoint": "http://" + tempo2.NetworkEndpoint(3200),
		},
	}

	tmplConfigFederated := map[string]any{
		"Instances": federatedInstances,
	}
	_, err = util.CopyTemplateToSharedDir(s, configFederatedFrontend, "config-federated.yaml", tmplConfigFederated)
	require.NoError(t, err)

	// Start federated query frontend
	federatedFrontend := util.NewTempoFederatedFrontend("-config.file=/shared/config-federated.yaml", "-log.level=debug")
	require.NoError(t, s.StartAndWaitReady(federatedFrontend))

	// Query federated frontend for the split trace
	federatedClient := httpclient.New("http://"+federatedFrontend.Endpoint(3200), "")
	federatedTrace, err := federatedClient.QueryTrace(traceID)
	require.NoError(t, err)
	require.NotNil(t, federatedTrace)

	// Verify the federated trace contains spans from both tempo instances
	// Total should be 2 spans (1 from tempo1 + 1 from tempo2)
	totalSpanCount := 0
	tempo1SpanNames := 0
	tempo2SpanNames := 0

	for _, rs := range federatedTrace.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				totalSpanCount++
				spanName := span.Name
				if len(spanName) >= 6 && spanName[:6] == "tempo1" {
					tempo1SpanNames++
				} else if len(spanName) >= 6 && spanName[:6] == "tempo2" {
					tempo2SpanNames++
				}
			}
		}
	}

	require.Equal(t, 2, totalSpanCount, "federated trace should have 2 total spans")
	require.Equal(t, 1, tempo1SpanNames, "federated trace should have 1 span with 'tempo1' prefix")
	require.Equal(t, 1, tempo2SpanNames, "federated trace should have 1 span with 'tempo2' prefix")
}
