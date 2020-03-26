package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/grafana/loki/pkg/logproto"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	prometheusListenAddress string
	prometheusPath          string

	lokiBaseURL string
	lokiQuery   string
	lokiUser    string
	lokiPass    string

	tempoBaseURL         string
	tempoBackoffDuration time.Duration
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
	flag.DurationVar(&tempoBackoffDuration, "tempo-backoff-duration", time.Second, "The amount of time to pause between tempo calls")
}

func main() {
	flag.Parse()

	glog.Error("Application Starting")

	testDurations := []time.Duration{
		24 * time.Hour,
		12 * time.Hour,
		6 * time.Hour,
		3 * time.Hour,
		time.Hour,
		30 * time.Minute,
	}

	ticker := time.NewTicker(15 * time.Second)
	go func() {
		for {
			<-ticker.C

			for _, duration := range testDurations {

				// query loki for trace ids
				lines, err := queryLoki(lokiBaseURL, lokiQuery, duration, lokiUser, lokiPass)
				if err != nil {
					glog.Error("error querying Loki ", err)
					metricErrorTotal.Inc()
					continue
				}
				ids := extractTraceIDs(lines)

				// query tempo for trace ids
				metrics, err := queryTempoAndAnalyze(tempoBaseURL, tempoBackoffDuration, ids)
				if err != nil {
					glog.Error("error querying Tempo ", err)
					metricErrorTotal.Inc()
					continue
				}

				metricTracesInspected.WithLabelValues(strconv.Itoa(int(duration.Seconds()))).Add(float64(metrics.requested))
				metricTracesErrors.WithLabelValues("notfound", strconv.Itoa(int(duration.Seconds()))).Add(float64(metrics.notfound))
				metricTracesErrors.WithLabelValues("missingspans", strconv.Itoa(int(duration.Seconds()))).Add(float64(metrics.missingSpans))
			}
		}
	}()

	http.Handle(prometheusPath, promhttp.Handler())
	log.Fatal(http.ListenAndServe(prometheusListenAddress, nil))
}

func queryTempoAndAnalyze(baseURL string, backoff time.Duration, traceIDs []string) (*traceMetrics, error) {
	tm := &traceMetrics{
		requested: len(traceIDs),
	}

	for _, id := range traceIDs {
		time.Sleep(backoff)

		glog.Error("tempo url ", baseURL+"/api/traces/"+id)
		resp, err := http.Get(baseURL + "/api/traces/" + id)
		if err != nil {
			return nil, fmt.Errorf("error querying tempo ", err)
		}

		trace := &tempopb.Trace{}
		err = json.NewDecoder(resp.Body).Decode(trace)
		if err != nil {
			return nil, fmt.Errorf("error decoding trace json ", err)
		}

		if len(trace.Batches) == 0 {
			tm.notfound++
			continue
		}

		// iterate through
		if hasMissingSpans(trace) {
			tm.missingSpans++
		}
	}

	return tm, nil
}

func hasMissingSpans(t *tempopb.Trace) bool {
	// collect all parent span IDs
	linkedSpanIDs := make([][]byte, 0)

	for _, b := range t.Batches {
		for _, s := range b.Spans {
			for _, l := range s.Links {
				linkedSpanIDs = append(linkedSpanIDs, l.SpanId)
			}
		}
	}

	for _, id := range linkedSpanIDs {
		found := false

	B:
		for _, b := range t.Batches {
			for _, s := range b.Spans {
				if bytes.Equal(s.SpanId, id) {
					found = true
					break B
				}
			}
		}

		if !found {
			return true
		}
	}

	return false
}

func queryLoki(baseURL string, query string, durationAgo time.Duration, user string, pass string) ([]string, error) {
	start := time.Now().Add(-durationAgo)
	end := start.Add(30 * time.Minute)
	url := baseURL + fmt.Sprintf("/api/prom/query?limit=10&start=%d&end=%d&query=%s", start.UnixNano(), end.UnixNano(), url.QueryEscape(query))

	glog.Error("loki url ", url)

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
			traceID := strings.TrimSpace(strings.TrimPrefix(match, "traceID="))
			if len(traceID)%2 == 1 {
				traceID = "0" + traceID
			}
			ids = append(ids, traceID)
		}
	}

	return ids
}
