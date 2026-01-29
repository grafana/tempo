package jsonnet

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/grafana/tanka/pkg/jsonnet/jpath"
)

var (
	importersCache    = make(map[string][]string)
	jsonnetFilesCache = make(map[string]map[string]*cachedJsonnetFile)
	symlinkCache      = make(map[string]string)
)

type cachedJsonnetFile struct {
	Base       string
	Imports    []string
	Content    string
	IsMainFile bool
}

// FindImporterForFiles finds the entrypoints (main.jsonnet files) that import the given files.
// It looks through imports transitively, so if a file is imported through a chain, it will still be reported.
// If the given file is a main.jsonnet file, it will be returned as well.
func FindImporterForFiles(root string, files []string) ([]string, error) {
	var err error
	root, err = filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	importers := map[string]struct{}{}

	// Handle files prefixed with `deleted:`. They need to be made absolute and we shouldn't try to find symlinks for them
	var filesToCheck, existingFiles []string
	for _, file := range files {
		if strings.HasPrefix(file, "deleted:") {
			deletedFile := strings.TrimPrefix(file, "deleted:")
			// Try with both the absolute path and the path relative to the root
			if !filepath.IsAbs(deletedFile) {
				absFilePath, err := filepath.Abs(deletedFile)
				if err != nil {
					return nil, err
				}
				filesToCheck = append(filesToCheck, absFilePath)
				filesToCheck = append(filesToCheck, filepath.Clean(filepath.Join(root, deletedFile)))
			}
			continue
		}

		existingFiles = append(existingFiles, file)
	}

	if existingFiles, err = expandSymlinksInFiles(root, existingFiles); err != nil {
		return nil, err
	}
	filesToCheck = append(filesToCheck, existingFiles...)

	// Loop through all given files and add their importers to the list
	for _, file := range filesToCheck {
		if filepath.Base(file) == jpath.DefaultEntrypoint {
			importers[file] = struct{}{}
		}

		newImporters, err := findImporters(root, file, map[string]struct{}{})
		if err != nil {
			return nil, err
		}
		for _, importer := range newImporters {
			importer, err = evalSymlinks(importer)
			if err != nil {
				return nil, err
			}
			importers[importer] = struct{}{}
		}
	}

	return mapToArray(importers), nil
}

// CountImporters lists all the files in the given directory and for each file counts the number of environments that import it.
func CountImporters(root string, dir string, recursive bool, filenameRegexStr string) (string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolving root: %w", err)
	}

	if filenameRegexStr == "" {
		filenameRegexStr = "^.*\\.(jsonnet|libsonnet)$"
	}
	filenameRegexp, err := regexp.Compile(filenameRegexStr)
	if err != nil {
		return "", fmt.Errorf("compiling filename regex: %w", err)
	}
	var files []string
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && !recursive && path != dir {
			return filepath.SkipDir
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		if !filenameRegexp.MatchString(path) {
			return nil
		}

		if info.Name() == jpath.DefaultEntrypoint {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		files = append(files, path)

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walking directory: %w", err)
	}

	importers := map[string]int{}
	for _, file := range files {
		importersList, err := FindImporterForFiles(root, []string{file})
		if err != nil {
			return "", fmt.Errorf("resolving imports: %w", err)
		}
		importers[file] = len(importersList)
	}

	// Print sorted by count
	type importer struct {
		File  string `json:"file"`
		Count int    `json:"count"`
	}
	var importersList []importer
	for file, count := range importers {
		importersList = append(importersList, importer{File: file, Count: count})
	}
	sort.Slice(importersList, func(i, j int) bool {
		if importersList[i].Count == importersList[j].Count {
			return importersList[i].File < importersList[j].File
		}
		return importersList[i].Count > importersList[j].Count
	})

	var sb strings.Builder
	for _, importer := range importersList {
		sb.WriteString(fmt.Sprintf("%s: %d\n", importer.File, importer.Count))
	}

	return sb.String(), nil
}

// expandSymlinksInFiles takes an array of files and adds to it:
// - all symlinks that point to the files
// - all files that are pointed to by the symlinks
func expandSymlinksInFiles(root string, files []string) ([]string, error) {
	filesMap := map[string]struct{}{}

	for _, file := range files {
		file, err := filepath.Abs(file)
		if err != nil {
			return nil, err
		}
		filesMap[file] = struct{}{}

		symlink, err := evalSymlinks(file)
		if err != nil {
			return nil, err
		}
		if symlink != file {
			filesMap[symlink] = struct{}{}
		}

		symlinks, err := findSymlinks(root, file)
		if err != nil {
			return nil, err
		}
		for _, symlink := range symlinks {
			filesMap[symlink] = struct{}{}
		}
	}

	return mapToArray(filesMap), nil
}

// evalSymlinks returns the path after following all symlinks.
// It caches the results to avoid unnecessary work.
func evalSymlinks(path string) (string, error) {
	var err error
	eval, ok := symlinkCache[path]
	if !ok {
		eval, err = filepath.EvalSymlinks(path)
		if err != nil {
			return "", err
		}
		symlinkCache[path] = eval
	}
	return eval, nil
}

// findSymlinks finds all symlinks that point to the given file.
// It's restricted to the given root directory.
// It's used in the case where a user wants to find which entrypoints import a given file.
// In that case, we also want to find the entrypoints that import a symlink to the file.
func findSymlinks(root, file string) ([]string, error) {
	var symlinks []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Mode()&os.ModeSymlink == os.ModeSymlink {
			eval, err := evalSymlinks(path)
			if err != nil {
				return err
			}
			if strings.Contains(file, eval) {
				symlinks = append(symlinks, strings.Replace(file, eval, path, 1))
			}
		}

		return nil
	})

	return symlinks, err
}

func findImporters(root string, searchForFile string, chain map[string]struct{}) ([]string, error) {
	// If we've already looked through this file in the current execution, don't do it again and return an empty list to end the recursion
	// Jsonnet supports cyclic imports (as long as the _attributes_ being used are not cyclic)
	if _, ok := chain[searchForFile]; ok {
		return nil, nil
	}
	chain[searchForFile] = struct{}{}

	// If we've already computed the importers for a file, return the cached result
	key := root + ":" + searchForFile
	if importers, ok := importersCache[key]; ok {
		return importers, nil
	}

	jsonnetFiles, err := createJsonnetFileCache(root)
	if err != nil {
		return nil, err
	}

	var importers []string
	var intermediateImporters []string

	// If the file is not a vendored or a lib file, we assume:
	// - it is used in a Tanka environment
	// - it will not be imported by any lib or vendor files
	// - the environment base (closest main file in parent dirs) will be considered an importer
	// - if no base is found, all main files in child dirs will be considered importers
	rootVendor := filepath.Join(root, "vendor")
	rootLib := filepath.Join(root, "lib")
	isFileLibOrVendored := func(file string) bool {
		return strings.HasPrefix(file, rootVendor) || strings.HasPrefix(file, rootLib)
	}
	searchedFileIsLibOrVendored := isFileLibOrVendored(searchForFile)
	if !searchedFileIsLibOrVendored {
		searchedDir := filepath.Dir(searchForFile)
		if entrypoint := findEntrypoint(searchedDir); entrypoint != "" {
			// Found the main file for the searched file, add it as an importer
			importers = append(importers, entrypoint)
		} else if _, err := os.Stat(searchedDir); err == nil {
			// No main file found, add all main files in child dirs as importers
			files, err := FindFiles(searchedDir, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to find files in %s: %w", searchedDir, err)
			}
			for _, file := range files {
				if filepath.Base(file) == jpath.DefaultEntrypoint {
					importers = append(importers, file)
				}
			}
		}
	}

	for jsonnetFilePath, jsonnetFileContent := range jsonnetFiles {
		if len(jsonnetFileContent.Imports) == 0 {
			continue
		}

		if !searchedFileIsLibOrVendored && isFileLibOrVendored(jsonnetFilePath) {
			// Skip the file if it's a vendored or lib file and the searched file is an environment file
			// Libs and vendored files cannot import environment files
			continue
		}

		isImporter := false
		// For all imports in all jsonnet files, check if they import the file we're looking for
		for _, importPath := range jsonnetFileContent.Imports {
			// If the filename is not the same as the file we are looking for, skip it
			if filepath.Base(importPath) != filepath.Base(searchForFile) {
				continue
			}

			// Remove any `./` or `../` that can be removed just by looking at the given path
			// ex: `./foo/bar.jsonnet` -> `foo/bar.jsonnet` or `/foo/../bar.jsonnet` -> `/bar.jsonnet`
			importPath = filepath.Clean(importPath)

			// Match on relative imports with ..
			// Jsonnet also matches relative imports that are one level deeper than they should be
			// Example: Given two envs (env1 and env2), the two following imports in `env1/main.jsonnet will work`: `../env2/main.jsonnet` and `../../env2/main.jsonnet`
			// This can lead to false positives, but ruling them out would require much more complex logic
			if strings.HasPrefix(importPath, "..") {
				shallowImport := filepath.Clean(filepath.Join(filepath.Dir(jsonnetFilePath), strings.Replace(importPath, "../", "", 1)))
				importPath = filepath.Clean(filepath.Join(filepath.Dir(jsonnetFilePath), importPath))

				isImporter = pathMatches(searchForFile, importPath) || pathMatches(searchForFile, shallowImport)
			}

			// Match on imports to lib/ or vendor/
			if !isImporter {
				isImporter = pathMatches(searchForFile, filepath.Join(root, "vendor", importPath)) || pathMatches(searchForFile, filepath.Join(root, "lib", importPath))
			}

			// Match on imports to the base dir where the file is located (e.g. in the env dir)
			if !isImporter {
				if jsonnetFileContent.Base == "" {
					base, err := jpath.FindBase(jsonnetFilePath, root)
					if err != nil {
						return nil, err
					}
					jsonnetFileContent.Base = base
				}
				isImporter = strings.HasPrefix(searchForFile, jsonnetFileContent.Base) && strings.HasSuffix(searchForFile, importPath)
			}

			// If the file we're looking in imports one of the files we're looking for, add it to the list
			// It can either be an importer that we want to return (from a main file) or an intermediate importer
			if isImporter {
				if jsonnetFileContent.IsMainFile {
					importers = append(importers, jsonnetFilePath)
				}
				intermediateImporters = append(intermediateImporters, jsonnetFilePath)
				break
			}
		}
	}

	// Process intermediate importers recursively
	// This will go on until we hit a main file, which will be returned
	if len(intermediateImporters) > 0 {
		for _, intermediateImporter := range intermediateImporters {
			newImporters, err := findImporters(root, intermediateImporter, chain)
			if err != nil {
				return nil, err
			}
			importers = append(importers, newImporters...)
		}
	}

	// If we've found a vendored file, check that it's not overridden by a vendored file in the environment root
	// In that case, we only want to keep the environment vendored file
	var filteredImporters []string
	if strings.HasPrefix(searchForFile, rootVendor) {
		for _, importer := range importers {
			relativePath, err := filepath.Rel(rootVendor, searchForFile)
			if err != nil {
				return nil, err
			}
			vendoredFileInEnvironment := filepath.Join(filepath.Dir(importer), "vendor", relativePath)
			if _, ok := jsonnetFilesCache[root][vendoredFileInEnvironment]; !ok {
				filteredImporters = append(filteredImporters, importer)
			}
		}
	} else {
		filteredImporters = importers
	}

	importersCache[key] = filteredImporters
	return filteredImporters, nil
}

func createJsonnetFileCache(root string) (map[string]*cachedJsonnetFile, error) {
	if val, ok := jsonnetFilesCache[root]; ok {
		return val, nil
	}
	jsonnetFilesCache[root] = make(map[string]*cachedJsonnetFile)

	files, err := FindFiles(root, nil)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		matches := importsRegexp.FindAllStringSubmatch(string(content), -1)

		cachedObj := &cachedJsonnetFile{
			Content:    string(content),
			IsMainFile: strings.HasSuffix(file, jpath.DefaultEntrypoint),
		}
		for _, match := range matches {
			cachedObj.Imports = append(cachedObj.Imports, match[2])
		}
		jsonnetFilesCache[root][file] = cachedObj
	}

	return jsonnetFilesCache[root], nil
}

// findEntrypoint finds the nearest main.jsonnet file in the given file's directory or parent directories
func findEntrypoint(searchedDir string) string {
	for {
		if _, err := os.Stat(searchedDir); err == nil {
			break
		}
		searchedDir = filepath.Dir(searchedDir)
	}
	searchedFileEntrypoint, err := jpath.Entrypoint(searchedDir)
	if err != nil {
		return ""
	}
	return searchedFileEntrypoint
}

func pathMatches(path1, path2 string) bool {
	if path1 == path2 {
		return true
	}

	var err error

	evalPath1, err := evalSymlinks(path1)
	if err != nil {
		return false
	}

	evalPath2, err := evalSymlinks(path2)
	if err != nil {
		return false
	}

	return evalPath1 == evalPath2
}

func mapToArray(m map[string]struct{}) []string {
	var arr []string
	for k := range m {
		arr = append(arr, k)
	}
	sort.Strings(arr)
	return arr
}
