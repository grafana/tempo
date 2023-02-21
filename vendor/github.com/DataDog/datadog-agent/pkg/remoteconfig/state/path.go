// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

package state

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	// matches datadog/<int>/<string>/<string>/<string> for datadog/<org_id>/<product>/<config_id>/<file>
	datadogPathRegexp       = regexp.MustCompile(`^datadog/(\d+)/([^/]+)/([^/]+)/([^/]+)$`)
	datadogPathRegexpGroups = 4

	// matches employee/<string>/<string>/<string> for employee/<org_id>/<product>/<config_id>/<file>
	employeePathRegexp       = regexp.MustCompile(`^employee/([^/]+)/([^/]+)/([^/]+)$`)
	employeePathRegexpGroups = 3
)

type source uint

const (
	sourceUnknown source = iota
	sourceDatadog
	sourceEmployee
)

type configPath struct {
	Source   source
	OrgID    int64
	Product  string
	ConfigID string
	Name     string
}

func parseConfigPath(path string) (configPath, error) {
	configType := parseConfigPathSource(path)
	switch configType {
	case sourceDatadog:
		return parseDatadogConfigPath(path)
	case sourceEmployee:
		return parseEmployeeConfigPath(path)
	}
	return configPath{}, fmt.Errorf("config path '%s' has unknown source", path)
}

func parseDatadogConfigPath(path string) (configPath, error) {
	matchedGroups := datadogPathRegexp.FindStringSubmatch(path)
	if len(matchedGroups) != datadogPathRegexpGroups+1 {
		return configPath{}, fmt.Errorf("config file path '%s' has wrong format", path)
	}
	rawOrgID := matchedGroups[1]
	orgID, err := strconv.ParseInt(rawOrgID, 10, 64)
	if err != nil {
		return configPath{}, fmt.Errorf("could not parse orgID '%s' in config file path: %v", rawOrgID, err)
	}
	rawProduct := matchedGroups[2]
	if len(rawProduct) == 0 {
		return configPath{}, fmt.Errorf("product is empty")
	}
	return configPath{
		Source:   sourceDatadog,
		OrgID:    orgID,
		Product:  rawProduct,
		ConfigID: matchedGroups[3],
		Name:     matchedGroups[4],
	}, nil
}

func parseEmployeeConfigPath(path string) (configPath, error) {
	matchedGroups := employeePathRegexp.FindStringSubmatch(path)
	if len(matchedGroups) != employeePathRegexpGroups+1 {
		return configPath{}, fmt.Errorf("config file path '%s' has wrong format", path)
	}
	rawProduct := matchedGroups[1]
	if len(rawProduct) == 0 {
		return configPath{}, fmt.Errorf("product is empty")
	}
	return configPath{
		Source:   sourceEmployee,
		Product:  rawProduct,
		ConfigID: matchedGroups[2],
		Name:     matchedGroups[3],
	}, nil
}

func parseConfigPathSource(path string) source {
	switch {
	case strings.HasPrefix(path, "datadog/"):
		return sourceDatadog
	case strings.HasPrefix(path, "employee/"):
		return sourceEmployee
	}
	return sourceUnknown
}
