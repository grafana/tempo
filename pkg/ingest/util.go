package ingest

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/backoff"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	// unknownBroker duplicates a constant from franz-go because it isn't exported.
	unknownBroker = "unknown broker"
)

// Regular expression used to parse the ingester numeric ID.
var ingesterIDRegexp = regexp.MustCompile("-([0-9]+)$")

// IngesterPartitionID returns the partition ID owner of the given ingester.
func IngesterPartitionID(ingesterID string) (int32, error) {
	match := ingesterIDRegexp.FindStringSubmatch(ingesterID)
	if len(match) == 0 {
		return 0, fmt.Errorf("ingester ID %s doesn't match regular expression %q", ingesterID, ingesterIDRegexp.String())
	}

	// Parse the ingester sequence number.
	ingesterSeq, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, fmt.Errorf("no ingester sequence number in ingester ID %s", ingesterID)
	}

	return int32(ingesterSeq), nil
}

func LiveStoreConsumerGroupID(instanceID string) (string, error) {
	match := ingesterIDRegexp.FindStringSubmatch(instanceID)
	if len(match) == 0 {
		return "", fmt.Errorf("instance ID %s doesn't match regular expression %q", instanceID, ingesterIDRegexp.String())
	}

	// Extract everything before the numeric suffix
	prefixEnd := len(instanceID) - len(match[0])
	return instanceID[:prefixEnd], nil
}

// It retry until Kafka broker is ready
func WaitForKafkaBroker(ctx context.Context, c *kgo.Client, l log.Logger) error {
	boff := backoff.New(ctx, backoff.Config{
		MinBackoff: 100 * time.Millisecond,
		MaxBackoff: time.Minute, // If there is a network hiccup, we prefer to wait longer retrying, than fail the service.
		MaxRetries: 10,
	})

	for boff.Ongoing() {
		err := c.Ping(ctx)
		if err == nil {
			break
		}
		level.Warn(l).Log("msg", "ping kafka; will retry", "err", err)
		boff.Wait()
	}
	if err := boff.ErrCause(); err != nil {
		return fmt.Errorf("kafka broker not ready after %d retries: %w", boff.NumRetries(), err)
	}
	return nil
}

func HandleKafkaError(err error, forceMetadataRefresh func()) {
	if err == nil {
		return
	}
	errString := err.Error()

	switch {
	// We're asking a broker which is no longer the leader. For a partition. We should refresh our metadata and try again.
	case errors.Is(err, kerr.NotLeaderForPartition):
		forceMetadataRefresh()
	// Maybe the replica hasn't replicated the log yet, or it is no longer a replica for this partition.
	// We should refresh and try again with a leader or replica which is up to date.
	case errors.Is(err, kerr.ReplicaNotAvailable):
		forceMetadataRefresh()
	// Maybe there's an ongoing election. We should refresh our metadata and try again with a leader in the current epoch.
	case errors.Is(err, kerr.UnknownLeaderEpoch):
		forceMetadataRefresh()
	case errors.Is(err, kerr.LeaderNotAvailable):
		forceMetadataRefresh()
	case errors.Is(err, kerr.BrokerNotAvailable):
		forceMetadataRefresh()
	// Topic or partition doesn't exist - metadata refresh needed to get current topology
	case errors.Is(err, kerr.UnknownTopicOrPartition):
		forceMetadataRefresh()
	// Network connectivity issues - broker may have changed
	case errors.Is(err, kerr.NetworkException):
		forceMetadataRefresh()
	// Coordinator moved to different broker
	case errors.Is(err, kerr.NotCoordinator):
		forceMetadataRefresh()
	case strings.Contains(errString, "i/o timeout"):
		forceMetadataRefresh()
	case strings.Contains(errString, unknownBroker):
		forceMetadataRefresh()
		// The client's metadata refreshed after we called Broker(). It should already be refreshed, so we can retry immediately.
	}
}
