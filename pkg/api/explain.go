package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/grafana/tempo/pkg/traceql"
)

// ExplainResponse is the JSON response for GET /api/explain.
type ExplainResponse struct {
	Plan *ExplainNode `json:"plan"`
}

// ExplainNode is a single node in the serialized logical plan tree.
type ExplainNode struct {
	Name     string         `json:"name"`
	Detail   string         `json:"detail"`
	Children []*ExplainNode `json:"children,omitempty"`
}

// ParseExplainRequest extracts the explain query params from an HTTP request.
// Returns query string, start (unix seconds), end (unix seconds), and any error.
func ParseExplainRequest(r *http.Request) (string, int64, int64, error) {
	q := r.URL.Query().Get(urlParamQuery)
	if q == "" {
		return "", 0, 0, fmt.Errorf("q parameter is required")
	}

	var start, end int64
	if s := r.URL.Query().Get(urlParamStart); s != "" {
		var err error
		start, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			return "", 0, 0, fmt.Errorf("invalid start parameter: %w", err)
		}
	}
	if e := r.URL.Query().Get(urlParamEnd); e != "" {
		var err error
		end, err = strconv.ParseInt(e, 10, 64)
		if err != nil {
			return "", 0, 0, fmt.Errorf("invalid end parameter: %w", err)
		}
	}

	return q, start, end, nil
}

// PlanNodeToExplainNode converts a traceql.PlanNode tree into a serializable ExplainNode tree.
func PlanNodeToExplainNode(n traceql.PlanNode) *ExplainNode {
	if n == nil {
		return nil
	}
	node := &ExplainNode{
		Name:   planNodeTypeName(n),
		Detail: n.String(),
	}
	for _, child := range n.Children() {
		node.Children = append(node.Children, PlanNodeToExplainNode(child))
	}
	return node
}

// planNodeTypeName returns the short Go type name of a PlanNode, e.g. "RateNode".
func planNodeTypeName(n traceql.PlanNode) string {
	t := fmt.Sprintf("%T", n)
	// Strip package prefix: "*traceql.RateNode" -> "RateNode"
	if i := strings.LastIndex(t, "."); i >= 0 {
		t = t[i+1:]
	}
	return strings.TrimPrefix(t, "*")
}
