package integration

// Collection of utilities to share between our various load tests

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/grafana/dskit/backoff"
	"github.com/grafana/e2e"
	jaeger_grpc "github.com/jaegertracing/jaeger/cmd/agent/app/reporter/grpc"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/tempopb"
	tempoUtil "github.com/grafana/tempo/pkg/util"
)

const (
	image      = "tempo:latest"
	queryImage = "tempo-query:latest"
)

// GetExtraArgs returns the extra args to pass to the Docker command used to run Tempo.
func GetExtraArgs() []string {
	// Get extra args from the TEMPO_EXTRA_ARGS env variable
	// falling back to an empty list
	if os.Getenv("TEMPO_EXTRA_ARGS") != "" {
		return strings.Fields(os.Getenv("TEMPO_EXTRA_ARGS"))
	}

	return nil
}

func buildArgsWithExtra(args, extraArgs []string) []string {
	if len(extraArgs) > 0 {
		args = append(args, extraArgs...)
	}
	if envExtraArgs := GetExtraArgs(); len(envExtraArgs) > 0 {
		args = append(args, envExtraArgs...)
	}

	return args
}

func NewTempoAllInOne(extraArgs ...string) *e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(e2e.ContainerSharedDir, "config.yaml")}
	args = buildArgsWithExtra(args, extraArgs)

	s := e2e.NewHTTPService(
		"tempo",
		image,
		e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299),
		3200,  // http all things
		9095,  // grpc tempo
		14250, // jaeger grpc ingest
		9411,  // zipkin ingest (used by load)
		4317,  // otlp grpc
	)

	s.SetBackoff(TempoBackoff())

	return s
}

func NewTempoDistributor(extraArgs ...string) *e2e.HTTPService {
	return NewNamedTempoDistributor("distributor", extraArgs...)
}

func NewNamedTempoDistributor(name string, extraArgs ...string) *e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(e2e.ContainerSharedDir, "config.yaml"), "-target=distributor"}
	args = buildArgsWithExtra(args, extraArgs)

	s := e2e.NewHTTPService(
		name,
		image,
		e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299),
		3200,
		14250,
	)

	s.SetBackoff(TempoBackoff())

	return s
}

func NewTempoIngester(replica int, extraArgs ...string) *e2e.HTTPService {
	return NewNamedTempoIngester("ingester", replica, extraArgs...)
}

func NewNamedTempoIngester(name string, replica int, extraArgs ...string) *e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(e2e.ContainerSharedDir, "config.yaml"), "-target=ingester"}
	args = buildArgsWithExtra(args, extraArgs)

	s := e2e.NewHTTPService(
		name+"-"+strconv.Itoa(replica),
		image,
		e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299),
		3200,
	)

	s.SetBackoff(TempoBackoff())

	return s
}

func NewTempoMetricsGenerator(extraArgs ...string) *e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(e2e.ContainerSharedDir, "config.yaml"), "-target=metrics-generator"}
	args = buildArgsWithExtra(args, extraArgs)

	s := e2e.NewHTTPService(
		"metrics-generator",
		image,
		e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299),
		3200,
	)

	s.SetBackoff(TempoBackoff())

	return s
}

func NewTempoQueryFrontend(extraArgs ...string) *e2e.HTTPService {
	return NewNamedTempoQueryFrontend("query-frontend", extraArgs...)
}

func NewNamedTempoQueryFrontend(name string, extraArgs ...string) *e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(e2e.ContainerSharedDir, "config.yaml"), "-target=query-frontend"}
	args = buildArgsWithExtra(args, extraArgs)

	s := e2e.NewHTTPService(
		name,
		image,
		e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299),
		3200,
	)

	s.SetBackoff(TempoBackoff())

	return s
}

func NewTempoQuerier(extraArgs ...string) *e2e.HTTPService {
	return NewNamedTempoQuerier("querier", extraArgs...)
}

func NewNamedTempoQuerier(name string, extraArgs ...string) *e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(e2e.ContainerSharedDir, "config.yaml"), "-target=querier"}
	args = buildArgsWithExtra(args, extraArgs)

	s := e2e.NewHTTPService(
		name,
		image,
		e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		e2e.NewHTTPReadinessProbe(3200, "/ready", 200, 299),
		3200,
	)

	s.SetBackoff(TempoBackoff())

	return s
}

func NewTempoScalableSingleBinary(replica int, extraArgs ...string) *e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(e2e.ContainerSharedDir, "config.yaml"), "-target=scalable-single-binary", "-querier.frontend-address=tempo-" + strconv.Itoa(replica) + ":9095"}
	args = buildArgsWithExtra(args, extraArgs)

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

func NewTempoQuery() *e2e.HTTPService {
	args := []string{
		"--query.base-path=/",
		"--grpc-storage-plugin.configuration-file=" + filepath.Join(e2e.ContainerSharedDir, "config-tempo-query.yaml"),
	}

	s := e2e.NewHTTPService(
		"tempo-query",
		queryImage,
		e2e.NewCommandWithoutEntrypoint("/go/bin/query-linux", args...),
		e2e.NewHTTPReadinessProbe(16686, "/", 200, 299),
		16686,
	)

	s.SetBackoff(TempoBackoff())

	return s
}

func CopyFileToSharedDir(s *e2e.Scenario, src, dst string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("unable to read local file %s: %w", src, err)
	}

	_, err = writeFileToSharedDir(s, dst, content)
	return err
}

func CopyTemplateToSharedDir(s *e2e.Scenario, src, dst string, data any) (string, error) {
	tmpl, err := template.ParseFiles(src)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return writeFileToSharedDir(s, dst, buf.Bytes())
}

func writeFileToSharedDir(s *e2e.Scenario, dst string, content []byte) (string, error) {
	dst = filepath.Join(s.SharedDir(), dst)

	// Ensure the entire path of directories exists
	err := os.MkdirAll(filepath.Dir(dst), os.ModePerm)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(dst, content, os.ModePerm)
	if err != nil {
		return "", err
	}

	return dst, nil
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
	conn, err := grpc.Dial(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}
	return jaeger_grpc.NewReporter(conn, nil, logger), err
}

func NewSearchGRPCClient(endpoint string) (tempopb.StreamingQuerierClient, error) {
	clientConn, err := grpc.DialContext(context.Background(), endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return tempopb.NewStreamingQuerierClient(clientConn), nil
}

func SearchAndAssertTrace(t *testing.T, client *httpclient.Client, info *tempoUtil.TraceInfo) {
	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	attr := tempoUtil.RandomAttrFromTrace(expected)

	// NOTE: SearchTags doesn't include live traces anymore
	// so don't check SearchTags

	// verify attribute value is present in tag values
	tagValuesResp, err := client.SearchTagValues(attr.Key)
	require.NoError(t, err)
	require.Contains(t, tagValuesResp.TagValues, attr.GetValue().GetStringValue())

	// verify trace can be found using attribute
	resp, err := client.Search(attr.GetKey() + "=" + attr.GetValue().GetStringValue())
	require.NoError(t, err)

	require.True(t, traceIDInResults(t, info.HexID(), resp))
}

func SearchTraceQLAndAssertTrace(t *testing.T, client *httpclient.Client, info *tempoUtil.TraceInfo) {
	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	attr := tempoUtil.RandomAttrFromTrace(expected)
	query := fmt.Sprintf(`{ .%s = "%s"}`, attr.GetKey(), attr.GetValue().GetStringValue())

	resp, err := client.SearchTraceQL(query)
	require.NoError(t, err)

	require.True(t, traceIDInResults(t, info.HexID(), resp))
}

func SearchStreamAndAssertTrace(t *testing.T, client tempopb.StreamingQuerierClient, info *tempoUtil.TraceInfo, start, end int64) {
	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	attr := tempoUtil.RandomAttrFromTrace(expected)
	query := fmt.Sprintf(`{ .%s = "%s"}`, attr.GetKey(), attr.GetValue().GetStringValue())

	resp, err := client.Search(context.Background(), &tempopb.SearchRequest{
		Query: query,
		Start: uint32(start),
		End:   uint32(end),
	})
	require.NoError(t, err)

	// drain the stream until everything is returned
	found := false
	for {
		searchResp, err := resp.Recv()
		if searchResp != nil {
			found = traceIDInResults(t, info.HexID(), searchResp)
			if found {
				break
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}
	require.True(t, found)
}

// by passing a time range and using a query_ingesters_until/backend_after of 0 we can force the queriers
// to look in the backend blocks
func SearchAndAssertTraceBackend(t *testing.T, client *httpclient.Client, info *tempoUtil.TraceInfo, start, end int64) {
	expected, err := info.ConstructTraceFromEpoch()
	require.NoError(t, err)

	attr := tempoUtil.RandomAttrFromTrace(expected)

	// verify trace can be found using attribute and time range
	resp, err := client.SearchWithRange(attr.GetKey()+"="+attr.GetValue().GetStringValue(), start, end)
	require.NoError(t, err)

	require.True(t, traceIDInResults(t, info.HexID(), resp))
}

func traceIDInResults(t *testing.T, hexID string, resp *tempopb.SearchResponse) bool {
	for _, s := range resp.Traces {
		equal, err := tempoUtil.EqualHexStringTraceIDs(s.TraceID, hexID)
		require.NoError(t, err)
		if equal {
			return true
		}
	}

	return false
}
