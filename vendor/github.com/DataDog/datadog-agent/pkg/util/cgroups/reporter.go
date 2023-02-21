// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux
// +build linux

package cgroups

// Reporter defines some hooks that can be used to track and report on cgroup parsing.
// It's typically useful for logging and debugging.
// Reporter implementations should support concurrent calls.
type Reporter interface {
	// HandleError is called when a non-blocking error has been encountered.
	// e is the encountered error
	HandleError(e error)

	// FileAccessed is called everytime time a file is opened
	FileAccessed(path string)
}

var reporter Reporter

// SetReporter allows to set a Reporter (set to nil to disable)
func SetReporter(r Reporter) {
	reporter = r
}

func reportError(e error) {
	if reporter != nil {
		reporter.HandleError(e)
	}
}

func reportFileAccessed(path string) {
	if reporter != nil {
		reporter.FileAccessed(path)
	}
}
