module github.com/joe-elliott/frigg

go 1.13

require (
	github.com/cortexproject/cortex v0.4.1-0.20191217132644-cd4009e2f8e7
	github.com/go-kit/kit v0.9.0
	github.com/golang/protobuf v1.3.2
	github.com/google/uuid v1.1.1
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645
	github.com/open-telemetry/opentelemetry-proto v0.0.0-20200114203242-839beca37552
	github.com/opentracing/opentracing-go v1.1.0
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.3.0
	github.com/prometheus/common v0.8.0
	github.com/prometheus/prometheus v1.8.2-0.20191126064551-80ba03c67da1 // indirect
	github.com/stretchr/testify v1.4.0
	github.com/weaveworks/common v0.0.0-20200116092424-8f725fc52d8d
	github.com/willf/bitset v1.1.10 // indirect
	github.com/willf/bloom v2.0.3+incompatible
	golang.org/x/net v0.0.0-20191112182307-2180aed22343
	google.golang.org/grpc v1.25.1
	gopkg.in/yaml.v2 v2.2.5
)

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.2.0+incompatible

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
