package utils

import (
	"fmt"
	"strconv"
)

// GetString returns the dict[key] or an empty string if the entry doesn't exists
func GetString(dict map[string]string, key string) string {
	if v, ok := dict[key]; ok {
		return v
	}
	return ""
}

// GetStringInterface returns the dict[key] or an empty string if the entry doesn't exists or the value is not a string
func GetStringInterface(dict map[string]interface{}, key string) string {
	if v, ok := dict[key]; ok {
		vString, ok := v.(string)
		if !ok {
			return ""
		}
		return vString
	}
	return ""
}

// GetUint64 returns the dict[key] or an empty uint64 if the entry doesn't exists
func GetUint64(dict map[string]string, key string, warnings *[]string) uint64 {
	if v, ok := dict[key]; ok {
		num, err := strconv.ParseUint(v, 10, 64)
		if err == nil {
			return num
		}
		*warnings = append(*warnings, fmt.Sprintf("could not parse '%s' from %s: %s", v, key, err))
		return 0
	}
	return 0
}

// GetFloat64 returns the dict[key] or an empty uint64 if the entry doesn't exists
func GetFloat64(dict map[string]string, key string, warnings *[]string) float64 {
	if v, ok := dict[key]; ok {
		num, err := strconv.ParseFloat(v, 10)
		if err == nil {
			return num
		}
		*warnings = append(*warnings, fmt.Sprintf("could not parse '%s' from %s: %s", v, key, err))
		return 0
	}
	return 0
}
