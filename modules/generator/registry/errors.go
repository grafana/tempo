package registry

import (
	"errors"

	"github.com/prometheus/prometheus/storage"
	"github.com/prometheus/prometheus/tsdb"
)

// getErrType maps known Prometheus write errors to coarse categories
func getErrType(err error) string {
	if err == nil {
		return "none"
	}

	switch {
	case errors.Is(err, storage.ErrNotFound):
		return "not_found"
	case errors.Is(err, tsdb.ErrInvalidSample),
		errors.Is(err, tsdb.ErrInvalidExemplar):
		return "invalid"
	case errors.Is(err, storage.ErrOutOfOrderSample),
		errors.Is(err, storage.ErrOutOfBounds),
		errors.Is(err, storage.ErrTooOldSample),
		errors.Is(err, storage.ErrOutOfOrderCT),
		errors.Is(err, storage.ErrCTNewerThanSample):
		return "out_of_order"
	case errors.Is(err, storage.ErrDuplicateSampleForTimestamp),
		errors.Is(err, storage.ErrDuplicateExemplar):
		return "duplicate"
	default:
		return "other"
	}
}
