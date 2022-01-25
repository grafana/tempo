package integration

// Collection of utilities to share between our various load tests

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/grafana/dskit/backoff"
	"github.com/grafana/e2e"
	"github.com/pkg/errors"
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
