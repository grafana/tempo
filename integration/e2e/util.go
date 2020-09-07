package e2e

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cortexproject/cortex/integration/e2e"
	"github.com/pkg/errors"
)

func getIntegrationTestHome() string {
	return os.Getenv("GOPATH") + "/src/github.com/grafana/tempo/integration/e2e"
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
	content, err := ioutil.ReadFile(filepath.Join(getIntegrationTestHome(), src))
	if err != nil {
		return errors.Wrapf(err, "unable to read local file %s", src)
	}

	return writeFileToSharedDir(s, dst, content)
}

