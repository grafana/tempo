package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"

	"github.com/grafana/frigg/pkg/friggpb"
)

var (
	host        string
	findTraceID string
)

func init() {
	flag.StringVar(&host, "host", "localhost:3100", "frigg host to connect to")
	flag.StringVar(&findTraceID, "find-trace", "", "finds a trace by id.  expected to be a hex value")
}

func main() {
	flag.Parse()

	if len(host) == 0 {
		fmt.Println("-host is required")
		return
	}

	if len(findTraceID) == 0 {
		fmt.Println("One of -find-trace, ... is required")
		return
	}

	var err error
	if len(findTraceID) > 0 {
		err = findTraceByID(host, findTraceID)
	}

	if err != nil {
		fmt.Printf("%v", err)
	}
}

func findTraceByID(host string, id string) error {
	resp, err := http.Get("http://" + host + "/api/traces/" + id)
	if err != nil {
		return err
	}

	out := &friggpb.Trace{}
	err = json.NewDecoder(resp.Body).Decode(out)
	if err != nil {
		return err
	}
	fmt.Printf("%+v", out)

	return nil
}
