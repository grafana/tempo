package vparquet5

import (
	"context"
	crand "crypto/rand"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	tempo_io "github.com/grafana/tempo/pkg/io"
	"github.com/grafana/tempo/pkg/tempopb"
	v1_common "github.com/grafana/tempo/pkg/tempopb/common/v1"
	v1_resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1_trace "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

// makeBenchmarkTrace generates a trace with realistic, compressible attribute
// data that mimics production telemetry. Unlike test.MakeTraceWithSpanCount
// which uses random strings (incompressible), this uses repeated realistic
// keys and limited-cardinality values so compression algorithms can demonstrate
// meaningful size reduction.
func makeBenchmarkTrace(batches, spansPerBatch int, traceID []byte) *tempopb.Trace {
	type attrPool struct {
		key    string
		values []string
	}

	attrPools := []attrPool{
		{"http.method", []string{"GET", "POST", "PUT", "DELETE", "PATCH"}},
		{"http.status_code", []string{"200", "201", "204", "301", "400", "401", "403", "404", "500", "502", "503"}},
		{"http.url", []string{"/api/v1/users", "/api/v1/orders", "/api/v1/products", "/api/v1/auth/login", "/api/v1/search", "/healthz"}},
		{"db.system", []string{"postgresql", "mysql", "redis", "mongodb", "elasticsearch"}},
		{"db.statement", []string{"SELECT * FROM users WHERE id = ?", "INSERT INTO orders (user_id, total) VALUES (?, ?)", "UPDATE products SET stock = ? WHERE id = ?", "DELETE FROM sessions WHERE expired_at < ?"}},
		{"net.peer.name", []string{"db-primary.internal", "db-replica.internal", "cache-01.internal", "search-01.internal", "api-gateway.internal"}},
		{"net.peer.port", []string{"5432", "3306", "6379", "27017", "9200", "8080", "443"}},
		{"rpc.system", []string{"grpc", "http", "kafka", "rabbitmq"}},
		{"rpc.service", []string{"UserService", "OrderService", "PaymentService", "NotificationService", "InventoryService"}},
		{"deployment.environment", []string{"production", "staging", "canary"}},
	}

	spanNames := []string{
		"GET /api/v1/users", "POST /api/v1/orders", "GET /api/v1/products",
		"SELECT users", "INSERT orders", "UPDATE products",
		"redis.get", "redis.set", "redis.del",
		"grpc.UserService/GetUser", "grpc.OrderService/CreateOrder",
		"kafka.produce", "kafka.consume",
		"GET /healthz", "POST /api/v1/auth/login",
	}

	serviceNames := []string{
		"frontend", "api-gateway", "user-service", "order-service",
		"payment-service", "inventory-service", "notification-service",
	}

	instrumentationLibs := []string{
		"opentelemetry-go", "opentelemetry-java", "opentelemetry-python",
	}

	trace := &tempopb.Trace{
		ResourceSpans: make([]*v1_trace.ResourceSpans, 0, batches),
	}

	now := time.Now()

	for b := range batches {
		serviceName := serviceNames[b%len(serviceNames)]

		batch := &v1_trace.ResourceSpans{
			Resource: &v1_resource.Resource{
				Attributes: []*v1_common.KeyValue{
					strAttr("service.name", serviceName),
					strAttr("service.version", fmt.Sprintf("1.%d.0", b%5)),
					strAttr("deployment.environment", attrPools[9].values[b%len(attrPools[9].values)]),
					strAttr("host.name", fmt.Sprintf("pod-%s-%02d", serviceName, b%10)),
					strAttr("telemetry.sdk.language", "go"),
				},
			},
		}

		libName := instrumentationLibs[b%len(instrumentationLibs)]
		ss := &v1_trace.ScopeSpans{
			Scope: &v1_common.InstrumentationScope{
				Name:    libName,
				Version: "1.0.0",
			},
			Spans: make([]*v1_trace.Span, 0, spansPerBatch),
		}

		for s := range spansPerBatch {
			spanID := make([]byte, 8)
			_, _ = crand.Read(spanID)

			startTime := uint64(now.Add(time.Duration(s) * time.Millisecond).UnixNano())
			endTime := startTime + uint64((1+rand.Intn(100))*int(time.Millisecond)) // nolint:gosec

			// Pick 4-6 attributes from the realistic pools
			numAttrs := 4 + rand.Intn(3) // nolint:gosec
			attrs := make([]*v1_common.KeyValue, 0, numAttrs)
			for a := range numAttrs {
				pool := attrPools[(s+a)%len(attrPools)]
				attrs = append(attrs, strAttr(pool.key, pool.values[(s+a)%len(pool.values)]))
			}

			span := &v1_trace.Span{
				TraceId:           traceID,
				SpanId:            spanID,
				ParentSpanId:      make([]byte, 8),
				Name:              spanNames[(b*spansPerBatch+s)%len(spanNames)],
				Kind:              v1_trace.Span_SpanKind(1 + s%4), // cycle through CLIENT, SERVER, PRODUCER, CONSUMER
				StartTimeUnixNano: startTime,
				EndTimeUnixNano:   endTime,
				Attributes:        attrs,
				Status: &v1_trace.Status{
					Code:    v1_trace.Status_StatusCode(s % 3), // cycle UNSET, OK, ERROR
					Message: "OK",
				},
			}

			ss.Spans = append(ss.Spans, span)
		}

		batch.ScopeSpans = []*v1_trace.ScopeSpans{ss}
		trace.ResourceSpans = append(trace.ResourceSpans, batch)
	}

	return trace
}

func strAttr(key, value string) *v1_common.KeyValue {
	return &v1_common.KeyValue{
		Key: key,
		Value: &v1_common.AnyValue{
			Value: &v1_common.AnyValue_StringValue{StringValue: value},
		},
	}
}

func BenchmarkCompression(b *testing.B) {
	compressions := []common.ParquetCompression{
		common.ParquetCompressionSnappy,
		common.ParquetCompressionLZ4,
		common.ParquetCompressionNone,
	}

	for _, compression := range compressions {
		b.Run(string(compression), func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				rawR, rawW, _, err := local.New(&local.Config{
					Path: b.TempDir(),
				})
				require.NoError(b, err)

				r := backend.NewReader(rawR)
				w := backend.NewWriter(rawW)

				cfg := &common.BlockConfig{
					BloomFP:             0.01,
					BloomShardSizeBytes: 100 * 1024,
					RowGroupSizeBytes:   20_000_000,
					ParquetCompression:  compression,
				}

				ctx := context.Background()
				inMeta := &backend.BlockMeta{
					TenantID:     tenantID,
					BlockID:      backend.NewUUID(),
					TotalObjects: 100,
				}

				sb := newStreamingBlock(ctx, cfg, inMeta, r, w, tempo_io.NewBufferedWriter)

				for t := 0; t < 100; t++ {
					id := make([]byte, 16)
					_, err := crand.Read(id)
					require.NoError(b, err)

					tr := makeBenchmarkTrace(100, 100, id)
					trp, _ := traceToParquet(inMeta, id, tr, nil)

					require.NoError(b, sb.Add(trp, 0, 0))
					if sb.EstimatedBufferedBytes() > 20_000_000 {
						_, err := sb.Flush()
						require.NoError(b, err)
					}
				}

				_, err = sb.Complete()
				require.NoError(b, err)

				b.ReportMetric(float64(sb.meta.Size_), "block_bytes")
			}
		})
	}
}
