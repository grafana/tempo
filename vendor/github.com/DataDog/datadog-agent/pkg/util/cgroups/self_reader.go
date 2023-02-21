// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux
// +build linux

package cgroups

import (
	"fmt"
	"strings"

	"github.com/karrick/godirwalk"
)

// SelfCgroupIdentifier is the identifier to be used to get self cgroup
const (
	selfSysPath          = "/sys"
	SelfCgroupIdentifier = "self"
)

type selfReaderFilter struct {
	readerFilter ReaderFilter
	procPath     string
}

func (f *selfReaderFilter) init(inContainer bool, baseController string) error {
	// If we run in a container, /sys/fs/cgroup directly contains the values for our own container
	if inContainer {
		f.readerFilter = func(path, name string) (string, error) {
			return SelfCgroupIdentifier, godirwalk.SkipThis
		}

		return nil
	}

	// If we don't run in a container, we expect to be in host cgroup namespace, otherwise this will not work
	// as the path retrieved from `/proc/self/cgroup` may not be the expected one
	relativePath, err := IdentiferFromCgroupReferences(f.procPath, SelfCgroupIdentifier, baseController, func(path, name string) (string, error) {
		return path, nil
	})
	if err != nil {
		return fmt.Errorf("unable to get self relative cgroup path, err: %w", err)
	}

	f.readerFilter = func(path, name string) (string, error) {
		if strings.HasSuffix(path, relativePath) {
			return SelfCgroupIdentifier, godirwalk.SkipThis
		}

		return "", nil
	}
	return nil
}

func (f *selfReaderFilter) filter(path, name string) (string, error) {
	return f.readerFilter(path, name)
}

// NewSelfReader allows to get current process cgroup stats
// selfProcPath should always be `/proc`, taken as parameter to allow unit testing
func NewSelfReader(selfProcPath string, inContainer bool, opts ...ReaderOption) (*Reader, error) {
	selfFilter := selfReaderFilter{
		procPath: selfProcPath,
	}

	// The self requires to always read `/proc` and `/sys`, even if `/host/sys` is present, for instance.
	// We use HostPrefix = `/sys` to filter out any other mount path.
	// If we're on host with cgroup not mounted at `/sys`, it will not work.
	opts = append(opts, WithReaderFilter(selfFilter.filter), WithProcPath(selfProcPath), WithHostPrefix(selfSysPath))
	selfReader, err := NewReader(opts...)
	if err != nil {
		return nil, err
	}

	// cgroupV2 uses a single, unified, path, no baseController is required
	baseController := selfReader.cgroupV1BaseController
	if selfReader.CgroupVersion() == 2 {
		baseController = ""
	}

	err = selfFilter.init(inContainer, baseController)
	if err != nil {
		return nil, err
	}

	err = selfReader.RefreshCgroups(0)
	if err != nil {
		return nil, err
	}

	return selfReader, nil
}
