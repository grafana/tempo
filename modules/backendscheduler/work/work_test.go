package work

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQueue(t *testing.T) {
	w := New()
	require.NotNil(t, w)

	err := w.AddJob(&Job{ID: "1"})
	require.NoError(t, err)

	err = w.AddJob(&Job{ID: "1"})
	require.Error(t, err)

	j := w.GetJob("1")
	require.NotNil(t, j)

	jj := w.GetJob("2")
	require.Nil(t, jj)

	err = w.AddJob(&Job{ID: "2"})
	require.NoError(t, err)

	jj = w.GetJob("2")
	require.NotNil(t, jj)

	require.Len(t, w.Jobs(), 2)

	jobs := w.Jobs()
	require.Equal(t, "1", jobs[0].ID)
	require.Equal(t, "2", jobs[1].ID)

	j.Complete()
	w.Prune()
	require.Len(t, w.Jobs(), 1)

	jj.Fail()
	w.Prune()
	require.Len(t, w.Jobs(), 0)

	err = w.AddJob(&Job{ID: "3"})
	require.NoError(t, err)

	j = w.GetJob("3")
	require.NotNil(t, j)
	require.Len(t, w.Jobs(), 1)
	w.RemoveJob(j.ID)
}
