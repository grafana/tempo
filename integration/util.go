package integration

// Collection of utilities to share between our various load tests

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/cortexproject/cortex/integration/e2e"
	cortex_e2e "github.com/cortexproject/cortex/integration/e2e"
	"github.com/cortexproject/cortex/pkg/util"
	"github.com/pkg/errors"
)

const (
	image        = "tempo:latest"
	azuriteImage = "mcr.microsoft.com/azure-storage/azurite"
)

func NewTempoAllInOne() *cortex_e2e.HTTPService {
	args := "-config.file=" + filepath.Join(cortex_e2e.ContainerSharedDir, "config.yaml")

	s := cortex_e2e.NewHTTPService(
		"tempo",
		image,
		cortex_e2e.NewCommandWithoutEntrypoint("/tempo", args),
		cortex_e2e.NewHTTPReadinessProbe(3100, "/ready", 200, 299),
		3100,  // http all things
		14250, // jaeger grpc ingest
		9411,  // zipkin ingest (used by load)
	)

	s.SetBackoff(tempoBackoff())

	return s
}

func NewAzurite() *cortex_e2e.HTTPService {
	s := cortex_e2e.NewHTTPService(
		"azurite",
		azuriteImage, // Create the the azurite container
		e2e.NewCommandWithoutEntrypoint("sh", "-c", "azurite -l /data --blobHost 0.0.0.0"),
		e2e.NewHTTPReadinessProbe(10000, "/devstoreaccount1?comp=list", 403, 403), //If we get 403 the Azurite is ready
		10000, // blob storage port
	)

	s.SetBackoff(tempoBackoff())

	return s
}

func NewTempoDistributor() *cortex_e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(cortex_e2e.ContainerSharedDir, "config.yaml"), "-target=distributor"}

	s := cortex_e2e.NewHTTPService(
		"distributor",
		image,
		cortex_e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		cortex_e2e.NewHTTPReadinessProbe(3100, "/ready", 200, 299),
		3100,
		14250,
	)

	s.SetBackoff(tempoBackoff())

	return s
}

func NewTempoIngester(replica int) *cortex_e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(cortex_e2e.ContainerSharedDir, "config.yaml"), "-target=ingester"}

	s := cortex_e2e.NewHTTPService(
		"ingester-"+strconv.Itoa(replica),
		image,
		cortex_e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		cortex_e2e.NewHTTPReadinessProbe(3100, "/ready", 200, 299),
		3100,
	)

	s.SetBackoff(tempoBackoff())

	return s
}

func NewTempoQuerier() *cortex_e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(cortex_e2e.ContainerSharedDir, "config.yaml"), "-target=querier"}

	s := cortex_e2e.NewHTTPService(
		"querier",
		image,
		cortex_e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		cortex_e2e.NewHTTPReadinessProbe(3100, "/ready", 200, 299),
		3100,
	)

	s.SetBackoff(tempoBackoff())

	return s
}

func WriteFileToSharedDir(s *e2e.Scenario, dst string, content []byte) error {
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

func CopyFileToSharedDir(s *e2e.Scenario, src, dst string) error {
	content, err := ioutil.ReadFile(src)
	if err != nil {
		return errors.Wrapf(err, "unable to read local file %s", src)
	}

	return WriteFileToSharedDir(s, dst, content)
}

func tempoBackoff() util.BackoffConfig {
	return util.BackoffConfig{
		MinBackoff: 500 * time.Millisecond,
		MaxBackoff: time.Second,
		MaxRetries: 300, // Sometimes the CI is slow ¯\_(ツ)_/¯
	}
}
