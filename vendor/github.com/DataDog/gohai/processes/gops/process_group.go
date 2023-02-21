// +build linux darwin

package gops

import (
	"sort"
)

// Represents a group of processes, grouped by name
type ProcessNameGroup struct {
	pids      []int32
	rss       uint64
	pctMem    float64
	vms       uint64
	name      string
	usernames map[string]bool
}

type ProcessNameGroups []*ProcessNameGroup

func (pg *ProcessNameGroup) Pids() []int32 {
	return pg.pids
}

func (pg *ProcessNameGroup) Name() string {
	return pg.name
}

func (pg *ProcessNameGroup) RSS() uint64 {
	return pg.rss
}

func (pg *ProcessNameGroup) PctMem() float64 {
	return pg.pctMem
}

func (pg *ProcessNameGroup) VMS() uint64 {
	return pg.vms
}

// Return a slice of the usernames, sorted alphabetically
func (pg *ProcessNameGroup) Usernames() []string {
	var usernameStringSlice sort.StringSlice
	for username := range pg.usernames {
		usernameStringSlice = append(usernameStringSlice, username)
	}

	sort.Sort(usernameStringSlice)

	return []string(usernameStringSlice)
}

func NewProcessNameGroup() *ProcessNameGroup {
	processNameGroup := new(ProcessNameGroup)
	processNameGroup.usernames = make(map[string]bool)

	return processNameGroup
}

// Group the processInfos by name and return a slice of ProcessNameGroup
func GroupByName(processInfos []*ProcessInfo) ProcessNameGroups {
	groupIndexByName := make(map[string]int)
	processNameGroups := make(ProcessNameGroups, 0, 10)

	for _, processInfo := range processInfos {
		if _, ok := groupIndexByName[processInfo.Name]; !ok {
			processNameGroups = append(processNameGroups, NewProcessNameGroup())
			groupIndexByName[processInfo.Name] = len(processNameGroups) - 1
		}

		processNameGroups[groupIndexByName[processInfo.Name]].add(processInfo)
	}

	return processNameGroups
}

func (pg *ProcessNameGroup) add(p *ProcessInfo) {
	pg.pids = append(pg.pids, p.PID)
	if pg.name == "" {
		pg.name = p.Name
	}
	pg.rss += p.RSS
	pg.pctMem += p.PctMem
	pg.vms += p.VMS
	pg.usernames[p.Username] = true
}

// Sort slices of process groups
func (s ProcessNameGroups) Len() int {
	return len(s)
}

func (s ProcessNameGroups) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type ByRSSDesc struct {
	ProcessNameGroups
}

func (s ByRSSDesc) Less(i, j int) bool {
	return s.ProcessNameGroups[i].RSS() > s.ProcessNameGroups[j].RSS()
}
