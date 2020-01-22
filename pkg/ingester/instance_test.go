package ingester

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/joe-elliott/frigg/pkg/ingester/wal"
	"github.com/joe-elliott/frigg/pkg/util/test"
	"github.com/joe-elliott/frigg/pkg/util/validation"

	"github.com/stretchr/testify/assert"
)

type ringCountMock struct {
	count int
}

func (m *ringCountMock) HealthyInstancesCount() int {
	return m.count
}

func TestInstance(t *testing.T) {
	limits, err := validation.NewOverrides(validation.Limits{})
	assert.NoError(t, err, "unexpected error creating limits")
	limiter := NewLimiter(limits, &ringCountMock{count: 1}, 1)

	tempDir, err := ioutil.TempDir("/tmp", "")
	assert.NoError(t, err, "unexpected error getting temp dir")
	defer os.RemoveAll(tempDir)
	wal := wal.New(wal.Config{
		Filepath: tempDir,
	})

	request := test.MakeRequest(10, []byte{})

	i := newInstance("fake", limiter, wal)
	i.Push(context.Background(), request)

	i.CutCompleteTraces(0, true)

	ready := i.IsBlockReady(5, 0)
	assert.True(t, ready, "block should be ready due to time")

	ready = i.IsBlockReady(0, 30*time.Hour)
	assert.True(t, ready, "block should be ready due to max traces")

	records, _ := i.GetBlock()
	assert.Equal(t, 1, len(records))

	err = i.ResetBlock()
	assert.NoError(t, err, "unexpected error resetting block")

	records, _ = i.GetBlock()
	assert.Equal(t, 0, len(records))
}
