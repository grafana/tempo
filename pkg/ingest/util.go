package ingest

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/twmb/franz-go/pkg/kerr"
)

const (
	// unknownBroker duplicates a constant from franz-go because it isn't exported.
	unknownBroker = "unknown broker"
)

// Regular expression used to parse the ingester numeric ID.
var ingesterIDRegexp = regexp.MustCompile("-([0-9]+)$")

// IngesterPartitionID returns the partition ID owner the the given ingester.
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
