package uaparser

import (
	"fmt"
	"regexp"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/yaml.v3"
)

var defaultRegexesDefinitions = sync.OnceValue(func() RegexDefinitions {
	var def RegexDefinitions
	if err := yaml.Unmarshal(DefinitionYaml, &def); err != nil {
		panic(fmt.Errorf("error parsing regexes definitions: %w", err))
	}

	return def
})

type RegexDefinitions struct {
	UA     []*uaParser     `yaml:"user_agent_parsers"`
	OS     []*osParser     `yaml:"os_parsers"`
	Device []*deviceParser `yaml:"device_parsers"`
}

func (rd *RegexDefinitions) Clone() *RegexDefinitions {
	if rd == nil {
		return nil
	}

	rd2 := &RegexDefinitions{
		UA:     make([]*uaParser, len(rd.UA)),
		OS:     make([]*osParser, len(rd.OS)),
		Device: make([]*deviceParser, len(rd.Device)),
	}

	for i, v := range rd.UA {
		rd2.UA[i] = v.Clone()
	}

	for i, v := range rd.OS {
		rd2.OS[i] = v.Clone()
	}

	for i, v := range rd.Device {
		rd2.Device[i] = v.Clone()
	}

	return rd2
}

type UserAgentSorter []*uaParser

func (a UserAgentSorter) Len() int      { return len(a) }
func (a UserAgentSorter) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a UserAgentSorter) Less(i, j int) bool {
	return atomic.LoadUint64(&a[i].MatchesCount) > atomic.LoadUint64(&a[j].MatchesCount)
}

type uaParser struct {
	Reg               *regexp.Regexp
	Expr              string  `yaml:"regex"`
	Flags             string  `yaml:"regex_flag"`
	FamilyReplacement string  `yaml:"family_replacement"`
	V1Replacement     string  `yaml:"v1_replacement"`
	V2Replacement     string  `yaml:"v2_replacement"`
	V3Replacement     string  `yaml:"v3_replacement"`
	_                 [4]byte // padding for alignment
	MatchesCount      uint64
}

func (uap *uaParser) Clone() *uaParser {
	if uap == nil {
		return nil
	}

	ua2 := *uap
	ua2.MatchesCount = 0

	return &ua2
}

func (uap *uaParser) setDefaults() {
	if uap.FamilyReplacement == "" {
		uap.FamilyReplacement = "$1"
	}
	if uap.V1Replacement == "" {
		uap.V1Replacement = "$2"
	}
	if uap.V2Replacement == "" {
		uap.V2Replacement = "$3"
	}
	if uap.V3Replacement == "" {
		uap.V3Replacement = "$4"
	}
}

type OsSorter []*osParser

func (a OsSorter) Len() int      { return len(a) }
func (a OsSorter) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a OsSorter) Less(i, j int) bool {
	return atomic.LoadUint64(&a[i].MatchesCount) > atomic.LoadUint64(&a[j].MatchesCount)
}

type osParser struct {
	Reg           *regexp.Regexp
	Expr          string  `yaml:"regex"`
	Flags         string  `yaml:"regex_flag"`
	OSReplacement string  `yaml:"os_replacement"`
	V1Replacement string  `yaml:"os_v1_replacement"`
	V2Replacement string  `yaml:"os_v2_replacement"`
	V3Replacement string  `yaml:"os_v3_replacement"`
	V4Replacement string  `yaml:"os_v4_replacement"`
	_             [4]byte // padding for alignment
	MatchesCount  uint64
}

func (osp *osParser) Clone() *osParser {
	if osp == nil {
		return nil
	}

	os2 := *osp
	os2.MatchesCount = 0

	return &os2
}

func (osp *osParser) setDefaults() {
	if osp.OSReplacement == "" {
		osp.OSReplacement = "$1"
	}
	if osp.V1Replacement == "" {
		osp.V1Replacement = "$2"
	}
	if osp.V2Replacement == "" {
		osp.V2Replacement = "$3"
	}
	if osp.V3Replacement == "" {
		osp.V3Replacement = "$4"
	}
	if osp.V4Replacement == "" {
		osp.V4Replacement = "$5"
	}
}

type DeviceSorter []*deviceParser

func (a DeviceSorter) Len() int      { return len(a) }
func (a DeviceSorter) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a DeviceSorter) Less(i, j int) bool {
	return atomic.LoadUint64(&a[i].MatchesCount) > atomic.LoadUint64(&a[j].MatchesCount)
}

type deviceParser struct {
	Reg               *regexp.Regexp
	Expr              string  `yaml:"regex"`
	Flags             string  `yaml:"regex_flag"`
	DeviceReplacement string  `yaml:"device_replacement"`
	BrandReplacement  string  `yaml:"brand_replacement"`
	ModelReplacement  string  `yaml:"model_replacement"`
	_                 [4]byte // padding for alignment
	MatchesCount      uint64
}

func (dp *deviceParser) Clone() *deviceParser {
	if dp == nil {
		return nil
	}

	device2 := *dp
	device2.MatchesCount = 0

	return &device2
}

func (dp *deviceParser) setDefaults() {
	if dp.DeviceReplacement == "" {
		dp.DeviceReplacement = "$1"
	}
	if dp.ModelReplacement == "" {
		dp.ModelReplacement = "$1"
	}
}

type Client struct {
	UserAgent *UserAgent
	Os        *Os
	Device    *Device
}

type parserConfig struct {
	Mode            LookupMode
	UseSort         bool
	DebugMode       bool
	CacheSize       int
	MissesThreshold uint64
	MatchIdxNotOk   int
}

type Parser struct {
	/* atomic operation are done on the following unit64.
	 * These must be 64bit aligned. On 32bit architectures
	 * this is only guaranteed to be on the beginning of a struct */
	UserAgentMisses uint64
	OsMisses        uint64
	DeviceMisses    uint64

	config *parserConfig
	cache  *cache

	*RegexDefinitions

	mu *sync.RWMutex
}

type LookupMode int

const (
	EOsLookUpMode          LookupMode = 1 /* 00000001 */
	EUserAgentLookUpMode   LookupMode = 2 /* 00000010 */
	EDeviceLookUpMode      LookupMode = 4 /* 00000100 */
	cMinMissesTreshold                = 100000
	cDefaultMissesTreshold            = 500000
	cDefaultMatchIdxNotOk             = 20
	cDefaultSortOption                = false
	cDefaultDebugMode                 = false
	cDefaultCacheSize                 = 1024
)

func New(options ...Option) (*Parser, error) {
	parser := &Parser{
		config: &parserConfig{
			Mode:            EOsLookUpMode | EUserAgentLookUpMode | EDeviceLookUpMode,
			UseSort:         cDefaultSortOption,
			DebugMode:       cDefaultDebugMode,
			CacheSize:       cDefaultCacheSize,
			MissesThreshold: cDefaultMissesTreshold,
			MatchIdxNotOk:   cDefaultMatchIdxNotOk,
		},
		mu: &sync.RWMutex{},
	}

	for _, o := range options {
		o(parser)
	}

	if parser.config.MatchIdxNotOk < 0 {
		parser.config.MatchIdxNotOk = 0
	}

	if parser.config.MissesThreshold <= cMinMissesTreshold {
		parser.config.MissesThreshold = cMinMissesTreshold
	}

	if parser.config.CacheSize < 0 {
		parser.config.CacheSize = cDefaultCacheSize
	}

	if parser.cache == nil {
		parser.cache = newCache(parser.config.CacheSize)
	}

	if parser.RegexDefinitions == nil {
		regexesDefinitions := defaultRegexesDefinitions()
		parser.RegexDefinitions = &regexesDefinitions
	}

	if parser.config.UseSort {
		parser.RegexDefinitions = parser.RegexDefinitions.Clone()
	}

	parser.mustCompile()

	return parser, nil
}

func (parser *Parser) mustCompile() { // until we can use yaml.UnmarshalYAML with embedded pointer struct
	for _, p := range parser.RegexDefinitions.UA {
		p.Reg = compileRegex(p.Flags, p.Expr)
		p.setDefaults()
	}
	for _, p := range parser.RegexDefinitions.OS {
		p.Reg = compileRegex(p.Flags, p.Expr)
		p.setDefaults()
	}
	for _, p := range parser.RegexDefinitions.Device {
		p.Reg = compileRegex(p.Flags, p.Expr)
		p.setDefaults()
	}
}

func (parser *Parser) Parse(line string) *Client {
	cli := new(Client)
	var wg sync.WaitGroup
	if EUserAgentLookUpMode&parser.config.Mode == EUserAgentLookUpMode {
		wg.Add(1)
		go func() {
			defer wg.Done()
			parser.mu.RLock()
			cli.UserAgent = parser.ParseUserAgent(line)
			parser.mu.RUnlock()
		}()
	}
	if EOsLookUpMode&parser.config.Mode == EOsLookUpMode {
		wg.Add(1)
		go func() {
			defer wg.Done()
			parser.mu.RLock()
			cli.Os = parser.ParseOs(line)
			parser.mu.RUnlock()
		}()
	}
	if EDeviceLookUpMode&parser.config.Mode == EDeviceLookUpMode {
		wg.Add(1)
		go func() {
			defer wg.Done()
			parser.mu.RLock()
			cli.Device = parser.ParseDevice(line)
			parser.mu.RUnlock()
		}()
	}
	wg.Wait()
	if parser.config.UseSort {
		checkAndSort(parser)
	}
	return cli
}

func (parser *Parser) ParseUserAgent(line string) *UserAgent {
	cachedUA, ok := parser.cache.userAgent.Get(line)
	if ok {
		return cachedUA.(*UserAgent)
	}
	ua := new(UserAgent)
	foundIdx := -1
	found := false
	for i, uaPattern := range parser.RegexDefinitions.UA {
		uaPattern.Match(line, ua)
		if len(ua.Family) > 0 {
			found = true
			foundIdx = i
			atomic.AddUint64(&uaPattern.MatchesCount, 1)
			break
		}
	}
	if !found {
		ua.Family = "Other"
	}
	if foundIdx > parser.config.MatchIdxNotOk {
		atomic.AddUint64(&parser.UserAgentMisses, 1)
	}
	parser.cache.userAgent.Add(line, ua)
	return ua
}

func (parser *Parser) ParseOs(line string) *Os {
	cachedOS, ok := parser.cache.os.Get(line)
	if ok {
		return cachedOS.(*Os)
	}

	os := new(Os)
	foundIdx := -1
	found := false
	for i, osPattern := range parser.RegexDefinitions.OS {
		osPattern.Match(line, os)
		if len(os.Family) > 0 {
			found = true
			foundIdx = i
			atomic.AddUint64(&osPattern.MatchesCount, 1)
			break
		}
	}
	if !found {
		os.Family = "Other"
	}
	if foundIdx > parser.config.MatchIdxNotOk {
		atomic.AddUint64(&parser.OsMisses, 1)
	}

	parser.cache.os.Add(line, os)
	return os
}

func (parser *Parser) ParseDevice(line string) *Device {
	cachedDevice, ok := parser.cache.device.Get(line)
	if ok {
		return cachedDevice.(*Device)
	}

	dvc := new(Device)
	foundIdx := -1
	found := false
	for i, dvcPattern := range parser.RegexDefinitions.Device {
		dvcPattern.Match(line, dvc)
		if len(dvc.Family) > 0 {
			found = true
			foundIdx = i
			atomic.AddUint64(&dvcPattern.MatchesCount, 1)
			break
		}
	}
	if !found {
		dvc.Family = "Other"
	}
	if foundIdx > parser.config.MatchIdxNotOk {
		atomic.AddUint64(&parser.DeviceMisses, 1)
	}

	parser.cache.device.Add(line, dvc)
	return dvc
}

func checkAndSort(parser *Parser) {
	parser.mu.Lock()
	if atomic.LoadUint64(&parser.UserAgentMisses) >= parser.config.MissesThreshold {
		if parser.config.DebugMode {
			fmt.Printf("%s\tSorting UserAgents slice\n", time.Now())
		}
		parser.UserAgentMisses = 0

		sort.Sort(UserAgentSorter(parser.RegexDefinitions.UA))
	}
	parser.mu.Unlock()
	parser.mu.Lock()
	if atomic.LoadUint64(&parser.OsMisses) >= parser.config.MissesThreshold {
		if parser.config.DebugMode {
			fmt.Printf("%s\tSorting OS slice\n", time.Now())
		}
		parser.OsMisses = 0

		sort.Sort(OsSorter(parser.RegexDefinitions.OS))
	}
	parser.mu.Unlock()
	parser.mu.Lock()
	if atomic.LoadUint64(&parser.DeviceMisses) >= parser.config.MissesThreshold {
		if parser.config.DebugMode {
			fmt.Printf("%s\tSorting Device slice\n", time.Now())
		}
		parser.DeviceMisses = 0

		sort.Sort(DeviceSorter(parser.RegexDefinitions.Device))
	}
	parser.mu.Unlock()
}

func compileRegex(flags, expr string) *regexp.Regexp {
	if flags == "" {
		return regexp.MustCompile(expr)
	} else {
		return regexp.MustCompile(fmt.Sprintf("(?%s)%s", flags, expr))
	}
}
