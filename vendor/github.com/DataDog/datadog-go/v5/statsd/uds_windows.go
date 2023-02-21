// +build windows

package statsd

import (
	"fmt"
	"io"
	"time"
)

// newUDSWriter is disabled on Windows as Unix sockets are not available.
func newUDSWriter(_ string, _ time.Duration) (io.WriteCloser, error) {
	return nil, fmt.Errorf("Unix socket is not available on Windows")
}
