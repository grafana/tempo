package manifest

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// SchemaError means that some expected fields were missing
type SchemaError struct {
	Fields   map[string]error
	Name     string
	Manifest Manifest
}

var (
	redf    = color.New(color.FgRed, color.Bold, color.Underline).Sprintf
	yellowf = color.New(color.FgYellow).Sprintf
	bluef   = color.New(color.FgBlue, color.Bold).Sprintf
)

// Error returns the fields the manifest at the path is missing
func (s *SchemaError) Error() string {
	if s.Name == "" {
		s.Name = "Resource"
	}

	msg := fmt.Sprintf("%s has missing or invalid fields:\n", redf(s.Name))

	for k, err := range s.Fields {
		if err == nil {
			continue
		}

		msg += fmt.Sprintf("  - %s: %s\n", yellowf(k), err)
	}

	if s.Manifest != nil {
		msg += bluef("\nPlease check below object:\n")
		msg += SampleString(s.Manifest.String()).Indent(2)
	}

	return msg
}

// SampleString is used for displaying code samples for error messages. It
// truncates the output to 10 lines
type SampleString string

func (s SampleString) String() string {
	lines := strings.Split(strings.TrimSpace(string(s)), "\n")
	truncate := len(lines) >= 10
	if truncate {
		lines = lines[0:10]
	}
	out := strings.Join(lines, "\n")
	if truncate {
		out += "\n..."
	}
	return out
}

func (s SampleString) Indent(n int) string {
	indent := strings.Repeat(" ", n)
	lines := strings.Split(s.String(), "\n")
	return indent + strings.Join(lines, "\n"+indent)
}

// ErrorDuplicateName means two resources share the same name using the given
// nameFormat.
type ErrorDuplicateName struct {
	name   string
	format string
}

func (e ErrorDuplicateName) Error() string {
	return fmt.Sprintf("Two resources share the same name '%s'. Please adapt the name template '%s'.", e.name, e.format)
}
