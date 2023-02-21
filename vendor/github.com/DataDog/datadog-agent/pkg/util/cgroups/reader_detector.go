// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux
// +build linux

package cgroups

import (
	"bufio"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	cgroupV2Key = "cgroupv2"
)

func discoverCgroupMountPoints(hostPrefix, procFsPath string) (map[string]string, error) {
	f, err := os.Open(filepath.Join(procFsPath, "/mounts"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	mountPointsv1 := make(map[string]string)
	var mountPointsv2 string

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()

		tokens := strings.Fields(line)
		if len(tokens) >= 3 {
			// Check if the filesystem type is 'cgroup' or 'cgroup2'
			fsType := tokens[2]
			if !strings.HasPrefix(fsType, "cgroup") {
				continue
			}

			cgroupPath := tokens[1]
			// Mounts are duplicated when /sys/fs is mounted inside a container (like /host/sys/fs)
			if !strings.HasPrefix(cgroupPath, hostPrefix) {
				continue
			}

			if fsType == "cgroup" {
				// Target can be comma-separate values like cpu,cpuacct
				tsp := strings.Split(path.Base(cgroupPath), ",")
				for _, target := range tsp {
					// In case multiple paths are mounted for a single controller, take the shortest one
					previousPath := mountPointsv1[target]
					if previousPath == "" || len(cgroupPath) < len(previousPath) {
						mountPointsv1[target] = cgroupPath
					}
				}
			} else if tokens[2] == "cgroup2" {
				mountPointsv2 = cgroupPath
			}
		}
	}

	if len(mountPointsv1) == 0 && mountPointsv2 != "" {
		return map[string]string{cgroupV2Key: mountPointsv2}, nil
	}

	return mountPointsv1, nil
}

func isCgroup1(cgroupMountPoints map[string]string) bool {
	return len(cgroupMountPoints) > 1
}

func isCgroup2(cgroupMountPoints map[string]string) bool {
	if _, found := cgroupMountPoints[cgroupV2Key]; found && len(cgroupMountPoints) == 1 {
		return true
	}

	return false
}
