package benchmark

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

const ResultSchemaVersion = 1

// CaseResult is what one query shape cost.
//
// Every measurement has the same shape whatever it came from, so a reader does
// not need a rule per source. Keys are "source.name": harness for what the
// benchmark timed itself, backend for object-store traffic, response for what
// a query API reported, process for what Tempo emitted to its registry.
type CaseResult struct {
	ID    string `json:"id"`
	API   string `json:"api"`
	Query string `json:"query,omitempty"`

	Executions int `json:"executions"`
	// Matched is the total result size. Two runs are only comparable if this
	// agrees, so it is recorded to be checked.
	Matched int64 `json:"matched"`

	Metrics metrics.Set `json:"metrics,omitempty"`

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
	SchemaVersion int        `json:"schemaVersion"`
	StartedAt     time.Time  `json:"startedAt"`
	DurationNs    int64      `json:"durationNs"`
	RunEnv        RunEnv     `json:"runEnv"`
	Options       RunOptions `json:"options"`
	// Shards is how many row-group shards the block split into. A case that
	// shards runs one execution per shard per pass, so this is what explains a
	// case's execution count next to Options.Repeat.
	Shards int          `json:"shards"`
	Cases  []CaseResult `json:"cases"`
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
