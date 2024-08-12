package backend

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/tempopb"
)

const (
	testTenantID = "fake"
)

func TestNewBlockMeta(t *testing.T) {
	testVersion := "blerg"
	testEncoding := EncLZ4_256k
	testDataEncoding := "blarg"

	id := uuid.New()
	b := NewBlockMeta(testTenantID, id, testVersion, testEncoding, testDataEncoding)

	assert.Equal(t, id, b.BlockID)
	assert.Equal(t, testTenantID, b.TenantID)
	assert.Equal(t, testVersion, b.Version)
	assert.Equal(t, testEncoding, b.Encoding)
	assert.Equal(t, testDataEncoding, b.DataEncoding)
}

func TestBlockMetaObjectAdded(t *testing.T) {
	now := time.Unix(time.Now().Unix(), 0)

	tests := []struct {
		ids             [][]byte
		starts          []uint32
		ends            []uint32
		expectedStart   time.Time
		expectedEnd     time.Time
		expectedObjects int
	}{
		{},
		{
			ids: [][]byte{
				{0x01},
			},
			starts: []uint32{
				uint32(now.Unix()),
			},
			ends: []uint32{
				uint32(now.Add(time.Minute).Unix()),
			},
			expectedStart:   now,
			expectedEnd:     now.Add(time.Minute),
			expectedObjects: 1,
		},
		{
			ids: [][]byte{
				{0x01},
				{0x02},
			},
			starts: []uint32{
				uint32(now.Unix()),
				uint32(now.Add(-time.Minute).Unix()),
			},
			ends: []uint32{
				uint32(now.Add(time.Hour).Unix()),
				uint32(now.Add(time.Minute).Unix()),
			},
			expectedStart:   now.Add(-time.Minute),
			expectedEnd:     now.Add(time.Hour),
			expectedObjects: 2,
		},
	}

	for _, tc := range tests {
		b := &BlockMeta{}

		for i := 0; i < len(tc.ids); i++ {
			b.ObjectAdded(tc.ids[i], tc.starts[i], tc.ends[i])
		}

		assert.Equal(t, tc.expectedStart, b.StartTime)
		assert.Equal(t, tc.expectedEnd, b.EndTime)
		assert.Equal(t, tc.expectedObjects, b.TotalObjects)
	}
}

func TestBlockMetaParsing(t *testing.T) {
	timeParse := func(s string) time.Time {
		date, err := time.Parse(time.RFC3339Nano, s)
		require.NoError(t, err)
		return date
	}

	meta := BlockMeta{
		Version:         "vParquet3",
		BlockID:         uuid.MustParse("00000000-0000-0000-0000-000000000000"),
		TenantID:        "single-tenant",
		StartTime:       timeParse("2021-01-01T00:00:00.0000000Z"),
		EndTime:         timeParse("2021-01-02T00:00:00.0000000Z"),
		TotalObjects:    10,
		Size:            12345,
		CompactionLevel: 1,
		Encoding:        EncZstd,
		IndexPageSize:   250000,
		TotalRecords:    124356,
		DataEncoding:    "",
		BloomShardCount: 244,
		FooterSize:      15775,
		DedicatedColumns: DedicatedColumns{
			{Scope: "resource", Name: "namespace", Type: "string"},
			{Scope: "span", Name: "http.method", Type: "string"},
			{Scope: "span", Name: "namespace", Type: "string"},
		},
	}

	expectedJSON := `{
    	"format": "vParquet3",
    	"blockID": "00000000-0000-0000-0000-000000000000",
    	"tenantID": "single-tenant",
		"startTime": "2021-01-01T00:00:00Z",
    	"endTime": "2021-01-02T00:00:00Z",
    	"totalObjects": 10,
    	"size": 12345,
    	"compactionLevel": 1,
		"encoding": "zstd",
    	"indexPageSize": 250000,
    	"totalRecords": 124356,
    	"dataEncoding": "",
    	"bloomShards": 244,
		"footerSize": 15775,
    	"dedicatedColumns": [
    		{"s": "resource", "n": "namespace"},
    		{"n": "http.method"},
    		{"n": "namespace"}
    	]
	}`

	metaJSON, err := json.Marshal(meta)
	require.NoError(t, err)
	assert.JSONEq(t, expectedJSON, string(metaJSON))

	var metaRoundtrip BlockMeta
	err = json.Unmarshal(metaJSON, &metaRoundtrip)
	require.NoError(t, err)
	assert.Equal(t, meta, metaRoundtrip)
}

func TestDedicatedColumnsFromTempopb(t *testing.T) {
	tests := []struct {
		name        string
		cols        []*tempopb.DedicatedColumn
		expected    DedicatedColumns
		expectedErr error
	}{
		{
			name: "no error",
			cols: []*tempopb.DedicatedColumn{
				{Scope: tempopb.DedicatedColumn_SPAN, Name: "test.span.1", Type: tempopb.DedicatedColumn_STRING},
				{Scope: tempopb.DedicatedColumn_RESOURCE, Name: "test.res.1", Type: tempopb.DedicatedColumn_STRING},
				{Scope: tempopb.DedicatedColumn_SPAN, Name: "test.span.2", Type: tempopb.DedicatedColumn_STRING},
			},
			expected: DedicatedColumns{
				{Scope: DedicatedColumnScopeSpan, Name: "test.span.1", Type: DedicatedColumnTypeString},
				{Scope: DedicatedColumnScopeResource, Name: "test.res.1", Type: DedicatedColumnTypeString},
				{Scope: DedicatedColumnScopeSpan, Name: "test.span.2", Type: DedicatedColumnTypeString},
			},
		},
		{
			name: "wrong type",
			cols: []*tempopb.DedicatedColumn{
				{Scope: tempopb.DedicatedColumn_RESOURCE, Name: "test.res.1", Type: tempopb.DedicatedColumn_Type(3)},
				{Scope: tempopb.DedicatedColumn_SPAN, Name: "test.span.2", Type: tempopb.DedicatedColumn_STRING},
			},
			expectedErr: errors.New("unable to convert dedicated column 'test.res.1': invalid value for tempopb.DedicatedColumn_Type '3'"),
		},
		{
			name: "wrong scope",
			cols: []*tempopb.DedicatedColumn{
				{Scope: tempopb.DedicatedColumn_RESOURCE, Name: "test.res.1", Type: tempopb.DedicatedColumn_STRING},
				{Scope: tempopb.DedicatedColumn_Scope(4), Name: "test.span.2", Type: tempopb.DedicatedColumn_STRING},
			},
			expectedErr: errors.New("unable to convert dedicated column 'test.span.2': invalid value for tempopb.DedicatedColumn_Scope '4'"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cols, err := DedicatedColumnsFromTempopb(tc.cols)
			if tc.expectedErr != nil {
				require.Error(t, err)
				assert.EqualError(t, err, tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, cols)
			}
		})
	}
}

func TestDedicatedColumns_ToTempopb(t *testing.T) {
	tests := []struct {
		name        string
		cols        DedicatedColumns
		expected    []*tempopb.DedicatedColumn
		expectedErr error
	}{
		{
			name: "no error",
			cols: DedicatedColumns{
				{Scope: DedicatedColumnScopeSpan, Name: "test.span.1", Type: DedicatedColumnTypeString},
				{Scope: DedicatedColumnScopeResource, Name: "test.res.1", Type: DedicatedColumnTypeString},
				{Scope: DedicatedColumnScopeSpan, Name: "test.span.2", Type: DedicatedColumnTypeString},
			},
			expected: []*tempopb.DedicatedColumn{
				{Scope: tempopb.DedicatedColumn_SPAN, Name: "test.span.1", Type: tempopb.DedicatedColumn_STRING},
				{Scope: tempopb.DedicatedColumn_RESOURCE, Name: "test.res.1", Type: tempopb.DedicatedColumn_STRING},
				{Scope: tempopb.DedicatedColumn_SPAN, Name: "test.span.2", Type: tempopb.DedicatedColumn_STRING},
			},
		},
		{
			name: "wrong type",
			cols: DedicatedColumns{
				{Scope: DedicatedColumnScopeSpan, Name: "test.span.1", Type: DedicatedColumnType("no-type")},
				{Scope: DedicatedColumnScopeResource, Name: "test.res.1", Type: DedicatedColumnTypeString},
			},
			expectedErr: errors.New("unable to convert dedicated column 'test.span.1': invalid value for dedicated column type 'no-type'"),
		},
		{
			name: "wrong scope",
			cols: DedicatedColumns{
				{Scope: DedicatedColumnScopeResource, Name: "test.res.1", Type: DedicatedColumnTypeString},
				{Scope: DedicatedColumnScope("no-scope"), Name: "test.span.2", Type: DedicatedColumnTypeString},
			},
			expectedErr: errors.New("unable to convert dedicated column 'test.span.2': invalid value for dedicated column scope 'no-scope'"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cols, err := tc.cols.ToTempopb()
			if tc.expectedErr != nil {
				require.Error(t, err)
				assert.EqualError(t, err, tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, cols)
			}
		})
	}
}
