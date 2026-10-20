package distributor

import (
	"context"
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/go-kit/log"
	dslog "github.com/grafana/dskit/log"
	"github.com/grafana/dskit/user"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/xpdata/entity"

	"github.com/grafana/tempo/v3/modules/distributor/receiver"
	"github.com/grafana/tempo/v3/modules/overrides"
	"github.com/grafana/tempo/v3/pkg/tempopb"
)

func conversionFixture(n int, rich bool) ptrace.Traces {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", "frontend")
	rs.SetSchemaUrl("resource-schema")
	rs.Resource().SetDroppedAttributesCount(2)
	if rich {
		ref := entity.ResourceEntityRefs(rs.Resource()).AppendEmpty()
		ref.SetType("service")
		ref.SetSchemaUrl("entity-schema")
		ref.IdKeys().Append("service.name")
		ref.DescriptionKeys().Append("region")
		rs.Resource().Attributes().PutStr("region", "us-west")
	}
	ss := rs.ScopeSpans().AppendEmpty()
	ss.SetSchemaUrl("scope-schema")
	ss.Scope().SetName("instrumentation")
	ss.Scope().SetVersion("1.2.3")
	ss.Scope().SetDroppedAttributesCount(3)
	for i := range n {
		s := ss.Spans().AppendEmpty()
		var tid pcommon.TraceID
		var sid pcommon.SpanID
		binary.BigEndian.PutUint64(tid[8:], uint64(i/10+1))
		binary.BigEndian.PutUint64(sid[:], uint64(i+1))
		s.SetTraceID(tid)
		s.SetSpanID(sid)
		s.SetName("request")
		s.SetKind(ptrace.SpanKindServer)
		s.SetStartTimestamp(pcommon.Timestamp(1700000000000000000 + i))
		s.SetEndTimestamp(s.StartTimestamp() + 1000)
		if rich {
			s.SetParentSpanID(pcommon.SpanID{1})
			s.TraceState().FromRaw("vendor=value")
			s.SetFlags(0x301)
			s.Status().SetCode(ptrace.StatusCodeError)
			s.Status().SetMessage("failure")
			s.SetDroppedAttributesCount(4)
			s.SetDroppedEventsCount(5)
			s.SetDroppedLinksCount(6)
			a := s.Attributes()
			a.PutStr("http.method", "GET")
			a.PutInt("code", 503)
			a.PutBool("retry", true)
			a.PutDouble("ratio", .5)
			a.PutEmptyBytes("payload").FromRaw([]byte{1, 2, 3})
			a.PutEmpty("empty")
			a.PutEmptyBytes("empty-bytes")
			a.PutEmptyMap("nested").PutStr("key", "value")
			a.PutEmptySlice("array").AppendEmpty().SetStr("value")
			e := s.Events().AppendEmpty()
			e.SetName("event")
			e.SetTimestamp(s.StartTimestamp())
			e.SetDroppedAttributesCount(7)
			e.Attributes().PutStr("message", "event-value")
			l := s.Links().AppendEmpty()
			l.SetTraceID(tid)
			l.SetSpanID(sid)
			l.SetFlags(0x301)
			l.TraceState().FromRaw("linked=value")
			l.SetDroppedAttributesCount(8)
			l.Attributes().PutStr("link", "value")
		}
	}
	return td
}

func conversionDistributor(b testing.TB) *Distributor {
	b.Helper()
	cfg := Config{}
	cfg.MaxAttributeBytes = 1 << 20
	cfg.DistributorRing.InstanceID = "bench"
	cfg.DistributorRing.HeartbeatPeriod = time.Second
	cfg.DistributorRing.InstanceInterfaceNames = []string{"lo0", "lo"}
	limits := overrides.Config{Defaults: overrides.Overrides{Ingestion: overrides.IngestionOverrides{RateStrategy: overrides.LocalIngestionRateStrategy, RateLimitBytes: 1e12, BurstSizeBytes: 1e12}}}
	ov, err := overrides.NewOverrides(limits, nil, prometheus.NewRegistry())
	require.NoError(b, err)
	var logLevel dslog.Level
	require.NoError(b, logLevel.Set("error"))
	d, err := New(cfg, LocalPushTargets{LiveStore: func(context.Context, *tempopb.PushBytesRequest) (*tempopb.PushResponse, error) {
		return &tempopb.PushResponse{}, nil
	}}, nil, ov, receiver.MultiTenancyMiddleware(), log.NewNopLogger(), logLevel, prometheus.NewRegistry())
	require.NoError(b, err)
	return d
}

func BenchmarkPushTracesConversion(b *testing.B) {
	for _, n := range []int{1, 100, 1000} {
		for _, rich := range []bool{false, true} {
			b.Run(fmt.Sprintf("spans%d/rich%t", n, rich), func(b *testing.B) {
				td := conversionFixture(n, rich)
				d := conversionDistributor(b)
				ctx := user.InjectOrgID(context.Background(), "bench")
				b.ReportAllocs()
				for b.Loop() {
					_, err := d.PushTraces(ctx, td)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

func BenchmarkPushTracesConversionParallel(b *testing.B) {
	for _, rich := range []bool{false, true} {
		b.Run(fmt.Sprintf("rich%t", rich), func(b *testing.B) {
			d := conversionDistributor(b)
			ctx := user.InjectOrgID(context.Background(), "bench")
			td := conversionFixture(100, rich)
			td.MarkReadOnly()
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					if _, err := d.PushTraces(ctx, td); err != nil {
						b.Error(err)
						return
					}
				}
			})
		})
	}
}

func BenchmarkPushTracesConversionLargeBytes(b *testing.B) {
	td := conversionFixture(100, true)
	spans := td.ResourceSpans().At(0).ScopeSpans().At(0).Spans()
	payload := make([]byte, 64<<10)
	for i := range spans.Len() {
		spans.At(i).Attributes().PutEmptyBytes("large-payload").FromRaw(payload)
	}
	d := conversionDistributor(b)
	ctx := user.InjectOrgID(context.Background(), "bench")
	b.ReportAllocs()
	for b.Loop() {
		if _, err := d.PushTraces(ctx, td); err != nil {
			b.Fatal(err)
		}
	}
}
