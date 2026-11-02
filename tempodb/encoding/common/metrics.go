package common

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	FindBlocksStageBloomChecked = "bloom_checked"
	FindBlocksStageBloomPass    = "bloom_pass"
	FindBlocksStageIndexPass    = "index_pass"
	FindBlocksStageRead         = "read"
)

var MetricFindBlocks = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "tempodb",
	Name:      "find_blocks_total",
	Help:      "Total number of blocks reaching each stage of a trace by ID lookup. index_pass includes blocks without an index or with index lookup disabled.",
}, []string{"tenant", "stage"})
