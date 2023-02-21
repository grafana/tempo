// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux
// +build linux

package cgroups

import (
	"fmt"
	"path/filepath"

	"github.com/karrick/godirwalk"
)

const (
	defaultBaseController = "memory"
)

type readerV1 struct {
	mountPoints map[string]string
	cgroupRoot  string
	filter      ReaderFilter
	pidMapper   pidMapper
}

func newReaderV1(procPath string, mountPoints map[string]string, baseController string, filter ReaderFilter) (*readerV1, error) {
	if baseController == "" {
		baseController = defaultBaseController
	}

	if path, found := mountPoints[baseController]; found {
		return &readerV1{
			mountPoints: mountPoints,
			cgroupRoot:  path,
			filter:      filter,
			pidMapper:   getPidMapper(procPath, path, baseController, filter),
		}, nil
	}

	return nil, &InvalidInputError{Desc: fmt.Sprintf("cannot create cgroup readerv1: %s controller not found", baseController)}
}

func (r *readerV1) parseCgroups() (map[string]Cgroup, error) {
	res := make(map[string]Cgroup)

	err := godirwalk.Walk(r.cgroupRoot, &godirwalk.Options{
		AllowNonDirectory: true,
		Unsorted:          true,
		Callback: func(fullPath string, de *godirwalk.Dirent) error {
			if de.IsDir() {
				id, err := r.filter(fullPath, de.Name())
				if id != "" {
					relPath, err := filepath.Rel(r.cgroupRoot, fullPath)
					if err != nil {
						return err
					}

					res[id] = newCgroupV1(id, relPath, r.mountPoints, r.pidMapper)
				}

				return err
			}

			return nil
		},
	})

	return res, err
}
