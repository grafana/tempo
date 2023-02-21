//go:build linux || darwin
// +build linux darwin

// Extract the information on running processes from gopsutil
package gops

import (
	"fmt"
	"runtime"

	// 3p
	log "github.com/cihub/seelog"

	// project
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

type ProcessInfo struct {
	PID      int32
	PPID     int32
	Name     string
	RSS      uint64
	PctMem   float64
	VMS      uint64
	Username string
}

// Return a slice of all the processes that are running
func GetProcesses() ([]*ProcessInfo, error) {
	processInfos := make([]*ProcessInfo, 0, 10)

	virtMemStat, err := mem.VirtualMemory()
	if err != nil {
		err = fmt.Errorf("Error fetching system memory stats: %s", err)
		return nil, err
	}
	totalMem := float64(virtMemStat.Total)

	pids, err := process.Pids()
	if err != nil {
		err = fmt.Errorf("Error fetching PIDs: %s", err)
		return nil, err
	}

	for _, pid := range pids {
		p, err := process.NewProcess(pid)
		if err != nil {
			// an error can occur here only if the process has disappeared,
			log.Debugf("Process with pid %d disappeared while scanning: %s", pid, err)
			continue
		}
		processInfo, err := newProcessInfo(p, totalMem)
		if err != nil {
			log.Debugf("Error fetching info for pid %d: %s", pid, err)
			continue
		}

		processInfos = append(processInfos, processInfo)
	}

	// platform-specific post-processing on the collected info
	postProcess(processInfos)

	return processInfos, nil
}

// Make a new ProcessInfo from a Process from gopsutil
func newProcessInfo(p *process.Process, totalMem float64) (*ProcessInfo, error) {
	memInfo, err := p.MemoryInfo()
	if err != nil {
		return nil, err
	}
	pid := p.Pid
	ppid, err := p.Ppid()
	if err != nil {
		return nil, err
	}

	name, err := p.Name()
	if err != nil {
		return nil, err
	}

	pctMem := 100. * float64(memInfo.RSS) / totalMem

	var username string
	if runtime.GOOS != "android" {
		username, err = p.Username()
		if err != nil {
			return nil, err
		}
	}

	return &ProcessInfo{pid, ppid, name, memInfo.RSS, pctMem, memInfo.VMS, username}, nil
}
