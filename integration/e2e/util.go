package e2e

import (
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/cortexproject/cortex/integration/e2e"
	cortex_e2e "github.com/cortexproject/cortex/integration/e2e"
	jaeger_grpc "github.com/jaegertracing/jaeger/cmd/agent/app/reporter/grpc"
	thrift "github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	image = "tempo:latest"
)

func newTempoAllInOne() (*cortex_e2e.HTTPService, error) {
	args := "-config.file=" + filepath.Join(cortex_e2e.ContainerSharedDir, "config.yaml")

	return cortex_e2e.NewHTTPService(
		"tempo",
		image,
		cortex_e2e.NewCommandWithoutEntrypoint("/tempo", args),
		cortex_e2e.NewHTTPReadinessProbe(3100, "/ready", 200, 505),
		3100,
		14250,
	), nil
}

func newTempoDistributor() (*cortex_e2e.HTTPService, error) {
	args := []string{"-config.file=" + filepath.Join(cortex_e2e.ContainerSharedDir, "config.yaml"), "-target=distributor"}

	return cortex_e2e.NewHTTPService(
		"distributor",
		image,
		cortex_e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		cortex_e2e.NewHTTPReadinessProbe(3100, "/ready", 200, 505),
		3100,
		14250,
	), nil
}

func newTempoIngester(replica int) (*cortex_e2e.HTTPService, error) {
	args := []string{"-config.file=" + filepath.Join(cortex_e2e.ContainerSharedDir, "config.yaml"), "-target=ingester"}

	return cortex_e2e.NewHTTPService(
		"ingester-"+strconv.Itoa(replica),
		image,
		cortex_e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		cortex_e2e.NewHTTPReadinessProbe(3100, "/ready", 200, 505),
		3100,
	), nil
}

func newTempoQuerier() (*cortex_e2e.HTTPService, error) {
	args := []string{"-config.file=" + filepath.Join(cortex_e2e.ContainerSharedDir, "config.yaml"), "-target=querier"}

	return cortex_e2e.NewHTTPService(
		"querier",
		image,
		cortex_e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		cortex_e2e.NewHTTPReadinessProbe(3100, "/ready", 200, 505),
		3100,
	), nil
}

func newJaegerGRPCClient(endpoint string) (*jaeger_grpc.Reporter, error) {
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

func makeThriftBatch() *thrift.Batch {
	var spans []*thrift.Span
	spans = append(spans, &thrift.Span{
		TraceIdLow:    rand.Int63(),
		TraceIdHigh:   0,
		SpanId:        rand.Int63(),
		ParentSpanId:  0,
		OperationName: "my operation",
		References:    nil,
		Flags:         0,
		StartTime:     time.Now().Unix(),
		Duration:      1,
		Tags:          nil,
		Logs:          nil,
	})
	return &thrift.Batch{Spans: spans}
}

func writeFileToSharedDir(s *e2e.Scenario, dst string, content []byte) error {
	dst = filepath.Join(s.SharedDir(), dst)

	// Ensure the entire path of directories exist.
	if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
		return err
	}

	return ioutil.WriteFile(
		dst,
		content,
		os.ModePerm)
}

func copyFileToSharedDir(s *e2e.Scenario, src, dst string) error {
	content, err := ioutil.ReadFile(src)
	if err != nil {
		return errors.Wrapf(err, "unable to read local file %s", src)
	}

	return writeFileToSharedDir(s, dst, content)
}
