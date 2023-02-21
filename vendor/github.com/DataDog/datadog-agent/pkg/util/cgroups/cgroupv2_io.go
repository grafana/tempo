// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux
// +build linux

package cgroups

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/DataDog/datadog-agent/pkg/util/pointer"
)

func (c *cgroupV2) GetIOStats(stats *IOStats) error {
	if stats == nil {
		return &InvalidInputError{Desc: "input stats cannot be nil"}
	}

	if !c.controllerActivated("io") {
		return &ControllerNotFoundError{Controller: "io"}
	}

	stats.Devices = make(map[string]DeviceIOStats)
	stats.ReadBytes = pointer.Ptr(uint64(0))
	stats.WriteBytes = pointer.Ptr(uint64(0))
	stats.ReadOperations = pointer.Ptr(uint64(0))
	stats.WriteOperations = pointer.Ptr(uint64(0))

	if err := parseColumnStats(c.fr, c.pathFor("io.stat"), parseV2IOFn(stats)); err != nil {
		reportError(err)
	}

	if err := parseColumnStats(c.fr, c.pathFor("io.max"), parseV2IOFn(stats)); err != nil {
		reportError(err)
	}

	if err := parsePSI(c.fr, c.pathFor("io.pressure"), &stats.PSISome, &stats.PSIFull); err != nil {
		reportError(err)
	}

	// In case we did not get any device info, clearing everything
	if len(stats.Devices) == 0 {
		stats.Devices = nil
		stats.ReadBytes = nil
		stats.WriteBytes = nil
		stats.ReadOperations = nil
		stats.WriteOperations = nil
	}

	return nil
}

// format for io.stat "259:0 rbytes=278528 wbytes=9700089856 rios=6 wios=2289428 dbytes=0 dios=0"
// format for io.max "8:16 rbps=2097152 wbps=max riops=max wiops=120"
func parseV2IOFn(stats *IOStats) func([]string) error {
	return func(fields []string) error {
		if len(fields) < 2 {
			reportError(newValueError("", fmt.Errorf("malformed line fields: '%v'", fields)))
		}

		written := false
		device := stats.Devices[fields[0]]

		for i := 1; i < len(fields); i++ {
			parts := strings.Split(fields[i], "=")
			if len(parts) != 2 {
				reportError(newValueError("", fmt.Errorf("malformed line fields: '%v'", fields)))
				continue
			}

			// max appears in io.max, it means no limit, not reporting in this case
			if parts[1] == "max" {
				continue
			}

			val, err := strconv.ParseUint(parts[1], 10, 64)
			if err != nil {
				reportError(newValueError("", fmt.Errorf("unexpected format in io file, part: %d, content: %v", i, fields[i])))
				continue
			}

			switch parts[0] {
			case "rbytes":
				written = true
				stats.ReadBytes = pointer.Ptr(*stats.ReadBytes + val)
				device.ReadBytes = &val
			case "wbytes":
				written = true
				stats.WriteBytes = pointer.Ptr(*stats.WriteBytes + val)
				device.WriteBytes = &val
			case "rios":
				written = true
				stats.ReadOperations = pointer.Ptr(*stats.ReadOperations + val)
				device.ReadOperations = &val
			case "wios":
				written = true
				stats.WriteOperations = pointer.Ptr(*stats.WriteOperations + val)
				device.WriteOperations = &val
			case "rbps":
				written = true
				device.ReadBytesLimit = &val
			case "wbps":
				written = true
				device.WriteBytesLimit = &val
			case "riops":
				written = true
				device.ReadOperationsLimit = &val
			case "wiops":
				written = true
				device.WriteOperationsLimit = &val
			}
		}

		if written {
			stats.Devices[fields[0]] = device
		}

		return nil
	}
}
