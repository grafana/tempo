package spanmetrics

type Subprocessor int

const (
	Latency Subprocessor = iota
	Count
	Size
)

func (s Subprocessor) String() string {
	switch s {
	case Latency:
		return "span-metrics-latency"
	case Count:
		return "span-metrics-count"
	case Size:
		return "span-metrics-size"
	default:
		return "unsupported"
	}
}
