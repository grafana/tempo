package benchmark

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

const ResultSchemaVersion = 1

// CaseResult is what one query shape cost.
//
// Latency is per execution; everything else is a total over the case, because
// process-wide counters cannot be attributed to a single execution and rusage
// is too coarse to resolve a fast one.
type CaseResult struct {
	ID    string `json:"id"`
	API   string `json:"api"`
	Query string `json:"query,omitempty"`

	Executions int `json:"executions"`
	// Matched is the total result size. A comparison between two runs is only
	// meaningful if this agrees, so it is recorded to be checked.
	Matched int64 `json:"matched"`

	WallNs Summary `json:"wallNs"`
	// Samples holds per-execution latencies, thinned to at most MaxSamples.
	Samples []int64 `json:"samples,omitempty"`

	CPUNs      int64 `json:"cpuNs"`
	AllocBytes int64 `json:"allocBytes"`
	AllocCount int64 `json:"allocCount"`

	InspectedBytes int64     `json:"inspectedBytes"`
	Backend        readStats `json:"backend"`

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

// ProfileRef identifies the profile a run consumed, so a result cannot be
// silently paired with a different block.
type ProfileRef struct {
	SchemaVersion int    `json:"schemaVersion"`
	Format        string `json:"format"`
	BlockID       string `json:"blockID"`
	TenantID      string `json:"tenantID"`
	RowGroups     int    `json:"rowGroups"`
	TraceIDMode   string `json:"traceIDMode"`
	PresentIDs    int    `json:"presentIDs"`
	AbsentIDs     int    `json:"absentIDs"`
}

type Result struct {
	SchemaVersion int          `json:"schemaVersion"`
	StartedAt     time.Time    `json:"startedAt"`
	DurationNs    int64        `json:"durationNs"`
	RunEnv        RunEnv       `json:"runEnv"`
	Profile       ProfileRef   `json:"profile"`
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
