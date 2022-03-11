module github.com/grafana/tempo/cmd/tempo-serverless/lambda

go 1.16

require (
	github.com/aws/aws-lambda-go v1.28.0
	github.com/gogo/protobuf v1.3.2
	github.com/grafana/tempo v0.0.0-00010101000000-000000000000
)

replace github.com/grafana/tempo => ../../../

replace (
	github.com/bradfitz/gomemcache => github.com/themihai/gomemcache v0.0.0-20180902122335-24332e2d58ab
	k8s.io/api => k8s.io/api v0.20.4
	k8s.io/client-go => k8s.io/client-go v0.20.4
)
