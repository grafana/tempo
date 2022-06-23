//go:build !purego

package parquet

//go:noescape
func memsetValuesAVX2(values []Value, model Value, _ uint64)

func memsetValues(values []Value, model Value) {
	if hasAVX2 {
		memsetValuesAVX2(values, model, 0)
	} else {
		for i := range values {
			values[i] = model
		}
	}
}
