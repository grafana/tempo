package handler

import (
	"net/http"

	"github.com/gogo/protobuf/jsonpb"
	serverless "github.com/grafana/tempo/cmd/tempo-serverless"

	// required by the goog
	_ "github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	resp, httpErr := serverless.Handler(r)
	if httpErr != nil {
		http.Error(w, httpErr.Err.Error(), httpErr.Status)
	}

	marshaller := &jsonpb.Marshaler{}
	err := marshaller.Marshal(w, resp)
	if httpErr != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
