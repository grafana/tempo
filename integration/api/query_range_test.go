package api

import (
	"errors"
	"fmt"
	"io"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jaegertracing/jaeger-idl/thrift-gen/jaeger"

	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/httpclient"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	configQueryRangeMaxSeries         = "config-query-range-max-series.yaml"
	configQueryRangeMaxSeriesDisabled = "config-query-range-max-series-disabled.yaml"
	configQueryRangeExemplars         = "config-query-range-exemplars.yaml"
	configQueryRangeEndCutoff         = "config-query-range-end-cutoff.yaml"
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

func TestQueryRangeExemplars(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		ConfigOverlay: configQueryRangeExemplars,
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		timer := time.NewTimer(10 * time.Second)
		defer timer.Stop()

		tracesSent := 0

		// send one batch every 500ms for 10 seconds
	sendLoop:
		for {
			select {
			case <-ticker.C:
				require.NoError(t, h.WriteJaegerBatch(
					util.MakeThriftBatchWithSpanCountAttributeAndName(
						1, "my operation",
						"res_val", "span_val",
						"res_attr", "span_attr",
					), ""),
				)
				require.NoError(t, h.WriteJaegerBatch(
					util.MakeThriftBatchWithSpanCountAttributeAndName(
						1, "my operation",
						"res_val2", "span_val2",
						"res_attr", "span_attr",
					), ""),
				)
				require.NoError(t, h.WriteJaegerBatch(
					util.MakeThriftBatchWithSpanCountAttributeAndName(
						1, "operation with high cardinality",
						uuid.New().String(), uuid.New().String(),
						"res_high_cardinality", "span_high_cardinality",
					), ""),
				)
				tracesSent += 3
			case <-timer.C:
				break sendLoop
			}
		}

		h.WaitTracesQueryable(t, tracesSent)

		for _, exemplarsCase := range []struct {
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
				t.Run(fmt.Sprintf("%s: %s", exemplarsCase.name, query), func(t *testing.T) {
					req := queryRangeRequest{
						Query:     query,
						Exemplars: exemplarsCase.exemplars,
					}
					callQueryRange(t, h, req, func(queryRangeRes *tempopb.QueryRangeResponse, err error) {
						require.NoError(t, err)
						require.NotNil(t, queryRangeRes)
						require.GreaterOrEqual(t, len(queryRangeRes.GetSeries()), 1)

						exemplarCount := 0

						for _, series := range queryRangeRes.GetSeries() {
							exemplarCount += len(series.GetExemplars())
						}
						assert.LessOrEqual(t, exemplarCount, exemplarsCase.expectedExemplars)
						assert.GreaterOrEqual(t, exemplarCount, 1)
					})
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
				callQueryRange(t, h, req, func(queryRangeRes *tempopb.QueryRangeResponse, err error) {
					require.NoError(t, err)
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
			})
		}

		// invalid query (both http and grpc)
		for name, req := range map[string]queryRangeRequest{
			"invalid query": {
				Query: "{. a}",
			},
		} {
			t.Run(name, func(t *testing.T) {
				req := req
				callQueryRange(t, h, req, func(_ *tempopb.QueryRangeResponse, err error) {
					require.ErrorContains(t, err, "unexpected END_ATTRIBUTE, expecting IDENTIFIER")
				})
			})
		}

		// invalid query (http only)
		//  grpc accepts 0 step and provides a default
		for name, req := range map[string]queryRangeRequest{
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
				apiClient := h.APIClientHTTP("")
				_, err := apiClient.MetricsQueryRange(req.Query, req.Start.UnixNano(), req.End.UnixNano(), req.Step, req.Exemplars)
				require.ErrorContains(t, err, "response: 400")
			})
		}

		// small step
		t.Run("small step", func(t *testing.T) {
			req := queryRangeRequest{Query: "{} | count_over_time()"}
			req.SetDefaults()
			req.Step = "35ms"

			callQueryRange(t, h, req, func(_ *tempopb.QueryRangeResponse, err error) {
				require.ErrorContains(t, err, "step of 35ms is too small, minimum step for given range is 36ms")
			})
		})

		// query with empty results
		for _, query := range []string{
			// existing attribute, no traces
			"{status=error} | count_over_time()",
			// non-existing attribute, no traces
			`{span.randomattr = "doesnotexist"} | count_over_time()`,
		} {
			t.Run(query, func(t *testing.T) {
				callQueryRange(t, h, queryRangeRequest{Query: query, Exemplars: 100}, func(queryRangeRes *tempopb.QueryRangeResponse, err error) {
					require.NoError(t, err)
					require.NotNil(t, queryRangeRes)
					// it has time series but they are empty and has no exemplars
					require.GreaterOrEqual(t, len(queryRangeRes.GetSeries()), 1)
					exemplarCount := 0
					for _, series := range queryRangeRes.GetSeries() {
						exemplarCount += len(series.GetExemplars())
					}
					require.Equal(t, 0, exemplarCount)
				})
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

				var expectedValue float64
				callQueryRange(t, h, req, func(queryRangeRes *tempopb.QueryRangeResponse, err error) {
					require.NoError(t, err)
					require.NotNil(t, queryRangeRes)
					require.Equal(t, 1, len(queryRangeRes.GetSeries()))

					expectedValue = testCase.converter(queryRangeRes.Series[0].Samples)
				})

				instantQueryRes, err := callInstantQuery(h.APIClientHTTP(""), req)
				require.NoError(t, err)
				require.NotNil(t, instantQueryRes)
				require.Equal(t, 1, len(instantQueryRes.GetSeries()))
				require.InDelta(t, expectedValue, instantQueryRes.GetSeries()[0].Value, 0.000001)
			})
		}

		t.Run("avg_over_time instant query", func(t *testing.T) {
			req := queryRangeRequest{
				Query: "{} | avg_over_time(duration)",
				Start: time.Now().Add(-5 * time.Minute),
				End:   time.Now().Add(time.Minute),
			}

			countReq := req
			countReq.Query = "{} | count_over_time()"
			countRes, err := callInstantQuery(h.APIClientHTTP(""), countReq)
			require.NoError(t, err)
			require.NotNil(t, countRes)
			require.Equal(t, 1, len(countRes.GetSeries()))
			count := countRes.GetSeries()[0].Value

			sumReq := req
			sumReq.Query = "{} | sum_over_time(duration)"
			sumRes, err := callInstantQuery(h.APIClientHTTP(""), sumReq)
			require.NoError(t, err)
			require.NotNil(t, sumRes)
			require.Equal(t, 1, len(sumRes.GetSeries()))
			sum := sumRes.GetSeries()[0].Value

			res, err := callInstantQuery(h.APIClientHTTP(""), req)
			require.NoError(t, err)
			require.NotNil(t, res)
			require.Equal(t, 1, len(res.GetSeries()))
			require.InDelta(t, sum/count, res.GetSeries()[0].Value, 0.000001)
		})

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

				instantQueryRes, err := callInstantQuery(h.APIClientHTTP(""), req)
				require.NoError(t, err)
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

				callQueryRange(t, h, req, func(queryRangeRes *tempopb.QueryRangeResponse, err error) {
					require.NoError(t, err)
					require.NotNil(t, queryRangeRes)
					series := queryRangeRes.GetSeries()
					require.Equal(t, 1, len(series), "Expected 1 series for count_over_time query")
					require.Equal(t, testCase.expectedIntervals, len(series[0].Samples))
				})
			})
		}
	})
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
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		DeploymentMode: util.DeploymentModeSingleBinary, // for unknown reasons this fails on microservices mode. TODO: figure that crap out
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)
		// Emit a single trace
		require.NoError(t, h.WriteJaegerBatch(util.MakeThriftBatch(), ""))

		h.WaitTracesQueryable(t, 1)

		// Query the trace by count. As we have only one trace, we should get one dot with value 1
		query := "{} | count_over_time()"
		callQueryRange(t, h, queryRangeRequest{Query: query, Exemplars: 100}, func(queryRangeRes *tempopb.QueryRangeResponse, err error) {
			require.NoError(t, err)
			require.NotNil(t, queryRangeRes)
			require.Equal(t, len(queryRangeRes.GetSeries()), 1)

			series := queryRangeRes.GetSeries()[0]
			assert.Equal(t, len(series.GetExemplars()), 1)

			var sum float64
			for _, sample := range series.GetSamples() {
				sum += sample.Value
			}
			require.InDelta(t, sum, 1, 0.000001)
		})
	})
}

func TestQueryRangeMaxSeries(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		ConfigOverlay: configQueryRangeMaxSeries,
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()

		tracesSent := 0
	sendLoop:
		for {
			select {
			case <-ticker.C:
				require.NoError(t, h.WriteJaegerBatch(util.MakeThriftBatch(), ""))
				tracesSent++
			case <-timer.C:
				break sendLoop
			}
		}

		require.Greater(t, tracesSent, 3)
		h.WaitTracesQueryable(t, tracesSent)

		callQueryRange(t, h, queryRangeRequest{
			Query:     "{} | rate() by (span:id)",
			Start:     time.Now().Add(-5 * time.Minute),
			End:       time.Now(),
			Step:      "5s",
			Exemplars: 100,
		}, func(queryRangeRes *tempopb.QueryRangeResponse, err error) {
			require.NoError(t, err)
			require.NotNil(t, queryRangeRes)
			require.Equal(t, tempopb.PartialStatus_PARTIAL, queryRangeRes.GetStatus())
			require.Equal(t, "Response exceeds maximum series limit of 3, a partial response is returned. Warning: the accuracy of each individual value is not guaranteed.", queryRangeRes.GetMessage())
			require.Equal(t, 3, len(queryRangeRes.GetSeries()))
		})
	})
}

func TestQueryRangeMaxSeriesDisabled(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		ConfigOverlay: configQueryRangeMaxSeriesDisabled,
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		spanCount := 0
	sendLoop:
		for {
			select {
			case <-ticker.C:
				require.NoError(t, h.WriteJaegerBatch(util.MakeThriftBatch(), ""))
				spanCount++
			case <-timer.C:
				break sendLoop
			}
		}

		// Wait for traces to be flushed to blocks. spanCount happens to make traces count
		h.WaitTracesQueryable(t, spanCount)

		callQueryRange(t, h, queryRangeRequest{
			Query:     "{} | rate() by (span:id)",
			Start:     time.Now().Add(-5 * time.Minute),
			End:       time.Now(),
			Step:      "5s",
			Exemplars: 100,
		}, func(queryRangeRes *tempopb.QueryRangeResponse, err error) {
			require.NoError(t, err)
			require.NotNil(t, queryRangeRes)
			require.Equal(t, tempopb.PartialStatus_COMPLETE, queryRangeRes.GetStatus())
			require.Equal(t, spanCount, len(queryRangeRes.GetSeries()))
		})
	})
}

func TestQueryRangeTypeHandling(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		// Emit spans where the attribute names but the values are
		// string and int with the same textual representation.
		t1 := util.MakeThriftBatch()
		t1.Spans[0].Tags = append(t1.Spans[0].Tags, &jaeger.Tag{
			Key:   "foo",
			VType: jaeger.TagType_STRING,
			VStr:  strptr("123"),
		})
		require.NoError(t, h.WriteJaegerBatch(t1, ""))

		t2 := util.MakeThriftBatch()
		t2.Spans[0].Tags = append(t2.Spans[0].Tags, &jaeger.Tag{
			Key:   "foo",
			VType: jaeger.TagType_LONG,
			VLong: int64ptr(123),
		})
		require.NoError(t, h.WriteJaegerBatch(t2, ""))

		// Wait for traces to be flushed to blocks
		h.WaitTracesQueryable(t, 2)

		callQueryRange(t, h, queryRangeRequest{
			Query:     "{} | rate() by (span:id)",
			Start:     time.Now().Add(-5 * time.Minute),
			End:       time.Now(),
			Step:      "5s",
			Exemplars: 100,
		}, func(queryRangeRes *tempopb.QueryRangeResponse, err error) {
			require.NoError(t, err)
			require.NotNil(t, queryRangeRes)
			require.Equal(t, tempopb.PartialStatus_COMPLETE, queryRangeRes.GetStatus())
			require.Equal(t, 2, len(queryRangeRes.GetSeries()))
		})
	})
}

func TestQueryRangeEndCutoff(t *testing.T) {
	util.RunIntegrationTests(t, util.TestHarnessConfig{
		ConfigOverlay: configQueryRangeEndCutoff,
	}, func(h *util.TempoHarness) {
		h.WaitTracesWritable(t)

		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		timer := time.NewTimer(10 * time.Second)
		defer timer.Stop()

		tracesSent := 0
	sendLoop:
		for {
			select {
			case <-ticker.C:
				require.NoError(t, h.WriteJaegerBatch(util.MakeThriftBatch(), ""))
				tracesSent++
			case <-timer.C:
				break sendLoop
			}
		}

		h.WaitTracesQueryable(t, tracesSent)

		// Query with end=now, which should be adjusted by query_end_cutoff (5s)
		cutoff := 5 * time.Second
		now := time.Now()

		req := queryRangeRequest{
			Query:     "{} | count_over_time()",
			Start:     now.Add(-15 * time.Second),
			End:       now,
			Step:      time.Second.String(),
			Exemplars: 100,
		}

		callQueryRange(t, h, req, func(queryRangeRes *tempopb.QueryRangeResponse, err error) {
			require.NoError(t, err)
			require.NotNil(t, queryRangeRes)
			require.Equal(t, 1, len(queryRangeRes.GetSeries()))

			series := queryRangeRes.GetSeries()[0]
			samples := series.GetSamples()
			require.Greater(t, len(samples), 0, "Expected at least one sample")

			// The cutoff should be applied, so the last sample should be at least 'cutoff' seconds before now
			maxAllowedTimestamp := now.Add(-cutoff).UnixMilli()
			for _, sample := range samples {
				assert.LessOrEqual(t, sample.TimestampMs, maxAllowedTimestamp,
					"Sample timestamp %d is after cutoff %d (diff: %d ms)",
					sample.TimestampMs, maxAllowedTimestamp, sample.TimestampMs-maxAllowedTimestamp)
			}
		})
	})
}

func callInstantQuery(apiClient *httpclient.Client, req queryRangeRequest) (*tempopb.QueryInstantResponse, error) {
	req.SetDefaults()
	return apiClient.MetricsQueryInstant(req.Query, req.Start.UnixNano(), req.End.UnixNano(), req.Exemplars)
}

func callQueryRange(t *testing.T, h *util.TempoHarness, req queryRangeRequest, fn func(*tempopb.QueryRangeResponse, error)) {
	req.SetDefaults()

	apiClient := h.APIClientHTTP("")
	fn(apiClient.MetricsQueryRange(req.Query, req.Start.UnixNano(), req.End.UnixNano(), req.Step, req.Exemplars))

	step := time.Duration(0)
	if req.Step != "" {
		var err error
		step, err = time.ParseDuration(req.Step)
		require.NoError(t, err)
	}

	grpcClient, ctx, err := h.APIClientGRPC("")
	require.NoError(t, err)

	stream, err := grpcClient.MetricsQueryRange(ctx, &tempopb.QueryRangeRequest{
		Query:     req.Query,
		Start:     uint64(req.Start.UnixNano()),
		End:       uint64(req.End.UnixNano()),
		Step:      uint64(step.Nanoseconds()),
		Exemplars: uint32(req.Exemplars),
	})
	require.NoError(t, err)

	finalResponse := &tempopb.QueryRangeResponse{
		Metrics: &tempopb.SearchMetrics{},
	}
	var finalError error
	for {
		t.Logf("recv")
		resp, err := stream.Recv()
		if resp != nil {
			t.Logf("resp: %+v, count: %d", resp, len(resp.GetSeries()))
			naiveQueryRangeCombine(resp, finalResponse)
			t.Logf("resp: %d, finalResponse: %d", len(resp.GetSeries()), len(finalResponse.GetSeries()))
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			finalError = err
			break
		}
	}
	fn(finalResponse, finalError)
}

func strptr(s string) *string {
	return &s
}

func int64ptr(i int64) *int64 {
	return &i
}

// naiveQueryRangeCombine makes assumptions about the data being sent from Tempo. it assumes that labels orders are always
// the same and that samples and exemplars do not need to be deduped.
func naiveQueryRangeCombine(rNew, rInto *tempopb.QueryRangeResponse) {
	rIntoSeries := map[string]*tempopb.TimeSeries{}
	for _, series := range rInto.GetSeries() {
		rIntoSeries[keyFromLabels(series.GetLabels())] = series
	}

	for _, newSeries := range rNew.GetSeries() {
		key := keyFromLabels(newSeries.GetLabels())
		if intoSeries, ok := rIntoSeries[key]; ok {
			intoSeries.Exemplars = append(intoSeries.Exemplars, newSeries.Exemplars...)
			intoSeries.Samples = append(intoSeries.Samples, newSeries.Samples...)
		} else {
			rIntoSeries[key] = newSeries
		}
	}

	// Rebuild the series slice from the map
	rInto.Series = make([]*tempopb.TimeSeries, 0, len(rIntoSeries))
	for _, series := range rIntoSeries {
		rInto.Series = append(rInto.Series, series)
	}

	if rInto.Message == "" {
		rInto.Message = rNew.Message
	}
	if rInto.Status == 0 {
		rInto.Status = rNew.Status
	}

	// metrics?
	rInto.Metrics.CompletedJobs += rNew.Metrics.CompletedJobs
	rInto.Metrics.InspectedBytes += rNew.Metrics.InspectedBytes
	rInto.Metrics.InspectedTraces += rNew.Metrics.InspectedTraces
	rInto.Metrics.InspectedSpans += rNew.Metrics.InspectedSpans
}

func keyFromLabels(labels []v1.KeyValue) string {
	key := ""
	for _, label := range labels {
		key += label.Key + "=" + label.Value.GetStringValue() + ","
	}
	return key
}
