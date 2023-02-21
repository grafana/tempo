// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

package state

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/DataDog/go-tuf/client"
	"github.com/DataDog/go-tuf/data"
	"github.com/DataDog/go-tuf/util"
	"github.com/DataDog/go-tuf/verify"
)

type tufRootsClient struct {
	rootClient      *client.Client
	rootLocalStore  client.LocalStore
	rootRemoteStore *rootClientRemoteStore
}

func newTufRootsClient(root []byte) (*tufRootsClient, error) {
	rootLocalStore := client.MemoryLocalStore()
	rootRemoteStore := &rootClientRemoteStore{}
	rootClient := client.NewClient(rootLocalStore, rootRemoteStore)

	err := rootClient.InitLocal(root)
	if err != nil {
		return nil, err
	}

	return &tufRootsClient{
		rootClient:      rootClient,
		rootLocalStore:  rootLocalStore,
		rootRemoteStore: rootRemoteStore,
	}, nil
}

func (trc *tufRootsClient) clone() (*tufRootsClient, error) {
	root, err := trc.latestRootRaw()
	if err != nil {
		return nil, err
	}

	return newTufRootsClient(root)
}

func (trc *tufRootsClient) updateRoots(newRoots [][]byte) error {
	if len(newRoots) == 0 {
		return nil
	}

	trc.rootRemoteStore.roots = append(trc.rootRemoteStore.roots, newRoots...)

	return trc.rootClient.UpdateRoots()
}

func (trc *tufRootsClient) latestRoot() (*data.Root, error) {
	raw, err := trc.latestRootRaw()
	if err != nil {
		return nil, err
	}

	return unsafeUnmarshalRoot(raw)
}

func (trc *tufRootsClient) latestRootRaw() ([]byte, error) {
	metas, err := trc.rootLocalStore.GetMeta()
	if err != nil {
		return nil, err
	}
	rawRoot := metas["root.json"]

	return rawRoot, nil
}

func (trc *tufRootsClient) validateTargets(rawTargets []byte) (*data.Targets, error) {
	root, err := trc.latestRoot()
	if err != nil {
		return nil, err
	}

	db := verify.NewDB()
	for _, key := range root.Keys {
		for _, id := range key.IDs() {
			if err := db.AddKey(id, key); err != nil {
				return nil, err
			}
		}
	}
	targetsRole, hasRoleTargets := root.Roles["targets"]
	if !hasRoleTargets {
		return nil, fmt.Errorf("root is missing a targets role")
	}
	role := &data.Role{Threshold: targetsRole.Threshold, KeyIDs: targetsRole.KeyIDs}
	if err := db.AddRole("targets", role); err != nil {
		return nil, fmt.Errorf("could not add targets role to db: %v", err)
	}
	var targets data.Targets
	err = db.Unmarshal(rawTargets, &targets, "targets", 0)
	if err != nil {
		return nil, err
	}

	return &targets, nil
}

type rootClientRemoteStore struct {
	roots [][]byte
}

func (s *rootClientRemoteStore) GetMeta(name string) (stream io.ReadCloser, size int64, err error) {
	metaPath, err := parseMetaPath(name)
	if err != nil {
		return nil, 0, err
	}
	if metaPath.role != roleRoot || !metaPath.versionSet {
		return nil, 0, client.ErrNotFound{File: name}
	}
	for _, root := range s.roots {
		parsedRoot, err := unsafeUnmarshalRoot(root)
		if err != nil {
			return nil, 0, err
		}
		if parsedRoot.Version == metaPath.version {
			return io.NopCloser(bytes.NewReader(root)), int64(len(root)), nil
		}
	}
	return nil, 0, client.ErrNotFound{File: name}
}

func (s *rootClientRemoteStore) GetTarget(path string) (stream io.ReadCloser, size int64, err error) {
	return nil, 0, client.ErrNotFound{File: path}
}

type role string

const (
	roleRoot role = "root"
)

type metaPath struct {
	role       role
	version    int64
	versionSet bool
}

func parseMetaPath(rawMetaPath string) (metaPath, error) {
	splitRawMetaPath := strings.SplitN(rawMetaPath, ".", 3)
	if len(splitRawMetaPath) != 2 && len(splitRawMetaPath) != 3 {
		return metaPath{}, fmt.Errorf("invalid metadata path '%s'", rawMetaPath)
	}
	suffix := splitRawMetaPath[len(splitRawMetaPath)-1]
	if suffix != "json" {
		return metaPath{}, fmt.Errorf("invalid metadata path (suffix) '%s'", rawMetaPath)
	}
	rawRole := splitRawMetaPath[len(splitRawMetaPath)-2]
	if rawRole == "" {
		return metaPath{}, fmt.Errorf("invalid metadata path (role) '%s'", rawMetaPath)
	}
	if len(splitRawMetaPath) == 2 {
		return metaPath{
			role: role(rawRole),
		}, nil
	}
	rawVersion, err := strconv.ParseInt(splitRawMetaPath[0], 10, 64)
	if err != nil {
		return metaPath{}, fmt.Errorf("invalid metadata path (version) '%s': %w", rawMetaPath, err)
	}
	return metaPath{
		role:       role(rawRole),
		version:    rawVersion,
		versionSet: true,
	}, nil
}

func validateTargetFileHash(targetMeta data.TargetFileMeta, targetFile []byte) error {
	if len(targetMeta.HashAlgorithms()) == 0 {
		return fmt.Errorf("target file has no hash")
	}
	generatedMeta, err := util.GenerateFileMeta(bytes.NewBuffer(targetFile), targetMeta.HashAlgorithms()...)
	if err != nil {
		return err
	}
	err = util.FileMetaEqual(targetMeta.FileMeta, generatedMeta)
	if err != nil {
		return err
	}
	return nil
}

func unsafeUnmarshalRoot(raw []byte) (*data.Root, error) {
	var signedRoot data.Signed
	err := json.Unmarshal(raw, &signedRoot)
	if err != nil {
		return nil, err
	}
	var root data.Root
	err = json.Unmarshal(signedRoot.Signed, &root)
	if err != nil {
		return nil, err
	}
	return &root, err
}

func unsafeUnmarshalTargets(raw []byte) (*data.Targets, error) {
	var signedTargets data.Signed
	err := json.Unmarshal(raw, &signedTargets)
	if err != nil {
		return nil, err
	}
	var targets data.Targets
	err = json.Unmarshal(signedTargets.Signed, &targets)
	if err != nil {
		return nil, err
	}
	return &targets, err
}

func extractRootVersion(raw []byte) (int64, error) {
	root, err := unsafeUnmarshalRoot(raw)
	if err != nil {
		return 0, err
	}

	return root.Version, nil
}
