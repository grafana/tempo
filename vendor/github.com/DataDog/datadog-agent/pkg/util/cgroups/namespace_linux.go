// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux
// +build linux

package cgroups

import (
	"os"
	"path/filepath"
	"syscall"
)

// From https://github.com/torvalds/linux/blob/5859a2b1991101d6b978f3feb5325dad39421f29/include/linux/proc_ns.h#L41-L49
// Currently, host namespace inode number are hardcoded, which can be used to detect
// if we're running in host namespace or not (does not work when running in DinD)
const (
	hostCgroupNamespaceInode = 0xEFFFFFFB
)

// IsProcessHostCgroupNamespace compares namespaceID with known, harcoded host PID Namespace inode
// Keeps same signature as `IsProcessHostNetwork` as we may need to change implementation depending on Kernel evolution
func IsProcessHostCgroupNamespace(procPath string, namespaceID uint64) *bool {
	b := namespaceID == hostCgroupNamespaceInode
	return &b
}

// getProcessNamespaceInode performs a stat() call on /proc/<pid>/ns/<namespace>
//
// This has been copied from pkg/util/system/namespace_linux.go's GetProcessNamespaceInode. This
// should ideally live only in pkg/util/system, but currently must be duplicated since the
// pkg/util/cgroups module needs to be isolated from (not require)
// github.com/DataDog/datadog-agent, which is the module where pkg/util/system lives.
//
// This is, in turn, necessary because modules that will use pkg/util/cgroups must not inherit
// github.com/DataDog/datadog-agent as a dependency, since github.com/DataDog/datadog-agent
// requires many replacements in order to work correctly.
//
// For example, pkg/trace needs pkg/util/cgroups, and OpenTelemetry uses pkg/trace. We do not want
// OpenTelemetry to have an indirect dependency on github.com/DataDog/datadog-agent
func getProcessNamespaceInode(procPath string, pid string, namespace string) (uint64, error) {
	nsPath := filepath.Join(procPath, pid, "ns", namespace)
	fi, err := os.Stat(nsPath)
	if err != nil {
		return 0, err
	}

	// We are on linux, casting in safe
	return fi.Sys().(*syscall.Stat_t).Ino, nil
}
