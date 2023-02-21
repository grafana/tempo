//go:build linux || darwin
// +build linux darwin

package filesystem

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

var dfCommand = "df"
var dfOptions = []string{"-l", "-k"}
var dfTimeout = 2 * time.Second

func getFileSystemInfo() (interface{}, error) {

	ctx, cancel := context.WithTimeout(context.Background(), dfTimeout)
	defer cancel()

	/* Grab filesystem data from df	*/
	cmd := exec.CommandContext(ctx, dfCommand, dfOptions...)

	// force output in the C locale (untranslated) so that we can recognize the headers
	cmd.Env = []string{"LC_ALL=C"}

	out, execErr := cmd.Output()
	var parseErr error
	var result []interface{}
	if out != nil {
		result, parseErr = parseDfOutput(string(out))
	}

	// if we managed to get _any_ data, just use it, ignoring other errors
	if result != nil && len(result) != 0 {
		return result, nil
	}

	// otherwise, prefer the parse error, as it is probably more detailed
	err := execErr
	if parseErr != nil {
		err = parseErr
	}
	if err == nil {
		err = errors.New("unknown error")
	}
	return nil, fmt.Errorf("df failed to collect filesystem data: %s", parseErr)
}

func parseDfOutput(out string) ([]interface{}, error) {
	lines := strings.Split(out, "\n")
	if len(lines) < 2 {
		return nil, errors.New("no output")
	}
	var fileSystemInfo = make([]interface{}, 0, len(lines)-2)

	// parse the header to find the offsets for each component we need
	hdr := lines[0]
	fieldErrFunc := func(field string) error {
		return fmt.Errorf("could not find '%s' in `%s %s` output",
			field, dfCommand, strings.Join(dfOptions, " "))
	}

	kbsizeOffset := strings.Index(hdr, "1K-blocks")
	if kbsizeOffset == -1 {
		kbsizeOffset = strings.Index(hdr, "1024-blocks")
		if kbsizeOffset == -1 {
			return nil, fieldErrFunc("`1K-blocks` or `1024-blocks`")
		}
	}

	mountedOnOffset := strings.Index(hdr, "Mounted on")
	if mountedOnOffset == -1 {
		return nil, fieldErrFunc("`Mounted on`")
	}

	// now parse the remaining lines using those offsets
	for _, line := range lines[1:] {
		if len(line) == 0 || len(line) < mountedOnOffset {
			continue
		}
		info := map[string]string{}

		// we assume that "Filesystem" is the leftmost field, and continues to the
		// beginning of "1K-blocks".
		info["name"] = strings.Trim(line[:kbsizeOffset], " ")

		// kbsize is right-aligned under "1K-blocks", so strip leading
		// whitespace and the discard everything after the next whitespace
		kbsizeAndMore := strings.TrimLeft(line[kbsizeOffset:], " ")
		info["kb_size"] = strings.SplitN(kbsizeAndMore, " ", 2)[0]

		// mounted_on is left-aligned under "Mounted on" and continues to EOL
		info["mounted_on"] = strings.Trim(line[mountedOnOffset:], " ")

		fileSystemInfo = append(fileSystemInfo, info)
	}
	return fileSystemInfo, nil
}
