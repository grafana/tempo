package frontend

/* jpe - make sure all functionality is replicated in the combiner package
import (
	"io"
	"net/http"

	"github.com/golang/protobuf/jsonpb" //nolint:all //deprecated
	"github.com/grafana/tempo/pkg/api"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
)

// tagsResultsHandler handled all request/response payloads for querier and for return to frontend for tags.
type tagsResultsHandler struct {
	limit           int
	resultsCombiner *util.DistinctStringCollector
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
*/
