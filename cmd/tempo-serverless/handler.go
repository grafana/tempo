package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

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
)

const configFile = "./config.json"

// jpe - test
// jpe - readme
// jpe - makefile (go.mod has to be rewritten to remove the replace on tempo)

// Handler is the main entrypoint
// Parameters
//  - searchRequest           - tags, min/maxDuration, limit
//  - BackendSearch 		  - start, end
//  - BackendSearchQuerier    - startPage, totalPages, blockID
//  - BackendSearchServerless - encoding, dataEncoding, indexPageSize, totalRecords, tenant
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

	enc, dataEncoding, indexPageSize, totalRecords, tenant, err := api.ParseBackendSearchServerless(r)
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
	cfgBytes, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	cfg := &tempodb.Config{}
	err = json.Unmarshal(cfgBytes, cfg)
	if err != nil {
		return nil, err
	}

	var r backend.RawReader

	switch cfg.Backend {
	case "local":
		r, _, _, err = local.New(cfg.Local)
	case "gcs":
		r, _, _, err = gcs.New(cfg.GCS)
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
