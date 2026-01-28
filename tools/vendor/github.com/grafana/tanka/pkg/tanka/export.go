package tanka

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/grafana/tanka/pkg/kubernetes/manifest"
	"github.com/grafana/tanka/pkg/spec/v1alpha1"
)

// BelRune is a string of the Ascii character BEL which made computers ring in ancient times
// We use it as "magic" char for the subfolder creation as it is a non printable character and thereby will never be
// in a valid filepath by accident. Only when we include it.
const BelRune = string(rune(7))

// When exporting manifests to files, it becomes increasingly hard to map manifests back to its environment, this file
// can be used to map the files back to their environment. This is aimed to be used by CI/CD but can also be used for
// debugging purposes.
const manifestFile = "manifest.json"

type ExportMergeStrategy string

const (
	ExportMergeStrategyNone          ExportMergeStrategy = ""
	ExportMergeStrategyFailConflicts ExportMergeStrategy = "fail-on-conflicts"
	ExportMergeStrategyReplaceEnvs   ExportMergeStrategy = "replace-envs"
)

// ExportEnvOpts specify options on how to export environments
type ExportEnvOpts struct {
	// formatting the filename based on the exported Kubernetes manifest
	Format string
	// extension of the filename
	Extension string
	// optional: options to parse Jsonnet
	Opts Opts
	// optional: filter environments based on labels
	Selector labels.Selector
	// optional: number of environments to process in parallel
	Parallelism int

	// What to do when exporting to an existing directory
	// - none: fail when directory is not empty
	// - fail-on-conflicts: fail when an exported file already exists
	// - replace-envs: delete files previously exported by the targeted envs and re-export them
	MergeStrategy ExportMergeStrategy
	// Environments (main.jsonnet files) that have been deleted since the last export.
	// This is used when using a merge strategy to delete the files of these deleted environments.
	MergeDeletedEnvs []string
}

func ExportEnvironments(envs []*v1alpha1.Environment, to string, opts *ExportEnvOpts) error {
	// Keep track of which file maps to which environment
	fileToEnv := map[string]string{}

	// dir must be empty
	empty, err := dirEmpty(to)
	if err != nil {
		return fmt.Errorf("checking target dir: %s", err)
	}
	if !empty && opts.MergeStrategy == ExportMergeStrategyNone {
		return fmt.Errorf("output dir `%s` not empty. Pass a different --merge-strategy to ignore this", to)
	}

	// delete files previously exported by the targeted envs.
	if opts.MergeStrategy == ExportMergeStrategyReplaceEnvs {
		if err := deletePreviouslyExportedManifestsFromTankaEnvs(to, envs); err != nil {
			return fmt.Errorf("deleting previously exported manifests: %w", err)
		}
	}

	// delete files that were exported by environments that have been deleted since the last export.
	if err := deletePreviouslyExportedManifests(to, opts.MergeDeletedEnvs); err != nil {
		return fmt.Errorf("deleting previously exported manifests from deleted environments: %w", err)
	}

	// get all environments for paths
	loadedEnvs, err := parallelLoadEnvironments(envs, parallelOpts{
		Opts:        opts.Opts,
		Selector:    opts.Selector,
		Parallelism: opts.Parallelism,
	})
	if err != nil {
		return err
	}

	for _, env := range loadedEnvs {
		// get the manifests
		loaded, err := LoadManifests(env, opts.Opts.Filters)
		if err != nil {
			return err
		}

		env := loaded.Env
		res := loaded.Resources

		// create raw manifest version of env for templating
		env.Data = nil
		raw, err := json.Marshal(env)
		if err != nil {
			return err
		}
		var menv manifest.Manifest
		if err := json.Unmarshal(raw, &menv); err != nil {
			return err
		}

		// create template
		manifestTemplate, err := createTemplate(opts.Format, menv)
		if err != nil {
			return fmt.Errorf("parsing format: %s", err)
		}

		// write each to a file
		for _, m := range res {
			// apply template
			name, err := applyTemplate(manifestTemplate, m)
			if err != nil {
				return fmt.Errorf("executing name template: %w", err)
			}

			// Create all subfolders in path
			relpath := name + "." + opts.Extension
			path := filepath.Join(to, relpath)

			fileToEnv[relpath] = env.Metadata.Namespace

			// Abort if already exists
			if exists, err := fileExists(path); err != nil {
				return err
			} else if exists {
				return fmt.Errorf("file '%s' already exists. Aborting", path)
			}

			// Write manifest
			data := m.String()
			if err := writeExportFile(path, []byte(data)); err != nil {
				return err
			}
		}
	}

	return exportManifestFile(to, fileToEnv, nil)
}

func fileExists(name string) (bool, error) {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func dirEmpty(dir string) (bool, error) {
	f, err := os.Open(dir)
	if os.IsNotExist(err) {
		return true, os.MkdirAll(dir, os.ModePerm)
	} else if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

func deletePreviouslyExportedManifestsFromTankaEnvs(path string, envs []*v1alpha1.Environment) error {
	envNames := []string{}
	for _, env := range envs {
		envNames = append(envNames, env.Metadata.Namespace)
	}
	return deletePreviouslyExportedManifests(path, envNames)
}

func deletePreviouslyExportedManifests(path string, tankaEnvNames []string) error {
	if len(tankaEnvNames) == 0 {
		return nil
	}

	envNamesMap := make(map[string]struct{})
	for _, envName := range tankaEnvNames {
		envNamesMap[envName] = struct{}{}
	}

	fileToEnvMap := make(map[string]string)

	manifestFilePath := filepath.Join(path, manifestFile)
	manifestContent, err := os.ReadFile(manifestFilePath)
	if errors.Is(err, fs.ErrNotExist) {
		log.Warn().Msgf("No manifest file found at %s, skipping deletion of previously exported manifests\n", manifestFilePath)
		return nil
	} else if err != nil {
		return err
	}

	if err := json.Unmarshal(manifestContent, &fileToEnvMap); err != nil {
		return err
	}

	var deletedManifestKeys []string
	for exportedManifest, manifestEnv := range fileToEnvMap {
		if _, ok := envNamesMap[manifestEnv]; ok {
			deletedManifestKeys = append(deletedManifestKeys, exportedManifest)
			if err := os.Remove(filepath.Join(path, exportedManifest)); err != nil {
				return err
			}
		}
	}

	return exportManifestFile(path, nil, deletedManifestKeys)
}

// exportManifestFile writes a manifest file that maps the exported files to their environment.
// If the file already exists, the newString entries will be merged with the existing ones.
func exportManifestFile(path string, newFileToEnvMap map[string]string, deletedKeys []string) error {
	if len(newFileToEnvMap) == 0 && len(deletedKeys) == 0 {
		return nil
	}

	currentFileToEnvMap := make(map[string]string)
	manifestFilePath := filepath.Join(path, manifestFile)
	if manifestContent, err := os.ReadFile(manifestFilePath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("reading existing manifest file: %w", err)
	} else if err == nil {
		// Only read the manifest file if it exists.
		// If it doesn't exist, currentFileToEnvMap will be empty, meaning that we're starting from a newString export dir.
		if err := json.Unmarshal(manifestContent, &currentFileToEnvMap); err != nil {
			return fmt.Errorf("unmarshalling existing manifest file: %w", err)
		}
	}

	for k, v := range newFileToEnvMap {
		currentFileToEnvMap[k] = v
	}
	for _, k := range deletedKeys {
		delete(currentFileToEnvMap, k)
	}

	// Write manifest file
	data, err := json.MarshalIndent(currentFileToEnvMap, "", "    ")
	if err != nil {
		return fmt.Errorf("marshalling manifest file: %w", err)
	}

	return writeExportFile(manifestFilePath, data)
}

func writeExportFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating filepath '%s': %s", filepath.Dir(path), err)
	}

	return os.WriteFile(path, data, 0644)
}

func createTemplate(format string, env manifest.Manifest) (*template.Template, error) {
	// Replace all os.path separators in string with BelRune for creating subfolders
	replaceFormat := replaceTmplText(format, string(os.PathSeparator), BelRune)

	envMap := template.FuncMap{"env": func() manifest.Manifest { return env }}

	template, err := template.New("").
		Funcs(sprig.TxtFuncMap()). // register Masterminds/sprig
		Funcs(envMap).             // register environment mapping
		Parse(replaceFormat)       // parse template
	if err != nil {
		return nil, err
	}
	return template, nil
}

func replaceTmplText(s, oldString, newString string) string {
	parts := []string{}
	l := strings.Index(s, "{{")
	r := strings.Index(s, "}}") + 2

	for l != -1 && l < r {
		// replace only in text between template action blocks
		text := strings.ReplaceAll(s[:l], oldString, newString)
		action := s[l:r]
		parts = append(parts, text, action)
		s = s[r:]
		l = strings.Index(s, "{{")
		r = strings.Index(s, "}}") + 2
	}
	parts = append(parts, strings.ReplaceAll(s, oldString, newString))
	return strings.Join(parts, "")
}

func applyTemplate(template *template.Template, m manifest.Manifest) (path string, err error) {
	buf := bytes.Buffer{}
	if err := template.Execute(&buf, m); err != nil {
		return "", err
	}

	// Replace all os.path separators in string in order to not accidentally create subfolders
	path = strings.ReplaceAll(buf.String(), string(os.PathSeparator), "-")
	// Replace the BEL character inserted with a path separator again in order to create a subfolder
	path = strings.ReplaceAll(path, BelRune, string(os.PathSeparator))

	return path, nil
}
