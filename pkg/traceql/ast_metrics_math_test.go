package traceql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyBinaryOp_Exemplars(t *testing.T) {
	exemplar := func(id string) Exemplar {
		return Exemplar{
			Labels:      Labels{{Name: "trace.id", Value: NewStaticString(id)}},
			Value:       1,
			TimestampMs: 100,
		}
	}
	eEvenL := exemplar("evL")
	eOddL := exemplar("odL")
	eEvenR := exemplar("evR")
	eOddR := exemplar("odR")
	eAll1 := exemplar("a1")
	eAll2 := exemplar("a2")

	svcEven := Labels{{Name: "service.name", Value: NewStaticString("even")}}
	svcOdd := Labels{{Name: "service.name", Value: NewStaticString("odd")}}

	series := func(lbls Labels, ex []Exemplar) TimeSeries {
		return TimeSeries{Labels: lbls, Values: []float64{2}, Exemplars: ex}
	}

	tests := []struct {
		name string
		lhs  SeriesSet
		rhs  SeriesSet
		want map[SeriesMapKey][]Exemplar
	}{
		{
			name: "both grouped same -> merge per key",
			lhs: SeriesSet{
				svcEven.MapKey(): series(svcEven, []Exemplar{eEvenL}),
				svcOdd.MapKey():  series(svcOdd, []Exemplar{eOddL}),
			},
			rhs: SeriesSet{
				svcEven.MapKey(): series(svcEven, []Exemplar{eEvenR}),
				svcOdd.MapKey():  series(svcOdd, []Exemplar{eOddR}),
			},
			want: map[SeriesMapKey][]Exemplar{
				svcEven.MapKey(): {eEvenL, eEvenR},
				svcOdd.MapKey():  {eOddL, eOddR},
			},
		},
		{
			name: "RHS broadcast -> drop RHS exemplars",
			lhs: SeriesSet{
				svcEven.MapKey(): series(svcEven, []Exemplar{eEvenL}),
				svcOdd.MapKey():  series(svcOdd, []Exemplar{eOddL}),
			},
			rhs: SeriesSet{
				noLabelsSeriesMapKey: series(Labels{}, []Exemplar{eAll1, eAll2}),
			},
			want: map[SeriesMapKey][]Exemplar{
				svcEven.MapKey(): {eEvenL},
				svcOdd.MapKey():  {eOddL},
			},
		},
		{
			name: "LHS broadcast -> drop LHS exemplars",
			lhs: SeriesSet{
				noLabelsSeriesMapKey: series(Labels{}, []Exemplar{eAll1, eAll2}),
			},
			rhs: SeriesSet{
				svcEven.MapKey(): series(svcEven, []Exemplar{eEvenR}),
				svcOdd.MapKey():  series(svcOdd, []Exemplar{eOddR}),
			},
			want: map[SeriesMapKey][]Exemplar{
				svcEven.MapKey(): {eEvenR},
				svcOdd.MapKey():  {eOddR},
			},
		},
		{
			name: "both ungrouped -> merge",
			lhs: SeriesSet{
				noLabelsSeriesMapKey: series(Labels{}, []Exemplar{eAll1}),
			},
			rhs: SeriesSet{
				noLabelsSeriesMapKey: series(Labels{}, []Exemplar{eAll2}),
			},
			want: map[SeriesMapKey][]Exemplar{
				noLabelsSeriesMapKey: {eAll1, eAll2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := applyBinaryOp(OpDiv, tt.lhs, tt.rhs)
			require.Len(t, out, len(tt.want))
			for k, want := range tt.want {
				require.Equal(t, want, out[k].Exemplars)
			}
		})
	}
}
