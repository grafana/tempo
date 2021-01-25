package frontend

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
	"github.com/weaveworks/common/user"

	"github.com/grafana/tempo/modules/querier"
	"github.com/grafana/tempo/pkg/util"
)

const (
	querierPrefix  = "/querier"
	queryDelimiter = "?"
)

func ShardingWare(queryShards int, logger log.Logger) Middleware {
	return MiddlewareFunc(func(next Handler) Handler {
		return shardQuery{
			next:        next,
			queryShards: queryShards,
			logger:      logger,
		}
	})
}

type shardQuery struct {
	next            Handler
	queryShards     int
	logger          log.Logger
	blockBoundaries [][]byte
}

// Do implements Handler
func (s shardQuery) Do(r *http.Request) (*http.Response, error) {
	userID, err := user.ExtractOrgID(r.Context())
	if err != nil {
		return nil, err
	}

	// only need to initialise boundaries once
	if len(s.blockBoundaries) == 0 {
		s.blockBoundaries = createBlockBoundaries(s.queryShards)
	}

	reqs := make([]*http.Request, s.queryShards)
	for i := 0; i < len(s.blockBoundaries)-1; i++ {
		reqs[i] = r.Clone(r.Context())
		q := reqs[i].URL.Query()
		q.Add(querier.BlockStartKey, hex.EncodeToString(s.blockBoundaries[i]))
		q.Add(querier.BlockEndKey, hex.EncodeToString(s.blockBoundaries[i+1]))

		if i == 0 {
			q.Add(querier.QueryIngestersKey, "true")
		} else {
			q.Add(querier.QueryIngestersKey, "false")
		}

		reqs[i].Header.Set(user.OrgIDHeaderName, userID)

		// adding to RequestURI only because weaveworks/common uses the RequestURI field to
		// translate from http.Request to httpgrpc.Request
		// https://github.com/weaveworks/common/blob/47e357f4e1badb7da17ad74bae63e228bdd76e8f/httpgrpc/server/server.go#L48
		reqs[i].RequestURI = querierPrefix + reqs[i].URL.RequestURI() + queryDelimiter + q.Encode()
	}

	rrs, err := doRequests(reqs, s.next)
	if err != nil {
		return nil, err
	}

	return mergeResponses(rrs)
}

// createBlockBoundaries splits the range of blockIDs into queryShards parts
func createBlockBoundaries(queryShards int) [][]byte {
	if queryShards == 0 {
		return nil
	}

	// create sharded queries
	blockBoundaries := make([][]byte, queryShards+1)
	for i := 0; i < queryShards+1; i++ {
		blockBoundaries[i] = make([]byte, 16)
	}
	const MaxUint = uint64(^uint8(0))
	for i := 0; i < queryShards; i++ {
		binary.LittleEndian.PutUint64(blockBoundaries[i][:8], (MaxUint/uint64(queryShards))*uint64(i))
		binary.LittleEndian.PutUint64(blockBoundaries[i][8:], 0)
	}
	const MaxUint64 = ^uint64(0)
	binary.LittleEndian.PutUint64(blockBoundaries[queryShards][:8], MaxUint64)
	binary.LittleEndian.PutUint64(blockBoundaries[queryShards][8:], MaxUint64)

	return blockBoundaries
}

// RequestResponse contains a request response and the respective request that was used.
type RequestResponse struct {
	Request  *http.Request
	Response *http.Response
}

// doRequests executes a list of requests in parallel.
func doRequests(reqs []*http.Request, downstream Handler) ([]RequestResponse, error) {
	respChan, errChan := make(chan RequestResponse), make(chan error)
	for _, req := range reqs {
		go func(req *http.Request) {
			resp, err := downstream.Do(req)
			if err != nil {
				errChan <- err
			} else {
				respChan <- RequestResponse{req, resp}
			}
		}(req)
	}

	resps := make([]RequestResponse, 0, len(reqs))
	var firstErr error
	for range reqs {
		select {
		case resp := <-respChan:
			resps = append(resps, resp)
		case err := <-errChan:
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return resps, firstErr
}

func mergeResponses(rrs []RequestResponse) (*http.Response, error) {
	var errCode = http.StatusOK
	var errBody io.ReadCloser
	var combinedTrace []byte
	var shardMissCount = 0
	for _, rr := range rrs {
		if rr.Response.StatusCode == http.StatusOK {
			body, err := ioutil.ReadAll(rr.Response.Body)
			rr.Response.Body.Close()
			if err != nil {
				return nil, errors.Wrap(err, "error reading response body at query frontend")
			}

			if len(combinedTrace) == 0 {
				combinedTrace = body
			} else {
				combinedTrace = util.CombineTraces(combinedTrace, body)
			}
		} else if rr.Response.StatusCode != http.StatusNotFound {
			errCode = rr.Response.StatusCode
			errBody = rr.Response.Body
		} else {
			shardMissCount++
		}
	}

	if shardMissCount == len(rrs) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       ioutil.NopCloser(strings.NewReader("trace not found in Tempo")),
			Header:     http.Header{},
		}, nil
	}

	if errCode == http.StatusOK {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader(combinedTrace)),
			Header:     http.Header{},
		}, nil
	}

	// Propagate any other errors as 5xx to the user so they can retry the query
	return &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       errBody,
		Header:     http.Header{},
	}, nil
}
