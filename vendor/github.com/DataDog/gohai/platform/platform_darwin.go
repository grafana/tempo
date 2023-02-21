package platform

import (
	"strings"

	log "github.com/cihub/seelog"
	"golang.org/x/sys/unix"
)

var unameOptions = []string{"-s", "-n", "-r", "-m", "-p"}

// processIsTranslated detects if the process using gohai is running under the Rosetta 2 translator
func processIsTranslated() (bool, error) {
	// https://developer.apple.com/documentation/apple_silicon/about_the_rosetta_translation_environment#3616845
	ret, err := unix.SysctlUint32("sysctl.proc_translated")

	if err == nil {
		return ret == 1, nil
	} else if err.(unix.Errno) == unix.ENOENT {
		return false, nil
	}
	return false, err
}

func updateArchInfo(archInfo map[string]string, values []string) {
	archInfo["kernel_name"] = values[0]
	archInfo["hostname"] = values[1]
	archInfo["kernel_release"] = values[2]
	archInfo["machine"] = values[3]
	archInfo["processor"] = strings.Trim(values[4], "\n")
	archInfo["os"] = values[0]

	if isTranslated, err := processIsTranslated(); err == nil && isTranslated {
		log.Debug("Running under Rosetta translator; overriding architecture values")
		archInfo["processor"] = "arm"
		archInfo["machine"] = "arm64"
	} else if err != nil {
		log.Debugf("Error when detecting Rosetta translator: %s", err)
	}
}
