package frontend

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/golang/protobuf/proto"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
)

func newTraceCombiner(logger log.Logger) traceCombiner {
	return traceCombiner{
		logger:     logger,
		combiner:   trace.NewCombiner(),
		statusMsg:  "trace not found",
		statusCode: http.StatusNotFound,
	}
}

type traceCombiner struct {
	overallError error
	logger       log.Logger
	combiner     *trace.Combiner
	statusMsg    string
	statusCode   int
}

func (tc traceCombiner) Consume(resp *http.Response) {
	tc.statusCode = resp.StatusCode
	url, _ := url.Parse(resp.Request.URL.String())
	uri := url.RequestURI()
	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		defer resp.Body.Close()

		if err != nil {
			_ = level.Error(tc.logger).Log("msg", "error reading response body status == ok", "url", uri, "err", err)
			tc.overallError = err
			return
		}

		traceResp := &tempopb.TraceByIDResponse{}
		err = proto.Unmarshal(body, traceResp)
		if err != nil {
			_ = level.Error(tc.logger).Log("msg", "error unmarshalling response", "url", uri, "err", err, "body", string(body))
			tc.overallError = err
			return
		}

		tc.combiner.Consume(traceResp.Trace)

	} else {
		bytesMsg, err := io.ReadAll(resp.Body)
		if err != nil {
			_ = level.Error(tc.logger).Log("msg", "error reading response body status != ok", "url", uri, "err", err)
		}
		tc.statusMsg = string(bytesMsg)
	}
}

func (tc traceCombiner) Result() (*http.Response, error) {
	if tc.overallError != nil {
		return nil, tc.overallError
	}

	overallTrace, _ := tc.combiner.Result()
	if overallTrace == nil || tc.statusCode != http.StatusOK {
		return &http.Response{
			StatusCode: tc.statusCode,
			Body:       io.NopCloser(strings.NewReader(tc.statusMsg)),
			Header:     http.Header{},
		}, nil
	}

	buff, err := proto.Marshal(&tempopb.TraceByIDResponse{
		Trace:   overallTrace,
		Metrics: &tempopb.TraceByIDMetrics{},
	})
	if err != nil {
		_ = level.Error(tc.logger).Log("msg", "error marshalling response to proto", "err", err)
		return nil, err
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			api.HeaderContentType: {api.HeaderAcceptProtobuf},
		},
		Body:          io.NopCloser(bytes.NewReader(buff)),
		ContentLength: int64(len(buff)),
	}, nil
}
