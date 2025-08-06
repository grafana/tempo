package api

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/google/uuid"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	configQueryRange                         = "config-query-range.yaml"
	configQueryRangeMaxSeries                = "config-query-range-max-series.yaml"
	configQueryRangeMaxSeriesDisabled        = "config-query-range-max-series-disabled.yaml"
	configQueryRangeMaxSeriesDisabledQuerier = "config-query-range-max-series-disabled-querier.yaml"
)

type queryRangeRequest struct {
	Query     string    `json:"query"`
	Start     time.Time `json:"start"`     // default: now - 5m
	End       time.Time `json:"end"`       // default: now + 1m
	Step      string    `json:"step"`      // default: 5s
	Exemplars int       `json:"exemplars"` // default: 100
	noDefault bool      `json:"-"`         // if true, SetDefaults() will not set defaults`
}

func (r *queryRangeRequest) SetDefaults() {
	if r.noDefault {
		return
	}
	if r.Start.IsZero() {
		r.Start = time.Now().Add(-5 * time.Minute)
	}
	if r.End.IsZero() {
		r.End = time.Now().Add(time.Minute)
	}
	if r.Step == "" {
		r.Step = "5s"
	}
	if r.Exemplars == 0 {
		r.Exemplars = 100
	}
}

// Set debugMode to true to print the response body
var debugMode = false

func TestQueryRangeExemplars(t *testing.T) {
	t.Parallel()

	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configQueryRange, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	jaegerClient, err := util.NewJaegerToOTLPExporter(tempo.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, jaegerClient)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	// send one batch every 500ms for 10 seconds
sendLoop:
	for {
		select {
		case <-ticker.C:
			require.NoError(t, jaegerClient.EmitBatch(context.Background(),
				util.MakeThriftBatchWithSpanCountAttributeAndName(
					1, "my operation",
					"res_val", "span_val",
					"res_attr", "span_attr",
				),
			))
			require.NoError(t, jaegerClient.EmitBatch(context.Background(),
				util.MakeThriftBatchWithSpanCountAttributeAndName(
					1, "my operation",
					"res_val2", "span_val2",
					"res_attr", "span_attr",
				),
			))
			require.NoError(t, jaegerClient.EmitBatch(context.Background(),
				util.MakeThriftBatchWithSpanCountAttributeAndName(
					1, "operation with high cardinality",
					uuid.New().String(), uuid.New().String(),
					"res_high_cardinality", "span_high_cardinality",
				),
			))
		case <-timer.C:
			break sendLoop
		}
	}

	// Wait for traces to be flushed to blocks
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempo_metrics_generator_processor_local_blocks_spans_total"}, e2e.WaitMissingMetrics))
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempo_metrics_generator_processor_local_blocks_cut_blocks"}, e2e.WaitMissingMetrics))

	for _, exeplarsCase := range []struct {
		name              string
		exemplars         int
		expectedExemplars int
	}{
		{
			name:              "default",
			exemplars:         0,
			expectedExemplars: 100, // if set to 0, then limits to 100
		},
		{
			name:              "5 exemplar",
			exemplars:         5,
			expectedExemplars: 5,
		},
		{
			name:              "25 exemplars",
			exemplars:         25,
			expectedExemplars: 25,
		},
		{
			name:              "capped exemplars",
			exemplars:         1000,
			expectedExemplars: 100, // capped to 100
		},
	} {
		for _, query := range []string{
			"{} | rate()",
			"{} | compare({status=error})",
			"{} | count_over_time()",
			"{} | min_over_time(duration)",
			"{} | max_over_time(duration)",
			"{} | avg_over_time(duration)",
			"{} | sum_over_time(duration)",
			"{} | quantile_over_time(duration, .5)",
			"{} | quantile_over_time(duration, .5, 0.9, 0.99)",

			"{} | count_over_time() by (span.span_attr)",
			"{} | count_over_time() by (resource.res_attr)",
			"{} | count_over_time() by (.span_attr)",
			"{} | count_over_time() by (.res_attr)",

			"{} | histogram_over_time(duration)",
			"{} | count_over_time() by (status)",
			"{status != error} | count_over_time() by (status)",
		} {
			t.Run(fmt.Sprintf("%s: %s", exeplarsCase.name, query), func(t *testing.T) {
				req := queryRangeRequest{
					Query:     query,
					Exemplars: exeplarsCase.exemplars,
				}
				queryRangeRes := callQueryRange(t, tempo.Endpoint(tempoPort), req)
				require.NotNil(t, queryRangeRes)
				require.GreaterOrEqual(t, len(queryRangeRes.GetSeries()), 1)
				if query == "{} | quantile_over_time(duration, .5, 0.9, 0.99)" {
					// Bug: https://github.com/grafana/tempo/issues/5167
					t.Skip("Bug in quantile_over_time in calculating exemplars")
				}

				exemplarCount := 0

				for _, series := range queryRangeRes.GetSeries() {
					exemplarCount += len(series.GetExemplars())
				}
				assert.LessOrEqual(t, exemplarCount, exeplarsCase.expectedExemplars)
				assert.GreaterOrEqual(t, exemplarCount, 1)
			})
		}
	}

	// check exemplars in more detail
	for _, testCase := range []struct {
		query                   string
		targetAttribute         string
		targetExemplarAttribute string
	}{
		{
			query:                   "{} | quantile_over_time(duration, .9) by (span.span_attr)",
			targetAttribute:         "span.span_attr",
			targetExemplarAttribute: "span.span_attr",
		},
		{
			query:                   "{} | quantile_over_time(duration, .9) by (resource.res_attr)",
			targetAttribute:         "resource.res_attr",
			targetExemplarAttribute: "resource.res_attr",
		},
		{
			query:                   "{} | quantile_over_time(duration, .9) by (.span_attr)",
			targetAttribute:         ".span_attr",
			targetExemplarAttribute: "span.span_attr",
		},
		{
			query:                   "{} | quantile_over_time(duration, .9) by (.res_attr)",
			targetAttribute:         ".res_attr",
			targetExemplarAttribute: "resource.res_attr",
		},
		{
			query:                   "{} | rate() by (span.span_attr)",
			targetAttribute:         "span.span_attr",
			targetExemplarAttribute: "span.span_attr",
		},
		{
			query:                   "{} | count_over_time() by (span.span_attr)",
			targetAttribute:         "span.span_attr",
			targetExemplarAttribute: "span.span_attr",
		},
		{
			query:                   "{} | min_over_time(duration) by (span.span_attr)",
			targetAttribute:         "span.span_attr",
			targetExemplarAttribute: "span.span_attr",
		},
		{
			query:                   "{} | max_over_time(duration) by (span.span_attr)",
			targetAttribute:         "span.span_attr",
			targetExemplarAttribute: "span.span_attr",
		},
		{
			query:                   "{} | avg_over_time(duration) by (span.span_attr)",
			targetAttribute:         "span.span_attr",
			targetExemplarAttribute: "span.span_attr",
		},
		{
			query:                   "{} | sum_over_time(duration) by (span.span_attr)",
			targetAttribute:         "span.span_attr",
			targetExemplarAttribute: "span.span_attr",
		},
	} {
		t.Run(testCase.query, func(t *testing.T) {
			req := queryRangeRequest{
				Query:     testCase.query,
				Exemplars: 100,
			}
			queryRangeRes := callQueryRange(t, tempo.Endpoint(tempoPort), req)
			require.NotNil(t, queryRangeRes)
			require.Equal(t, len(queryRangeRes.GetSeries()), 3) // value 1, value 2 and nil (high cardinality's span has no such attribute)

			// Verify that all exemplars in this series belongs to the right series
			// by matching attribute values
			var skippedForNilAttr bool
			for _, series := range queryRangeRes.Series {
				// search attribute value for the series
				var expectedAttrValue string
				for _, label := range series.Labels {
					if label.Key == testCase.targetAttribute {
						expectedAttrValue = label.Value.GetStringValue()
						break
					}
				}
				if (expectedAttrValue == "" || expectedAttrValue == "nil") && !skippedForNilAttr { // one attribute is empty, so we skip it
					skippedForNilAttr = true
					continue
				}
				require.NotEmpty(t, expectedAttrValue)

				// check attribute value in exemplars
				for _, exemplar := range series.Exemplars {
					var actualAttrValue string
					for _, label := range exemplar.Labels {
						if label.Key == testCase.targetExemplarAttribute {
							actualAttrValue = label.Value.GetStringValue()
							break
						}
					}
					require.Equal(t, expectedAttrValue, actualAttrValue)
				}
			}
		})
	}

	// invalid query
	for name, req := range map[string]queryRangeRequest{
		"invalid query": {
			Query: "{. a}",
		},
		"step=0 (default step)": {
			Query: "{} | count_over_time()",
			Step:  "0",
		},
		"step=0s (default step)": {
			Query: "{} | count_over_time()",
			Step:  "0s",
		},
	} {
		t.Run(name, func(t *testing.T) {
			req := req
			req.SetDefaults()
			res := doRequest(t, tempo.Endpoint(tempoPort), "api/metrics/query_range", req)
			require.Equal(t, 400, res.StatusCode)
		})
	}

	// small step
	t.Run("small step", func(t *testing.T) {
		req := queryRangeRequest{Query: "{} | count_over_time()"}
		req.SetDefaults()
		req.Step = "35ms"
		res := doRequest(t, tempo.Endpoint(tempoPort), "api/metrics/query_range", req)
		require.Equal(t, 400, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, string(body), "step of 35ms is too small, minimum step for given range is 36ms")
	})

	// query with empty results
	for _, query := range []string{
		// existing attribute, no traces
		"{status=error} | count_over_time()",
		// non-existing attribute, no traces
		`{span.randomattr = "doesnotexist"} | count_over_time()`,
	} {
		t.Run(query, func(t *testing.T) {
			queryRangeRes := callQueryRange(t, tempo.Endpoint(tempoPort), queryRangeRequest{Query: query, Exemplars: 100})
			require.NotNil(t, queryRangeRes)
			// it has time series but they are empty and has no exemplars
			require.GreaterOrEqual(t, len(queryRangeRes.GetSeries()), 1)
			exemplarCount := 0
			for _, series := range queryRangeRes.GetSeries() {
				exemplarCount += len(series.GetExemplars())
			}
			require.Equal(t, 0, exemplarCount)
		})
	}

	for _, testCase := range []struct {
		name      string
		query     string
		step      string
		converter func([]tempopb.Sample) float64
	}{
		{
			name:      "count_over_time",
			query:     "{} | count_over_time()",
			step:      "1s",
			converter: sumSamples,
		},
		{
			name:      "sum_over_time",
			query:     "{} | sum_over_time(duration)",
			step:      "1s",
			converter: sumSamples,
		},
		{
			name:      "max_over_time",
			query:     "{} | max_over_time(duration)",
			step:      "1s",
			converter: maxSamples,
		},
		{
			name:      "min_over_time",
			query:     "{} | min_over_time(duration)",
			step:      "1s",
			converter: minSamples,
		},
		{
			name:      "1m step",
			query:     "{} | count_over_time()",
			step:      "1m",
			converter: sumSamples,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			req := queryRangeRequest{
				Query: testCase.query,
				// Query range truncates the start and end to the step, while instant query does not.
				// We need to truncate the start and end to the step to align the interval for query range and instant query.
				Start: time.Now().Add(-5 * time.Minute).Truncate(time.Minute),
				End:   time.Now().Add(time.Minute).Truncate(time.Minute),
				Step:  testCase.step,
			}

			queryRangeRes := callQueryRange(t, tempo.Endpoint(tempoPort), req)
			require.NotNil(t, queryRangeRes)
			require.Equal(t, 1, len(queryRangeRes.GetSeries()))

			expectedValue := testCase.converter(queryRangeRes.Series[0].Samples)

			instantQueryRes := callInstantQuery(t, tempo.Endpoint(tempoPort), req)
			require.NotNil(t, instantQueryRes)
			require.Equal(t, 1, len(instantQueryRes.GetSeries()))
			require.InDelta(t, expectedValue, instantQueryRes.GetSeries()[0].Value, 0.000001)
		})
	}

	for _, testCase := range []struct {
		name        string
		query       string
		expectedNum int
	}{
		{
			name:        "top 1 by span attribute",
			query:       "{ } | rate() by (span.span_high_cardinality) | topk(1)",
			expectedNum: 1,
		},
		{
			name:        "top 10 by span attribute",
			query:       "{ } | rate() by (span.span_high_cardinality) | topk(10)",
			expectedNum: 10,
		},
		{
			name:        "top 2 by resource attribute",
			query:       "{ } | rate() by (resource.res_high_cardinality) | topk(2)",
			expectedNum: 2,
		},
		{
			name:        "bottom 1 by resource attribute",
			query:       "{ } | rate() by (resource.res_high_cardinality) | bottomk(1)",
			expectedNum: 1,
		},
		{
			name:        "bootom 10 by resource attribute",
			query:       "{ } | rate() by (resource.res_high_cardinality) | bottomk(10)",
			expectedNum: 10,
		},
		{
			name:        "bottom 2 by span attribute",
			query:       "{ } | rate() by (span.span_high_cardinality) | bottomk(2)",
			expectedNum: 2,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			req := queryRangeRequest{
				Query: testCase.query,
				Start: time.Now().Add(-5 * time.Minute),
				End:   time.Now().Add(time.Minute),
				Step:  "1m",
			}

			instantQueryRes := callInstantQuery(t, tempo.Endpoint(tempoPort), req)
			require.NotNil(t, instantQueryRes)
			require.Equal(t, testCase.expectedNum, len(instantQueryRes.GetSeries()))
		})
	}

	for _, testCase := range []struct {
		name              string
		end               time.Time
		step              string
		expectedIntervals int
	}{
		// |---start|---|---end|
		{name: "aligned", end: time.Now().Truncate(time.Minute), step: "1m", expectedIntervals: 3},
		// |---|---start---|---|---end---|
		{name: "unaligned", end: time.Now(), step: "1m", expectedIntervals: 4},
		{name: "default step", end: time.Now(), step: "", expectedIntervals: 122},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			req := queryRangeRequest{
				Query:     "{} | count_over_time()",
				Start:     testCase.end.Add(-2 * time.Minute),
				End:       testCase.end,
				Step:      testCase.step,
				noDefault: true,
			}

			queryRangeRes := callQueryRange(t, tempo.Endpoint(tempoPort), req)
			require.NotNil(t, queryRangeRes)
			series := queryRangeRes.GetSeries()
			require.Equal(t, 1, len(series), "Expected 1 series for count_over_time query")
			require.Equal(t, testCase.expectedIntervals, len(series[0].Samples))
		})
	}
}

func sumSamples(samples []tempopb.Sample) float64 {
	var sum float64
	for _, sample := range samples {
		sum += sample.Value
	}
	return sum
}

func maxSamples(samples []tempopb.Sample) float64 {
	maxValue := math.Inf(-1)
	for _, sample := range samples {
		if sample.Value > maxValue {
			maxValue = sample.Value
		}
	}
	return maxValue
}

func minSamples(samples []tempopb.Sample) float64 {
	minValue := math.Inf(1)
	for _, sample := range samples {
		if sample.Value < minValue {
			minValue = sample.Value
		}
	}
	return minValue
}

// TestQueryRangeSingleTrace checks count for a single trace
// Single trace creates a block with startTime == endTime
// which covers a few edge cases under the hood.
func TestQueryRangeSingleTrace(t *testing.T) {
	t.Parallel()

	s, err := e2e.NewScenario("tempo_e2e_single_trace")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configQueryRange, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	jaegerClient, err := util.NewJaegerToOTLPExporter(tempo.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, jaegerClient)

	// Emit a single trace
	require.NoError(t, jaegerClient.EmitBatch(context.Background(), util.MakeThriftBatch()))

	// Wait for traces to be flushed to blocks
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempo_metrics_generator_processor_local_blocks_spans_total"}, e2e.WaitMissingMetrics))
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempo_metrics_generator_processor_local_blocks_cut_blocks"}, e2e.WaitMissingMetrics))

	// Query the trace by count. As we have only one trace, we should get one dot with value 1
	query := "{} | count_over_time()"
	queryRangeRes := callQueryRange(t, tempo.Endpoint(tempoPort), queryRangeRequest{Query: query, Exemplars: 100})
	require.NotNil(t, queryRangeRes)
	require.Equal(t, len(queryRangeRes.GetSeries()), 1)

	series := queryRangeRes.GetSeries()[0]
	assert.Equal(t, len(series.GetExemplars()), 1)

	var sum float64
	for _, sample := range series.GetSamples() {
		sum += sample.Value
	}
	require.InDelta(t, sum, 1, 0.000001)
}

func TestQueryRangeMaxSeries(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configQueryRangeMaxSeries, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	jaegerClient, err := util.NewJaegerToOTLPExporter(tempo.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, jaegerClient)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
sendLoop:
	for {
		select {
		case <-ticker.C:
			require.NoError(t, jaegerClient.EmitBatch(context.Background(), util.MakeThriftBatch()))
		case <-timer.C:
			break sendLoop
		}
	}

	// Wait for traces to be flushed to blocks
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempo_metrics_generator_processor_local_blocks_spans_total"}, e2e.WaitMissingMetrics))
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempo_metrics_generator_processor_local_blocks_cut_blocks"}, e2e.WaitMissingMetrics))

	query := "{} | rate() by (span:id)"
	url := fmt.Sprintf(
		"http://%s/api/metrics/query_range?q=%s&start=%d&end=%d&step=%s",
		tempo.Endpoint(3200),
		url.QueryEscape(query),
		time.Now().Add(-5*time.Minute).UnixNano(),
		time.Now().Add(time.Minute).UnixNano(),
		"5s",
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	// Read body and print it
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	fmt.Println(string(body))

	queryRangeRes := &tempopb.QueryRangeResponse{}
	readBody := strings.NewReader(string(body))
	err = new(jsonpb.Unmarshaler).Unmarshal(readBody, queryRangeRes)
	require.NoError(t, err)
	require.NotNil(t, queryRangeRes)

	// max series is 3 so we should get a partial response with 3 series
	require.Equal(t, tempopb.PartialStatus_PARTIAL, queryRangeRes.GetStatus())
	require.Equal(t, "Response exceeds maximum series limit of 3, a partial response is returned. Warning: the accuracy of each individual value is not guaranteed.", queryRangeRes.GetMessage())
	require.Equal(t, 3, len(queryRangeRes.GetSeries()))
}

func TestQueryRangeMaxSeriesDisabled(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configQueryRangeMaxSeriesDisabled, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	jaegerClient, err := util.NewJaegerToOTLPExporter(tempo.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, jaegerClient)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	spanCount := 0
sendLoop:
	for {
		select {
		case <-ticker.C:
			require.NoError(t, jaegerClient.EmitBatch(context.Background(), util.MakeThriftBatch()))
			spanCount++
		case <-timer.C:
			break sendLoop
		}
	}

	// Wait for traces to be flushed to blocks
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempo_metrics_generator_processor_local_blocks_spans_total"}, e2e.WaitMissingMetrics))
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempo_metrics_generator_processor_local_blocks_cut_blocks"}, e2e.WaitMissingMetrics))

	query := "{} | rate() by (span:id)"
	url := fmt.Sprintf(
		"http://%s/api/metrics/query_range?q=%s&start=%d&end=%d&step=%s",
		tempo.Endpoint(3200),
		url.QueryEscape(query),
		time.Now().Add(-5*time.Minute).UnixNano(),
		time.Now().Add(time.Minute).UnixNano(),
		"5s",
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	// Read body and print it
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	fmt.Println(string(body))

	queryRangeRes := &tempopb.QueryRangeResponse{}
	readBody := strings.NewReader(string(body))
	err = new(jsonpb.Unmarshaler).Unmarshal(readBody, queryRangeRes)
	require.NoError(t, err)
	require.NotNil(t, queryRangeRes)

	// max series is disabled so we should get a complete response with all series
	require.Equal(t, tempopb.PartialStatus_COMPLETE, queryRangeRes.GetStatus())
	require.Equal(t, spanCount, len(queryRangeRes.GetSeries()))
}

func TestQueryRangeMaxSeriesDisabledQuerier(t *testing.T) {
	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, util.CopyFileToSharedDir(s, configQueryRangeMaxSeriesDisabledQuerier, "config.yaml"))
	tempo := util.NewTempoAllInOne()
	require.NoError(t, s.StartAndWaitReady(tempo))

	jaegerClient, err := util.NewJaegerToOTLPExporter(tempo.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, jaegerClient)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	spanCount := 0
sendLoop:
	for {
		select {
		case <-ticker.C:
			require.NoError(t, jaegerClient.EmitBatch(context.Background(), util.MakeThriftBatch()))
			spanCount++
		case <-timer.C:
			break sendLoop
		}
	}

	// Wait for traces to be flushed to blocks
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempo_metrics_generator_processor_local_blocks_spans_total"}, e2e.WaitMissingMetrics))
	require.NoError(t, tempo.WaitSumMetricsWithOptions(e2e.GreaterOrEqual(1), []string{"tempo_metrics_generator_processor_local_blocks_cut_blocks"}, e2e.WaitMissingMetrics))

	// Wait for the traces to be written to the WAL
	time.Sleep(time.Second * 3)

	util.CallFlush(t, tempo)
	time.Sleep(blockFlushTimeout)
	util.CallFlush(t, tempo)

	require.NoError(t, tempo.WaitSumMetrics(e2e.GreaterOrEqual(5), "tempo_ingester_blocks_flushed_total"))

	query := "{} | rate() by (span:id)"
	url := fmt.Sprintf(
		"http://%s/api/metrics/query_range?q=%s&start=%d&end=%d&step=%s",
		tempo.Endpoint(3200),
		url.QueryEscape(query),
		time.Now().Add(-5*time.Minute).UnixNano(),
		time.Now().Add(time.Minute).UnixNano(),
		"5s",
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	// Read body and print it
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	fmt.Println(string(body))

	queryRangeRes := &tempopb.QueryRangeResponse{}
	readBody := strings.NewReader(string(body))
	err = new(jsonpb.Unmarshaler).Unmarshal(readBody, queryRangeRes)
	require.NoError(t, err)
	require.NotNil(t, queryRangeRes)

	// max series is disabled so we should get a complete response with all series
	require.Equal(t, tempopb.PartialStatus_COMPLETE, queryRangeRes.GetStatus())
	require.Equal(t, spanCount, len(queryRangeRes.GetSeries()))
}

func callInstantQuery(t *testing.T, endpoint string, req queryRangeRequest) tempopb.QueryInstantResponse {
	req.SetDefaults()
	res := doRequest(t, endpoint, "api/metrics/query", req)
	require.Equal(t, http.StatusOK, res.StatusCode)

	// Read body and print it
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	if debugMode {
		t.Logf("Response body: %s", string(body))
	}

	instantQueryRes := tempopb.QueryInstantResponse{}
	readBody := strings.NewReader(string(body))
	err = new(jsonpb.Unmarshaler).Unmarshal(readBody, &instantQueryRes)
	require.NoError(t, err)
	return instantQueryRes
}

func callQueryRange(t *testing.T, endpoint string, req queryRangeRequest) tempopb.QueryRangeResponse {
	req.SetDefaults()
	res := doRequest(t, endpoint, "api/metrics/query_range", req)
	require.Equal(t, http.StatusOK, res.StatusCode)

	// Read body and print it
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	if debugMode {
		t.Logf("Response body: %s", string(body))
	}

	queryRangeRes := tempopb.QueryRangeResponse{}
	readBody := strings.NewReader(string(body))
	err = new(jsonpb.Unmarshaler).Unmarshal(readBody, &queryRangeRes)
	require.NoError(t, err)
	return queryRangeRes
}

func doRequest(t *testing.T, host, endpoint string, req queryRangeRequest) *http.Response {
	req.Query = fmt.Sprintf("%s with(exemplars=true)", req.Query)
	url := buildURL(host, endpoint, req)
	rawReq, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	res, err := http.DefaultClient.Do(rawReq)
	require.NoError(t, err)
	return res
}

func buildURL(host, endpoint string, req queryRangeRequest) string {
	return fmt.Sprintf(
		"http://%s/%s?query=%s&start=%d&end=%d&step=%s&exemplars=%d",
		host, endpoint,
		url.QueryEscape(req.Query),
		req.Start.UnixNano(),
		req.End.UnixNano(),
		req.Step,
		req.Exemplars,
	)
}
