package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/grafana/loki/pkg/logproto"
)

var (
	prometheusListenAddress string
	prometheusPath          string

	lokiBaseURL string
	lokiQuery   string
	lokiUser    string
	lokiPass    string
)

func init() {
	flag.StringVar(&prometheusPath, "prometheus-path", "/metrics", "The path to publish Prometheus metrics to.")
	flag.StringVar(&prometheusListenAddress, "prometheus-listen-address", ":80", "The address to listen on for Prometheus scrapes.")

	flag.StringVar(&lokiBaseURL, "loki-base-url", "", "The path to publish Prometheus metrics to.")
	flag.StringVar(&lokiQuery, "loki-query", "", "The address to listen on for Prometheus scrapes.")
	flag.StringVar(&lokiUser, "loki-user", "", "The address to listen on for Prometheus scrapes.")
	flag.StringVar(&lokiPass, "loki-pass", "", "The address to listen on for Prometheus scrapes.")
}

func main() {
	flag.Parse()

	glog.Error("Application Starting")

	// query loki for trace ids
	lines := queryLoki(lokiBaseURL, lokiQuery, lokiUser, lokiPass)
	ids := extractTraceIDs(lines)

	for _, id := range ids {
		fmt.Println(id)
	}

	// query tempo for trace ids

	// do they exist

	// http.Handle(prometheusPath, promhttp.Handler())
	// log.Fatal(http.ListenAndServe(prometheusListenAddress, nil))
}

func queryLoki(baseURL string, query string, user string, pass string) []string {
	start := time.Now().Add(-time.Hour)
	end := time.Now()
	url := baseURL + fmt.Sprintf("/api/prom/query?limit=1000&start=%d&end=%d&query=%s", start.UnixNano(), end.UnixNano(), url.QueryEscape(query))

	glog.Error("url ", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		glog.Error("error building request ", err)
		return nil
	}
	req.SetBasicAuth(user, pass)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		glog.Error("error querying ", err)
		return nil
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			glog.Error("error closing body ", err)
		}
	}()

	if resp.StatusCode/100 != 2 {
		buf, _ := ioutil.ReadAll(resp.Body)
		glog.Error("error response from server: ", string(buf), err)
		return nil
	}
	var decoded logproto.QueryResponse
	err = json.NewDecoder(resp.Body).Decode(&decoded)
	if err != nil {
		glog.Error("error decoding response", err)
		return nil
	}

	lines := make([]string, 0)
	for _, stream := range decoded.Streams {
		for _, entry := range stream.Entries {
			lines = append(lines, entry.Line)
		}
	}

	return lines
}

func extractTraceIDs(lines []string) []string {
	regex := regexp.MustCompile("traceID=(.*?) ")
	ids := make([]string, 0, len(lines))

	for _, l := range lines {
		match := regex.FindString(l)

		if match != "" {
			ids = append(ids, strings.TrimPrefix(match, "traceID="))
		}
	}

	return ids
}
