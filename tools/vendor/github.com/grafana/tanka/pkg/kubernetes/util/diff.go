package util

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/grafana/tanka/pkg/kubernetes/manifest"
)

// DiffName computes the filename for use with `DiffStr`
func DiffName(m manifest.Manifest) string {
	return strings.ReplaceAll(fmt.Sprintf("%s.%s.%s.%s",
		m.APIVersion(),
		m.Kind(),
		m.Metadata().Namespace(),
		m.Metadata().Name(),
	), "/", "-")
}

// DiffStr computes the differences between the strings `is` and `should` using the
// UNIX `diff(1)` utility.
func DiffStr(name, is, should string) (string, error) {
	dir, err := os.MkdirTemp("", "diff")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)

	if err := os.WriteFile(filepath.Join(dir, "LIVE-"+name), []byte(is), os.ModePerm); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "MERGED-"+name), []byte(should), os.ModePerm); err != nil {
		return "", err
	}

	buf := bytes.Buffer{}
	merged := filepath.Join(dir, "MERGED-"+name)
	live := filepath.Join(dir, "LIVE-"+name)
	cmd := exec.Command("diff", "-u", "-N", live, merged)
	cmd.Stdout = &buf
	err = cmd.Run()

	// the diff utility exits with `1` if there are differences. We need to not fail there.
	if exitError, ok := err.(*exec.ExitError); ok && err != nil {
		if exitError.ExitCode() != 1 {
			return "", err
		}
	}

	out := buf.String()
	if out != "" {
		out = fmt.Sprintf("diff -u -N %s %s\n%s", live, merged, out)
	}

	return out, nil
}

// Diffstat creates a histogram of a diff
func DiffStat(d string) (string, error) {
	lines := strings.Split(d, "\n")
	type diff struct {
		added, removed int
	}

	maxFilenameLength := 0
	maxChanges := 0
	var fileNames []string
	diffMap := map[string]diff{}

	currentFileName := ""
	totalAdded, added, totalRemoved, removed := 0, 0, 0, 0
	for i, line := range lines {
		if strings.HasPrefix(line, "diff ") {
			splitLine := strings.Split(line, " ")
			currentFileName = findStringsCommonSuffix(splitLine[len(splitLine)-2], splitLine[len(splitLine)-1])
			added, removed = 0, 0
			continue
		}

		if strings.HasPrefix(line, "+ ") {
			added++
		} else if strings.HasPrefix(line, "- ") {
			removed++
		}

		if currentFileName != "" && (i == len(lines)-1 || strings.HasPrefix(lines[i+1], "diff ")) {
			totalAdded += added
			totalRemoved += removed
			if added+removed > maxChanges {
				maxChanges = added + removed
			}

			fileNames = append(fileNames, currentFileName)
			diffMap[currentFileName] = diff{added, removed}
			if len(currentFileName) > maxFilenameLength {
				maxFilenameLength = len(currentFileName)
			}
		}
	}
	sort.Strings(fileNames)

	builder := strings.Builder{}
	for _, fileName := range fileNames {
		f := diffMap[fileName]
		builder.WriteString(fmt.Sprintf("%-*s | %4d %s\n", maxFilenameLength, fileName, f.added+f.removed, printPlusAndMinuses(f.added, f.removed, maxChanges)))
	}
	builder.WriteString(fmt.Sprintf("%d files changed, %d insertions(+), %d deletions(-)", len(fileNames), totalAdded, totalRemoved))

	return builder.String(), nil
}

// FilteredErr is a filtered Stderr. If one of the regular expressions match, the current input is discarded.
type FilteredErr []*regexp.Regexp

func (r FilteredErr) Write(p []byte) (n int, err error) {
	for _, re := range r {
		if re.Match(p) {
			// silently discard
			return len(p), nil
		}
	}
	return os.Stderr.Write(p)
}

// printPlusAndMinuses prints colored plus and minus signs for the given number of added and removed lines.
// The number of characters is calculated based on the maximum number of changes in all files (maxChanges).
// The number of characters is capped at 40.
func printPlusAndMinuses(added, removed int, maxChanges int) string {
	addedAndRemoved := float64(added + removed)
	chars := math.Ceil(addedAndRemoved / float64(maxChanges) * 40)

	added = minInt(added, int(float64(added)/addedAndRemoved*chars))
	removed = minInt(removed, int(chars)-added)

	return color.New(color.FgGreen).Sprint(strings.Repeat("+", added)) +
		color.New(color.FgRed).Sprint(strings.Repeat("-", removed))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// findStringsCommonSuffix returns the common suffix of the two strings (removing leading `/` or `-`)
// e.g. findStringsCommonSuffix("foo/bar/baz", "other/bar/baz") -> "bar/baz"
func findStringsCommonSuffix(a, b string) string {
	if a == b {
		return a
	}

	if len(a) > len(b) {
		a, b = b, a
	}

	for i := 0; i < len(a); i++ {
		if a[len(a)-i-1] != b[len(b)-i-1] {
			return strings.TrimLeft(a[len(a)-i:], "/-")
		}
	}

	return ""
}
