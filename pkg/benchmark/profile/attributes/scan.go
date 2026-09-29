package attributes

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"time"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	v1_common "github.com/grafana/tempo/v3/pkg/tempopb/common/v1"
	v1_trace "github.com/grafana/tempo/v3/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/v3/pkg/traceql"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
)

// serviceNameAttribute has a column of its own in every vParquet version.
const serviceNameAttribute = "service.name"

// Build ranks the block's n attributes with the most bytes.
func Build(ctx context.Context, meta *backend.BlockMeta, r backend.Reader, n int) (*Profiles, error) {
	iter, err := openTraceIterator(ctx, meta, r)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	return profileAttributes(ctx, iter, meta.DedicatedColumns, n)
}

// profileAttributes ranks the n attributes with the most bytes in the traces.
//
// Bytes are counted per stored value, as analyse block counts them when it
// picks a block's dedicated columns: a resource attribute once per resource, a
// span attribute once per span. Density and selectivity are per span, because
// a resource attribute matches every span beneath it.
func profileAttributes(ctx context.Context, iter common.Iterator, dedicated backend.DedicatedColumns, n int) (*Profiles, error) {
	s := &attributeScan{accs: map[attrKey]*attrAccumulator{}}
	for {
		_, tr, err := iter.Next(ctx)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading traces: %w", err)
		}
		if tr == nil {
			break
		}
		s.observeTrace(tr)
	}

	if s.spans == 0 {
		return nil, errors.New("block has no spans to profile attributes from")
	}
	return s.rank(dedicated, n), nil
}

type attributeScan struct {
	accs  map[attrKey]*attrAccumulator
	spans uint64

	// Reused across occurrences.
	attrs []scopedValue
	vals  []traceql.Static
}

type scopedValue struct {
	key attrKey
	val traceql.Static
}

func (s *attributeScan) observeTrace(tr *tempopb.Trace) {
	for _, rs := range tr.ResourceSpans {
		var spans uint64
		for _, ss := range rs.ScopeSpans {
			spans += uint64(len(ss.Spans))
		}
		// A resource without spans matches no query.
		if spans == 0 {
			continue
		}
		s.spans += spans

		if rs.Resource != nil {
			s.addAttributes(ScopeResource, rs.Resource.Attributes)
			s.flush(spans)
		}
		for _, ss := range rs.ScopeSpans {
			for _, sp := range ss.Spans {
				s.addAttributes(ScopeSpan, sp.Attributes)
				s.addIntrinsics(sp)
				s.flush(1)
			}
		}
	}
}

func (s *attributeScan) addAttributes(scope string, kvs []*v1_common.KeyValue) {
	for _, kv := range kvs {
		if kv.Value == nil {
			continue
		}
		if arr := kv.Value.GetArrayValue(); arr != nil {
			for _, e := range arr.Values {
				if e != nil {
					s.add(scope, kv.Key, traceql.StaticFromAnyValue(e))
				}
			}
			continue
		}
		s.add(scope, kv.Key, traceql.StaticFromAnyValue(kv.Value))
	}
}

func (s *attributeScan) addIntrinsics(sp *v1_trace.Span) {
	var duration time.Duration
	if sp.EndTimeUnixNano > sp.StartTimeUnixNano {
		duration = time.Duration(sp.EndTimeUnixNano - sp.StartTimeUnixNano)
	}

	s.add(ScopeIntrinsic, traceql.IntrinsicName.String(), traceql.NewStaticString(sp.Name))
	s.add(ScopeIntrinsic, traceql.IntrinsicStatus.String(), traceql.NewStaticStatus(otlpStatus(sp.Status)))
	s.add(ScopeIntrinsic, traceql.IntrinsicKind.String(), traceql.NewStaticKind(otlpKind(sp.Kind)))
	s.add(ScopeIntrinsic, traceql.IntrinsicDuration.String(), traceql.NewStaticDuration(duration))
}

func (s *attributeScan) add(scope, name string, v traceql.Static) {
	if typ, ok := attributeType(v.Type); ok {
		s.attrs = append(s.attrs, scopedValue{attrKey{scope, name, typ}, v})
	}
}

// flush records the values added since the last flush as one occurrence each
// of their attributes, covering the given spans. Values are grouped by
// attribute first, since an array arrives as one value per element.
func (s *attributeScan) flush(spans uint64) {
	slices.SortFunc(s.attrs, func(a, b scopedValue) int { return compareAttrKeys(a.key, b.key) })
	for i := 0; i < len(s.attrs); {
		k := s.attrs[i].key
		s.vals = s.vals[:0]
		for ; i < len(s.attrs) && s.attrs[i].key == k; i++ {
			s.vals = append(s.vals, s.attrs[i].val)
		}

		acc, ok := s.accs[k]
		if !ok {
			acc = newAttrAccumulator(k)
			s.accs[k] = acc
		}
		acc.observe(s.vals, spans)
	}
	s.attrs = s.attrs[:0]
}

// rank finishes only the top n, since finishing sorts each one's values.
func (s *attributeScan) rank(dedicated backend.DedicatedColumns, n int) *Profiles {
	accs := slices.Collect(maps.Values(s.accs))
	slices.SortFunc(accs, func(a, b *attrAccumulator) int {
		return cmp.Or(cmp.Compare(b.totalBytes, a.totalBytes), compareAttrKeys(a.key, b.key))
	})
	accs = accs[:min(n, len(accs))]

	ranked := make([]Profile, 0, len(accs))
	for _, acc := range accs {
		p := acc.finish(s.spans)
		p.Dedicated = hasOwnColumn(acc.key, dedicated)
		ranked = append(ranked, p)
	}
	return &Profiles{Spans: s.spans, Ranked: ranked}
}

func compareAttrKeys(a, b attrKey) int {
	return cmp.Or(cmp.Compare(a.scope, b.scope), cmp.Compare(a.name, b.name), cmp.Compare(a.typ, b.typ))
}

func attributeType(t traceql.StaticType) (string, bool) {
	switch t {
	case traceql.TypeString:
		return TypeString, true
	case traceql.TypeInt:
		return TypeInt, true
	case traceql.TypeFloat:
		return TypeFloat, true
	case traceql.TypeBoolean:
		return TypeBool, true
	case traceql.TypeDuration:
		return TypeDuration, true
	case traceql.TypeStatus:
		return TypeStatus, true
	case traceql.TypeKind:
		return TypeKind, true
	}
	return "", false
}

func otlpStatus(s *v1_trace.Status) traceql.Status {
	if s == nil {
		return traceql.StatusUnset
	}
	switch s.Code {
	case v1_trace.Status_STATUS_CODE_OK:
		return traceql.StatusOk
	case v1_trace.Status_STATUS_CODE_ERROR:
		return traceql.StatusError
	default:
		return traceql.StatusUnset
	}
}

func otlpKind(k v1_trace.Span_SpanKind) traceql.Kind {
	switch k {
	case v1_trace.Span_SPAN_KIND_INTERNAL:
		return traceql.KindInternal
	case v1_trace.Span_SPAN_KIND_SERVER:
		return traceql.KindServer
	case v1_trace.Span_SPAN_KIND_CLIENT:
		return traceql.KindClient
	case v1_trace.Span_SPAN_KIND_PRODUCER:
		return traceql.KindProducer
	case v1_trace.Span_SPAN_KIND_CONSUMER:
		return traceql.KindConsumer
	default:
		return traceql.KindUnspecified
	}
}

// hasOwnColumn reports whether the attribute is stored in a column of its own
// rather than the generic key/value columns. A dedicated column holds one type,
// so the attribute's values of another type are still generic.
func hasOwnColumn(k attrKey, dedicated backend.DedicatedColumns) bool {
	if k.scope == ScopeIntrinsic {
		return true
	}
	if k.scope == ScopeResource && k.name == serviceNameAttribute && k.typ == TypeString {
		return true
	}
	for _, c := range dedicated {
		if string(c.Scope) == k.scope && c.Name == k.name && string(c.Type) == k.typ {
			return true
		}
	}
	return false
}
