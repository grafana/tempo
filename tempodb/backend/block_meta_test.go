package backend

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	uuid "github.com/google/uuid"
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

	assert.Equal(t, id, (uuid.UUID)(b.BlockID))
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
		expectedObjects int64
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
			b.ObjectAdded(tc.starts[i], tc.ends[i])
		}

		assert.Equal(t, tc.expectedStart, b.StartTime)
		assert.Equal(t, tc.expectedEnd, b.EndTime)
		assert.Equal(t, tc.expectedObjects, b.TotalObjects)
	}
}

func TestBlockMetaJSONRoundTrip(t *testing.T) {
	timeParse := func(s string) time.Time {
		date, err := time.Parse(time.RFC3339Nano, s)
		require.NoError(t, err)
		return date
	}

	meta := BlockMeta{
		Version:         "vParquet3",
		BlockID:         MustParse("00000000-0000-0000-0000-000000000000"),
		TenantID:        "single-tenant",
		StartTime:       timeParse("2021-01-01T00:00:00.0000000Z"),
		EndTime:         timeParse("2021-01-02T00:00:00.0000000Z"),
		TotalObjects:    10,
		Size_:           12345,
		CompactionLevel: 1,
		Encoding:        EncZstd,
		IndexPageSize:   250000,
		TotalRecords:    124356,
		DataEncoding:    "",
		BloomShardCount: 244,
		FooterSize:      15775,
		DedicatedColumns: DedicatedColumns{
			{Scope: "resource", Name: "namespace", Type: "string"},
			{Scope: "resource", Name: "net.host.port", Type: "int"},
			{Scope: "span", Name: "http.method", Type: "string"},
			{Scope: "span", Name: "namespace", Type: "string"},
			{Scope: "span", Name: "http.response.body.size", Type: "int"},
			{Scope: "span", Name: "http.request.header.accept", Type: "string", Options: DedicatedColumnOptions{DedicatedColumnOptionArray}},
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
    		{"s": "resource", "n": "net.host.port", "t": "int"},
    		{"n": "http.method"},
    		{"n": "namespace"},
    		{"n": "http.response.body.size", "t": "int"},
    		{"n": "http.request.header.accept", "o": ["array"]}
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
				{Scope: tempopb.DedicatedColumn_SPAN, Name: "test.span.3", Type: tempopb.DedicatedColumn_INT},
				{Scope: tempopb.DedicatedColumn_SPAN, Name: "test.span.4", Type: tempopb.DedicatedColumn_INT, Options: tempopb.DedicatedColumn_ARRAY},
			},
			expected: DedicatedColumns{
				{Scope: DedicatedColumnScopeSpan, Name: "test.span.1", Type: DedicatedColumnTypeString},
				{Scope: DedicatedColumnScopeResource, Name: "test.res.1", Type: DedicatedColumnTypeString},
				{Scope: DedicatedColumnScopeSpan, Name: "test.span.2", Type: DedicatedColumnTypeString},
				{Scope: DedicatedColumnScopeSpan, Name: "test.span.3", Type: DedicatedColumnTypeInt},
				{Scope: DedicatedColumnScopeSpan, Name: "test.span.4", Type: DedicatedColumnTypeInt, Options: DedicatedColumnOptions{DedicatedColumnOptionArray}},
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
				{Scope: DedicatedColumnScopeResource, Name: "test.res.2", Type: DedicatedColumnTypeInt},
				{Scope: DedicatedColumnScopeSpan, Name: "test.span.2", Type: DedicatedColumnTypeString},
				{Scope: DedicatedColumnScopeSpan, Name: "test.span.3", Type: DedicatedColumnTypeInt, Options: DedicatedColumnOptions{}},
				{Scope: DedicatedColumnScopeSpan, Name: "test.span.4", Type: DedicatedColumnTypeInt, Options: DedicatedColumnOptions{DedicatedColumnOptionArray}},
			},
			expected: []*tempopb.DedicatedColumn{
				{Scope: tempopb.DedicatedColumn_SPAN, Name: "test.span.1", Type: tempopb.DedicatedColumn_STRING},
				{Scope: tempopb.DedicatedColumn_RESOURCE, Name: "test.res.1", Type: tempopb.DedicatedColumn_STRING},
				{Scope: tempopb.DedicatedColumn_RESOURCE, Name: "test.res.2", Type: tempopb.DedicatedColumn_INT},
				{Scope: tempopb.DedicatedColumn_SPAN, Name: "test.span.2", Type: tempopb.DedicatedColumn_STRING},
				{Scope: tempopb.DedicatedColumn_SPAN, Name: "test.span.3", Type: tempopb.DedicatedColumn_INT},
				{Scope: tempopb.DedicatedColumn_SPAN, Name: "test.span.4", Type: tempopb.DedicatedColumn_INT, Options: tempopb.DedicatedColumn_ARRAY},
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
		{
			name: "wrong option",
			cols: DedicatedColumns{
				{Scope: DedicatedColumnScopeResource, Name: "test.res.1", Type: DedicatedColumnTypeString},
				{Scope: DedicatedColumnScopeSpan, Name: "test.span.2", Type: DedicatedColumnTypeString, Options: DedicatedColumnOptions{DedicatedColumnOptionArray, DedicatedColumnOption("no-option")}},
			},
			expectedErr: errors.New("unable to convert dedicated column 'test.span.2': invalid value for dedicated column option 'no-option'"),
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

func TestDedicatedColumnsMarshalRoundTrip(t *testing.T) {
	roundTripTestCases := []struct {
		name      string
		skipProto bool
		cols      DedicatedColumns
	}{
		{
			name: "null",
		},
		{
			name:      "empty",
			cols:      DedicatedColumns{},
			skipProto: true,
		},
		{
			name: "single",
			cols: DedicatedColumns{
				{Scope: DedicatedColumnScopeSpan, Name: "test.span.1", Type: DedicatedColumnTypeString},
			},
		},
		{
			name: "multiple",
			cols: DedicatedColumns{
				{Scope: DedicatedColumnScopeResource, Name: "test.res.1", Type: DedicatedColumnTypeString},
				{Scope: DedicatedColumnScopeSpan, Name: "test.span.1", Type: DedicatedColumnTypeString},
				{Scope: DedicatedColumnScopeSpan, Name: "test.span.2", Type: DedicatedColumnTypeString},
				{Scope: DedicatedColumnScopeSpan, Name: "test.span.3", Type: DedicatedColumnTypeInt},
				{Scope: DedicatedColumnScopeSpan, Name: "test.span.4", Type: DedicatedColumnTypeString, Options: DedicatedColumnOptions{DedicatedColumnOptionArray}},
			},
		},
	}

	t.Run("json", func(t *testing.T) {
		for _, tc := range roundTripTestCases {
			t.Run(tc.name, func(t *testing.T) {
				data, err := json.Marshal(tc.cols)
				require.NoError(t, err)

				var cols DedicatedColumns
				err = json.Unmarshal(data, &cols)
				require.NoError(t, err)
				assert.Equal(t, tc.cols, cols)
			})
		}
	})

	t.Run("proto", func(t *testing.T) {
		for _, tc := range roundTripTestCases {
			if tc.skipProto {
				continue
			}
			t.Run(tc.name, func(t *testing.T) {
				// DedicatedColumns does not implement proto.Message, needs to be wrapped in BlockMeta
				bm1 := BlockMeta{DedicatedColumns: tc.cols}

				data, err := proto.Marshal(&bm1)
				require.NoError(t, err)

				var bm2 BlockMeta
				err = proto.Unmarshal(data, &bm2)
				require.NoError(t, err)
				assert.Equal(t, tc.cols, bm2.DedicatedColumns)
			})
		}
	})
}

func TestDedicatedColumns_Validate(t *testing.T) {
	testCases := []struct {
		name    string
		cols    DedicatedColumns
		isValid bool
	}{
		{name: "nil", isValid: true},
		{name: "empty", cols: DedicatedColumns{}, isValid: true},
		{
			name: "minimal valid",
			cols: DedicatedColumns{
				{Name: "test.span.str", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
			},
			isValid: true,
		},
		{
			name: "valid with options",
			cols: DedicatedColumns{
				{Name: "test.span.str", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString, Options: DedicatedColumnOptions{DedicatedColumnOptionArray}},
			},
			isValid: true,
		},
		{
			name: "all valid scopes and types",
			cols: DedicatedColumns{
				{Name: "test.span.str", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.span.int", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeInt},
				{Name: "test.res.str", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.res.int", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeInt},
			},
			isValid: true,
		},
		{
			name: "maximum allowed columns",
			cols: DedicatedColumns{
				{Name: "test.res.str-01", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.res.str-02", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.res.str-03", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.res.str-04", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.res.str-05", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.res.str-06", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.res.str-07", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.res.str-08", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.res.str-09", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.res.str-10", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},

				{Name: "test.res.int-01", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeInt},
				{Name: "test.res.int-02", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeInt},
				{Name: "test.res.int-03", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeInt},
				{Name: "test.res.int-04", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeInt},
				{Name: "test.res.int-05", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeInt},

				{Name: "test.span.str-01", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.span.str-02", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.span.str-03", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.span.str-04", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.span.str-05", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.span.str-06", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.span.str-07", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.span.str-08", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.span.str-09", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.span.str-10", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},

				{Name: "test.span.int-01", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeInt},
				{Name: "test.span.int-02", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeInt},
				{Name: "test.span.int-03", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeInt},
				{Name: "test.span.int-04", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeInt},
				{Name: "test.span.int-05", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeInt},
			},
			isValid: true,
		},
		{
			name: "duplicated names different scope",
			cols: DedicatedColumns{
				{Name: "test.same", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.span.str", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.same", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
			},
			isValid: true,
		},
		{
			name: "duplicated names same scope",
			cols: DedicatedColumns{
				{Name: "test.same", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.span.str", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.same", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
			},
		},
		{
			name: "empty name",
			cols: DedicatedColumns{
				{Name: "test.span.str", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
			},
		},
		{
			name: "duplicated names same scope",
			cols: DedicatedColumns{
				{Name: "test.span.str", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
			},
		},
		{
			name: "invalid scope",
			cols: DedicatedColumns{
				{Name: "test.span.str", Scope: "link", Type: DedicatedColumnTypeString},
				{Name: "test.span.str", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
			},
		},
		{
			name: "invalid type",
			cols: DedicatedColumns{
				{Name: "test.span.str", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.span.str", Scope: DedicatedColumnScopeSpan, Type: "float"},
			},
		},
		{
			name: "invalid option",
			cols: DedicatedColumns{
				{Name: "test.span.str", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.span.str", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString, Options: DedicatedColumnOptions{DedicatedColumnOption("no-option")}},
			},
		},
		{
			name: "too many resource str cols",
			cols: DedicatedColumns{
				{Name: "test.span.str", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.res.str-01", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.res.str-02", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.res.str-03", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.res.str-04", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.res.str-05", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.res.str-06", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.res.str-07", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.res.str-08", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.res.str-09", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.res.str-10", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.res.str-11", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
			},
		},
		{
			name: "too many resource int cols",
			cols: DedicatedColumns{
				{Name: "test.span.str", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.res.int-01", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeInt},
				{Name: "test.res.int-02", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeInt},
				{Name: "test.res.int-03", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeInt},
				{Name: "test.res.int-04", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeInt},
				{Name: "test.res.int-05", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeInt},
				{Name: "test.res.int-06", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeInt},
			},
		},
		{
			name: "too many span str cols",
			cols: DedicatedColumns{
				{Name: "test.res.str", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.span.str-01", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.span.str-02", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.span.str-03", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.span.str-04", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.span.str-05", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.span.str-06", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.span.str-07", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.span.str-08", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.span.str-09", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.span.str-10", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
				{Name: "test.span.str-11", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeString},
			},
		},
		{
			name: "too many span int cols",
			cols: DedicatedColumns{
				{Name: "test.res.str", Scope: DedicatedColumnScopeResource, Type: DedicatedColumnTypeString},
				{Name: "test.span.int-01", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeInt},
				{Name: "test.span.int-02", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeInt},
				{Name: "test.span.int-03", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeInt},
				{Name: "test.span.int-04", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeInt},
				{Name: "test.span.int-05", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeInt},
				{Name: "test.span.int-06", Scope: DedicatedColumnScopeSpan, Type: DedicatedColumnTypeInt},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cols.Validate()
			if tc.isValid {
				require.NoError(t, err, "dedicated columns expected to be valid, but got error: %s", err)
			} else {
				require.Error(t, err, "dedicated columns expected to be invalid, but got no error")
			}
		})
	}
}
