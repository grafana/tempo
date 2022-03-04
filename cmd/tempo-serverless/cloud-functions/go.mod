module github.com/grafana/tempo/cmd/tempo-serverless/cloud-functions

go 1.16

require (
	github.com/GoogleCloudPlatform/functions-framework-go v1.3.0
	github.com/gogo/protobuf v1.3.2
	github.com/grafana/tempo v1.2.0-rc.0.0.20211029120833-dee59ebe564c
)

require (
	cloud.google.com/go/functions v1.0.0 // indirect
	cloud.google.com/go/kms v1.1.0 // indirect
)

replace github.com/grafana/tempo => ../../../

replace (
	github.com/bradfitz/gomemcache => github.com/themihai/gomemcache v0.0.0-20180902122335-24332e2d58ab
	k8s.io/api => k8s.io/api v0.20.4
	k8s.io/client-go => k8s.io/client-go v0.20.4
)
