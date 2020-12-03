package integration

// Collection of utilities to share between our various load tests

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/cortexproject/cortex/integration/e2e"
	cortex_e2e "github.com/cortexproject/cortex/integration/e2e"
	"github.com/pkg/errors"
)

const (
	image = "tempo:latest"
)

func NewTempoAllInOne() *cortex_e2e.HTTPService {
	args := "-config.file=" + filepath.Join(cortex_e2e.ContainerSharedDir, "config.yaml")

	return cortex_e2e.NewHTTPService(
		"tempo",
		image,
		cortex_e2e.NewCommandWithoutEntrypoint("/tempo", args),
		cortex_e2e.NewHTTPReadinessProbe(3100, "/ready", 200, 505),
		3100,
		14250,
	)
}

func NewTempoDistributor() *cortex_e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(cortex_e2e.ContainerSharedDir, "config.yaml"), "-target=distributor"}

	return cortex_e2e.NewHTTPService(
		"distributor",
		image,
		cortex_e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		cortex_e2e.NewHTTPReadinessProbe(3100, "/ready", 200, 505),
		3100,
		14250,
	)
}

func NewTempoIngester(replica int) *cortex_e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(cortex_e2e.ContainerSharedDir, "config.yaml"), "-target=ingester"}

	return cortex_e2e.NewHTTPService(
		"ingester-"+strconv.Itoa(replica),
		image,
		cortex_e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		cortex_e2e.NewHTTPReadinessProbe(3100, "/ready", 200, 505),
		3100,
	)
}

func NewTempoQuerier() *cortex_e2e.HTTPService {
	args := []string{"-config.file=" + filepath.Join(cortex_e2e.ContainerSharedDir, "config.yaml"), "-target=querier"}

	return cortex_e2e.NewHTTPService(
		"querier",
		image,
		cortex_e2e.NewCommandWithoutEntrypoint("/tempo", args...),
		cortex_e2e.NewHTTPReadinessProbe(3100, "/ready", 200, 505),
		3100,
	)
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
