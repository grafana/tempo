package search

import (
	"github.com/grafana/tempo/pkg/tempofb"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type DataCombiner struct{}

var _ common.ObjectCombiner = (*DataCombiner)(nil)

var staticCombiner = DataCombiner{}

func (*DataCombiner) Combine(_ string, searchData ...[]byte) ([]byte, bool) {

	if len(searchData) <= 0 {
		return nil, false
	}

	if len(searchData) == 1 {
		return searchData[0], false
	}

	// Squash all datas into 1
	data := tempofb.SearchEntryMutable{}
	kv := &tempofb.KeyValues{} // buffer
	for _, sb := range searchData {
		if len(sb) == 0 {
			continue
		}
		sd := tempofb.SearchEntryFromBytes(sb)
		for i, ii := 0, sd.TagsLength(); i < ii; i++ {
			sd.Tags(kv, i)
			for j, jj := 0, kv.ValueLength(); j < jj; j++ {
				data.AddTag(string(kv.Key()), string(kv.Value(j)))
			}
		}

		data.SetStartTimeUnixNano(sd.StartTimeUnixNano())
		data.SetEndTimeUnixNano(sd.EndTimeUnixNano())
		data.TraceID = sd.Id()
	}

	return data.ToBytes(), true
}
