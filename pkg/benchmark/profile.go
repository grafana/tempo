// Package benchmark profiles a Tempo block for read-path benchmarking.
//
// A profile records what had to be measured from the block: its metadata, the
// authoritative row-group count, and a sample of trace IDs. Everything a run
// derives from those — shards, query windows, steps — is left to the runner, so
// one profile serves every variant of an experiment.
package benchmark

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/grafana/tempo/v3/pkg/util"
	"github.com/grafana/tempo/v3/tempodb/backend"
)

const ProfileSchemaVersion = 1

// traceIDHexLen is the length of a padded 16-byte trace ID in hex.
const traceIDHexLen = 32

const (
	TraceIDModeSample = "sample"
	TraceIDModeAll    = "all"
)

// TraceIDProfile describes which trace IDs a trace-by-ID benchmark can use.
type TraceIDProfile struct {
	Mode    string   `json:"mode"`
	Present []string `json:"present,omitempty"`
	// Absent IDs are verified missing, so the miss path is measured against a
	// real negative rather than a guess.
	Absent []string `json:"absent,omitempty"`
}

type BuildInfo struct {
	TempoVersion string `json:"tempoVersion,omitempty"`
	GitSHA       string `json:"gitSHA,omitempty"`
}

// BlockProfile is what a benchmark needs to know about a block. Block is
// recorded in full because two runs are only comparable if they agree on every
// property of the block except the one under test.
type BlockProfile struct {
	SchemaVersion int                `json:"schemaVersion"`
	GeneratedAt   time.Time          `json:"generatedAt"`
	GeneratedBy   BuildInfo          `json:"generatedBy"`
	Block         *backend.BlockMeta `json:"block"`
	// RowGroups comes from the parquet footer, not Block.TotalRecords, which
	// Tempo's own sharding calls an estimate.
	RowGroups int            `json:"rowGroups"`
	TraceIDs  TraceIDProfile `json:"traceIDs"`
}

func (p *BlockProfile) Write(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(p)
}

func LoadProfile(r io.Reader) (*BlockProfile, error) {
	var p BlockProfile
	if err := json.NewDecoder(r).Decode(&p); err != nil {
		return nil, err
	}
	if p.SchemaVersion != ProfileSchemaVersion {
		return nil, fmt.Errorf("profile schema version %d is not supported, expected %d", p.SchemaVersion, ProfileSchemaVersion)
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &p, nil
}

// Validate runs on load and after profiling, so an unusable profile is
// rejected where it is produced rather than midway through a run.
func (p *BlockProfile) Validate() error {
	if p.Block == nil {
		return errors.New("profile has no block metadata")
	}
	if p.RowGroups <= 0 {
		return fmt.Errorf("profile reports %d row groups", p.RowGroups)
	}

	switch p.TraceIDs.Mode {
	case TraceIDModeSample, TraceIDModeAll:
	default:
		return fmt.Errorf("unknown trace ID mode %q", p.TraceIDs.Mode)
	}
	if p.TraceIDs.Mode == TraceIDModeAll && len(p.TraceIDs.Present) > 0 {
		return errors.New(`trace ID mode is "all" but present IDs are embedded`)
	}

	for _, id := range slices.Concat(p.TraceIDs.Present, p.TraceIDs.Absent) {
		// Require the padded form: HexStringToTraceID accepts a short string,
		// so a truncated ID would otherwise pass and decode to the wrong trace.
		if len(id) != traceIDHexLen {
			return fmt.Errorf("trace ID %q is %d characters, expected %d", id, len(id), traceIDHexLen)
		}
		if _, err := util.HexStringToTraceID(id); err != nil {
			return fmt.Errorf("invalid trace ID %q: %w", id, err)
		}
	}
	return nil
}
