package integration

// Collection of utilities to share between our various load tests

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	jaeger_grpc "github.com/jaegertracing/jaeger/cmd/agent/app/reporter/grpc"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/grafana/dskit/backoff"
	"github.com/grafana/e2e"
	"github.com/grafana/tempo/pkg/tempopb"
	tempoUtil "github.com/grafana/tempo/pkg/util"
)

const (
	image = "tempo:latest"
)

func NewTempoAllInOne() *e2e.HTTPService {
	args := "-config.file=" + filepath.Join(e2e.ContainerSharedDir, "config.yaml")

	s := e2e.NewHTTPService(
		"tempo",
		image,
		e2e.NewCommandWithoutEntrypoint("/tempo", args),
		e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299),
		3200,  // http all things
		14250, // jaeger grpc ingest
		9411,  // zipkin ingest (used by load)
		4317,  // otlp grpc
	)

	s.SetBackoff(TempoBackoff())

	return s
}

func NewTempoDistributor() *e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(e2e.ContainerSharedDir, "config.yaml"), "-target=distributor"}

	s := e2e.NewHTTPService(
		"distributor",
		image,
		e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299),
		3200,
		14250,
	)

	s.SetBackoff(TempoBackoff())

	return s
}

func NewTempoIngester(replica int) *e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(e2e.ContainerSharedDir, "config.yaml"), "-target=ingester"}

	s := e2e.NewHTTPService(
		"ingester-"+strconv.Itoa(replica),
		image,
		e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299),
		3200,
	)

	s.SetBackoff(TempoBackoff())

	return s
}

func NewTempoQueryFrontend() *e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(e2e.ContainerSharedDir, "config.yaml"), "-target=query-frontend"}

	s := e2e.NewHTTPService(
		"query-frontend",
		image,
		e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299),
		3200,
	)

	s.SetBackoff(TempoBackoff())

	return s
}

func NewTempoQuerier() *e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(e2e.ContainerSharedDir, "config.yaml"), "-target=querier"}

	s := e2e.NewHTTPService(
		"querier",
		image,
		e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299),
		3200,
	)

	s.SetBackoff(TempoBackoff())

	return s
}

func NewTempoScalableSingleBinary(replica int) *e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(e2e.ContainerSharedDir, "config.yaml"), "-target=scalable-single-binary", "-querier.frontend-address=tempo-" + strconv.Itoa(replica) + ":9095"}

	s := e2e.NewHTTPService(
		"tempo-"+strconv.Itoa(replica),
		image,
		e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299),
		3200,  // http all things
		14250, // jaeger grpc ingest
		// 9411,  // zipkin ingest (used by load)
	)

	s.SetBackoff(TempoBackoff())

	return s
}

func WriteFileToSharedDir(s *e2e.Scenario, dst string, content []byte) error {
	dst = filepath.Join(s.SharedDir(), dst)

	// Ensure the entire path of directories exist.
	if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
		return err
	}

	return os.WriteFile(
		dst,
		content,
		os.ModePerm)
}

func CopyFileToSharedDir(s *e2e.Scenario, src, dst string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return errors.Wrapf(err, "unable to read local file %s", src)
	}

	return WriteFileToSharedDir(s, dst, content)
}

func TempoBackoff() backoff.Config {
	return backoff.Config{
		MinBackoff: 500 * time.Millisecond,
		MaxBackoff: time.Second,
		MaxRetries: 300, // Sometimes the CI is slow ¯\_(ツ)_/¯
	}
}

func NewJaegerGRPCClient(endpoint string) (*jaeger_grpc.Reporter, error) {
	// new jaeger grpc exporter
	conn, err := grpc.Dial(endpoint, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}
	return jaeger_grpc.NewReporter(conn, nil, logger), err
}

func SearchAndAssertTrace(t *testing.T, client *tempoUtil.Client, info *tempoUtil.TraceInfo) {
	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	attr := tempoUtil.RandomAttrFromTrace(expected)

	// verify attribute is present in tags
	tagsResp, err := client.SearchTags()
	require.NoError(t, err)
	require.Contains(t, tagsResp.TagNames, attr.Key)

	// verify attribute value is present in tag values
	tagValuesResp, err := client.SearchTagValues(attr.Key)
	require.NoError(t, err)
	require.Contains(t, tagValuesResp.TagValues, strings.ToLower(attr.GetValue().GetStringValue()))

	// verify trace can be found using attribute
	resp, err := client.Search(attr.GetKey() + "=" + attr.GetValue().GetStringValue())
	require.NoError(t, err)

	hasHex := func(hexId string, resp *tempopb.SearchResponse) bool {
		for _, s := range resp.Traces {
			equal, err := tempoUtil.EqualHexStringTraceIDs(s.TraceID, hexId)
			require.NoError(t, err)
			if equal {
				return true
			}
		}

		return false
	}

	require.True(t, hasHex(info.HexID(), resp))
}

// by passing a time range and using a query_ingesters_until/backend_after of 0 we can force the queriers
// to look in the backend blocks
func SearchAndAssertTraceBackend(t *testing.T, client *tempoUtil.Client, info *tempoUtil.TraceInfo, start int64, end int64) {
	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	attr := tempoUtil.RandomAttrFromTrace(expected)

	// verify trace can be found using attribute and time range
	resp, err := client.SearchWithRange(attr.GetKey()+"="+attr.GetValue().GetStringValue(), start, end)
	require.NoError(t, err)

	hasHex := func(hexId string, resp *tempopb.SearchResponse) bool {
		for _, s := range resp.Traces {
			equal, err := tempoUtil.EqualHexStringTraceIDs(s.TraceID, hexId)
			require.NoError(t, err)
			if equal {
				return true
			}
		}

		return false
	}

	require.True(t, hasHex(info.HexID(), resp))
}
