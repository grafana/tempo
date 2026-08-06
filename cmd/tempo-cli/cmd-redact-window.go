package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// relDurTerm matches one term of a relative duration, e.g. "7d" or "30m". Supports w/d beyond
// Go's time.ParseDuration (which stops at h), for Grafana-style ranges like now-7d.
var relDurTerm = regexp.MustCompile(`(\d+)(w|d|h|m|s)`)

var relUnitToDuration = map[string]time.Duration{
	"w": 7 * 24 * time.Hour,
	"d": 24 * time.Hour,
	"h": time.Hour,
	"m": time.Minute,
	"s": time.Second,
}

// parseTimeSpec resolves a window bound to absolute unix nanoseconds, relative to now. It accepts:
//   - "" → 0 (unbounded on that side)
//   - "now"
//   - "now-<dur>" / "now+<dur>" where <dur> is a Grafana-style duration (w/d/h/m/s, e.g. 7d, 1h30m)
//   - an absolute RFC3339 timestamp
//
// Relative forms are resolved client-side so the server only ever sees a frozen absolute window.
const nowKeyword = "now"

func parseTimeSpec(spec string, now time.Time) (int64, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return 0, nil
	}
	if spec == nowKeyword {
		return now.UnixNano(), nil
	}
	if rest, ok := strings.CutPrefix(spec, nowKeyword+"-"); ok {
		d, err := parseRelativeDuration(rest)
		if err != nil {
			return 0, err
		}
		return now.Add(-d).UnixNano(), nil
	}
	if rest, ok := strings.CutPrefix(spec, nowKeyword+"+"); ok {
		d, err := parseRelativeDuration(rest)
		if err != nil {
			return 0, err
		}
		return now.Add(d).UnixNano(), nil
	}
	t, err := time.Parse(time.RFC3339, spec)
	if err != nil {
		return 0, fmt.Errorf("invalid time %q: want %q, a relative offset like %q, or an RFC3339 timestamp", spec, nowKeyword, nowKeyword+"-7d")
	}
	return t.UnixNano(), nil
}

// parseRelativeDuration sums Grafana-style duration terms (w/d/h/m/s). Rejects any leftover input
// so typos like "5x" or "" don't silently resolve to zero.
func parseRelativeDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	var total time.Duration
	consumed := 0
	for _, m := range relDurTerm.FindAllStringSubmatchIndex(s, -1) {
		if m[0] != consumed {
			break // a gap means unparseable input before this term
		}
		n, err := strconv.Atoi(s[m[2]:m[3]])
		if err != nil {
			return 0, err
		}
		total += time.Duration(n) * relUnitToDuration[s[m[4]:m[5]]]
		consumed = m[1]
	}
	if consumed != len(s) {
		return 0, fmt.Errorf("invalid duration %q: use w/d/h/m/s (e.g. 7d, 1h30m)", s)
	}
	return total, nil
}
