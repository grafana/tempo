package compare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"

	"github.com/grafana/tempo/v3/pkg/benchmark"
)

// Kind says what a setting is about, which decides how a difference in it
// reads.
type Kind int

const (
	// Setup is what an experiment varies: the run options.
	Setup Kind = iota
	// Build is the version and commit a run was built from, varied on purpose
	// just as options are.
	Build
	// Environment is where a run happened. Latencies from two environments are
	// hard to compare, so a difference here is flagged.
	Environment
	// Derived follows from other settings, like the shard count from the bytes
	// per request.
	Derived
)

// defaultValue is what an option left at zero reads as, since zero means
// Tempo's default and is omitted from a result.
const defaultValue = "default"

// byteOptions are the run options that are sizes in bytes, written in binary
// units because that is how they are passed on the command line.
var byteOptions = map[string]bool{
	"targetBytesPerRequest": true,
	"readBufferSize":        true,
	"chunkSizeBytes":        true,
}

// Setting is one thing a run was set up with, under its name in the result.
type Setting struct {
	Field string
	Value string
	Kind  Kind

	// number is the value as a number, for putting runs in its order. It is
	// not set for a value that is not a number, or an option left at default.
	number   float64
	isNumber bool
}

func (s Setting) String() string {
	return s.Field + " " + s.Value
}

// Change is a setting a run has that the baseline does not.
type Change struct {
	Field    string
	From, To string
	Kind     Kind
}

func (ch Change) String() string {
	s := fmt.Sprintf("%s %s → %s", ch.Field, ch.From, ch.To)
	if ch.Kind == Environment {
		return "⚠ " + s
	}
	return s
}

// Detail is one piece of how a run is described, with the kind of setting it is
// about so a caller can style it.
type Detail struct {
	Text string
	Kind Kind
}

// Describe says how a run was set up against the baseline: what it changed, or,
// for the baseline itself, what the others are measured from.
func (c *Comparison) Describe(run, baseline int) []Detail {
	if run == baseline {
		details := []Detail{{Text: "baseline", Kind: Setup}}
		for _, s := range c.baselineSettings(baseline) {
			details = append(details, Detail{Text: s.String(), Kind: Derived})
		}
		return details
	}

	chs := changes(c.settings[baseline], c.settings[run])
	if len(chs) == 0 {
		return []Detail{{Text: "same setup as the baseline", Kind: Derived}}
	}
	details := make([]Detail, len(chs))
	for i, ch := range chs {
		details[i] = Detail{Text: ch.String(), Kind: ch.Kind}
	}
	return details
}

// DetailText runs a description's details together as plain text.
func DetailText(details []Detail) string {
	texts := make([]string, len(details))
	for i, d := range details {
		texts[i] = d.Text
	}
	return strings.Join(texts, " · ")
}

// baselineSettings are what the others are measured from: the baseline's value
// of every setting that varies, then its build and where it ran.
func (c *Comparison) baselineSettings(baseline int) []Setting {
	fields := slices.Clone(c.Varying)
	for _, f := range []string{"gitSHA", "goVersion", "goMaxProcs", "hostname"} {
		if !slices.Contains(fields, f) {
			fields = append(fields, f)
		}
	}
	out := make([]Setting, len(fields))
	for i, f := range fields {
		out[i] = lookup(c.settings[baseline], f)
	}
	return out
}

// settings lists what a run was set up with: its options, in the order they
// are declared, then its build, where it ran, and what followed from those.
func settings(r *benchmark.Result) ([]Setting, error) {
	options, err := optionSettings(r.Options)
	if err != nil {
		return nil, err
	}
	orUnknown := func(s string) string {
		if s == "" {
			return "unknown"
		}
		return s
	}
	env := r.RunEnv
	return append(options,
		Setting{Field: "tempoVersion", Value: orUnknown(env.TempoVersion), Kind: Build},
		Setting{Field: "gitSHA", Value: orUnknown(shortSHA(env.GitSHA)), Kind: Build},
		Setting{Field: "goVersion", Value: orUnknown(env.GoVersion), Kind: Environment},
		numberSetting("goMaxProcs", env.GoMaxProcs, Environment),
		Setting{Field: "hostname", Value: orUnknown(env.Hostname), Kind: Environment},
		numberSetting("shards", r.Shards, Derived),
	), nil
}

// optionSettings reads the options through their JSON form, so a new option
// shows up without a change here. Decoding token by token keeps the order they
// are declared in.
func optionSettings(opts benchmark.RunOptions) ([]Setting, error) {
	b, err := json.Marshal(opts)
	if err != nil {
		return nil, fmt.Errorf("encoding run options: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	if _, err := dec.Token(); err != nil {
		return nil, fmt.Errorf("decoding run options: %w", err)
	}

	var out []Setting
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("decoding run options: %w", err)
		}
		key, ok := tok.(string)
		if !ok {
			return nil, fmt.Errorf("decoding run options: unexpected %v", tok)
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, fmt.Errorf("decoding run option %s: %w", key, err)
		}
		out = append(out, optionSetting(key, raw))
	}
	return out, nil
}

func optionSetting(key string, raw json.RawMessage) Setting {
	s := Setting{Field: key, Value: string(raw), Kind: Setup}
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		s.Value = str
		return s
	}
	n, err := strconv.ParseFloat(string(raw), 64)
	if err != nil {
		return s
	}
	s.number, s.isNumber = n, true
	if byteOptions[key] && n >= 0 && n == float64(uint64(n)) {
		s.Value = formatIBytes(uint64(n))
	}
	return s
}

func numberSetting(field string, n int, kind Kind) Setting {
	return Setting{Field: field, Value: strconv.Itoa(n), Kind: kind, number: float64(n), isNumber: true}
}

// formatIBytes writes a size exactly when it is a whole number of binary units,
// as sizes passed on the command line usually are.
func formatIBytes(n uint64) string {
	units := []struct {
		shift  uint
		suffix string
	}{{30, "GiB"}, {20, "MiB"}, {10, "KiB"}}
	for _, u := range units {
		if n >= 1<<u.shift && n%(1<<u.shift) == 0 {
			return fmt.Sprintf("%d%s", n>>u.shift, u.suffix)
		}
	}
	if n < 1<<10 {
		return fmt.Sprintf("%dB", n)
	}
	return strings.ReplaceAll(humanize.IBytes(n), " ", "")
}

func shortSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

// fieldsOf is every field any of the runs has, grouped by kind, each group in
// the order the fields first appear.
func fieldsOf(all ...[]Setting) []Setting {
	var fields []Setting
	seen := map[string]bool{}
	for _, ss := range all {
		for _, s := range ss {
			if !seen[s.Field] {
				seen[s.Field] = true
				fields = append(fields, Setting{Field: s.Field, Kind: s.Kind})
			}
		}
	}
	slices.SortStableFunc(fields, func(a, b Setting) int { return int(a.Kind) - int(b.Kind) })
	return fields
}

// lookup finds a field among a run's settings. A field it lacks is an option it
// left at default, since everything else is always recorded.
func lookup(ss []Setting, field string) Setting {
	for _, s := range ss {
		if s.Field == field {
			return s
		}
	}
	return Setting{Field: field, Value: defaultValue, Kind: Setup}
}

// changes lists what run was set up with differently from base.
func changes(base, run []Setting) []Change {
	var out []Change
	for _, f := range fieldsOf(base, run) {
		from, to := lookup(base, f.Field), lookup(run, f.Field)
		if from.Value != to.Value {
			out = append(out, Change{Field: f.Field, From: from.Value, To: to.Value, Kind: f.Kind})
		}
	}
	return out
}

// varying lists the fields whose value is not the same in every run.
func varying(all [][]Setting) []Setting {
	var out []Setting
	for _, f := range fieldsOf(all...) {
		first := lookup(all[0], f.Field).Value
		for _, ss := range all[1:] {
			if lookup(ss, f.Field).Value != first {
				out = append(out, f)
				break
			}
		}
	}
	return out
}
