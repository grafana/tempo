package impi

import (
	"bufio"
	"errors"
	"fmt"
	"go/parser"
	"go/scanner"
	"go/token"
	"io"
	"io/ioutil"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

// This regex is to appear in generated code.
var generatedRegex = regexp.MustCompile("// Code generated .* DO NOT EDIT\\.")

type verifier struct {
	verifyOptions *VerifyOptions
}

type importInfoGroup struct {
	importInfos []*importInfo
}

type importType int

const (
	importTypeUnknown = importType(iota)
	importTypeStd
	importTypeLocal
	importTypeThirdParty
	importTypeLocalOrThirdParty
)

var importTypeName = []string{
	"Unknown",
	"Std",
	"Local",
	"Third party",
	"Local or third party",
}

type verificationScheme interface {

	// getMaxNumGroups returns max number of groups the scheme allows
	getMaxNumGroups() int

	// getMixedGroupsAllowed returns whether a group can contain imports of different types
	getMixedGroupsAllowed() bool

	// getAllowedImportOrders returns which group orders are allowed
	getAllowedImportOrders() [][]importType
}

type importInfo struct {
	lineNum        int
	lineValue      string
	path           string
	classifiedType importType
}

func newVerifier() (*verifier, error) {
	return &verifier{}, nil
}

func (v *verifier) verify(sourceFileReader io.ReadSeeker, verifyOptions *VerifyOptions) error {
	v.verifyOptions = verifyOptions

	if verifyOptions.IgnoreGenerated {
		// The line specifying that the code was generated can be found anywhere
		// within a file. In practice, it is the first line.
		fileContents, err := ioutil.ReadAll(sourceFileReader)
		if err != nil {
			return err
		}

		if generatedRegex.Match(fileContents) {
			return nil
		}

		if _, err := sourceFileReader.Seek(0, 0); err != nil {
			return err
		}
	}

	// get lines on which imports start and end
	importLineNumbers, err := v.getImportPos(sourceFileReader)
	if err != nil {
		return err
	}

	// if there's nothing, do nothing
	if len(importLineNumbers) == 0 {
		return nil
	}

	// get import lines - the value of the source file from the first import to the last
	importInfos, err := v.readImportInfos(importLineNumbers[0],
		importLineNumbers[len(importLineNumbers)-1],
		sourceFileReader)

	if err != nil {
		return err
	}

	// group the import lines we got based on newlines separating the groups
	importInfoGroups := v.groupImportInfos(importInfos, importLineNumbers)

	// classify import info types - for each info type assign an "importType"
	v.classifyImportTypes(importInfoGroups)

	// get scheme by type
	verificationScheme, err := v.getVerificationScheme()
	if err != nil {
		return err
	}

	// verify that we don't have too many groups
	if verificationScheme.getMaxNumGroups() < len(importInfoGroups) {
		return fmt.Errorf("Expected no more than 3 groups, got %d", len(importInfoGroups))
	}

	// if the scheme disallowed mixed groups, check that there are no mixed groups
	if !verificationScheme.getMixedGroupsAllowed() {
		if err := v.verifyNonMixedGroups(importInfoGroups); err != nil {
			return err
		}

		// verify group order
		if err := v.verifyGroupOrder(importInfoGroups, verificationScheme.getAllowedImportOrders()); err != nil {
			return err
		}
	}

	// verify that all groups are sorted amongst themselves
	if err := v.verifyImportInfoGroupsOrder(importInfoGroups); err != nil {
		return err
	}

	return nil
}

func (v *verifier) groupImportInfos(importInfos []importInfo, importLineNumbers []int) []importInfoGroup {

	// initialize an import group with the first group already inserted
	importInfoGroups := []importInfoGroup{
		{},
	}

	// set current group - it'll change as new groups are found
	currentImportGroupIndex := 0

	// split the imports into groups, where groups are separated with empty lines
	for _, importInfoInstance := range importInfos {

		// if we found an empty line - open a new group
		if len(importInfoInstance.lineValue) == 0 {
			importInfoGroups = append(importInfoGroups, importInfoGroup{})
			currentImportGroupIndex++

			// skip line
			continue
		}

		// if this line doesn't hold a valid import (e.g. comment, comment block) - just ignore it. this helps
		// us use the parser outputs as the source of whether or not this is an import or a comment
		if findIntInIntSlice(importLineNumbers, importInfoInstance.lineNum) == -1 {
			continue
		}

		// add import info copy
		importInfoGroups[currentImportGroupIndex].importInfos = append(importInfoGroups[currentImportGroupIndex].importInfos, &importInfo{
			lineNum:   importInfoInstance.lineNum,
			lineValue: importInfoInstance.lineValue,
			path:      importInfoInstance.path,
		})
	}

	return v.filterImportC(importInfoGroups)
}

// filter out single `import "C"` from groups since it needs to be on it's own line
func (v *verifier) filterImportC(importInfoGroups []importInfoGroup) []importInfoGroup {
	var filteredGroups []importInfoGroup

	for _, importInfoGroup := range importInfoGroups {
		if len(importInfoGroup.importInfos) == 1 && importInfoGroup.importInfos[0].path == "C" {
			continue
		}
		filteredGroups = append(filteredGroups, importInfoGroup)
	}

	return filteredGroups
}

func (v *verifier) getImportPos(sourceFileReader io.ReadSeeker) ([]int, error) {
	sourceFileSet := token.NewFileSet()

	sourceNode, err := parser.ParseFile(sourceFileSet, "", sourceFileReader, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}

	var importLineNumbers []int

	for _, importSpec := range sourceNode.Imports {
		importLineNumbers = append(importLineNumbers, sourceFileSet.Position(importSpec.Pos()).Line)
	}

	return importLineNumbers, nil
}

func (v *verifier) readImportInfos(startLineNum int, endLineNum int, reader io.ReadSeeker) ([]importInfo, error) {
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	var importInfos []importInfo
	s := bufio.NewScanner(reader)

	for lineNum := 1; s.Scan(); lineNum++ {
		if lineNum >= startLineNum && lineNum <= endLineNum {
			lineValue := s.Bytes()
			path, err := extractImportPath(lineValue)
			if err != nil {
				return nil, err
			}

			importInfos = append(importInfos, importInfo{
				lineNum:   lineNum,
				lineValue: string(lineValue),
				path:      path,
			})
		}
	}

	return importInfos, nil
}

func extractImportPath(line []byte) (string, error) {
	s := scanner.Scanner{}
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(line))
	s.Init(file, line, nil, 0)

	var val string
	for {
		_, tok, lit := s.Scan()
		switch tok {
		case token.EOF:
			return strings.Replace(val, `"`, "", -1), nil
		case token.STRING:
			if val != "" {
				return "", fmt.Errorf("parsing failed, multiple strings on import line: %v", string(line))
			}
			val = lit
		}
	}
}

func findIntInIntSlice(slice []int, value int) int {
	for sliceValueIndex, sliceValue := range slice {
		if sliceValue == value {
			return sliceValueIndex
		}
	}

	return -1
}

func (v *verifier) verifyImportInfoGroupsOrder(importInfoGroups []importInfoGroup) error {
	var errorString string

	for importInfoGroupIndex, importInfoGroup := range importInfoGroups {
		var importPaths []string

		// create slice of strings so we can compare
		for _, importInfo := range importInfoGroup.importInfos {
			importPaths = append(importPaths, importInfo.path)
		}

		// check that group is sorted
		if !sort.StringsAreSorted(importPaths) {

			// created a sorted copy for logging
			sortedImportGroup := make([]string, len(importPaths))
			copy(sortedImportGroup, importPaths)
			sort.Sort(sort.StringSlice(sortedImportGroup))

			errorString += fmt.Sprintf("\n- Import group %d is not sorted\n-- Got:\n%s\n\n-- Expected:\n%s\n",
				importInfoGroupIndex,
				strings.Join(importPaths, "\n"),
				strings.Join(sortedImportGroup, "\n"))
		}
	}

	if len(errorString) != 0 {
		return errors.New(errorString)
	}

	return nil
}

func (v *verifier) classifyImportTypes(importInfoGroups []importInfoGroup) {
	for _, importInfoGroup := range importInfoGroups {

		// create slice of strings so we can compare
		for _, importInfo := range importInfoGroup.importInfos {

			// if the value doesn't contain dot, it's a standard import
			if !strings.Contains(importInfo.path, ".") {
				importInfo.classifiedType = importTypeStd
				continue
			}

			// if there's no prefix specified, it's either standard or local
			if len(v.verifyOptions.LocalPrefix) == 0 {
				importInfo.classifiedType = importTypeLocalOrThirdParty
				continue
			}

			if strings.HasPrefix(importInfo.path, v.verifyOptions.LocalPrefix) {
				importInfo.classifiedType = importTypeLocal
			} else {
				importInfo.classifiedType = importTypeThirdParty
			}
		}
	}
}

func (v *verifier) getVerificationScheme() (verificationScheme, error) {
	switch v.verifyOptions.Scheme {
	case ImportGroupVerificationSchemeStdLocalThirdParty:
		return newStdLocalThirdPartyScheme(), nil
	case ImportGroupVerificationSchemeStdThirdPartyLocal:
		return newStdThirdPartyLocalScheme(), nil
	default:
		return nil, errors.New("Unsupported verification scheme")
	}
}

func (v *verifier) verifyNonMixedGroups(importInfoGroups []importInfoGroup) error {
	for importInfoGroupIndex, importInfoGroup := range importInfoGroups {
		importGroupImportType := importInfoGroup.importInfos[0].classifiedType

		for _, importInfo := range importInfoGroup.importInfos {
			if importInfo.classifiedType != importGroupImportType {
				return fmt.Errorf("Imports of different types are not allowed in the same group (%d): %s != %s",
					importInfoGroupIndex,
					importInfoGroup.importInfos[0].lineValue,
					importInfo.lineValue)
			}
		}
	}

	return nil
}

func (v *verifier) verifyGroupOrder(importInfoGroups []importInfoGroup, allowedImportOrders [][]importType) error {
	var existingImportOrder []importType

	// use the first import type as indicative of the following. TODO: to support ImportGroupVerificationSchemeStdNonStd
	// this will need to do a full pass
	for _, importInfoGroup := range importInfoGroups {
		existingImportOrder = append(existingImportOrder, importInfoGroup.importInfos[0].classifiedType)
	}

	for _, allowedImportOrder := range allowedImportOrders {
		if reflect.DeepEqual(allowedImportOrder, existingImportOrder) {
			return nil
		}
	}

	// convert to string for a clearer error
	existingImportOrderString := []string{}
	for _, importType := range existingImportOrder {
		existingImportOrderString = append(existingImportOrderString, importTypeName[importType])
	}

	return fmt.Errorf("Import groups are not in the proper order: %q", existingImportOrderString)
}
