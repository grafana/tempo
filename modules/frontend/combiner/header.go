package combiner

import "net/http"

const (
	TempoCacheHeader = "X-Tempo-Cache"

	TempoCacheHit = "HIT"

	TempoCacheMiss = "MISS"
)

func IsCacheHit(resp *http.Response) bool {
	if resp == nil {
		return false
	}
	return resp.Header.Get(TempoCacheHeader) == TempoCacheHit
}
