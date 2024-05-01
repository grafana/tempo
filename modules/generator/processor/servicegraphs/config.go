package servicegraphs

import (
	"flag"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	Name = "service-graphs"
)

type Config struct {
	// Wait is the value to wait for an edge to be completed
	Wait time.Duration `yaml:"wait"`
	// MaxItems is the amount of edges that will be stored in the storeMap
	MaxItems int `yaml:"max_items"`

	// Workers is the amount of workers that will be used to process the edges
	Workers int `yaml:"workers"`

	// Buckets for latency histogram in seconds.
	HistogramBuckets []float64 `yaml:"histogram_buckets"`

	// Additional dimensions (labels) to be added to the metric along with the default ones.
	// If client and server spans have the same attribute and EnableClientServerPrefix is not enabled,
	// behaviour is undetermined (either value could get used)
	Dimensions []string `yaml:"dimensions"`

	// If enabled, additional dimensions (labels) will be prefixed with either
	// "client_" or "server_" depending on the span kind. Up to two labels will be added
	// per dimension.
	EnableClientServerPrefix bool `yaml:"enable_client_server_prefix"`

	// PeerAttributes are attributes that will be used to create a peer edge
	// Attributes are searched in the order they are provided
	PeerAttributes []string `yaml:"peer_attributes"`

	// If enabled attribute value will be used for metric calculation
	SpanMultiplierKey string `yaml:"span_multiplier_key"`

	// EnableVirtualNodeLabel enables additional labels for uninstrumented services
	EnableVirtualNodeLabel bool `yaml:"enable_virtual_node_label"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(string, *flag.FlagSet) {
	cfg.Wait = 10 * time.Second
	cfg.MaxItems = 10_000
	cfg.Workers = 10
	// TODO: Revisit this default value.
	cfg.HistogramBuckets = prometheus.ExponentialBuckets(0.1, 2, 8)

	peerAttr := make([]string, 0, len(defaultPeerAttributes))
	for _, attr := range defaultPeerAttributes {
		peerAttr = append(peerAttr, string(attr))
	}
	cfg.PeerAttributes = peerAttr
}
