package util

import (
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
	"github.com/grafana/tempo/pkg/tempopb"
)

func QueryTrace(baseURL, id string) (*tempopb.Trace, error) {
	resp, err := http.Get(baseURL + "/api/traces/" + id)
	if err != nil {
		return nil, fmt.Errorf("error querying tempo %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			glog.Error("error closing body ", err)
		}
	}()

	trace := &tempopb.Trace{}
	unmarshaller := &jsonpb.Unmarshaler{}
	err = unmarshaller.Unmarshal(resp.Body, trace)
	if err != nil {
		return nil, fmt.Errorf("error decoding trace json, err: %v, traceID: %s", err, id)
	}

	return trace, nil
}
