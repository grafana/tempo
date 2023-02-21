// +build linux darwin

package gops

import (
	"sort"
)

func minInt(x, y int) int {
	if x < y {
		return x
	}

	return y
}

// Return an ordered slice of the process groups that use the most RSS
func TopRSSProcessGroups(limit int) (ProcessNameGroups, error) {
	procs, err := GetProcesses()
	if err != nil {
		return nil, err
	}

	procGroups := ByRSSDesc{GroupByName(procs)}

	sort.Sort(procGroups)

	return procGroups.ProcessNameGroups[:minInt(limit, len(procGroups.ProcessNameGroups))], nil
}
