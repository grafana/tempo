package kong

import (
	"fmt"
	"regexp"
)

var interpolationRegex = regexp.MustCompile(`((?:\${([[:alpha:]_][[:word:]]*))(?:=([^}]+))?})|(\$)|([^$]+)`)

// Interpolate variables from vars into s for substrings in the form ${var} or ${var=default}.
func interpolate(s string, vars Vars, updatedVars map[string]string) (string, error) {
	out := ""
	matches := interpolationRegex.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return s, nil
	}
	for key, val := range updatedVars {
		if vars[key] != val {
			vars = vars.CloneWith(updatedVars)
			break
		}
	}
	for _, match := range matches {
		if name := match[2]; name != "" {
			value, ok := vars[name]
			if !ok {
				// No default value.
				if match[3] == "" {
					return "", fmt.Errorf("undefined variable ${%s}", name)
				}
				value = match[3]
			}
			out += value
		} else {
			out += match[0]
		}
	}
	return out, nil
}
