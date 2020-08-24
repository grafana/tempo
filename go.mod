module github.com/grafana/tempo

go 1.15

require (
	cloud.google.com/go/storage v1.3.0
	github.com/census-instrumentation/opencensus-proto v0.2.1
	github.com/cortexproject/cortex v0.7.0
	github.com/fatih/color v1.9.0 // indirect
	github.com/go-kit/kit v0.9.0
	github.com/gogo/protobuf v1.3.1
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.3.4
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.3
	github.com/grafana/loki v1.3.0
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645
	github.com/hashicorp/go-hclog v0.12.0
	github.com/hashicorp/go-plugin v1.0.1 // indirect
	github.com/jaegertracing/jaeger v1.17.0
	github.com/karrick/godirwalk v1.16.1
	github.com/mattn/go-colorable v0.1.6 // indirect
	github.com/olekukonko/tablewriter v0.0.2
	github.com/open-telemetry/opentelemetry-collector v0.2.7-0.20200311232134-5334b3a8ff08
	github.com/open-telemetry/opentelemetry-proto v0.0.0-20200308012146-674ae1c8703f
	github.com/opentracing/opentracing-go v1.1.1-0.20200124165624-2876d2018785
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.5.0
	github.com/prometheus/common v0.9.1
	github.com/spf13/cobra v0.0.6 // indirect
	github.com/spf13/viper v1.6.2
	github.com/stretchr/testify v1.4.0
	github.com/uber-go/atomic v1.4.0
	github.com/weaveworks/common v0.0.0-20200206153930-760e36ae819a
	github.com/willf/bitset v1.1.10 // indirect
	github.com/willf/bloom v2.0.3+incompatible
	go.uber.org/atomic v1.5.0
	go.uber.org/ratelimit v0.1.0
	go.uber.org/zap v1.10.0
	golang.org/x/net v0.0.0-20191112182307-2180aed22343
	golang.org/x/tools v0.0.0-20191127201027-ecd32218bd7f // indirect
	google.golang.org/api v0.14.0
	google.golang.org/grpc v1.25.1
	gopkg.in/yaml.v2 v2.2.8
)

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.2.0+incompatible

// Override reference that causes an error from Go proxy - see https://github.com/golang/go/issues/33558
replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190620085101-78d2af792bab

// the version otel collector uses.  required for it to build
replace github.com/apache/thrift => github.com/apache/thrift v0.0.0-20161221203622-b2a4d4ae21c7

// Override reference causing proxy error.  Otherwise it attempts to download https://proxy.golang.org/golang.org/x/net/@v/v0.0.0-20190813000000-74dc4d7220e7.info
replace golang.org/x/net => golang.org/x/net v0.0.0-20190923162816-aa69164e4478
