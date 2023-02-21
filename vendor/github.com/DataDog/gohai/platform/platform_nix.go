//go:build linux || darwin
// +build linux darwin

package platform

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// GetArchInfo() returns basic host architecture information
func GetArchInfo() (archInfo map[string]string, err error) {
	archInfo = map[string]string{}

	out, err := exec.Command("uname", unameOptions...).Output()
	if err != nil {
		return nil, err
	}
	line := fmt.Sprintf("%s", out)
	values := regexp.MustCompile(" +").Split(line, 7)
	updateArchInfo(archInfo, values)

	out, err = exec.Command("uname", "-v").Output()
	if err != nil {
		return nil, err
	}
	archInfo["kernel_version"] = strings.Trim(string(out), "\n")

	return
}
