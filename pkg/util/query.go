package util

import (
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
	"github.com/grafana/tempo/pkg/tempopb"
)

const orgIDHeader = "X-Scope-OrgID"

func QueryTrace(baseURL, id, orgID string) (*tempopb.Trace, error) {
	req, err := http.NewRequest("GET", baseURL+"/api/traces/"+id, nil)
	if err != nil {
		return nil, err
	}
	if len(orgID) > 0 {
		req.Header.Set(orgIDHeader, orgID)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error querying tempo %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			glog.Error("error closing body ", err)
		}
	}()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("not found traceID: %s", id)
	}

	trace := &tempopb.Trace{}
	unmarshaller := &jsonpb.Unmarshaler{}
	err = unmarshaller.Unmarshal(resp.Body, trace)
	if err != nil {
		return nil, fmt.Errorf("error decoding trace json, err: %v, traceID: %s", err, id)
	}

	return trace, nil
}
