package work

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQueue(t *testing.T) {
	q := NewQueue()
	require.NotNil(t, q)

	err := q.AddJob(&Job{ID: "1"})
	require.NoError(t, err)

	err = q.AddJob(&Job{ID: "1"})
	require.Error(t, err)

	j := q.GetJob("1")
	require.NotNil(t, j)

	jj := q.GetJob("2")
	require.Nil(t, jj)

	err = q.AddJob(&Job{ID: "2"})
	require.NoError(t, err)

	jj = q.GetJob("2")
	require.NotNil(t, jj)

	require.Len(t, q.Jobs(), 2)

	jobs := q.Jobs()
	require.Equal(t, "1", jobs[0].ID)
	require.Equal(t, "2", jobs[1].ID)

	j.Complete()
	q.Prune()
	require.Len(t, q.Jobs(), 1)

	jj.Fail()
	q.Prune()
	require.Len(t, q.Jobs(), 0)

	err = q.AddJob(&Job{ID: "3"})
	require.NoError(t, err)

	j = q.GetJob("3")
	require.NotNil(t, j)
	require.Len(t, q.Jobs(), 1)
	q.RemoveJob(j.ID)
}
