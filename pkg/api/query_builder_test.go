package api

import (
	"math/rand/v2"
	"net/url"
	"testing"

	"github.com/grafana/tempo/pkg/util/test"
	"github.com/stretchr/testify/require"
)

func TestQueryBuilder(t *testing.T) {
	numParams := rand.IntN(10) + 1

	qb := newQueryBuilder("")
	params := url.Values{}

	for i := 0; i < numParams; i++ {
		key := test.RandomString()
		value := test.RandomString()

		qb.addParam(key, value)
		params.Add(key, value)
	}

	// url sorts params but query builder does not. parse the query builder
	// string with url to guarantee its valid and also to sort the params
	actualQuery, err := url.ParseQuery(qb.query())
	require.NoError(t, err)

	actual := url.URL{}
	actual.RawQuery = actualQuery.Encode()

	expected := url.URL{}
	expected.RawQuery = params.Encode()

	require.Equal(t, expected.RawQuery, actual.RawQuery)
}
