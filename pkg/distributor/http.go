package distributor

import (
	"encoding/json"
	"math"
	"net/http"

	"github.com/weaveworks/common/httpgrpc"

	"github.com/cortexproject/cortex/pkg/util"

	"github.com/grafana/frigg/pkg/friggpb"
)

var contentType = http.CanonicalHeaderKey("Content-Type")

const applicationJSON = "application/json"

// PushHandler reads a snappy-compressed proto from the HTTP body.
func (d *Distributor) PushHandler(w http.ResponseWriter, r *http.Request) {
	var req friggpb.PushRequest

	switch r.Header.Get(contentType) {
	case applicationJSON:
		err := json.NewDecoder(r.Body).Decode(&req)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

	default:
		if _, err := util.ParseProtoReader(r.Context(), r.Body, int(r.ContentLength), math.MaxInt32, &req, util.RawSnappy); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	_, err := d.Push(r.Context(), &req)
	if err == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	resp, ok := httpgrpc.HTTPResponseFromError(err)
	if ok {
		http.Error(w, string(resp.Body), int(resp.Code))
	} else {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
