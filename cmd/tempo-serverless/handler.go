package handler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/tempodb"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/azure"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/backend/s3"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"

	// required by the goog
	_ "github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
)

const envConfigPrefix = "TEMPO"

// jpe - test
// jpe - readme
// jpe - makefile
//     - go.mod has to be rewritten to remove the replace on tempo
//     - go mod download github.com/grafana/tempo

// Handler is the main entrypoint
// Parameters
//  - searchRequest           - tags, min/maxDuration, limit
//  - BackendSearch 		  - start, end
//  - BackendSearchQuerier    - startPage, totalPages, blockID
//  - BackendSearchServerless - encoding, dataEncoding, indexPageSize, totalRecords, tenant, version
// Response
//  - tempopb.SearchResponse
func Handler(w http.ResponseWriter, r *http.Request) {
	// jpe consolidate with querier/cmd line code
	searchReq, err := api.ParseSearchRequest(r, 20, 0) // jpe this is hardcoded
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// parseSearchRequest doesn't respect these as "reserved" tags. let's remove them here.
	// this will all be cleaned up when search paths are consolidated.
	delete(searchReq.Tags, api.URLParamBlockID)
	delete(searchReq.Tags, api.URLParamStartPage)
	delete(searchReq.Tags, api.URLParamTotalPages)
	delete(searchReq.Tags, api.URLParamStart)
	delete(searchReq.Tags, api.URLParamEnd)
	delete(searchReq.Tags, api.URLParamEncoding)
	delete(searchReq.Tags, api.URLParamIndexPageSize)
	delete(searchReq.Tags, api.URLParamTotalRecords)
	delete(searchReq.Tags, api.URLParamTenant)
	delete(searchReq.Tags, api.URLParamDataEncoding)
	delete(searchReq.Tags, api.URLParamVersion)

	start, end, limit, err := api.ParseBackendSearch(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	startPage, totalPages, blockID, err := api.ParseBackendSearchQuerier(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	enc, dataEncoding, indexPageSize, totalRecords, tenant, version, err := api.ParseBackendSearchServerless(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// load local backend config file. by convention this must be in the same folder at ./config.json
	reader, err := loadBackend()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// fake out meta, we're only filling in the fields here we need
	// which is kind of cheating
	meta := &backend.BlockMeta{
		Version:       version,
		TenantID:      tenant,
		Encoding:      enc,
		IndexPageSize: indexPageSize,
		TotalRecords:  totalRecords,
		BlockID:       blockID,
		DataEncoding:  dataEncoding,
	}

	block, err := encoding.NewBackendBlock(meta, reader)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// jpe parameterize?, consolidate loop with querier/cmd line code
	iter, err := block.PartialIterator(1_000_000, int(startPage), int(totalPages))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	iter = encoding.NewPrefetchIterator(r.Context(), iter, 10000)

	resp := &tempopb.SearchResponse{
		Metrics: &tempopb.SearchMetrics{},
	}
	for {
		id, obj, err := iter.Next(r.Context())
		if err == io.EOF || id == nil || obj == nil {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp.Metrics.InspectedTraces++
		resp.Metrics.InspectedBytes += uint64(len(obj))

		metadata, err := model.Matches(id, obj, dataEncoding, uint32(start), uint32(end), searchReq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if metadata == nil {
			continue
		}

		resp.Traces = append(resp.Traces, metadata)
		if len(resp.Traces) >= int(limit) {
			break
		}
	}

	marshaller := &jsonpb.Marshaler{}
	err = marshaller.Marshal(w, resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func loadBackend() (backend.Reader, error) {
	// horrible viper dance since it won't unmarshal to a struct from env: https://github.com/spf13/viper/issues/188
	v := viper.NewWithOptions()
	b, err := yaml.Marshal(defaultConfig())
	if err != nil {
		return nil, err
	}
	v.SetConfigType("yaml")
	if err := v.MergeConfig(bytes.NewReader(b)); err != nil {
		return nil, err
	}

	v.AutomaticEnv()
	v.SetEnvPrefix(envConfigPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	fmt.Println(v.AllSettings())

	cfg := &tempodb.Config{}
	v.Unmarshal(cfg, setTagName)

	var r backend.RawReader

	switch cfg.Backend {
	case "local":
		r, _, _, err = local.New(cfg.Local)
	case "gcs":
		r, _, _, err = gcs.New(cfg.GCS) // jpe this call GetAttr to confirm backend is working. we should remove that for the lambda
	case "s3":
		r, _, _, err = s3.New(cfg.S3)
	case "azure":
		r, _, _, err = azure.New(cfg.Azure)
	default:
		err = fmt.Errorf("unknown backend %s", cfg.Backend)
	}

	if err != nil {
		return nil, err
	}

	return backend.NewReader(r), nil
}

func defaultConfig() *tempodb.Config {
	return &tempodb.Config{
		Local: &local.Config{},
		GCS:   &gcs.Config{},
		S3:    &s3.Config{},
		Azure: &azure.Config{},
	}
}

func setTagName(d *mapstructure.DecoderConfig) {
	d.TagName = "yaml"
}
