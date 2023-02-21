// +build linux darwin

package processes

import (
	"strings"
	"time"

	"github.com/DataDog/gohai/processes/gops"
)

type ProcessField [7]interface{}

// Return a JSON payload that's compatible with the legacy "processes" resource check
func getProcesses(limit int) ([]interface{}, error) {
	processGroups, err := gops.TopRSSProcessGroups(limit)
	if err != nil {
		return nil, err
	}

	snapData := make([]ProcessField, len(processGroups))

	for i, processGroup := range processGroups {
		processField := ProcessField{
			strings.Join(processGroup.Usernames(), ","),
			0, // pct_cpu, requires two consecutive samples to be computed, so not fetched for now
			processGroup.PctMem(),
			processGroup.VMS(),
			processGroup.RSS(),
			processGroup.Name(),
			len(processGroup.Pids()),
		}
		snapData[i] = processField
	}

	return []interface{}{time.Now().Unix(), snapData}, nil
}
