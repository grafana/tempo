// +build !windows

package statsd

import (
	"errors"
	"io"
	"time"
)

func newWindowsPipeWriter(pipepath string, writeTimeout time.Duration) (io.WriteCloser, error) {
	return nil, errors.New("Windows Named Pipes are only supported on Windows")
}
