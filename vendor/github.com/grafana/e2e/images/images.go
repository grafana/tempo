// Package images provides Docker image version constants for e2e test services.
package images

var (
	// These are variables so that they can be modified.
	Memcached        = "memcached:1.6.12"
	Redis            = "redis:7.0.7"
	Minio            = "minio/minio:RELEASE.2021-10-13T00-23-17Z"
	KES              = "minio/kes:v0.17.1"
	Consul           = "consul:1.8.15"
	ETCD             = "gcr.io/etcd-development/etcd:v3.4.13"
	DynamoDB         = "amazon/dynamodb-local:1.17.0"
	BigtableEmulator = "shopify/bigtable-emulator:0.1.0"
	Cassandra        = "rinscy/cassandra:3.11.0"
	SwiftEmulator    = "bouncestorage/swift-aio:55ba4331"
	Kafka            = "apache/kafka:4.0.0"
	Dex              = "dexidp/dex:v2.41.1"
)
