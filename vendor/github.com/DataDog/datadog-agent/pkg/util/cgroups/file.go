// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux
// +build linux

package cgroups

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const (
	spaceSeparator = " "
)

var defaultFileReader = &osFileReader{}

type fileReader interface {
	open(path string) (file, error)
}

type file interface {
	io.Reader
	Close() error
}

type osFileReader struct{}

func (fr *osFileReader) open(path string) (file, error) {
	reportFileAccessed(path)
	return os.Open(path)
}

type stopParsingError struct{}

func (e *stopParsingError) Error() string {
	return "stopping file parsing" // should never be used
}

// returning an error will stop parsing and return the error
// with the exception of stopParsingError that will return without error
type parser func(string) error

func parseFile(fr fileReader, path string, p parser) error {
	f, err := fr.open(path)
	if err != nil {
		return newFileSystemError(path, err)
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		err := p(line)
		if err != nil {
			if errors.Is(err, &stopParsingError{}) {
				return nil
			}

			return err
		}
	}

	return nil
}

func parseSingleSignedStat(fr fileReader, path string, val **int64) error {
	return parseFile(fr, path, func(line string) error {
		// handle cgroupv2 max value, we usually consider max == no value (limit)
		if line == "max" {
			return &stopParsingError{}
		}

		value, err := strconv.ParseInt(line, 10, 64)
		if err != nil {
			return newValueError(line, err)
		}
		*val = &value
		return &stopParsingError{}
	})
}

func parseSingleUnsignedStat(fr fileReader, path string, val **uint64) error {
	return parseFile(fr, path, func(line string) error {
		// handle cgroupv2 max value, we usually consider max == no value (limit)
		if line == "max" {
			return &stopParsingError{}
		}

		value, err := strconv.ParseUint(line, 10, 64)
		if err != nil {
			return newValueError(line, err)
		}
		*val = &value
		return &stopParsingError{}
	})
}

func parseColumnStats(fr fileReader, path string, valueParser func([]string) error) error {
	err := parseFile(fr, path, func(line string) error {
		splits := strings.Fields(line)
		return valueParser(splits)
	})

	return err
}

// columns are 0-indexed, we skip malformed lines
func parse2ColumnStats(fr fileReader, path string, keyColumn, valueColumn int, valueParser func(string, string) error) error {
	lastIdx := valueColumn
	if keyColumn > lastIdx {
		lastIdx = keyColumn
	}

	err := parseFile(fr, path, func(line string) error {
		splits := strings.SplitN(line, spaceSeparator, lastIdx+1)
		if len(splits) <= lastIdx {
			return nil
		}

		return valueParser(splits[keyColumn], splits[valueColumn])
	})

	return err
}

// format is "some avg10=0.00 avg60=0.00 avg300=0.00 total=0"
func parsePSI(fr fileReader, path string, somePsi, fullPsi *PSIStats) error {
	return parseColumnStats(fr, path, func(fields []string) error {
		if len(fields) != 5 {
			reportError(newValueError("", fmt.Errorf("unexpected format for psi file at: %s, line content: %v", path, fields)))
			return nil
		}

		var psiStats *PSIStats

		switch fields[0] {
		case "some":
			psiStats = somePsi
		case "full":
			psiStats = fullPsi
		default:
			reportError(newValueError("", fmt.Errorf("unexpected psi type (some|full) for psi file at: %s, type: %s", path, fields[0])))
		}

		// User did not provide stat for this type or unknown PSI type
		if psiStats == nil {
			return nil
		}

		for i := 1; i < 5; i++ {
			parts := strings.Split(fields[i], "=")
			if len(parts) != 2 {
				reportError(newValueError("", fmt.Errorf("unexpected format for psi file at: %s, part: %d, content: %v", path, i, fields[i])))
				continue
			}

			psi, err := strconv.ParseFloat(parts[1], 64)
			if err != nil {
				reportError(newValueError("", fmt.Errorf("unexpected format for psi file at: %s, part: %d, content: %v", path, i, fields[i])))
				continue
			}

			switch parts[0] {
			case "avg10":
				psiStats.Avg10 = &psi
			case "avg60":
				psiStats.Avg60 = &psi
			case "avg300":
				psiStats.Avg300 = &psi
			case "total":
				total, err := strconv.ParseUint(parts[1], 10, 64)
				if err != nil {
					reportError(newValueError("", fmt.Errorf("unexpected format for psi file at: %s, part: %d, content: %v", path, i, fields[i])))
					continue
				}
				psiStats.Total = &total
			}
		}

		return nil
	})
}
