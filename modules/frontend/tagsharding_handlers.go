package frontend

import (
	"io"
	"net/http"

	"github.com/golang/protobuf/jsonpb" //nolint:all //deprecated
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend"
)

/* tagsSearchRequest request interface for transform tags and tags V2 requests into a querier request */
type tagsSearchRequest struct {
	request tempopb.SearchTagsRequest
}

func (r *tagsSearchRequest) start() uint32 {
	return r.request.Start
}

func (r *tagsSearchRequest) end() uint32 {
	return r.request.End
}

func (r *tagsSearchRequest) newWithRange(start, end uint32) tagSearchReq {
	newReq := r.request
	newReq.Start = start
	newReq.End = end

	return &tagsSearchRequest{
		request: newReq,
	}
}

func (r *tagsSearchRequest) buildSearchTagRequest(subR *http.Request) (*http.Request, error) {
	return api.BuildSearchTagsRequest(subR, &r.request)
}

func (r *tagsSearchRequest) buildTagSearchBlockRequest(subR *http.Request, blockID string,
	startPage int, pages int, m *backend.BlockMeta,
) (*http.Request, error) {
	return api.BuildSearchTagsBlockRequest(subR, &tempopb.SearchTagsBlockRequest{
		BlockID:       blockID,
		StartPage:     uint32(startPage),
		PagesToSearch: uint32(pages),
		Encoding:      m.Encoding.String(),
		IndexPageSize: m.IndexPageSize,
		TotalRecords:  m.TotalRecords,
		DataEncoding:  m.DataEncoding,
		Version:       m.Version,
		Size_:         m.Size,
		FooterSize:    m.FooterSize,
	})
}

/* TagValue V2 handler and request implementation */
type tagValueSearchRequest struct {
	request tempopb.SearchTagValuesRequest
}

func (r *tagValueSearchRequest) start() uint32 {
	return r.request.Start
}

func (r *tagValueSearchRequest) end() uint32 {
	return r.request.End
}

func (r *tagValueSearchRequest) newWithRange(start, end uint32) tagSearchReq {
	newReq := r.request
	newReq.Start = start
	newReq.End = end

	return &tagValueSearchRequest{
		request: newReq,
	}
}

func (r *tagValueSearchRequest) buildSearchTagRequest(subR *http.Request) (*http.Request, error) {
	return api.BuildSearchTagValuesRequest(subR, &r.request)
}

func (r *tagValueSearchRequest) buildTagSearchBlockRequest(subR *http.Request, blockID string,
	startPage int, pages int, m *backend.BlockMeta,
) (*http.Request, error) {
	return api.BuildSearchTagValuesBlockRequest(subR, &tempopb.SearchTagValuesBlockRequest{
		BlockID:       blockID,
		StartPage:     uint32(startPage),
		PagesToSearch: uint32(pages),
		Encoding:      m.Encoding.String(),
		IndexPageSize: m.IndexPageSize,
		TotalRecords:  m.TotalRecords,
		DataEncoding:  m.DataEncoding,
		Version:       m.Version,
		Size_:         m.Size,
		FooterSize:    m.FooterSize,
	})
}

// tagsResultsHandler handled all request/response payloads for querier and for return to frontend for tags.
type tagsResultsHandler struct {
	limit           int
	resultsCombiner *util.DistinctStringCollector
}

func parseTagsRequest(r *http.Request) (tagSearchReq, error) {
	searchReq, err := api.ParseSearchTagsRequest(r)
	if err != nil {
		return nil, err
	}
	return &tagsSearchRequest{
		request: *searchReq,
	}, nil
}

func (h *tagsResultsHandler) shouldQuit() bool {
	return h.resultsCombiner.Exceeded()
}

func (h *tagsResultsHandler) addResponse(r io.ReadCloser) error {
	results := &tempopb.SearchTagsResponse{}
	err := (&jsonpb.Unmarshaler{AllowUnknownFields: true}).Unmarshal(r, results)
	if err != nil {
		return err
	}
	for _, t := range results.TagNames {
		h.resultsCombiner.Collect(t)
	}
	return nil
}

func (h *tagsResultsHandler) marshalResult() (string, error) {
	m := &jsonpb.Marshaler{}
	bodyString, err := m.MarshalToString(&tempopb.SearchTagsResponse{
		TagNames: h.resultsCombiner.Strings(),
	})
	if err != nil {
		return "", err
	}
	return bodyString, nil
}

func tagsResultHandlerFactory(limit int) tagResultsHandler {
	return &tagsResultsHandler{
		limit:           limit,
		resultsCombiner: util.NewDistinctStringCollector(limit),
	}
}

// tagValuesResultsHandler handled all request/response payloads for querier and for return to frontend for tag values.
type tagValuesResultsHandler struct {
	limit           int
	resultsCombiner *util.DistinctStringCollector
}

func parseTagValuesRequest(r *http.Request) (tagSearchReq, error) {
	searchReq, err := api.ParseSearchTagValuesRequest(r)
	if err != nil {
		return nil, err
	}
	return &tagValueSearchRequest{
		request: *searchReq,
	}, nil
}

func (h *tagValuesResultsHandler) shouldQuit() bool {
	return h.resultsCombiner.Exceeded()
}

func (h *tagValuesResultsHandler) addResponse(r io.ReadCloser) error {
	results := &tempopb.SearchTagValuesResponse{}
	err := (&jsonpb.Unmarshaler{AllowUnknownFields: true}).Unmarshal(r, results)
	if err != nil {
		return err
	}
	for _, t := range results.TagValues {
		h.resultsCombiner.Collect(t)
	}
	return nil
}

func (h *tagValuesResultsHandler) marshalResult() (string, error) {
	m := &jsonpb.Marshaler{}
	bodyString, err := m.MarshalToString(&tempopb.SearchTagValuesResponse{
		TagValues: h.resultsCombiner.Strings(),
	})
	if err != nil {
		return "", err
	}
	return bodyString, nil
}

func tagValuesResultHandlerFactory(limit int) tagResultsHandler {
	return &tagValuesResultsHandler{
		limit:           limit,
		resultsCombiner: util.NewDistinctStringCollector(limit),
	}
}

// tagsV2ResultsHandler handled all request/response payloads for querier and for return to frontend for tags for v2
type tagsV2ResultsHandler struct {
	limit           int
	resultsCombiner map[string]*util.DistinctStringCollector
}

func (h *tagsV2ResultsHandler) shouldQuit() bool {
	for _, combiner := range h.resultsCombiner {
		if combiner.Exceeded() {
			return true
		}
	}
	return false
}

func (h *tagsV2ResultsHandler) addResponse(r io.ReadCloser) error {
	results := &tempopb.SearchTagsV2Response{}
	err := (&jsonpb.Unmarshaler{AllowUnknownFields: true}).Unmarshal(r, results)
	if err != nil {
		return err
	}
	for _, scope := range results.Scopes {
		tagsNames := h.resultsCombiner[scope.Name]
		if tagsNames == nil {
			h.resultsCombiner[scope.Name] = util.NewDistinctStringCollector(h.limit)
		}
		for _, tag := range scope.Tags {
			h.resultsCombiner[scope.Name].Collect(tag)
		}
	}
	return nil
}

func (h *tagsV2ResultsHandler) marshalResult() (string, error) {
	var scopes []*tempopb.SearchTagsV2Scope
	for name, tags := range h.resultsCombiner {
		scopes = append(scopes, &tempopb.SearchTagsV2Scope{
			Tags: tags.Strings(),
			Name: name,
		})
	}

	m := &jsonpb.Marshaler{}
	bodyString, err := m.MarshalToString(&tempopb.SearchTagsV2Response{
		Scopes: scopes,
	})
	if err != nil {
		return "", err
	}

	return bodyString, nil
}

func tagsV2ResultHandlerFactory(limit int) tagResultsHandler {
	return &tagsV2ResultsHandler{
		limit:           limit,
		resultsCombiner: map[string]*util.DistinctStringCollector{},
	}
}

type tagValuesV2ResultsHandler struct {
	limit           int
	resultsCombiner *util.DistinctValueCollector[tempopb.TagValue]
}

func (h *tagValuesV2ResultsHandler) shouldQuit() bool {
	return h.resultsCombiner.Exceeded()
}

func (h *tagValuesV2ResultsHandler) addResponse(r io.ReadCloser) error {
	results := &tempopb.SearchTagValuesV2Response{}
	err := (&jsonpb.Unmarshaler{AllowUnknownFields: true}).Unmarshal(r, results)
	if err != nil {
		return err
	}

	for _, t := range results.TagValues {
		h.resultsCombiner.Collect(*t)
	}

	return nil
}

func (h *tagValuesV2ResultsHandler) marshalResult() (string, error) {
	m := &jsonpb.Marshaler{}

	var resp []*tempopb.TagValue
	for _, v := range h.resultsCombiner.Values() {
		tgv := v
		resp = append(resp, &tgv)
	}

	bodyString, err := m.MarshalToString(&tempopb.SearchTagValuesV2Response{
		TagValues: resp,
	})
	if err != nil {
		return "", err
	}

	return bodyString, nil
}

func tagValuesV2ResultHandlerFactory(limit int) tagResultsHandler {
	return &tagValuesV2ResultsHandler{
		limit:           limit,
		resultsCombiner: util.NewDistinctValueCollector[tempopb.TagValue](limit, func(v tempopb.TagValue) int { return len(v.Type) + len(v.Value) }),
	}
}
