package term

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/fatih/color"
)

// Colordiff colorizes unified diff output (diff -u -N)
func Colordiff(d string) *bytes.Buffer {
	exps := map[string]func(s string) bool{
		"add":  regexp.MustCompile(`^\+.*`).MatchString,
		"del":  regexp.MustCompile(`^\-.*`).MatchString,
		"head": regexp.MustCompile(`^diff -u -N.*`).MatchString,
		"hid":  regexp.MustCompile(`^@.*`).MatchString,
	}

	buf := bytes.Buffer{}
	lines := strings.Split(d, "\n")

	for _, l := range lines {
		switch {
		case exps["add"](l):
			color.New(color.FgGreen).Fprintln(&buf, l)
		case exps["del"](l):
			color.New(color.FgRed).Fprintln(&buf, l)
		case exps["head"](l):
			color.New(color.FgBlue, color.Bold).Fprintln(&buf, l)
		case exps["hid"](l):
			color.New(color.FgMagenta, color.Bold).Fprintln(&buf, l)
		default:
			fmt.Fprintln(&buf, l)
		}
	}

	return &buf
}
