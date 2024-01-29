package frontend

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTagsResultsHandler(t *testing.T) {
	start := uint32(100)
	end := uint32(200)
	bm := backend.NewBlockMeta("test", uuid.New(), "wdwad", backend.EncGZIP, "asdf")
	bm.StartTime = time.Unix(int64(start), 0)
	bm.EndTime = time.Unix(int64(end), 0)
	bm.Size = defaultTargetBytesPerRequest * 2
	bm.TotalRecords = 2

	encodingURLPart := "&dataEncoding=asdf&encoding=gzip&end=200&"
	totalRecordsURLPart := "totalRecords=2&version=wdwad"
	blockIDURLPart := "blockID="

	tests := []struct {
		name                 string
		request              string
		factory              tagResultHandlerFactory
		result1              string
		result2              string
		wrongResult          string
		expectedResult       string
		expectedReq          func(r *http.Request) *http.Request
		expectedBlockURL     string
		overflowRes1         string
		overflowRes2         string
		limit                int
		assertResultFunction func(t *testing.T, result, expected string)
		parseRequestFunction parseRequestFunction
	}{
		{
			name:           "TagsResultHandler",
			factory:        tagsResultHandlerFactory,
			request:        "/?start=100&end=200",
			result1:        "{ \"tagNames\":[\"tag1\"]}",
			result2:        "{ \"tagNames\":[\"tag2\",\"tag3\"]}",
			expectedResult: "{\"tagNames\":[\"tag1\",\"tag2\",\"tag3\"]}",
			expectedReq: func(r *http.Request) *http.Request {
				expectedReq, _ := api.BuildSearchTagsRequest(r, &tempopb.SearchTagsRequest{
					Scope: "all",
					Start: start,
					End:   end,
				})
				return expectedReq
			},
			expectedBlockURL: blockIDURLPart + bm.BlockID.String() + encodingURLPart +
				"footerSize=0&indexPageSize=0&pagesToSearch=1&scope=all&size=209715200&start=100&startPage=0&" +
				totalRecordsURLPart,
			overflowRes1: "{ \"tagNames\":[\"tag1\"]}",
			overflowRes2: "{ \"tagNames\":[\"tag2\"]}",
			limit:        5,
			assertResultFunction: func(t *testing.T, result, expected string) {
				resultStruct := tempopb.SearchTagsResponse{}
				expectedStruct := tempopb.SearchTagsResponse{}

				err := json.Unmarshal([]byte(result), &resultStruct)
				require.NoError(t, err)
				err = json.Unmarshal([]byte(expected), &expectedStruct)
				require.NoError(t, err)

				sort.Strings(expectedStruct.TagNames)
				sort.Strings(resultStruct.TagNames)
				assert.Equal(t, expectedStruct, resultStruct)
			},
			parseRequestFunction: parseTagsRequest,
		},
		{
			name:           "TagValuesResultHandler",
			factory:        tagValuesResultHandlerFactory,
			request:        "/?start=100&end=200",
			result1:        "{ \"tagValues\":[\"tag1\"]}",
			result2:        "{ \"tagValues\":[\"tag2\",\"tag3\"]}",
			expectedResult: "{\"tagValues\":[\"tag1\",\"tag2\",\"tag3\"]}",
			expectedReq: func(r *http.Request) *http.Request {
				expectedReq, _ := api.BuildSearchTagValuesRequest(r, &tempopb.SearchTagValuesRequest{
					Start: start,
					End:   end,
				})
				return expectedReq
			},
			expectedBlockURL: blockIDURLPart + bm.BlockID.String() + encodingURLPart +
				"footerSize=0&indexPageSize=0&pagesToSearch=1&q=&size=209715200&start=100&startPage=0&" +
				totalRecordsURLPart,
			overflowRes1: "{ \"tagValues\":[\"tag1\"]}",
			overflowRes2: "{ \"tagValues\":[\"tag2\"]}",
			limit:        5,
			assertResultFunction: func(t *testing.T, result, expected string) {
				resultStruct := tempopb.SearchTagValuesResponse{}
				expectedStruct := tempopb.SearchTagValuesResponse{}

				err := json.Unmarshal([]byte(result), &resultStruct)
				require.NoError(t, err)
				err = json.Unmarshal([]byte(expected), &expectedStruct)
				require.NoError(t, err)

				sort.Strings(expectedStruct.TagValues)
				sort.Strings(resultStruct.TagValues)
				assert.Equal(t, expectedStruct, resultStruct)
			},
			parseRequestFunction: parseTagValuesRequest,
		},
		{
			name:           "TagValuesV2ResultHandler",
			request:        "/.service.name/?start=100&end=200",
			factory:        tagValuesV2ResultHandlerFactory,
			result1:        "{\"tagValues\":[{\"type\":\"string\",\"value\":\"v1\"}]}",
			result2:        "{\"tagValues\":[{\"type\":\"string\",\"value\":\"v2\"},{\"type\":\"string\",\"value\":\"v3\"}]}",
			expectedResult: "{\"tagValues\":[{\"type\":\"string\",\"value\":\"v1\"},{\"type\":\"string\",\"value\":\"v2\"},{\"type\":\"string\",\"value\":\"v3\"}]}",
			expectedReq: func(r *http.Request) *http.Request {
				expectedReq, _ := api.BuildSearchTagValuesRequest(r, &tempopb.SearchTagValuesRequest{
					Start: start,
					End:   end,
				})
				return expectedReq
			},
			expectedBlockURL: blockIDURLPart + bm.BlockID.String() + encodingURLPart +
				"footerSize=0&indexPageSize=0&pagesToSearch=1&q=&size=209715200&start=100&startPage=0&" +
				totalRecordsURLPart,
			overflowRes1: "{\"tagValues\":[{\"type\":\"string\",\"value\":\"tag1\"}]}",
			overflowRes2: "{\"tagValues\":[{\"type\":\"string\",\"value\":\"tag2\"}]}",
			limit:        15,
			assertResultFunction: func(t *testing.T, result, expected string) {
				resultStruct := tempopb.SearchTagValuesV2Response{}
				expectedStruct := tempopb.SearchTagValuesV2Response{}

				err := json.Unmarshal([]byte(result), &resultStruct)
				require.NoError(t, err)
				err = json.Unmarshal([]byte(expected), &expectedStruct)
				require.NoError(t, err)

				sort.SliceStable(resultStruct.TagValues, func(i, j int) bool {
					return resultStruct.TagValues[i].Value < resultStruct.TagValues[j].Value
				})

				sort.SliceStable(expectedStruct.TagValues, func(i, j int) bool {
					return expectedStruct.TagValues[i].Value < expectedStruct.TagValues[j].Value
				})

				assert.Equal(t, expectedStruct, resultStruct)
			},
			parseRequestFunction: parseTagValuesRequest,
		},
		{
			name:           "TagsV2ResultHandler",
			request:        "/.service.name/?start=100&end=200",
			factory:        tagsV2ResultHandlerFactory,
			result1:        "{\"scopes\":[{\"name\":\"scope1\",\"tags\":[\"v1\"]}]}",
			result2:        "{\"scopes\":[{\"name\":\"scope1\",\"tags\":[\"v1\",\"v2\"]}]}",
			expectedResult: "{\"scopes\":[{\"name\":\"scope1\",\"tags\":[\"v1\",\"v2\"]}]}",
			expectedReq: func(r *http.Request) *http.Request {
				expectedReq, _ := api.BuildSearchTagValuesRequest(r, &tempopb.SearchTagValuesRequest{
					Start: start,
					End:   end,
				})
				return expectedReq
			},
			expectedBlockURL: blockIDURLPart + bm.BlockID.String() + encodingURLPart +
				"footerSize=0&indexPageSize=0&pagesToSearch=1&q=&scope=&size=209715200&start=100&startPage=0&" +
				totalRecordsURLPart,
			overflowRes1: "{\"scopes\":[{\"name\":\"scope1\",\"tags\":[\"tag1\"]}]}",
			overflowRes2: "{\"scopes\":[{\"name\":\"scope1\",\"tags\":[\"tag2\"]}]}",
			limit:        5,
			assertResultFunction: func(t *testing.T, result, expected string) {
				resultStruct := tempopb.SearchTagsV2Response{}
				expectedStruct := tempopb.SearchTagsV2Response{}

				err := json.Unmarshal([]byte(result), &resultStruct)
				require.NoError(t, err)
				err = json.Unmarshal([]byte(expected), &expectedStruct)
				require.NoError(t, err)

				sort.Strings(resultStruct.Scopes[0].Tags)
				sort.Strings(expectedStruct.Scopes[0].Tags)

				assert.Equal(t, expectedStruct, resultStruct)
			},
			parseRequestFunction: parseTagsRequest,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", tc.request, nil)
			r = mux.SetURLVars(r, map[string]string{
				"tagName": "service.name",
			})

			handler := tc.factory(100)

			err := handler.addResponse(io.NopCloser(bytes.NewBufferString("{}")))
			assert.NoError(t, err)

			err = handler.addResponse(io.NopCloser(bytes.NewBufferString(tc.result1)))
			assert.NoError(t, err)

			err = handler.addResponse(io.NopCloser(bytes.NewBufferString(tc.result2)))
			assert.NoError(t, err)

			err = handler.addResponse(io.NopCloser(bytes.NewBufferString("{ ]}")))
			assert.Error(t, err)

			res, err := handler.marshalResult()
			require.NoError(t, err)
			tc.assertResultFunction(t, tc.expectedResult, res)

			// Test parse request
			req, err := tc.parseRequestFunction(r)
			require.NoError(t, err)
			assert.Equal(t, start, req.start())
			assert.Equal(t, end, req.end())

			// Test build
			backReq, err := req.buildSearchTagRequest(r)
			assert.NoError(t, err)

			assert.Equal(t, backReq, tc.expectedReq(r))

			blockReq, _ := req.buildTagSearchBlockRequest(r, bm.BlockID.String(), 0, 1, bm)
			assert.Equal(t, tc.expectedBlockURL, blockReq.URL.RawQuery)

			handlerOverflow := tc.factory(tc.limit)
			err = handlerOverflow.addResponse(io.NopCloser(bytes.NewBufferString(tc.overflowRes1)))
			require.NoError(t, err)

			assert.Equal(t, false, handlerOverflow.shouldQuit())
			err = handlerOverflow.addResponse(io.NopCloser(bytes.NewBufferString(tc.overflowRes2)))
			require.NoError(t, err)
			assert.Equal(t, true, handlerOverflow.shouldQuit())
		})
	}
}
