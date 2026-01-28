package jsonnet

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/gobwas/glob"
	"github.com/google/go-jsonnet/linter"
	"github.com/grafana/tanka/pkg/jsonnet/implementations/goimpl"
	"github.com/grafana/tanka/pkg/jsonnet/jpath"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// LintOpts modifies the behaviour of Lint
type LintOpts struct {
	// Excludes are a list of globs to exclude files while searching for Jsonnet
	// files
	Excludes []glob.Glob

	// Parallelism determines the number of workers that will process files
	Parallelism int

	Out io.Writer
}

// Lint takes a list of files and directories, processes them and prints
// out to stderr if there are linting warnings
func Lint(fds []string, opts *LintOpts) error {
	if opts.Parallelism <= 0 {
		return errors.New("parallelism must be greater than 0")
	}

	if opts.Out == nil {
		opts.Out = os.Stdout
	}

	var paths []string
	for _, f := range fds {
		fs, err := FindFiles(f, opts.Excludes)
		if err != nil {
			return errors.Wrap(err, "finding Jsonnet files")
		}
		paths = append(paths, fs...)
	}

	type result struct {
		success bool
		output  string
	}
	fileCh := make(chan string, len(paths))
	resultCh := make(chan result, len(paths))
	lintWorker := func(fileCh <-chan string, resultCh chan result) {
		for file := range fileCh {
			buf, success := lintWithRecover(file)
			resultCh <- result{success: success, output: buf.String()}
		}
	}

	for i := 0; i < opts.Parallelism; i++ {
		go lintWorker(fileCh, resultCh)
	}

	for _, file := range paths {
		fileCh <- file
	}
	close(fileCh)

	lintingFailed := false
	for i := 0; i < len(paths); i++ {
		result := <-resultCh
		lintingFailed = lintingFailed || !result.success
		if result.output != "" {
			fmt.Fprint(opts.Out, result.output)
		}
	}

	if lintingFailed {
		return errors.New("Linting has failed for at least one file")
	}
	return nil
}

func lintWithRecover(file string) (buf bytes.Buffer, success bool) {
	file, err := filepath.Abs(file)
	if err != nil {
		fmt.Fprintf(&buf, "got an error getting the absolute path for %s: %v\n\n", file, err)
		return
	}

	log.Debug().Str("file", file).Msg("linting file")
	startTime := time.Now()
	defer func() {
		if err := recover(); err != nil {
			fmt.Fprintf(&buf, "caught a panic while linting %s: %v\n\n", file, err)
		}
		log.Debug().Str("file", file).Dur("duration_ms", time.Since(startTime)).Msg("linted file")
	}()

	content, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(&buf, "got an error reading file %s: %v\n\n", file, err)
		return
	}

	jpaths, _, _, err := jpath.Resolve(file, true)
	if err != nil {
		fmt.Fprintf(&buf, "got an error getting jpath for %s: %v\n\n", file, err)
		return
	}
	vm := goimpl.MakeRawVM(jpaths, nil, nil, 0)

	failed := linter.LintSnippet(vm, &buf, []linter.Snippet{{FileName: file, Code: string(content)}})
	return buf, !failed
}
