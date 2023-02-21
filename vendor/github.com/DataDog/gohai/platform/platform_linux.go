package platform

import "strings"

var unameOptions = []string{"-s", "-n", "-r", "-m", "-p", "-i", "-o"}

func updateArchInfo(archInfo map[string]string, values []string) {
	archInfo["kernel_name"] = values[0]
	archInfo["hostname"] = values[1]
	archInfo["kernel_release"] = values[2]
	archInfo["machine"] = values[3]
	archInfo["processor"] = values[4]
	archInfo["hardware_platform"] = values[5]
	archInfo["os"] = strings.Trim(values[6], "\n")
}
