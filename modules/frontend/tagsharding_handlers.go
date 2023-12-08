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

func (r *tagsSearchRequest) adjustRange(start, end uint32) tagSearchReq {
	newReq := r.request
	newReq.Start = start
	newReq.End = end

	return &tagsSearchRequest{
		request: newReq,
	}
}

func (r *tagsSearchRequest) buildSearchTagRequest(subR *http.Request) (*http.Request, error) {
	return api.BuildSearchTagRequest(subR, &r.request)
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

func (r *tagValueSearchRequest) adjustRange(start, end uint32) tagSearchReq {
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

// TagsResultsHandler handled all request/response payloads for querier and for return to frontend for tags.
type TagsResultsHandler struct {
	limit           int
	resultsCombiner *util.DistinctStringCollector
}

func (h *TagsResultsHandler) parseRequest(r *http.Request) (tagSearchReq, error) {
	searchReq, err := api.ParseSearchTagsRequest(r)
	if err != nil {
		return nil, err
	}
	return &tagsSearchRequest{
		request: *searchReq,
	}, nil
}

func (h *TagsResultsHandler) shouldQuit() bool {
	return h.resultsCombiner.Exceeded()
}

func (h *TagsResultsHandler) addResponse(r io.ReadCloser) error {
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

func (h *TagsResultsHandler) marshalResult() (string, error) {
	m := &jsonpb.Marshaler{}
	bodyString, err := m.MarshalToString(&tempopb.SearchTagsResponse{
		TagNames: h.resultsCombiner.Strings(),
	})
	if err != nil {
		return "", err
	}
	return bodyString, nil
}

func TagsResultHandlerFactory(limit int) tagResultsHandler {
	return &TagsResultsHandler{
		limit:           limit,
		resultsCombiner: util.NewDistinctStringCollector(limit),
	}
}

// TagValuesResultsHandler handled all request/response payloads for querier and for return to frontend for tag values.
type TagValuesResultsHandler struct {
	limit           int
	resultsCombiner *util.DistinctStringCollector
}

func (h *TagValuesResultsHandler) parseRequest(r *http.Request) (tagSearchReq, error) {
	searchReq, err := api.ParseSearchTagValuesRequest(r)
	if err != nil {
		return nil, err
	}
	return &tagValueSearchRequest{
		request: *searchReq,
	}, nil
}

func (h *TagValuesResultsHandler) shouldQuit() bool {
	return h.resultsCombiner.Exceeded()
}

func (h *TagValuesResultsHandler) addResponse(r io.ReadCloser) error {
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

func (h *TagValuesResultsHandler) marshalResult() (string, error) {
	m := &jsonpb.Marshaler{}
	bodyString, err := m.MarshalToString(&tempopb.SearchTagValuesResponse{
		TagValues: h.resultsCombiner.Strings(),
	})
	if err != nil {
		return "", err
	}
	return bodyString, nil
}

func TagValuesResultHandlerFactory(limit int) tagResultsHandler {
	return &TagValuesResultsHandler{
		limit:           limit,
		resultsCombiner: util.NewDistinctStringCollector(limit),
	}
}

// TagsV2ResultsHandler handled all request/response payloads for querier and for return to frontend for tags for v2
type TagsV2ResultsHandler struct {
	limit           int
	resultsCombiner map[string]*util.DistinctStringCollector
}

func (h *TagsV2ResultsHandler) parseRequest(r *http.Request) (tagSearchReq, error) {
	searchReq, err := api.ParseSearchTagsRequest(r)
	if err != nil {
		return nil, err
	}
	return &tagsSearchRequest{
		request: *searchReq,
	}, nil
}

func (h *TagsV2ResultsHandler) shouldQuit() bool {
	for _, combiner := range h.resultsCombiner {
		if combiner.Exceeded() {
			return true
		}
	}
	return false
}

func (h *TagsV2ResultsHandler) addResponse(r io.ReadCloser) error {
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

func (h *TagsV2ResultsHandler) marshalResult() (string, error) {
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

func TagsV2ResultHandlerFactory(limit int) tagResultsHandler {
	return &TagsV2ResultsHandler{
		limit:           limit,
		resultsCombiner: map[string]*util.DistinctStringCollector{},
	}
}

type TagValuesV2ResultsHandler struct {
	limit           int
	resultsCombiner *util.DistinctValueCollector[tempopb.TagValue]
}

func (h *TagValuesV2ResultsHandler) parseRequest(r *http.Request) (tagSearchReq, error) {
	searchReq, err := api.ParseSearchTagValuesRequest(r)
	if err != nil {
		return nil, err
	}
	return &tagValueSearchRequest{
		request: *searchReq,
	}, nil
}

func (h *TagValuesV2ResultsHandler) shouldQuit() bool {
	return h.resultsCombiner.Exceeded()
}

func (h *TagValuesV2ResultsHandler) addResponse(r io.ReadCloser) error {
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

func (h *TagValuesV2ResultsHandler) marshalResult() (string, error) {
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

func TagValuesV2ResultHandlerFactory(limit int) tagResultsHandler {
	return &TagValuesV2ResultsHandler{
		limit:           limit,
		resultsCombiner: util.NewDistinctValueCollector[tempopb.TagValue](limit, func(v tempopb.TagValue) int { return len(v.Type) + len(v.Value) }),
	}
}
