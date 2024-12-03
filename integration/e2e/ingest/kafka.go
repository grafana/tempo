package ingest

import (
	"github.com/grafana/e2e"
)

const kafkaImage = "confluentinc/cp-kafka:7.7.1"

func NewKafka() *e2e.HTTPService {
	envVars := map[string]string{
		"CLUSTER_ID":                             "zH1GDqcNTzGMDCXm5VZQdg",
		"KAFKA_BROKER_ID":                        "1",
		"KAFKA_NUM_PARTITIONS":                   "1000",
		"KAFKA_PROCESS_ROLES":                    "broker,controller",
		"KAFKA_LISTENERS":                        "PLAINTEXT://:9092,CONTROLLER://:9093,PLAINTEXT_HOST://:29092",
		"KAFKA_ADVERTISED_LISTENERS":             "PLAINTEXT://kafka:9092,PLAINTEXT_HOST://localhost:29092",
		"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP":   "PLAINTEXT:PLAINTEXT,CONTROLLER:PLAINTEXT,PLAINTEXT_HOST:PLAINTEXT",
		"KAFKA_INTER_BROKER_LISTENER_NAME":       "PLAINTEXT",
		"KAFKA_CONTROLLER_LISTENER_NAMES":        "CONTROLLER",
		"KAFKA_CONTROLLER_QUORUM_VOTERS":         "1@kafka:9093",
		"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR": "1",
		"KAFKA_LOG_RETENTION_CHECK_INTERVAL_MS":  "10000",
	}

	service := e2e.NewConcreteService(
		"kafka",
		kafkaImage,
		e2e.NewCommand("/etc/confluent/docker/run"),
		// e2e.NewCmdReadinessProbe(e2e.NewCommand("kafka-topics", "--bootstrap-server", "broker:29092", "--list")),
		e2e.NewCmdReadinessProbe(e2e.NewCommand("sh", "-c", "nc -z localhost 9092 || exit 1")), // TODO: A bit unstable, sometimes it fails
		9092,
		29092,
	)

	service.SetEnvVars(envVars)

	httpService := &e2e.HTTPService{
		ConcreteService: service,
	}

	return httpService
}
