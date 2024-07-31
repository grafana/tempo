package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gogo/protobuf/jsonpb"
	serverless "github.com/grafana/tempo/v2/cmd/tempo-serverless"
)

func main() {
	log.Print("starting server...")
	http.HandleFunc("/", handler)

	// Determine port for HTTP service.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}

	// Start HTTP server.
	log.Printf("listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
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
