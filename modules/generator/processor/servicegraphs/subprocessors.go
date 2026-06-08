package servicegraphs

import (
	"strings"

	"github.com/grafana/tempo/modules/generator/processor"
)

type Subprocessor int

const (
	Request Subprocessor = iota
	Latency
	ConnectionInfo
)

var SupportedSubprocessors = []Subprocessor{
	Request,
	Latency,
	ConnectionInfo,
}

func (s Subprocessor) String() string {
	switch s {
	case Request:
		return processor.ServiceGraphsRequestName
	case Latency:
		return processor.ServiceGraphsLatencyName
	case ConnectionInfo:
		return processor.ServiceGraphsConnectionInfoName
	default:
		return "unsupported"
	}
}

func ParseSubprocessor(s string) bool {
	for _, p := range SupportedSubprocessors {
		if strings.EqualFold(p.String(), s) {
			return true
		}
	}
	return false
}
