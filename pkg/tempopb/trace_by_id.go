package tempopb

import "time"

// TimeRange returns the request's start and end, each a zero time.Time when not set.
func (m *TraceByIDRequest) TimeRange() (time.Time, time.Time) {
	var start, end time.Time
	if m.Start != 0 {
		start = time.Unix(int64(m.Start), 0)
	}
	if m.End != 0 {
		end = time.Unix(int64(m.End), 0)
	}
	return start, end
}
