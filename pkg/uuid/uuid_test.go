package uuid

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_roundTrip(t *testing.T) {
	u := UUID{uuid.New()}
	t.Logf("u %x", u)

	require.Equal(t, 16, u.Size())

	b := make([]byte, 16)
	l, err := u.MarshalTo(b)
	require.NoError(t, err)
	require.Equal(t, 16, l)
	require.Equal(t, 16, len(b))

	u2 := UUID{}
	err = u2.Unmarshal(b)
	t.Logf("u2 %x", u2)
	require.NoError(t, err)
	assert.Equal(t, u.UUID, u2.UUID)
}
