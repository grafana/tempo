package impi

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/kisielk/gotool"
)

// Impi is a single instance that can perform verification on a path
type Impi struct {
	numWorkers      int
	resultChan      chan interface{}
	filePathsChan   chan string
	stopChan        chan bool
	verifyOptions   *VerifyOptions
	SkipPathRegexes []*regexp.Regexp
}

// ImportGroupVerificationScheme specifies what to check when inspecting import groups
type ImportGroupVerificationScheme int

const (

	// ImportGroupVerificationSchemeSingle allows for a single, sorted group
	ImportGroupVerificationSchemeSingle = ImportGroupVerificationScheme(iota)

	// ImportGroupVerificationSchemeStdNonStd allows for up to two groups in the following order:
	// - standard imports
	// - non-standard imports
	ImportGroupVerificationSchemeStdNonStd

	// ImportGroupVerificationSchemeStdLocalThirdParty allows for up to three groups in the following order:
	// - standard imports
	// - local imports (where local prefix is specified in verification options)
	// - non-standard imports
	ImportGroupVerificationSchemeStdLocalThirdParty

	// ImportGroupVerificationSchemeStdThirdPartyLocal allows for up to three groups in the following order:
	// - standard imports
	// - non-standard imports
	// - local imports (where local prefix is specified in verification options)
	ImportGroupVerificationSchemeStdThirdPartyLocal
)

// VerifyOptions specifies how to perform verification
type VerifyOptions struct {
	SkipTests       bool
	Scheme          ImportGroupVerificationScheme
	LocalPrefix     string
	SkipPaths       []string
	IgnoreGenerated bool
}

// VerificationError holds an error and a file path on which the error occurred
type VerificationError struct {
	error
	FilePath string
}

// ErrorReporter receives error reports as they are detected by the workers
type ErrorReporter interface {
	Report(VerificationError)
}

// NewImpi creates a new impi instance
func NewImpi(numWorkers int) (*Impi, error) {
	newImpi := &Impi{
		numWorkers:    numWorkers,
		resultChan:    make(chan interface{}, 1024),
		filePathsChan: make(chan string),
		stopChan:      make(chan bool),
	}

	return newImpi, nil
}

// Verify will iterate over the path and start verifying import correctness within
// all .go files in the path. Path follows go tool semantics (e.g. ./...)
func (i *Impi) Verify(rootPath string, verifyOptions *VerifyOptions, errorReporter ErrorReporter) error {

	// save stuff for current session
	i.verifyOptions = verifyOptions

	// compile skip regex
	for _, skipPath := range verifyOptions.SkipPaths {
		skipPathRegex, err := regexp.Compile(skipPath)
		if err != nil {
			return err
		}

		i.SkipPathRegexes = append(i.SkipPathRegexes, skipPathRegex)
	}

	// spin up the workers do handle all the data in the channel. workers will die
	if err := i.createWorkers(i.numWorkers); err != nil {
		return err
	}

	// populate paths channel from path. paths channel will contain .go source file paths
	if err := i.populatePathsChan(rootPath); err != nil {
		return err
	}

	// wait for worker completion. if an error was reported, return error
	if numErrors := i.waitWorkerCompletion(errorReporter); numErrors != 0 {
		return fmt.Errorf("Found %d errors", numErrors)
	}

	return nil
}

func (i *Impi) populatePathsChan(rootPath string) error {
	// TODO: this should be done in parallel

	// get all the packages in the root path, following go 1.9 semantics
	packagePaths := gotool.ImportPaths([]string{rootPath})

	if len(packagePaths) == 0 {
		return fmt.Errorf("Could not find packages in %s", packagePaths)
	}

	// iterate over these paths:
	// - for files, just shove to paths
	// - for dirs, find all go sources
	for _, packagePath := range packagePaths {
		if isDir(packagePath) {

			// iterate over files in directory
			fileInfos, err := ioutil.ReadDir(packagePath)
			if err != nil {
				return err
			}

			for _, fileInfo := range fileInfos {
				if fileInfo.IsDir() {
					continue
				}

				i.addFilePathToFilePathsChan(path.Join(packagePath, fileInfo.Name()))
			}

		} else {

			// shove path to channel if passes filter
			i.addFilePathToFilePathsChan(packagePath)
		}
	}

	// close the channel to signify we won't add any more data
	close(i.filePathsChan)

	return nil
}

func (i *Impi) waitWorkerCompletion(errorReporter ErrorReporter) int {
	numWorkersComplete := 0
	numErrorsReported := 0

	for result := range i.resultChan {
		switch typedResult := result.(type) {
		case VerificationError:
			errorReporter.Report(typedResult)
			numErrorsReported++
		case bool:
			numWorkersComplete++
		}

		// if we're done, break the loop
		if numWorkersComplete == i.numWorkers {
			break
		}
	}

	return numErrorsReported
}

func (i *Impi) createWorkers(numWorkers int) error {
	for workerIndex := 0; workerIndex < numWorkers; workerIndex++ {
		go i.verifyPathsFromChan()
	}

	return nil
}

func (i *Impi) verifyPathsFromChan() error {

	// create a verifier with which we'll verify modules
	verifier, err := newVerifier()
	if err != nil {
		return err
	}

	// while we're not done
	for filePath := range i.filePathsChan {

		// open the file
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}

		// verify the path and report an error if one is found
		if err = verifier.verify(file, i.verifyOptions); err != nil {
			verificationError := VerificationError{
				error:    err,
				FilePath: filePath,
			}

			// write to results channel
			i.resultChan <- verificationError
		}
	}

	// a boolean in the result chan signifies that we're done
	i.resultChan <- true

	return nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}

func (i *Impi) addFilePathToFilePathsChan(filePath string) {

	// skip non-go files
	if !strings.HasSuffix(filePath, ".go") {
		return
	}

	// skip tests if not desired
	if strings.HasSuffix(filePath, "_test.go") && i.verifyOptions.SkipTests {
		return
	}

	// cmd/impi/main.go should check the patters
	for _, skipPathRegex := range i.SkipPathRegexes {
		if skipPathRegex.Match([]byte(filePath)) {
			return
		}
	}

	// write to paths chan
	i.filePathsChan <- filePath
}
