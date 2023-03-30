// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package scrubber

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type scrubCallback = func(string, interface{}) (bool, interface{})

func walkSlice(data []interface{}, callback scrubCallback) {
	for _, k := range data {
		switch v := k.(type) {
		case map[interface{}]interface{}:
			walkHash(v, callback)
		case []interface{}:
			walkSlice(v, callback)
		}
	}
}

func walkHash(data map[interface{}]interface{}, callback scrubCallback) {
	for k, v := range data {
		if keyString, ok := k.(string); ok {
			if match, newValue := callback(keyString, v); match {
				data[keyString] = newValue
				continue
			}
		}

		switch v := data[k].(type) {
		case map[interface{}]interface{}:
			walkHash(v, callback)
		case []interface{}:
			walkSlice(v, callback)
		}
	}
}

// walk will go through loaded yaml and call callback on every strings allowing
// the callback to overwrite the string value
func walk(data *interface{}, callback scrubCallback) {
	switch v := (*data).(type) {
	case map[interface{}]interface{}:
		walkHash(v, callback)
	case []interface{}:
		walkSlice(v, callback)
	}
}

// ScrubYaml scrubs credentials from the given YAML by loading the data and scrubbing the object instead of the
// serialized string.
func (c *Scrubber) ScrubYaml(input []byte) ([]byte, error) {
	var data *interface{}
	err := yaml.Unmarshal(input, &data)

	// if we can't load the yaml run the default scrubber on the input
	if err == nil {
		walk(data, func(key string, value interface{}) (bool, interface{}) {
			for _, replacer := range c.singleLineReplacers {
				if replacer.YAMLKeyRegex == nil {
					continue
				}
				if replacer.YAMLKeyRegex.Match([]byte(key)) {
					if replacer.ProcessValue != nil {
						return true, replacer.ProcessValue(value)
					}
					return true, defaultReplacement
				}
			}
			return false, ""
		})

		newInput, err := yaml.Marshal(data)
		if err == nil {
			input = newInput
		} else {
			// Since the scrubber is a dependency of the logger we can use it here.
			fmt.Fprintf(os.Stderr, "error scrubbing YAML, falling back on text scrubber: %s\n", err)
		}
	}
	return c.ScrubBytes(input)
}
