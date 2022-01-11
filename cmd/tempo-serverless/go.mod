module github.com/grafana/tempo/cmd/tempo-serverless

go 1.16

require (
	github.com/GoogleCloudPlatform/functions-framework-go v1.3.0
	github.com/gogo/protobuf v1.3.2
	github.com/grafana/tempo v1.2.0-rc.0.0.20211029120833-dee59ebe564c
	github.com/mitchellh/mapstructure v1.4.3
	github.com/spf13/viper v1.9.0
	gopkg.in/yaml.v2 v2.4.0
)

require (
	cloud.google.com/go/functions v1.0.0 // indirect
	cloud.google.com/go/kms v1.1.0 // indirect
	github.com/google/uuid v1.3.0
	github.com/grafana/dskit v0.0.0-20211021180445-3bd016e9d7f1
	github.com/weaveworks/common v0.0.0-20210913144402-035033b78a78
)

replace github.com/grafana/tempo => ../../

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
replace google.golang.org/grpc => google.golang.org/grpc v1.38.0
