package parquet

import "reflect"

var noTags = ParquetTags{}

// ParquetTags represents the superset of all the parquet struct tags that can be used
// to configure a field.
type ParquetTags struct {
	Parquet        string
	ParquetKey     string
	ParquetValue   string
	ParquetElement string
}

// fromStructTag parses the parquet struct tags from a reflect.StructTag and returns
// a parquetTags struct.
func fromStructTag(tag reflect.StructTag) ParquetTags {
	parquetTags := ParquetTags{}
	if val := tag.Get("parquet"); val != "" {
		parquetTags.Parquet = val
	}
	if val := tag.Get("parquet-key"); val != "" {
		parquetTags.ParquetKey = val
	}
	if val := tag.Get("parquet-value"); val != "" {
		parquetTags.ParquetValue = val
	}
	if val := tag.Get("parquet-element"); val != "" {
		parquetTags.ParquetElement = val
	}
	return parquetTags
}

// getMapKeyNodeTags returns the parquet tags for configuring the keys of a map.
func (p ParquetTags) getMapKeyNodeTags() ParquetTags {
	return ParquetTags{
		Parquet: p.ParquetKey,
	}
}

// getMapValueNodeTags returns the parquet tags for configuring the values of a map.
func (p ParquetTags) getMapValueNodeTags() ParquetTags {
	return ParquetTags{
		Parquet: p.ParquetValue,
	}
}

// getListElementNodeTags returns the parquet tags for configuring the elements of a list.
func (p ParquetTags) getListElementNodeTags() ParquetTags {
	return ParquetTags{
		Parquet: p.ParquetElement,
	}
}
