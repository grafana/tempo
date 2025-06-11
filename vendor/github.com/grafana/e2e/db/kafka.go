package e2edb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/grafana/e2e"
	"github.com/grafana/e2e/images"
)

type KafkaService struct {
	*e2e.HTTPService
}

func NewKafka() *KafkaService {
	return &KafkaService{
		HTTPService: e2e.NewHTTPService(
			"kafka",
			images.Kafka,
			nil, // No custom command.
			NewKafkaReadinessProbe(9092),
			9092,
		),
	}
}

func (s *KafkaService) Start(networkName, sharedDir string) (err error) {
	// Configures Kafka right before starting it so that we have the networkName to correctly compute
	// the advertised host.
	s.HTTPService.SetEnvVars(map[string]string{
		// Configure Kafka to run in KRaft mode (without Zookeeper).
		"CLUSTER_ID":                      "NqnEdODVKkiLTfJvqd1uqQ==", // A random ID (16 bytes of a base64-encoded UUID).
		"KAFKA_BROKER_ID":                 "1",
		"KAFKA_NODE_ID":                   "1",
		"KAFKA_LISTENERS":                 "PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:29093,PLAINTEXT_HOST://localhost:29092", // Host and port to which Kafka binds to for listening.
		"KAFKA_PROCESS_ROLES":             "broker,controller",
		"KAFKA_CONTROLLER_QUORUM_VOTERS":  "1@kafka:29093",
		"KAFKA_CONTROLLER_LISTENER_NAMES": "CONTROLLER",

		// Configure the advertised host and post.
		"KAFKA_ADVERTISED_LISTENERS": fmt.Sprintf("PLAINTEXT://%s-kafka:9092,PLAINTEXT_HOST://localhost:29092", networkName),

		// RF=1.
		"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR":         "1",
		"KAFKA_TRANSACTION_STATE_LOG_MIN_ISR":            "1",
		"KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR": "1",
		"KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS":         "0",

		// No TLS.
		"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP": "CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT,PLAINTEXT_HOST:PLAINTEXT",
		"KAFKA_INTER_BROKER_LISTENER_NAME":     "PLAINTEXT",

		// Enough partitions for integration tests.
		"KAFKA_NUM_PARTITIONS": "3",

		"LOG4J_ROOT_LOGLEVEL": "WARN",
	})

	return s.HTTPService.Start(networkName, sharedDir)
}

// KafkaReadinessProbe checks readiness by ensure a Kafka broker is up and running.
type KafkaReadinessProbe struct {
	port int
}

func NewKafkaReadinessProbe(port int) *KafkaReadinessProbe {
	return &KafkaReadinessProbe{
		port: port,
	}
}

func (p *KafkaReadinessProbe) Ready(service *e2e.ConcreteService) (err error) {
	const timeout = time.Second

	endpoint := service.Endpoint(p.port)
	if endpoint == "" {
		return fmt.Errorf("cannot get service endpoint for port %d", p.port)
	} else if endpoint == "stopped" {
		return errors.New("service has stopped")
	}

	client, err := kgo.NewClient(kgo.SeedBrokers(endpoint), kgo.DialTimeout(timeout))
	if err != nil {
		return err
	}

	// Ensure we close the client once done.
	defer client.Close()

	admin := kadm.NewClient(client)

	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err = admin.ApiVersions(ctxWithTimeout)
	return err
}
