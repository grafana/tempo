module github.com/grafana/tempo

go 1.16

require (
	cloud.google.com/go/storage v1.15.0
	contrib.go.opencensus.io/exporter/prometheus v0.2.0
	github.com/Azure/azure-pipeline-go v0.2.2
	github.com/Azure/azure-storage-blob-go v0.8.0
	github.com/alecthomas/kong v0.2.11
	github.com/cespare/xxhash v1.1.0
	github.com/cortexproject/cortex v1.9.0
	github.com/cristalhq/hedgedhttp v0.6.0
	github.com/drone/envsubst v1.0.3
	github.com/dustin/go-humanize v1.0.0
	github.com/go-kit/kit v0.10.0
	github.com/gogo/protobuf v1.3.2
	github.com/gogo/status v1.0.3
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.5.2
	github.com/golang/snappy v0.0.3
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.1-0.20191002090509-6af20e3a5340 // indirect
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645
	github.com/hashicorp/go-hclog v0.14.0
	github.com/hashicorp/go-plugin v1.3.0 // indirect
	github.com/jaegertracing/jaeger v1.21.0
	github.com/jsternberg/zap-logfmt v1.2.0
	github.com/klauspost/compress v1.11.7
	github.com/minio/minio-go/v7 v7.0.10
	github.com/olekukonko/tablewriter v0.0.2
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pierrec/lz4/v4 v4.1.3
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.10.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.21.0
	github.com/prometheus/prometheus v1.8.2-0.20210421143221-52df5ef7a3be
	github.com/segmentio/fasthash v0.0.0-20180216231524-a72b379d632e
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/uber-go/atomic v1.4.0
	github.com/uber/jaeger-client-go v2.25.0+incompatible
	github.com/weaveworks/common v0.0.0-20210419092856-009d1eebd624
	github.com/willf/bitset v1.1.10 // indirect
	github.com/willf/bloom v2.0.3+incompatible
	go.opencensus.io v0.23.0
	go.opentelemetry.io/collector v0.21.0
	go.uber.org/atomic v1.7.0
	go.uber.org/goleak v1.1.10
	go.uber.org/zap v1.16.0
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba
	google.golang.org/api v0.45.0
	google.golang.org/grpc v1.39.0
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

// additional Cortex or upstream required replaces
replace (
	github.com/bradfitz/gomemcache => github.com/themihai/gomemcache v0.0.0-20180902122335-24332e2d58ab
	github.com/gocql/gocql => github.com/grafana/gocql v0.0.0-20200605141915-ba5dc39ece85
	k8s.io/api => k8s.io/api v0.20.4
	k8s.io/client-go => k8s.io/client-go v0.20.4
)

// Pin to the latest release of grpc-go with GenerateAndRegisterManualResolver
// This function is used by jeagertracing/jaeger, but we can't update jaeger
// without updating the open-telemetry/collector as well
replace google.golang.org/grpc => google.golang.org/grpc v1.33.3

// Pin github.com/soheilhy/cmux to control grpc required version.
// Before v0.1.5 it contained examples in the root folder that imported grpc without a version,
// and therefore were importing grpc latest (which is problematic because we need <v1.29.1)
replace github.com/soheilhy/cmux => github.com/soheilhy/cmux v0.1.5
