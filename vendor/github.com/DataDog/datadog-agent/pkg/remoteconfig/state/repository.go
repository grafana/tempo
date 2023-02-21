// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

package state

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/DataDog/go-tuf/data"
)

var (
	// ErrMalformedEmbeddedRoot occurs when the TUF root provided is invalid
	ErrMalformedEmbeddedRoot = errors.New("malformed embedded TUF root file provided")
)

// RepositoryState contains all of the information about the current config files
// stored by the client to be able to make an update request to an Agent
type RepositoryState struct {
	Configs            []ConfigState
	CachedFiles        []CachedFile
	TargetsVersion     int64
	RootsVersion       int64
	OpaqueBackendState []byte
}

// ConfigState describes an applied config by the agent client.
type ConfigState struct {
	Product     string
	ID          string
	Version     uint64
	ApplyStatus ApplyStatus
}

// CachedFile describes a cached file stored by the agent client
//
// Note: You may be wondering why this exists when `ConfigState` exists
// as well. The API for requesting updates does not mandate that a client
// cache config files. This implementation just happens to do so.
type CachedFile struct {
	Path   string
	Length uint64
	Hashes map[string][]byte
}

// An Update contains all the data needed to update a client's remote config repository state
type Update struct {
	// TUFRoots contains, in order, updated roots that this repository needs to keep up with TUF validation
	TUFRoots [][]byte
	// TUFTargets is the latest TUF Targets file and is used to validate raw config files
	TUFTargets []byte
	// TargetFiles stores the raw config files by their full TUF path
	TargetFiles map[string][]byte
	// ClientcConfigs is a list of TUF path's corresponding to config files designated for this repository
	ClientConfigs []string
}

// Repository is a remote config client used in a downstream process to retrieve
// remote config updates from an Agent.
type Repository struct {
	// TUF related data
	latestTargets      *data.Targets
	tufRootsClient     *tufRootsClient
	opaqueBackendState []byte

	// Unverified mode
	tufVerificationEnabled bool
	latestRootVersion      int64

	// Config file storage
	metadata map[string]Metadata
	configs  map[string]map[string]interface{}
}

// NewRepository creates a new remote config repository that will track
// both TUF metadata and raw config files for a client.
func NewRepository(embeddedRoot []byte) (*Repository, error) {
	if embeddedRoot == nil {
		return nil, ErrMalformedEmbeddedRoot
	}

	configs := make(map[string]map[string]interface{})
	for _, product := range allProducts {
		configs[product] = make(map[string]interface{})
	}

	tufRootsClient, err := newTufRootsClient(embeddedRoot)
	if err != nil {
		return nil, err
	}

	return &Repository{
		latestTargets:          data.NewTargets(),
		tufRootsClient:         tufRootsClient,
		metadata:               make(map[string]Metadata),
		configs:                configs,
		tufVerificationEnabled: true,
	}, nil
}

// NewUnverifiedRepository creates a new remote config repository that will
// track config files for a client WITHOUT verifying any TUF related metadata.
func NewUnverifiedRepository() (*Repository, error) {
	configs := make(map[string]map[string]interface{})
	for _, product := range allProducts {
		configs[product] = make(map[string]interface{})
	}

	return &Repository{
		latestTargets:          data.NewTargets(),
		metadata:               make(map[string]Metadata),
		configs:                configs,
		tufVerificationEnabled: false,
	}, nil
}

// Update processes the ClientGetConfigsResponse from the Agent and updates the
// configuration state
func (r *Repository) Update(update Update) ([]string, error) {
	var err error
	var updatedTargets *data.Targets
	var tmpRootClient *tufRootsClient

	// TUF: Update the roots and verify the TUF Targets file (optional)
	//
	// We don't want to partially update the state, so we need a temporary client to hold the new root
	// data until we know it's valid. Since verification is optional, if the repository was configured
	// to not do TUF verification we only deserialize the TUF targets file.
	if r.tufVerificationEnabled {
		tmpRootClient, err = r.tufRootsClient.clone()
		if err != nil {
			return nil, err
		}
		err = tmpRootClient.updateRoots(update.TUFRoots)
		if err != nil {
			return nil, err
		}

		updatedTargets, err = tmpRootClient.validateTargets(update.TUFTargets)
		if err != nil {
			return nil, err
		}
	} else {
		updatedTargets, err = unsafeUnmarshalTargets(update.TUFTargets)
		if err != nil {
			return nil, err
		}
	}

	clientConfigsMap := make(map[string]struct{})
	for _, f := range update.ClientConfigs {
		clientConfigsMap[f] = struct{}{}
	}

	result := newUpdateResult()

	// 2: Check the config list and mark any missing configs as "to be removed"
	for _, configs := range r.configs {
		for path := range configs {
			if _, ok := clientConfigsMap[path]; !ok {
				result.removed = append(result.removed, path)
			}
		}
	}

	// 3: For all the files referenced in this update
	for _, path := range update.ClientConfigs {
		targetFileMetadata, ok := updatedTargets.Targets[path]
		if !ok {
			return nil, fmt.Errorf("missing config file in TUF targets - %s", path)
		}

		// 3.a: Extract the product and ID from the path
		parsedPath, err := parseConfigPath(path)
		if err != nil {
			return nil, err
		}

		storedMetadata, exists := r.metadata[path]
		if exists && hashesEqual(targetFileMetadata.Hashes, storedMetadata.Hashes) {
			continue
		}

		// 3.d: Ensure that the raw configuration file is present in the
		// update payload.
		raw, ok := update.TargetFiles[path]
		if !ok {
			return nil, fmt.Errorf("missing update file - %s", path)
		}

		// TUF: Validate the hash of the raw target file and ensure that it matches
		// the TUF metadata
		err = validateTargetFileHash(targetFileMetadata, raw)
		if err != nil {
			return nil, fmt.Errorf("error validating %s hash with TUF metadata - %v", path, err)
		}

		// 3.e: Deserialize the configuration.
		// 3.f: Store the update details for application later
		//
		// Note: We don't have to worry about extra fields as mentioned
		// in the RFC because the encoding/json library handles that for us.
		m, err := newConfigMetadata(parsedPath, targetFileMetadata)
		if err != nil {
			return nil, err
		}
		config, err := parseConfig(parsedPath.Product, raw, m)
		if err != nil {
			return nil, err
		}
		result.metadata[path] = m
		result.changed[parsedPath.Product][path] = config
	}

	// 4.a: Store the new targets.signed.custom.opaque_client_state
	// TUF: Store the updated roots now that everything has validated
	if r.tufVerificationEnabled {
		r.tufRootsClient = tmpRootClient
	} else if update.TUFRoots != nil && len(update.TUFRoots) > 0 {
		v, err := extractRootVersion(update.TUFRoots[len(update.TUFRoots)-1])
		if err != nil {
			return nil, err
		}
		r.latestRootVersion = v
	}
	r.latestTargets = updatedTargets
	if r.latestTargets.Custom != nil {
		r.opaqueBackendState = extractOpaqueBackendState(*r.latestTargets.Custom)
	}

	// Upstream may not want to take any actions if the update result doesn't
	// change any configs.
	if result.isEmpty() {
		return nil, nil
	}

	changedProducts := make([]string, 0)
	for product, configs := range result.changed {
		if len(configs) > 0 {
			changedProducts = append(changedProducts, product)
		}
	}

	// 4.b/4.rave the new state and apply cleanups
	r.applyUpdateResult(update, result)

	return changedProducts, nil
}

// UpdateApplyStatus updates the config's metadata to reflect its processing state
// Can be used after a call to Update() in order to tell the repository which config was acked, which
// wasn't and which errors occurred while processing.
// Note: it is the responsibility of the caller to ensure that no new Update() call was made between
// the first Update() call and the call to UpdateApplyStatus() so as to keep the repository state accurate.
func (r *Repository) UpdateApplyStatus(cfgPath string, status ApplyStatus) {
	if m, ok := r.metadata[cfgPath]; ok {
		m.ApplyStatus = status
	}
}

func (r *Repository) getConfigs(product string) map[string]interface{} {
	configs, ok := r.configs[product]
	if !ok {
		return nil
	}

	return configs
}

// applyUpdateResult changes the state of the client based on the given update.
//
// The update is guaranteed to succeed at this point, having been vetted and the details
// needed to apply the update stored in the `updateResult`.
func (r *Repository) applyUpdateResult(update Update, result updateResult) {
	// 4.b Save all the updated and new config files
	for product, configs := range result.changed {
		for path, config := range configs {
			m := r.configs[product]
			m[path] = config
		}
	}
	for path, metadata := range result.metadata {
		r.metadata[path] = metadata
	}

	// 5.b Clean up the cache of any removed configs
	for _, path := range result.removed {
		delete(r.metadata, path)
		for _, configs := range r.configs {
			delete(configs, path)
		}
	}
}

// CurrentState returns all of the information needed to
// make an update for new configurations.
func (r *Repository) CurrentState() (RepositoryState, error) {
	var configs []ConfigState
	var cached []CachedFile

	for path, metadata := range r.metadata {
		configs = append(configs, configStateFromMetadata(metadata))
		cached = append(cached, cachedFileFromMetadata(path, metadata))
	}

	var latestRootVersion int64
	if r.tufVerificationEnabled {
		root, err := r.tufRootsClient.latestRoot()
		if err != nil {
			return RepositoryState{}, err
		}
		latestRootVersion = root.Version
	} else {
		latestRootVersion = r.latestRootVersion
	}

	return RepositoryState{
		Configs:            configs,
		CachedFiles:        cached,
		TargetsVersion:     r.latestTargets.Version,
		RootsVersion:       latestRootVersion,
		OpaqueBackendState: r.opaqueBackendState,
	}, nil
}

// An updateResult allows the client to apply the update as a transaction
// after validating all required preconditions
type updateResult struct {
	removed  []string
	metadata map[string]Metadata
	changed  map[string]map[string]interface{}
}

func newUpdateResult() updateResult {
	changed := make(map[string]map[string]interface{})

	for _, p := range allProducts {
		changed[p] = make(map[string]interface{})
	}

	return updateResult{
		removed:  make([]string, 0),
		metadata: make(map[string]Metadata),
		changed:  changed,
	}
}

func (ur updateResult) Log() {
	log.Printf("Removed Configs: %v", ur.removed)

	var b strings.Builder
	b.WriteString("Changed configs: [")
	for path := range ur.metadata {
		b.WriteString(path)
		b.WriteString(" ")
	}
	b.WriteString("]")

	log.Println(b.String())
}

func (ur updateResult) isEmpty() bool {
	return len(ur.removed) == 0 && len(ur.metadata) == 0
}

func configStateFromMetadata(m Metadata) ConfigState {
	return ConfigState{
		Product:     m.Product,
		ID:          m.ID,
		Version:     m.Version,
		ApplyStatus: m.ApplyStatus,
	}
}

func cachedFileFromMetadata(path string, m Metadata) CachedFile {
	return CachedFile{
		Path:   path,
		Length: m.RawLength,
		Hashes: m.Hashes,
	}
}

// hashesEqual checks if the hash values in the TUF metadata file match the stored
// hash values for a given config
func hashesEqual(tufHashes data.Hashes, storedHashes map[string][]byte) bool {
	for algorithm, value := range tufHashes {
		v, ok := storedHashes[algorithm]
		if !ok {
			continue
		}

		if !bytes.Equal(value, v) {
			return false
		}
	}

	return true
}

func extractOpaqueBackendState(targetsCustom []byte) []byte {
	state := struct {
		State []byte `json:"opaque_backend_state"`
	}{nil}

	err := json.Unmarshal(targetsCustom, &state)
	if err != nil {
		return []byte{}
	}

	return state.State
}
