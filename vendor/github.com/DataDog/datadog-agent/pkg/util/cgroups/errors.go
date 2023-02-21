// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux
// +build linux

package cgroups

import "fmt"

// InvalidInputError is returned when an input parameter has an invalid value (typically, passing a nil pointer)
type InvalidInputError struct {
	Desc string
}

func (e *InvalidInputError) Error() string {
	return "invalid input: " + e.Desc
}

// ControllerNotFoundError is returned when trying to access a cgroup controller that's not visible in /proc/mount (cgroup v1 only)
type ControllerNotFoundError struct {
	Controller string
}

func (e *ControllerNotFoundError) Error() string {
	return "mount point for cgroup controller not found: " + e.Controller
}

// Is returns whether two ControllerNotFoundError targets the same controller
func (e *ControllerNotFoundError) Is(target error) bool {
	t, ok := target.(*ControllerNotFoundError)
	if !ok {
		return false
	}
	return e.Controller == t.Controller
}

// FileSystemError is returned when an error occurs when reading cgroup filesystem
type FileSystemError struct {
	FilePath string
	Err      error
}

func newFileSystemError(path string, err error) *FileSystemError {
	return &FileSystemError{
		FilePath: path,
		Err:      err,
	}
}

func (e *FileSystemError) Error() string {
	return fmt.Sprintf("fs error, path: %s, err: %s", e.FilePath, e.Err.Error())
}

func (e *FileSystemError) Unwrap() error {
	return e.Err
}

// ValueError is reported when the parser encounters an unexpected content in a cgroup file.
// This error is not blocking and will only be retported through the reporter.
// Seeing this error means that there's either in bug in the parsing code (most likely), unsupported new cgroup feature, Kernel bug (less likely)
type ValueError struct {
	Data string
	Err  error
}

func newValueError(data string, err error) *ValueError {
	return &ValueError{
		Data: data,
		Err:  err,
	}
}

func (e *ValueError) Error() string {
	return fmt.Sprintf("value error, data: '%s', err: %s", e.Data, e.Err.Error())
}

func (e *ValueError) Unwrap() error {
	return e.Err
}
