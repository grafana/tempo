package work

import (
	"testing"

	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/require"
)

var tenant = "test"

func TestStatus(t *testing.T) {
	j := &Job{
		ID: uuid.NewString(),
	}

	require.Equal(t, j.GetID(), j.ID)

	require.False(t, j.IsFailed())
	require.False(t, j.IsComplete())
	require.False(t, j.IsRunning())
	require.True(t, j.IsPending())
	j.Start()
	require.True(t, j.IsRunning())

	j.Start()
	require.False(t, j.IsFailed())

	j.Fail()
	require.True(t, j.IsFailed())

	j.Complete()
	require.False(t, j.IsFailed())
	require.True(t, j.IsComplete())
}

func TestCompactionDetail(t *testing.T) {
	cases := []struct {
		name     string
		jobType  tempopb.JobType
		detail   tempopb.JobDetail
		input    []string
		expected []string
	}{
		{
			name:     "non compaction job",
			input:    []string{"block1", "block2"},
			expected: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			j := &Job{
				ID:        uuid.NewString(),
				Type:      tc.jobType,
				JobDetail: tc.detail,
			}

			j.SetCompactionOutput(tc.input)
			require.Equal(t, j.GetCompactionOutput(), tc.expected)
		})
	}
}

func TestOnBlock(t *testing.T) {
	j := &Job{
		ID: uuid.NewString(),
	}

	require.False(t, j.OnBlock("block1"))
	j.Type = tempopb.JobType_JOB_TYPE_COMPACTION
	j.JobDetail = tempopb.JobDetail{
		Tenant:     tenant,
		Compaction: &tempopb.CompactionDetail{},
	}

	idOne := uuid.NewString()

	b := j.OnBlock(idOne)
	require.False(t, b)

	j.SetCompactionOutput([]string{idOne})
	require.False(t, b)
	j.Start()

	b = j.OnBlock(idOne)
	require.True(t, b)

	j.Complete()

	b = j.OnBlock(idOne)
	require.False(t, b)

	j.Fail()

	b = j.OnBlock(idOne)
	require.False(t, b)

	require.True(t, j.IsFailed())
}
