package statsd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"sync"
)

const (
	// cgroupPath is the path to the cgroup file where we can find the container id if one exists.
	cgroupPath = "/proc/self/cgroup"
)

const (
	uuidSource      = "[0-9a-f]{8}[-_][0-9a-f]{4}[-_][0-9a-f]{4}[-_][0-9a-f]{4}[-_][0-9a-f]{12}"
	containerSource = "[0-9a-f]{64}"
	taskSource      = "[0-9a-f]{32}-\\d+"
)

var (
	// expLine matches a line in the /proc/self/cgroup file. It has a submatch for the last element (path), which contains the container ID.
	expLine = regexp.MustCompile(`^\d+:[^:]*:(.+)$`)

	// expContainerID matches contained IDs and sources. Source: https://github.com/Qard/container-info/blob/master/index.js
	expContainerID = regexp.MustCompile(fmt.Sprintf(`(%s|%s|%s)(?:.scope)?$`, uuidSource, containerSource, taskSource))

	// containerID holds the container ID.
	containerID = ""
)

// parseContainerID finds the first container ID reading from r and returns it.
func parseContainerID(r io.Reader) string {
	scn := bufio.NewScanner(r)
	for scn.Scan() {
		path := expLine.FindStringSubmatch(scn.Text())
		if len(path) != 2 {
			// invalid entry, continue
			continue
		}
		if parts := expContainerID.FindStringSubmatch(path[1]); len(parts) == 2 {
			return parts[1]
		}
	}
	return ""
}

// readContainerID attempts to return the container ID from the provided file path or empty on failure.
func readContainerID(fpath string) string {
	f, err := os.Open(fpath)
	if err != nil {
		return ""
	}
	defer f.Close()
	return parseContainerID(f)
}

// getContainerID returns the container ID configured at the client creation
// It can either be auto-discovered with origin detection or provided by the user.
// User-defined container ID is prioritized.
func getContainerID() string {
	return containerID
}

var initOnce sync.Once

// initContainerID initializes the container ID.
// It can either be provided by the user or read from cgroups.
func initContainerID(userProvidedID string, cgroupFallback bool) {
	initOnce.Do(func() {
		if userProvidedID != "" {
			containerID = userProvidedID
			return
		}

		if cgroupFallback {
			containerID = readContainerID(cgroupPath)
		}
	})
}
