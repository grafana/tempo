package cpu

import (
	"bufio"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var prefix = "" // only used for testing
var listRangeRegex = regexp.MustCompile("([0-9]+)-([0-9]+)$")

// sysCpuInt reads an integer from a file in /sys/devices/system/cpu
func sysCpuInt(path string) (uint64, bool) {
	content, err := ioutil.ReadFile(prefix + "/sys/devices/system/cpu/" + path)
	if err != nil {
		return 0, false
	}

	value, err := strconv.ParseUint(strings.TrimSpace(string(content)), 0, 64)
	if err != nil {
		return 0, false
	}

	return value, true
}

// sysCpuSize reads an value with a K/M/G suffix from a file in /sys/devices/system/cpu
func sysCpuSize(path string) (uint64, bool) {
	content, err := ioutil.ReadFile(prefix + "/sys/devices/system/cpu/" + path)
	if err != nil {
		return 0, false
	}

	s := strings.TrimSpace(string(content))
	mult := uint64(1)
	switch s[len(s)-1] {
	case 'K':
		mult = 1024
	case 'M':
		mult = 1024 * 1024
	case 'G':
		mult = 1024 * 1024 * 1024
	}
	if mult > 1 {
		s = s[:len(s)-1]
	}

	value, err := strconv.ParseUint(s, 0, 64)
	if err != nil {
		return 0, false
	}

	return value * mult, true
}

// sysCpuList reads a list of integers, comma-seprated with ranges (`0-5,7-11`)
// from a file in /sys/devices/system/cpu.  The return value is the set of
// integers included in the list (for the example above, {0, 1, 2, 3, 4, 5, 7,
// 8, 9, 10, 11}).
func sysCpuList(path string) (map[uint64]struct{}, bool) {
	content, err := ioutil.ReadFile(prefix + "/sys/devices/system/cpu/" + path)
	if err != nil {
		return nil, false
	}

	result := map[uint64]struct{}{}
	contentStr := strings.TrimSpace(string(content))
	if len(contentStr) == 0 {
		return result, true
	}

	for _, elt := range strings.Split(contentStr, ",") {
		if submatches := listRangeRegex.FindStringSubmatch(elt); submatches != nil {
			// Handle the NN-NN form, inserting each included integer into the set
			first, err := strconv.ParseUint(submatches[1], 0, 64)
			if err != nil {
				return nil, false
			}
			last, err := strconv.ParseUint(submatches[2], 0, 64)
			if err != nil {
				return nil, false
			}
			for i := first; i <= last; i++ {
				result[i] = struct{}{}
			}
		} else {
			// Handle a simple integer, just inserting it into the set
			i, err := strconv.ParseUint(elt, 0, 64)
			if err != nil {
				return nil, false
			}
			result[i] = struct{}{}
		}
	}

	return result, true
}

// readProcCpuInfo reads /proc/cpuinfo.  The file is structured as a set of
// blank-line-separated stanzas, and each stanza is a map of string to string,
// with whitespace stripped.
func readProcCpuInfo() ([]map[string]string, error) {
	file, err := os.Open(prefix + "/proc/cpuinfo")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var stanzas []map[string]string
	var stanza map[string]string

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			stanza = nil
			continue
		}

		pair := strings.SplitN(line, ":", 2)
		if len(pair) != 2 {
			continue
		}
		if stanza == nil {
			stanza = make(map[string]string)
			stanzas = append(stanzas, stanza)
		}
		stanza[strings.TrimSpace(pair[0])] = strings.TrimSpace(pair[1])
	}

	if scanner.Err() != nil {
		err = scanner.Err()
		return nil, err
	}

	// On some platforms, such as rPi, there are stanzas in this file that do
	// not correspond to processors.  It doesn't seem this file is intended for
	// machine consumption!  So, we filter those out.
	var results []map[string]string
	for _, stanza := range stanzas {
		if _, found := stanza["processor"]; !found {
			continue
		}
		results = append(results, stanza)
	}

	return results, nil
}
