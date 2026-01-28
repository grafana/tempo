package cli

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"
	"text/template"
	"unicode"
)

func (c *Command) help(reason error) error {
	if c.Flags().Lookup("help") == nil {
		_ = initHelpFlag(c)
	}

	return ErrHelp{
		Message: reason.Error(),
		usage:   c.Usage(),
		error:   true,
	}
}

// ErrHelp wraps an actual error, showing the usage of the command afterwards
type ErrHelp struct {
	Message string
	usage   string
	error   bool
}

func (e ErrHelp) Error() string {
	pat := "%s\n\n%s"
	if e.error {
		pat = "Error: " + pat
	}

	return fmt.Sprintf(pat, e.Message, e.usage)
}

// helpable is a internal wrapper type of Command that defines functions
// required to generate the help output using a text/template.
type helpable struct {
	Command
}

func (h *helpable) Generate() string {
	tmpl := template.New("")

	tmpl = tmpl.Funcs(template.FuncMap{
		"trimRightSpace": func(s string) string {
			return strings.TrimRightFunc(s, unicode.IsSpace)
		},
		"rpad": func(s string, padding int) string {
			template := fmt.Sprintf("%%-%ds", padding)
			return fmt.Sprintf(template, s)
		},
	})

	tmpl = template.Must(tmpl.Parse(`
Usage:
{{- if .HasChildren }}
  {{.CommandPath}} [command]
{{ else }}
  {{ .Use }}
{{ end }}

{{if .HasChildren }}
Available Commands:
{{- range .Children }}
  {{rpad .Name .CommandPadding }} {{.Short}}
{{- end}}
{{- end}}

{{if .Flags }}
Flags:
{{.Flags.FlagUsages}}
{{ end}}

{{ if .HasChildren }}
Use "{{.CommandPath}} [command] --help" for more information about a command.
{{ end}}
`))

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, h); err != nil {
		log.Fatalln(err)
	}

	// at least two newlines
	return strings.TrimSpace(regexp.MustCompile(`\n\n+`).ReplaceAllString(buf.String(), "\n\n"))
}

// CommandPadding computes the padding required to make all subcommand help
// texts line up
func (h *helpable) CommandPadding() int {
	pad := 9
	for _, c := range h.parentPtr.children {
		l := len(c.Name())
		if l > pad {
			pad = l
		}
	}
	return pad + 2
}

// Use returns the UseLine for the given command.
// Parent command names are prepended
func (h *helpable) Use() string {
	use := h.Command.Use
	if h.parentPtr != nil {
		use = h.parentPtr.helpable().CommandPath() + " " + h.Command.Use
	}

	return fmt.Sprintf("%s [flags]", use)
}

// HasChildren reports whether this command has children
func (h *helpable) HasChildren() bool {
	return h.children != nil
}

// Children returns the children of this command.
func (h *helpable) Children() []*helpable {
	m := make([]*helpable, len(h.children))
	for i, c := range h.children {
		m[i] = c.helpable()
	}
	return m
}

// CommandPath returns the names of this and all parent commands joined, in the
// same order as they would be specified on the command line.
func (h *helpable) CommandPath() string {
	if h.parentPtr != nil {
		return fmt.Sprintf("%s %s", h.parentPtr.helpable().CommandPath(), h.Name())
	}
	return h.Name()
}
