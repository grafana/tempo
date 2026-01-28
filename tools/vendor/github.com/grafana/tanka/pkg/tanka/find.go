package tanka

import (
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"github.com/grafana/tanka/pkg/jsonnet"
	"github.com/grafana/tanka/pkg/jsonnet/jpath"
	"github.com/grafana/tanka/pkg/spec/v1alpha1"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/labels"
)

// FindOpts are optional arguments for FindEnvs
type FindOpts struct {
	JsonnetOpts
	JsonnetImplementation string
	Selector              labels.Selector
	Parallelism           int
}

// FindEnvs returns metadata of all environments recursively found in 'path'.
// Each directory is tested and included if it is a valid environment, either
// static or inline. If a directory is a valid environment, its subdirectories
// are not checked.
func FindEnvs(path string, opts FindOpts) ([]*v1alpha1.Environment, error) {
	return findEnvsFromPaths([]string{path}, opts)
}

// FindEnvsFromPaths does the same as FindEnvs but takes a list of paths instead
func FindEnvsFromPaths(paths []string, opts FindOpts) ([]*v1alpha1.Environment, error) {
	return findEnvsFromPaths(paths, opts)
}

func findEnvsFromPaths(paths []string, opts FindOpts) ([]*v1alpha1.Environment, error) {
	if opts.Parallelism <= 0 {
		opts.Parallelism = runtime.NumCPU()
	}

	log.Debug().Int("parallelism", opts.Parallelism).Int("paths", len(paths)).Msg("Finding Tanka environments")
	startTime := time.Now()

	jsonnetFiles, err := findJsonnetFilesFromPaths(paths, opts)
	if err != nil {
		return nil, fmt.Errorf("finding jsonnet files: %w", err)
	}

	findJsonnetFilesEndTime := time.Now()

	envs, err := findEnvsFromJsonnetFiles(jsonnetFiles, opts)
	if err != nil {
		return nil, fmt.Errorf("finding environments: %w", err)
	}

	findEnvsEndTime := time.Now()

	log.Debug().
		Int("environments", len(envs)).
		Dur("ms_to_find_jsonnet_files", findJsonnetFilesEndTime.Sub(startTime)).
		Dur("ms_to_find_environments", findEnvsEndTime.Sub(findJsonnetFilesEndTime)).
		Msg("Found Tanka environments")

	return envs, nil
}

// find all jsonnet files within given paths
func findJsonnetFilesFromPaths(paths []string, opts FindOpts) ([]string, error) {
	type findJsonnetFilesOut struct {
		jsonnetFiles []string
		err          error
	}

	pathChan := make(chan string, len(paths))
	findJsonnetFilesChan := make(chan findJsonnetFilesOut)
	for i := 0; i < opts.Parallelism; i++ {
		go func() {
			for path := range pathChan {
				jsonnetFiles, err := jsonnet.FindFiles(path, nil)
				var mainFiles []string
				for _, file := range jsonnetFiles {
					if filepath.Base(file) == jpath.DefaultEntrypoint {
						mainFiles = append(mainFiles, file)
					}
				}
				findJsonnetFilesChan <- findJsonnetFilesOut{jsonnetFiles: mainFiles, err: err}
			}
		}()
	}

	// push paths to channel
	for _, path := range paths {
		pathChan <- path
	}
	close(pathChan)

	// collect jsonnet files
	var jsonnetFiles []string
	var errs []error
	for range paths {
		res := <-findJsonnetFilesChan
		if res.err != nil {
			errs = append(errs, res.err)
			continue
		}
		jsonnetFiles = append(jsonnetFiles, res.jsonnetFiles...)
	}
	close(findJsonnetFilesChan)

	if len(errs) != 0 {
		return jsonnetFiles, ErrParallel{errors: errs}
	}

	return jsonnetFiles, nil
}

// find all environments within jsonnet files
func findEnvsFromJsonnetFiles(jsonnetFiles []string, opts FindOpts) ([]*v1alpha1.Environment, error) {
	type findEnvsOut struct {
		envs []*v1alpha1.Environment
		err  error
	}

	jsonnetFilesChan := make(chan string, len(jsonnetFiles))
	findEnvsChan := make(chan findEnvsOut)

	for i := 0; i < opts.Parallelism; i++ {
		go func() {
			// We need to create a copy of the opts for each goroutine because
			// the content may be modified down the line:
			jsonnetOpts := opts.JsonnetOpts.Clone()

			for jsonnetFile := range jsonnetFilesChan {
				// try if this has envs
				list, err := List(jsonnetFile, Opts{JsonnetOpts: jsonnetOpts, JsonnetImplementation: opts.JsonnetImplementation})
				if err != nil &&
					// expected when looking for environments
					!errors.As(err, &jpath.ErrorNoBase{}) &&
					!errors.As(err, &jpath.ErrorFileNotFound{}) {
					findEnvsChan <- findEnvsOut{err: fmt.Errorf("%s:\n %w", jsonnetFile, err)}
					continue
				}
				filtered := []*v1alpha1.Environment{}
				// optionally filter
				if opts.Selector != nil && !opts.Selector.Empty() {
					for _, e := range list {
						if !opts.Selector.Matches(e.Metadata) {
							continue
						}
						filtered = append(filtered, e)
					}
				} else {
					filtered = append(filtered, list...)
				}
				findEnvsChan <- findEnvsOut{envs: filtered, err: nil}
			}
		}()
	}

	// push jsonnet files to channel
	for _, jsonnetFile := range jsonnetFiles {
		jsonnetFilesChan <- jsonnetFile
	}
	close(jsonnetFilesChan)

	// collect environments
	var envs []*v1alpha1.Environment
	var errs []error
	for i := 0; i < len(jsonnetFiles); i++ {
		res := <-findEnvsChan
		if res.err != nil {
			errs = append(errs, res.err)
			continue
		}
		envs = append(envs, res.envs...)
	}
	close(findEnvsChan)

	if len(errs) != 0 {
		return envs, ErrParallel{errors: errs}
	}

	return envs, nil
}
