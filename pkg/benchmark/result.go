package benchmark

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

const ResultSchemaVersion = 1

// CaseResult is what one query shape cost. Latency is per execution;
// everything else is a total over the case, because the counters are
// process-wide and rusage is too coarse to resolve one fast execution.
type CaseResult struct {
	ID    string `json:"id"`
	API   string `json:"api"`
	Query string `json:"query,omitempty"`

	Executions int `json:"executions"`
	// Matched is the total result size. Two runs are only comparable if this
	// agrees, so it is recorded to be checked.
	Matched int64 `json:"matched"`

	WallNs Summary `json:"wallNs"`
	// Samples holds per-execution latencies, thinned to at most MaxSamples.
	Samples []int64 `json:"samples,omitempty"`

	CPUNs      int64 `json:"cpuNs"`
	AllocBytes int64 `json:"allocBytes"`
	AllocCount int64 `json:"allocCount"`

	// Response is what the query APIs reported, summed over the case and keyed
	// by Tempo's own metric names. A map, so a metric added to Tempo is carried
	// through without a change here.
	Response map[string]int64 `json:"response,omitempty"`
	// Process is the Prometheus metrics Tempo emitted while the case ran.
	Process metrics.Process `json:"process,omitempty"`
	// Backend counts and times the object-store calls. Responses do not cover
	// the trace-by-ID path, and a bloom miss returns none at all, so this is
	// measured at the reader instead.
	Backend metrics.ReadStats `json:"backend"`

	Error string `json:"error,omitempty"`
}

// RunEnv records where a run happened, since latencies are only comparable
// within one machine.
type RunEnv struct {
	TempoVersion string `json:"tempoVersion,omitempty"`
	GitSHA       string `json:"gitSHA,omitempty"`
	GoVersion    string `json:"goVersion"`
	GoMaxProcs   int    `json:"goMaxProcs"`
	Hostname     string `json:"hostname,omitempty"`
}

type Result struct {
	SchemaVersion int          `json:"schemaVersion"`
	StartedAt     time.Time    `json:"startedAt"`
	DurationNs    int64        `json:"durationNs"`
	RunEnv        RunEnv       `json:"runEnv"`
	Options       RunOptions   `json:"options"`
	Cases         []CaseResult `json:"cases"`
}

func (r *Result) Write(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func LoadResult(rd io.Reader) (*Result, error) {
	var r Result
	if err := json.NewDecoder(rd).Decode(&r); err != nil {
		return nil, err
	}
	if r.SchemaVersion != ResultSchemaVersion {
		return nil, fmt.Errorf("result schema version %d is not supported, expected %d", r.SchemaVersion, ResultSchemaVersion)
	}
	return &r, nil
}
