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
	"github.com/grafana/tempo/pkg/util"
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
	tempoOrgID           string
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
	flag.StringVar(&tempoOrgID, "tempo-org-id", "", "The orgID to query in Tempo")
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
		trace, err := util.QueryTrace(baseURL, id, tempoOrgID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				glog.Error("trace not found ", id)
				tm.notfound++
				continue
			}
			return nil, err
		}

		if len(trace.Batches) == 0 {
			glog.Error("trace not found", id)
			tm.notfound++
			continue
		}

		// iterate through
		if hasMissingSpans(trace) {
			glog.Error("has missing spans", id)
			tm.missingSpans++
		}
	}

	return tm, nil
}

func hasMissingSpans(t *tempopb.Trace) bool {
	// collect all parent span IDs
	linkedSpanIDs := make([][]byte, 0)

	for _, b := range t.Batches {
		for _, ils := range b.InstrumentationLibrarySpans {
			for _, s := range ils.Spans {
				if len(s.ParentSpanId) > 0 {
					linkedSpanIDs = append(linkedSpanIDs, s.ParentSpanId)
				}
			}
		}
	}

	for _, id := range linkedSpanIDs {
		found := false

	B:
		for _, b := range t.Batches {
			for _, ils := range b.InstrumentationLibrarySpans {
				for _, s := range ils.Spans {
					if bytes.Equal(s.SpanId, id) {
						found = true
						break B
					}
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
	start := time.Now().Add(-durationAgo).Add(-30 * time.Second) // offsetting 30 seconds prevents it from querying logs from now which naturally have a high percentage of errors
	end := start.Add(30 * time.Minute)
	url := baseURL + fmt.Sprintf("/api/prom/query?limit=10&start=%d&end=%d&query=%s", start.UnixNano(), end.UnixNano(), url.QueryEscape(query))

	glog.Error("loki url ", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error building request %v", err)
	}
	req.SetBasicAuth(user, pass)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error querying %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			glog.Error("error closing body ", err)
		}
	}()

	if resp.StatusCode/100 != 2 {
		buf, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("error response from server: %s %v", string(buf), err)
	}
	var decoded logproto.QueryResponse
	err = json.NewDecoder(resp.Body).Decode(&decoded)
	if err != nil {
		return nil, fmt.Errorf("error decoding response %v", err)
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
