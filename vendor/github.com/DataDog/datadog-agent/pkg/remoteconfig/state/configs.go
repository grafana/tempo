// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

package state

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/DataDog/go-tuf/data"
)

/*
	To add support for a new product:

	1. Add the definition of the product to the const() block of products and the `allProducts` list.
	2. Define the serialized configuration struct as well as a function to parse the config from a []byte.
	3. Add the product to the `parseConfig` function
	4. Add a method on the `Repository` to retrieved typed configs for the product.
*/

var allProducts = []string{ProductAPMSampling, ProductCWSDD, ProductCWSCustom, ProductASMFeatures, ProductASMDD, ProductASMData, ProductAPMTracing}

const (
	// ProductAPMSampling is the apm sampling product
	ProductAPMSampling = "APM_SAMPLING"
	// ProductCWSDD is the cloud workload security product managed by datadog employees
	ProductCWSDD = "CWS_DD"
	// ProductCWSCustom is the cloud workload security product managed by datadog customers
	ProductCWSCustom = "CWS_CUSTOM"
	// ProductASMFeatures is the ASM product used form ASM activation through remote config
	ProductASMFeatures = "ASM_FEATURES"
	// ProductASMDD is the application security monitoring product managed by datadog employees
	ProductASMDD = "ASM_DD"
	// ProductASMData is the ASM product used to configure WAF rules data
	ProductASMData = "ASM_DATA"
	// ProductAPMTracing is the apm tracing product
	ProductAPMTracing = "APM_TRACING"
)

// ErrNoConfigVersion occurs when a target file's custom meta is missing the config version
var ErrNoConfigVersion = errors.New("version missing in custom file meta")

func parseConfig(product string, raw []byte, metadata Metadata) (interface{}, error) {
	var c interface{}
	var err error
	switch product {
	case ProductAPMSampling:
		c, err = parseConfigAPMSampling(raw, metadata)
	case ProductASMFeatures:
		c, err = parseASMFeaturesConfig(raw, metadata)
	case ProductCWSDD:
		c, err = parseConfigCWSDD(raw, metadata)
	case ProductCWSCustom:
		c, err = parseConfigCWSCustom(raw, metadata)
	case ProductASMDD:
		c, err = parseConfigASMDD(raw, metadata)
	case ProductASMData:
		c, err = parseConfigASMData(raw, metadata)
	case ProductAPMTracing:
		c, err = parseConfigAPMTracing(raw, metadata)
	default:
		return nil, fmt.Errorf("unknown product - %s", product)
	}

	return c, err
}

// APMSamplingConfig is a deserialized APM Sampling configuration file
// along with its associated remote config metadata.
type APMSamplingConfig struct {
	Config   []byte
	Metadata Metadata
}

func parseConfigAPMSampling(data []byte, metadata Metadata) (APMSamplingConfig, error) {
	// We actually don't parse the payload here, we delegate this responsibility to the trace agent
	return APMSamplingConfig{
		Config:   data,
		Metadata: metadata,
	}, nil
}

// APMConfigs returns the currently active APM configs
func (r *Repository) APMConfigs() map[string]APMSamplingConfig {
	typedConfigs := make(map[string]APMSamplingConfig)

	configs := r.getConfigs(ProductAPMSampling)

	for path, conf := range configs {
		// We control this, so if this has gone wrong something has gone horribly wrong
		typed, ok := conf.(APMSamplingConfig)
		if !ok {
			panic("unexpected config stored as APMSamplingConfig")
		}

		typedConfigs[path] = typed
	}

	return typedConfigs
}

// ConfigCWSDD is a deserialized CWS DD configuration file along with its
// associated remote config metadata
type ConfigCWSDD struct {
	Config   []byte
	Metadata Metadata
}

func parseConfigCWSDD(data []byte, metadata Metadata) (ConfigCWSDD, error) {
	return ConfigCWSDD{
		Config:   data,
		Metadata: metadata,
	}, nil
}

// CWSDDConfigs returns the currently active CWSDD config files
func (r *Repository) CWSDDConfigs() map[string]ConfigCWSDD {
	typedConfigs := make(map[string]ConfigCWSDD)

	configs := r.getConfigs(ProductCWSDD)

	for path, conf := range configs {
		// We control this, so if this has gone wrong something has gone horribly wrong
		typed, ok := conf.(ConfigCWSDD)
		if !ok {
			panic("unexpected config stored as CWSDD Config")
		}

		typedConfigs[path] = typed
	}

	return typedConfigs
}

// ConfigCWSCustom is a deserialized CWS Custom configuration file along with its
// associated remote config metadata
type ConfigCWSCustom struct {
	Config   []byte
	Metadata Metadata
}

func parseConfigCWSCustom(data []byte, metadata Metadata) (ConfigCWSCustom, error) {
	return ConfigCWSCustom{
		Config:   data,
		Metadata: metadata,
	}, nil
}

// CWSCustomConfigs returns the currently active CWSCustom config files
func (r *Repository) CWSCustomConfigs() map[string]ConfigCWSCustom {
	typedConfigs := make(map[string]ConfigCWSCustom)

	configs := r.getConfigs(ProductCWSCustom)

	for path, conf := range configs {
		// We control this, so if this has gone wrong something has gone horribly wrong
		typed, ok := conf.(ConfigCWSCustom)
		if !ok {
			panic("unexpected config stored as CWSDD Config")
		}

		typedConfigs[path] = typed
	}

	return typedConfigs
}

// ConfigASMDD is a deserialized ASM DD configuration file along with its
// associated remote config metadata
type ConfigASMDD struct {
	Config   []byte
	Metadata Metadata
}

func parseConfigASMDD(data []byte, metadata Metadata) (ConfigASMDD, error) {
	return ConfigASMDD{
		Config:   data,
		Metadata: metadata,
	}, nil
}

// ASMDDConfigs returns the currently active ASMDD configs
func (r *Repository) ASMDDConfigs() map[string]ConfigASMDD {
	typedConfigs := make(map[string]ConfigASMDD)

	configs := r.getConfigs(ProductASMDD)

	for path, conf := range configs {
		// We control this, so if this has gone wrong something has gone horribly wrong
		typed, ok := conf.(ConfigASMDD)
		if !ok {
			panic("unexpected config stored as ASMDD Config")
		}

		typedConfigs[path] = typed
	}

	return typedConfigs
}

// ASMFeaturesConfig is a deserialized configuration file that indicates whether ASM should be enabled
// within a tracer, along with its associated remote config metadata.
type ASMFeaturesConfig struct {
	Config   ASMFeaturesData
	Metadata Metadata
}

// ASMFeaturesData describes the enabled state of ASM features
type ASMFeaturesData struct {
	ASM struct {
		Enabled bool `json:"enabled"`
	} `json:"asm"`
}

func parseASMFeaturesConfig(data []byte, metadata Metadata) (ASMFeaturesConfig, error) {
	var f ASMFeaturesData

	err := json.Unmarshal(data, &f)
	if err != nil {
		return ASMFeaturesConfig{}, nil
	}

	return ASMFeaturesConfig{
		Config:   f,
		Metadata: metadata,
	}, nil
}

// ASMFeaturesConfigs returns the currently active ASMFeatures configs
func (r *Repository) ASMFeaturesConfigs() map[string]ASMFeaturesConfig {
	typedConfigs := make(map[string]ASMFeaturesConfig)

	configs := r.getConfigs(ProductASMFeatures)

	for path, conf := range configs {
		// We control this, so if this has gone wrong something has gone horribly wrong
		typed, ok := conf.(ASMFeaturesConfig)
		if !ok {
			panic("unexpected config stored as ASMFeaturesConfig")
		}

		typedConfigs[path] = typed
	}

	return typedConfigs
}

// ApplyState represents the status of a configuration application by a remote configuration client
// Clients need to either ack the correct application of received configurations, or communicate that
// they haven't applied it yet, or communicate any error that may have happened while doing so
type ApplyState uint64

const (
	ApplyStateUnknown ApplyState = iota
	ApplyStateUnacknowledged
	ApplyStateAcknowledged
	ApplyStateError
)

// ApplyStatus is the processing status for a given configuration.
// It basically represents whether a config was successfully processed and apply, or if an error occurred
type ApplyStatus struct {
	State ApplyState
	Error string
}

// ASMDataConfig is a deserialized configuration file that holds rules data that can be used
// by the ASM WAF for specific features (example: ip blocking).
type ASMDataConfig struct {
	Config   ASMDataRulesData
	Metadata Metadata
}

// ASMDataRulesData is a serializable array of rules data entries
type ASMDataRulesData struct {
	RulesData []ASMDataRuleData `json:"rules_data"`
}

// ASMDataRuleData is an entry in the rules data list held by an ASMData configuration
type ASMDataRuleData struct {
	ID   string                 `json:"id"`
	Type string                 `json:"type"`
	Data []ASMDataRuleDataEntry `json:"data"`
}

// ASMDataRuleDataEntry represents a data entry in a rule data file
type ASMDataRuleDataEntry struct {
	Expiration int64  `json:"expiration,omitempty"`
	Value      string `json:"value"`
}

func parseConfigASMData(data []byte, metadata Metadata) (ASMDataConfig, error) {
	cfg := ASMDataConfig{
		Metadata: metadata,
	}
	err := json.Unmarshal(data, &cfg.Config)
	return cfg, err
}

// ASMDataConfigs returns the currently active ASMData configs
func (r *Repository) ASMDataConfigs() map[string]ASMDataConfig {
	typedConfigs := make(map[string]ASMDataConfig)
	configs := r.getConfigs(ProductASMData)

	for path, cfg := range configs {
		// We control this, so if this has gone wrong something has gone horribly wrong
		typed, ok := cfg.(ASMDataConfig)
		if !ok {
			panic("unexpected config stored as ASMDataConfig")
		}
		typedConfigs[path] = typed
	}

	return typedConfigs
}

type APMTracingConfig struct {
	Config   []byte
	Metadata Metadata
}

func parseConfigAPMTracing(data []byte, metadata Metadata) (APMTracingConfig, error) {
	// Delegate the parsing responsibility to the cluster agent
	return APMTracingConfig{
		Config:   data,
		Metadata: metadata,
	}, nil
}

// APMTracingConfigs returns the currently active APMTracing configs
func (r *Repository) APMTracingConfigs() map[string]APMTracingConfig {
	typedConfigs := make(map[string]APMTracingConfig)
	configs := r.getConfigs(ProductAPMTracing)
	for path, conf := range configs {
		// We control this, so if this has gone wrong something has gone horribly wrong
		typed, ok := conf.(APMTracingConfig)
		if !ok {
			panic("unexpected config stored as APMTracingConfig")
		}
		typedConfigs[path] = typed
	}
	return typedConfigs
}

// Metadata stores remote config metadata for a given configuration
type Metadata struct {
	Product     string
	ID          string
	Name        string
	Version     uint64
	RawLength   uint64
	Hashes      map[string][]byte
	ApplyStatus ApplyStatus
}

func newConfigMetadata(parsedPath configPath, tfm data.TargetFileMeta) (Metadata, error) {
	var m Metadata
	m.ID = parsedPath.ConfigID
	m.Product = parsedPath.Product
	m.Name = parsedPath.Name
	m.RawLength = uint64(tfm.Length)
	m.Hashes = make(map[string][]byte)
	for k, v := range tfm.Hashes {
		m.Hashes[k] = []byte(v)
	}
	v, err := fileMetaVersion(tfm)
	if err != nil {
		return Metadata{}, err
	}
	m.Version = v

	return m, nil
}

type fileMetaCustom struct {
	Version *uint64 `json:"v"`
}

func fileMetaVersion(fm data.TargetFileMeta) (uint64, error) {
	if fm.Custom == nil {
		return 0, ErrNoConfigVersion
	}
	fmc, err := parseFileMetaCustom(*fm.Custom)
	if err != nil {
		return 0, err
	}

	return *fmc.Version, nil
}

func parseFileMetaCustom(rawCustom []byte) (fileMetaCustom, error) {
	var custom fileMetaCustom
	err := json.Unmarshal(rawCustom, &custom)
	if err != nil {
		return fileMetaCustom{}, err
	}
	if custom.Version == nil {
		return fileMetaCustom{}, ErrNoConfigVersion
	}
	return custom, nil
}
