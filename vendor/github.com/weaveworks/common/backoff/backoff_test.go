package backoff_test

import (
	"errors"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/weaveworks/common/backoff"
)

func TestLog(t *testing.T) {
	hook := test.NewGlobal()

	err := errors.New("sample")
	cases := []struct {
		returns           []error
		expectedLastLevel log.Level
		expectedCount     int
	}{
		{[]error{nil}, log.InfoLevel, 1},
		{[]error{nil, nil /*x*/}, log.InfoLevel, 1},
		{[]error{nil, err}, log.WarnLevel, 2},
		{[]error{nil, err, nil}, log.InfoLevel, 3},
		{[]error{nil, err, nil, nil /*x*/}, log.InfoLevel, 3},
		{[]error{nil, err}, log.WarnLevel, 2},
		{[]error{nil, err, err}, log.WarnLevel, 3},
		{[]error{nil, err, err, err /*x*/}, log.WarnLevel, 3},
		{[]error{nil, err, err, err /*x*/, nil}, log.InfoLevel, 4},
	}

	for ci, c := range cases {
		hook.Reset()

		ri := 0
		bo := backoff.New(func() (done bool, err error) {
			if ri >= len(c.returns) {
				done, err = true, nil // abort
			} else {
				done, err = false, c.returns[ri]
			}
			ri++
			return
		}, "test")
		bo.SetInitialBackoff(1 * time.Millisecond)
		bo.SetMaxBackoff(4 * time.Millisecond) // exceeded at 3rd consecutive error
		bo.Start()

		if len(hook.Entries) != c.expectedCount {
			t.Errorf("case #%d failed: expected log count %d but got %d", ci, c.expectedCount, len(hook.Entries))
		} else if hook.LastEntry().Level != c.expectedLastLevel {
			t.Errorf("case #%d failed: expected level %d but got %d", ci, c.expectedLastLevel, hook.LastEntry().Level)
		}
	}
}
