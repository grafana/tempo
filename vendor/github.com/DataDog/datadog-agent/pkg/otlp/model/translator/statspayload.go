// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package translator

import (
	"fmt"
	"strings"

	"github.com/DataDog/sketches-go/ddsketch"
	"github.com/DataDog/sketches-go/ddsketch/mapping"
	"github.com/DataDog/sketches-go/ddsketch/pb/sketchpb"
	"github.com/DataDog/sketches-go/ddsketch/store"
	"github.com/golang/protobuf/proto"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"

	"github.com/DataDog/datadog-agent/pkg/otlp/model/source"
	"github.com/DataDog/datadog-agent/pkg/trace/pb"
)

// keyAPMStats specifies the key name of the resource attribute which identifies resource metrics
// as being an APM Stats Payload. The presence of the key results in them being treated and consumed
// differently by the Translator.
const keyAPMStats = "_dd.apm_stats"

// This group of constants specifies the metric attribute keys used for APM Stats aggregation keys.
const (
	statsKeyHostname         = "dd.hostname"
	statsKeyEnv              = "dd.env"
	statsKeyVersion          = "dd.version"
	statsKeyLang             = "dd.lang"
	statsKeyTracerVersion    = "dd.tracer_version"
	statsKeyRuntimeID        = "dd.runtime_id"
	statsKeySequence         = "dd.sequence"
	statsKeyAgentAggregation = "dd.agent_aggregation"
	statsKeyService          = "dd.service"
	statsKeyContainerID      = "dd.container_id"
	statsKeyTags             = "dd.tags"
	statsKeySynthetics       = "dd.synthetics"
	statsKeySpanName         = "dd.name"
	statsKeySpanResource     = "dd.resource"
	statsKeyHTTPStatusCode   = "dd.http_status_code"
	statsKeySpanType         = "dd.type"
	statsKeySpanDBType       = "dd.db_type"
)

// This group of constants specifies the metric names used to store APM Stats as metrics.
const (
	metricNameHits         = "dd.apm_stats.hits"
	metricNameErrors       = "dd.apm_stats.errors"
	metricNameDuration     = "dd.apm_stats.duration"
	metricNameTopLevelHits = "dd.apm_stats.top_level_hits"
	metricNameOkSummary    = "dd.apm_stats.ok_summary"
	metricNameErrorSummary = "dd.apm_stats.error_summary"
)

// StatsPayloadToMetrics converts an APM Stats Payload to a set of OTLP Metrics.
func (t *Translator) StatsPayloadToMetrics(sp pb.StatsPayload) pmetric.Metrics {
	mmx := pmetric.NewMetrics()
	// We ignore Agent{Hostname,Env,Version} and fill those in later. We want those
	// values to be consistent with the ones that appear on traces and logs. They are
	// only known in the Datadog exporter or the Datadog Agent OTLP Ingest.
	var npayloads, nbuckets, ngroups int
	for _, cp := range sp.Stats {
		npayloads++
		rmx := mmx.ResourceMetrics().AppendEmpty()
		attr := rmx.Resource().Attributes()
		attr.PutBool(keyAPMStats, true)
		putStr(attr, statsKeyHostname, cp.Hostname)
		putStr(attr, statsKeyEnv, cp.Env)
		putStr(attr, statsKeyVersion, cp.Version)
		putStr(attr, statsKeyLang, cp.Lang)
		putStr(attr, statsKeyTracerVersion, cp.TracerVersion)
		putStr(attr, statsKeyRuntimeID, cp.RuntimeID)
		putInt(attr, statsKeySequence, int64(cp.Sequence))
		putStr(attr, statsKeyAgentAggregation, cp.AgentAggregation)
		putStr(attr, statsKeyService, cp.Service)
		putStr(attr, statsKeyContainerID, cp.ContainerID)
		putStr(attr, statsKeyTags, strings.Join(cp.Tags, ","))

		for _, sb := range cp.Stats {
			nbuckets++
			smx := rmx.ScopeMetrics().AppendEmpty()
			for _, cgs := range sb.Stats {
				ngroups++
				mxs := smx.Metrics()
				for name, val := range map[string]uint64{
					metricNameHits:         cgs.Hits,
					metricNameErrors:       cgs.Errors,
					metricNameDuration:     cgs.Duration,
					metricNameTopLevelHits: cgs.TopLevelHits,
				} {
					appendSum(mxs, name, int64(val), sb.Start, sb.Start+sb.Duration, &cgs)
				}
				if err := appendSketch(mxs, metricNameOkSummary, cgs.OkSummary, sb.Start, sb.Start+sb.Duration, &cgs); err != nil {
					t.logger.Error("Error exporting APM Stats ok_summary", zap.Error(err))
				}
				if err := appendSketch(mxs, metricNameErrorSummary, cgs.ErrorSummary, sb.Start, sb.Start+sb.Duration, &cgs); err != nil {
					t.logger.Error("Error exporting APM Stats error_summary", zap.Error(err))
				}
			}
		}
	}
	return mmx
}

// appendSum appends the value val as a sum with the given name to the metric slice. It uses the appropriate fields
// from tags to set attributes.
func appendSum(mslice pmetric.MetricSlice, name string, val int64, start, end uint64, tags *pb.ClientGroupedStats) {
	mx := mslice.AppendEmpty()
	mx.SetName(name)
	sum := mx.SetEmptySum()
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityDelta)
	sum.SetIsMonotonic(true)

	dp := sum.DataPoints().AppendEmpty()
	dp.SetStartTimestamp(pcommon.Timestamp(start))
	dp.SetTimestamp(pcommon.Timestamp(end))
	dp.SetIntValue(int64(val))
	putGroupedStatsAttr(dp.Attributes(), tags)
}

// appendSketch appends the proto-encoded DDSketch from sketchBytes to the given metric slice as an ExponentialHistogram
// with the given name, start, and end timestamps. The fields from tags are set as attributes.
// It is assumed that the DDSketch was created by the trace-agent as a "LogCollapsingLowestDenseDDSketch" with a relative
// accuracy of 0.001.
func appendSketch(mslice pmetric.MetricSlice, name string, sketchBytes []byte, start, end uint64, tags *pb.ClientGroupedStats) error {
	if sketchBytes == nil {
		// no error, just nothing to do
		return nil
	}
	var msg sketchpb.DDSketch
	if err := proto.Unmarshal(sketchBytes, &msg); err != nil {
		return err
	}
	dds, err := ddsketch.FromProto(&msg)
	if err != nil {
		return err
	}
	if dds.IsEmpty() {
		// no error, just nothing to do
		return nil
	}
	mx := mslice.AppendEmpty()
	mx.SetName(name)
	hist := mx.SetEmptyExponentialHistogram()
	hist.SetAggregationTemporality(pmetric.AggregationTemporalityDelta)

	dp := hist.DataPoints().AppendEmpty()
	dp.SetStartTimestamp(pcommon.Timestamp(start))
	dp.SetTimestamp(pcommon.Timestamp(end))
	putGroupedStatsAttr(dp.Attributes(), tags)
	dp.SetCount(uint64(dds.GetCount()))
	if max, err := dds.GetMaxValue(); err == nil {
		dp.SetMax(max)
	}
	if min, err := dds.GetMinValue(); err == nil {
		dp.SetMin(min)
	}
	dp.SetSum(dds.GetSum())
	dp.SetZeroCount(uint64(dds.GetZeroCount()))
	// Relative accuracy (ra) in the trace-agent is 0.001. See:
	// https://github.com/DataDog/datadog-agent/blob/198e1a/pkg/trace/stats/statsraw.go#L19-L21
	// Gamma is computed as (1+ra)/(1-ra), which results in 1.02020202. Another formula for gamma
	// is 2^2^-scale. Using that formula, we conclude that a scale of 5 should be suitable, which
	// is equal to a gamma of 1.0218971486541166.
	//
	// It will not be equally accurate, but the error margin is negligible. Generally, the
	// ExponentialHistogram is simply a data recipient being converted back to the original DDSketch,
	// at which point we know the original gamma and the resulting sketch will be equivalent. The
	// only scenario when the histogram could be used is if someone exports it using a non-Datadog
	// exporter.
	dp.SetScale(5)
	storeToBuckets(dds.GetNegativeValueStore(), dp.Negative())
	storeToBuckets(dds.GetPositiveValueStore(), dp.Positive())
	return nil
}

// storeToBuckets converts a DDSketch store to an ExponentialHistogram data point buckets.
func storeToBuckets(s store.Store, b pmetric.ExponentialHistogramDataPointBuckets) {
	offset, err := s.MinIndex()
	if err != nil {
		return
	}
	max, err := s.MaxIndex()
	if err != nil {
		return
	}
	b.SetOffset(int32(offset))
	counts := make([]uint64, max-offset+1)
	s.ForEach(func(index int, count float64) bool {
		counts[index-offset] = uint64(count)
		return false
	})
	b.BucketCounts().FromRaw(counts)
}

func putGroupedStatsAttr(m pcommon.Map, cgs *pb.ClientGroupedStats) {
	putStr(m, statsKeyService, cgs.Service)
	putStr(m, statsKeySpanName, cgs.Name)
	putStr(m, statsKeySpanResource, cgs.Resource)
	putInt(m, statsKeyHTTPStatusCode, int64(cgs.HTTPStatusCode))
	putStr(m, statsKeySpanType, cgs.Type)
	putStr(m, statsKeySpanDBType, cgs.DBType)
	if cgs.Synthetics {
		m.PutBool(statsKeySynthetics, true)
	}
}

func putStr(m pcommon.Map, k, v string) {
	if v == "" {
		return
	}
	m.PutStr(k, v)
}

func putInt(m pcommon.Map, k string, v int64) {
	if v == 0 {
		return
	}
	m.PutInt(k, v)
}

// aggregationKey specifies a set of values by which a certain aggregationMetric is grouped.
type aggregationKey struct {
	Service        string
	Name           string
	Resource       string
	HTTPStatusCode uint32
	Type           string
	DBType         string
	Synthetics     bool
}

// aggregationValue specifies the set of metrics corresponding to a certain aggregationKey.
type aggregationValue struct {
	Hits         uint64
	Errors       uint64
	Duration     uint64
	OkSummary    []byte
	ErrorSummary []byte
	TopLevelHits uint64
}

// aggregations stores aggregation values (stats) grouped by their corresponding keys.
type aggregations struct {
	agg map[aggregationKey]*aggregationValue
}

// Value returns the aggregation value corresponding to the key found in map m.
func (a *aggregations) Value(m pcommon.Map) *aggregationValue {
	var sntx bool
	if v, ok := m.Get(statsKeySynthetics); ok {
		sntx = v.Bool()
	}
	key := aggregationKey{
		Service:        getStr(m, statsKeyService),
		Name:           getStr(m, statsKeySpanName),
		Resource:       getStr(m, statsKeySpanResource),
		HTTPStatusCode: uint32(getInt(m, statsKeyHTTPStatusCode)),
		Type:           getStr(m, statsKeySpanType),
		DBType:         getStr(m, statsKeySpanDBType),
		Synthetics:     sntx,
	}
	if a.agg == nil {
		a.agg = make(map[aggregationKey]*aggregationValue)
	}
	if _, ok := a.agg[key]; !ok {
		a.agg[key] = new(aggregationValue)
	}
	return a.agg[key]
}

// Stats returns the set of pb.ClientGroupedStats based on all the aggregated key/value
// pairs.
func (a *aggregations) Stats() []pb.ClientGroupedStats {
	cgs := make([]pb.ClientGroupedStats, 0, len(a.agg))
	for k, v := range a.agg {
		cgs = append(cgs, pb.ClientGroupedStats{
			Service:        k.Service,
			Name:           k.Name,
			Resource:       k.Resource,
			HTTPStatusCode: k.HTTPStatusCode,
			Type:           k.Type,
			DBType:         k.DBType,
			Synthetics:     k.Synthetics,
			Hits:           v.Hits,
			Errors:         v.Errors,
			Duration:       v.Duration,
			OkSummary:      v.OkSummary,
			ErrorSummary:   v.ErrorSummary,
			TopLevelHits:   v.TopLevelHits,
		})
	}
	return cgs
}

// UnsetHostnamePlaceholder is the string used as a hostname when the hostname can not be extracted from span attributes
// by the processor. Upon decoding the metrics, the Translator will use its configured fallback SourceProvider to replace
// it with the correct hostname.
//
// This isn't the most ideal approach to the problem, but provides the better user experience by avoiding the need to
// duplicate the "exporter::datadog::hostname" configuration field as "processor::datadog::hostname". The hostname can
// also not be left empty in case of failure to obtain it, because empty has special meaning. An empty hostname means
// that we are in a Lambda environment. Thus, we must use a placeholder.
const UnsetHostnamePlaceholder = "__unset__"

// statsPayloadFromMetrics converts Resource Metrics to an APM Client Stats Payload.
func (t *Translator) statsPayloadFromMetrics(rmx pmetric.ResourceMetrics) (pb.ClientStatsPayload, error) {
	attr := rmx.Resource().Attributes()
	if v, ok := attr.Get(keyAPMStats); !ok || !v.Bool() {
		return pb.ClientStatsPayload{}, fmt.Errorf("was asked to convert metrics to stats payload, but identifier key %q was not present. Skipping.", keyAPMStats)
	}
	hostname := getStr(attr, statsKeyHostname)
	tags := strings.Split(getStr(attr, statsKeyTags), ",")
	if hostname == UnsetHostnamePlaceholder {
		src, err := t.source(attr)
		if err != nil {
			return pb.ClientStatsPayload{}, err
		}
		switch src.Kind {
		case source.HostnameKind:
			hostname = src.Identifier
		case source.AWSECSFargateKind:
			hostname = ""
			tags = append(tags, src.Tag())
		}
	}
	cp := pb.ClientStatsPayload{
		Hostname:         hostname,
		Env:              getStr(attr, statsKeyEnv),
		Version:          getStr(attr, statsKeyVersion),
		Lang:             getStr(attr, statsKeyLang),
		TracerVersion:    getStr(attr, statsKeyTracerVersion),
		RuntimeID:        getStr(attr, statsKeyRuntimeID),
		Sequence:         getInt(attr, statsKeySequence),
		AgentAggregation: getStr(attr, statsKeyAgentAggregation),
		Service:          getStr(attr, statsKeyService),
		ContainerID:      getStr(attr, statsKeyContainerID),
		Tags:             tags,
	}
	smxs := rmx.ScopeMetrics()
	for j := 0; j < smxs.Len(); j++ {
		mxs := smxs.At(j).Metrics()
		var (
			buck pb.ClientStatsBucket
			agg  aggregations
		)
		for k := 0; k < mxs.Len(); k++ {
			m := mxs.At(k)
			switch m.Type() {
			case pmetric.MetricTypeSum:
				key, val := t.extractSum(m.Sum(), &buck)
				switch m.Name() {
				case metricNameHits:
					agg.Value(key).Hits = val
				case metricNameErrors:
					agg.Value(key).Errors = val
				case metricNameDuration:
					agg.Value(key).Duration = val
				case metricNameTopLevelHits:
					agg.Value(key).TopLevelHits = val
				}
			case pmetric.MetricTypeExponentialHistogram:
				key, val := t.extractSketch(m.ExponentialHistogram(), &buck)
				switch m.Name() {
				case metricNameOkSummary:
					agg.Value(key).OkSummary = val
				case metricNameErrorSummary:
					agg.Value(key).ErrorSummary = val
				}
			default:
				return pb.ClientStatsPayload{}, fmt.Errorf(`metric named %q in Stats Payload should be of type "Sum" or "ExponentialHistogram" but is %q instead`, m.Name(), m.Type())
			}
		}
		buck.Stats = agg.Stats()
		cp.Stats = append(cp.Stats, buck)
	}
	return cp, nil
}

// extractSketch extracts a proto-encoded version of the DDSketch found in the first data point of the given
// ExponentialHistogram along with its attributes and updates the timestamps in the provided stats bucket.
func (t *Translator) extractSketch(eh pmetric.ExponentialHistogram, buck *pb.ClientStatsBucket) (pcommon.Map, []byte) {
	dps := eh.DataPoints()
	if dps.Len() == 0 {
		t.logger.Debug("Stats payload exponential histogram with no data points.")
		return pcommon.NewMap(), nil
	}
	if dps.Len() > 1 {
		t.logger.Debug("Stats payload metrics should not have more than one data point. This could be an error.")
	}
	dp := dps.At(0)
	t.recordStatsBucketTimestamp(buck, dp.StartTimestamp(), dp.Timestamp())
	positive := toStore(dp.Positive())
	negative := toStore(dp.Negative())
	// use relative accuracy 0.01; same as pkg/trace/stats/statsraw.go
	index, err := mapping.NewLogarithmicMapping(0.01)
	if err != nil {
		t.logger.Debug("Error creating LogarithmicMapping.", zap.Error(err))
		return dp.Attributes(), nil
	}
	sketch := ddsketch.NewDDSketch(index, positive, negative)
	if err := sketch.AddWithCount(0, float64(dp.ZeroCount())); err != nil {
		t.logger.Debug("Error adding zero counts.", zap.Error(err))
		return dp.Attributes(), nil
	}
	pb := sketch.ToProto()
	b, err := proto.Marshal(pb)
	if err != nil {
		t.logger.Debug("Error marshalling stats payload sketch into proto.", zap.Error(err))
		return dp.Attributes(), nil
	}
	return dp.Attributes(), b
}

// extractSum extracts the attributes and the integer value found in the first data point of the given sum
// and updates the given buckets timestamps.
func (t *Translator) extractSum(sum pmetric.Sum, buck *pb.ClientStatsBucket) (pcommon.Map, uint64) {
	dps := sum.DataPoints()
	if dps.Len() == 0 {
		t.logger.Debug("APM stats payload sum with no data points.")
		return pcommon.NewMap(), 0
	}
	if dps.Len() > 1 {
		t.logger.Debug("APM stats metrics should not have more than one data point. This could be an error.")
	}
	dp := dps.At(0)
	t.recordStatsBucketTimestamp(buck, dp.StartTimestamp(), dp.Timestamp())
	return dp.Attributes(), uint64(dp.IntValue()) // more than one makes no sense
}

// recordStatsBucketTimestamp records the start & end timestamps from the given data point into the given stats bucket.
func (t *Translator) recordStatsBucketTimestamp(buck *pb.ClientStatsBucket, startt, endt pcommon.Timestamp) {
	start := uint64(startt)
	if buck.Start != 0 && buck.Start != start {
		t.logger.Debug("APM stats data point start timestamp did not match bucket. This could be an error.")
	}
	buck.Start = start
	duration := uint64(endt) - uint64(startt)
	buck.Duration = duration
	if buck.Duration != 0 && buck.Duration != duration {
		t.logger.Debug("APM Stats data point duration did not match bucket. This could be an error.")
	}
}

func getStr(m pcommon.Map, k string) string {
	v, ok := m.Get(k)
	if !ok {
		return ""
	}
	return v.Str()
}

func getInt(m pcommon.Map, k string) uint64 {
	v, ok := m.Get(k)
	if !ok {
		return 0
	}
	return uint64(v.Int())
}
