package uaparser

import (
	"os"

	"gopkg.in/yaml.v3"
)

// NewWithOptions is deprecated.
// Deprecated: Use New and option functions instead.
func NewWithOptions(regexFile string, mode, treshold, topCnt int, useSort, debugMode bool, cacheSize int) (*Parser, error) {
	uaRegexes, err := os.ReadFile(regexFile)
	if err != nil {
		return nil, err
	}

	var def RegexDefinitions

	if err := yaml.Unmarshal(uaRegexes, &def); err != nil {
		return nil, err
	}

	return New(
		WithRegexDefinitions(def),
		WithMode(LookupMode(mode)),
		WithMatchIdxNotOk(topCnt),
		WithSort(useSort, WithMissesThreshold(uint64(treshold))),
		WithDebug(debugMode),
		WithCacheSize(cacheSize),
	)
}

// NewFromSaved is deprecated.
// Deprecated: Use New() instead.
func NewFromSaved() *Parser {
	parser, err := New()
	if err != nil {
		// if the YAML is malformed, it's a programmatic error inside what
		// we've statically-compiled in our binary. Panic!
		panic(err.Error())
	}

	return parser
}

// NewFromBytes is deprecated.
// Deprecated: Use New(WithRegexDefinitions(...)) instead
func NewFromBytes(data []byte) (*Parser, error) {
	var def RegexDefinitions

	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, err
	}

	return New(WithRegexDefinitions(def))
}
