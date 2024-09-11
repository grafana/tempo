package uuid

import (
	"testing"

	google_uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_roundTrip(t *testing.T) {
	u := New()
	t.Logf("u %x", u)

	require.Equal(t, 16, u.Size())

	// Marshalto
	b := make([]byte, 16)
	l, err := u.MarshalTo(b)
	require.NoError(t, err)
	require.Equal(t, 16, l)
	require.Equal(t, 16, len(b))

	u2 := UUID{}
	err = u2.Unmarshal(b)
	require.NoError(t, err)
	assert.Equal(t, u.UUID, u2.UUID)

	// Marshal
	b2, err := u2.Marshal()
	require.NoError(t, err)
	require.Equal(t, 16, len(b2))

	u3 := UUID{}
	err = u3.Unmarshal(b2)
	require.NoError(t, err)
	assert.Equal(t, u.UUID, u2.UUID, u3.UUID)

	// MarshalJSON
	b3, err := u3.MarshalJSON()
	require.NoError(t, err)

	u4 := UUID{}
	err = u4.UnmarshalJSON(b3)
	require.NoError(t, err)
	assert.Equal(t, u.UUID, u2.UUID, u3.UUID, u4.UUID)
}

func Test_helpers(t *testing.T) {
	u := google_uuid.New()
	s := MustParse(u.String())
	require.Equal(t, u, s.UUID)

	s2, err := Parse(u.String())
	require.NoError(t, err)
	require.Equal(t, u, s2.UUID)

	_, err = Parse("x")
	require.Error(t, err)
}
