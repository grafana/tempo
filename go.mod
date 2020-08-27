module github.com/grafana/tempo

go 1.15

require (
	cloud.google.com/go/storage v1.6.0
	github.com/census-instrumentation/opencensus-proto v0.3.0
	github.com/cortexproject/cortex v0.7.0
	github.com/go-kit/kit v0.10.0
	github.com/gogo/protobuf v1.3.1
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.4.2
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.4
	github.com/grafana/loki v1.3.0
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645
	github.com/hashicorp/go-hclog v0.14.0
	github.com/jaegertracing/jaeger v1.18.2-0.20200707061226-97d2319ff2be
	github.com/karrick/godirwalk v1.16.1
	github.com/olekukonko/tablewriter v0.0.2
	github.com/open-telemetry/opentelemetry-proto v0.4.0
	github.com/opentracing/opentracing-go v1.1.1-0.20200124165624-2876d2018785
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/common v0.11.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1
	github.com/uber-go/atomic v1.4.0
	github.com/weaveworks/common v0.0.0-20200206153930-760e36ae819a
	github.com/willf/bitset v1.1.10 // indirect
	github.com/willf/bloom v2.0.3+incompatible
	go.opentelemetry.io/collector v0.6.1
	go.uber.org/atomic v1.6.0
	go.uber.org/ratelimit v0.1.0
	go.uber.org/zap v1.15.0
	golang.org/x/net v0.0.0-20200625001655-4c5254603344
	google.golang.org/api v0.29.0
	google.golang.org/grpc v1.31.0
	gopkg.in/yaml.v2 v2.3.0
)

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.2.0+incompatible

// Override reference that causes an error from Go proxy - see https://github.com/golang/go/issues/33558
replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190620085101-78d2af792bab

// the version otel collector uses.  required for it to build
// replace github.com/apache/thrift => github.com/apache/thrift v0.0.0-20161221203622-b2a4d4ae21c7

// Override reference causing proxy error.  Otherwise it attempts to download https://proxy.golang.org/golang.org/x/net/@v/v0.0.0-20190813000000-74dc4d7220e7.info
replace golang.org/x/net => golang.org/x/net v0.0.0-20190923162816-aa69164e4478

//Cortex:  We can't upgrade to grpc 1.30.0 until go.etcd.io/etcd will support it.
//  This causes go mod tidy to fail b/c there are two modules which export the same module module (google.golang.org/grpc/examples)
//  Part of grpc in 1.29.1 (https://github.com/grpc/grpc-go/tree/master/examples) but made it's own module in 1.30.0
//  PR to upgrade: https://github.com/etcd-io/etcd/pull/12155
//  go mod tidy issue: https://github.com/golang/go/issues/27899
replace google.golang.org/grpc => google.golang.org/grpc v1.29.1
