package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/grafana/loki/pkg/logproto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	prometheusListenAddress string
	prometheusPath          string

	lokiBaseURL string
	lokiQuery   string
	lokiUser    string
	lokiPass    string

	tempoBaseURL string
)

type traceMetrics struct {
	requested    int
	notfound     int
	missingSpans int
}

func init() {
	flag.StringVar(&prometheusPath, "prometheus-path", "/metrics", "The path to publish Prometheus metrics to.")
	flag.StringVar(&prometheusListenAddress, "prometheus-listen-address", ":80", "The address to listen on for Prometheus scrapes.")

	flag.StringVar(&lokiBaseURL, "loki-base-url", "", "The base URL (scheme://hostname) at which to find loki.")
	flag.StringVar(&lokiQuery, "loki-query", "", "The query to use to find traceIDs in Loki.")
	flag.StringVar(&lokiUser, "loki-user", "", "The user to use for Loki basic auth.")
	flag.StringVar(&lokiPass, "loki-pass", "", "The password to use for Loki basic auth.")

	flag.StringVar(&tempoBaseURL, "tempo-base-url", "", "The base URL (scheme://hostname) at which to find tempo.")
}

func main() {
	flag.Parse()

	glog.Error("Application Starting")

	ticker := time.NewTicker(15 * time.Second)
	go func() {
		for {
			// query loki for trace ids
			lines, err := queryLoki(lokiBaseURL, lokiQuery, lokiUser, lokiPass)
			if err != nil {
				glog.Error("error querying Loki ", err)
				metricErrorTotal.Inc()
				continue
			}
			ids := extractTraceIDs(lines)

			// query tempo for trace ids
			metrics, err := queryTempoAndAnalyze(tempoBaseURL, ids)
			if err != nil {
				glog.Error("error querying Tempo ", err)
				metricErrorTotal.Inc()
				continue
			}

			metricTracesInspected.Add(float64(metrics.requested))
			metricTracesErrors.WithLabelValues("notfound").Add(float64(metrics.notfound))
			metricTracesErrors.WithLabelValues("missingspans").Add(float64(metrics.missingSpans))

			<-ticker.C
		}
	}()

	http.Handle(prometheusPath, promhttp.Handler())
	log.Fatal(http.ListenAndServe(prometheusListenAddress, nil))
}

func queryTempoAndAnalyze(baseURL string, traceIDs []string) (traceMetrics, error) {
	return traceMetrics{}, nil
}

func queryLoki(baseURL string, query string, user string, pass string) ([]string, error) {
	start := time.Now().Add(-time.Hour)
	end := time.Now()
	url := baseURL + fmt.Sprintf("/api/prom/query?limit=1000&start=%d&end=%d&query=%s", start.UnixNano(), end.UnixNano(), url.QueryEscape(query))

	glog.Error("url ", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error building request ", err)
	}
	req.SetBasicAuth(user, pass)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error querying ", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			glog.Error("error closing body ", err)
		}
	}()

	if resp.StatusCode/100 != 2 {
		buf, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("error response from server: ", string(buf), err)
	}
	var decoded logproto.QueryResponse
	err = json.NewDecoder(resp.Body).Decode(&decoded)
	if err != nil {
		return nil, fmt.Errorf("error decoding response ", err)
	}

	lines := make([]string, 0)
	for _, stream := range decoded.Streams {
		for _, entry := range stream.Entries {
			lines = append(lines, entry.Line)
		}
	}

	return lines, nil
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
