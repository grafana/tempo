package azure

import (
	blob "github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/pkg/errors"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/azure/config"
	v1 "github.com/grafana/tempo/tempodb/backend/azure/v1"
	v2 "github.com/grafana/tempo/tempodb/backend/azure/v2"
)

// NewNoConfirm gets the Azure blob container without testing it
func NewNoConfirm(cfg *config.Config) (backend.RawReader, backend.RawWriter, backend.Compactor, error) {
	if cfg.UseV2SDK {
		rw, err := v2.New(cfg, false)
		return rw, rw, rw, err
	}

	rw, err := v1.New(cfg, false)
	return rw, rw, rw, err
}

// New gets the Azure blob container
func New(cfg *config.Config) (backend.RawReader, backend.RawWriter, backend.Compactor, error) {
	if cfg.UseV2SDK {
		rw, err := v2.New(cfg, true)
		return rw, rw, rw, err
	}

	rw, err := v1.New(cfg, true)
	return rw, rw, rw, err
}

// NewVersionedReaderWriter creates a client to perform versioned requests. Note that write requests are
// best-effort for now. We need to update the SDK to make use of the precondition headers.
// https://github.com/grafana/tempo/issues/2705
func NewVersionedReaderWriter(cfg *config.Config) (backend.VersionedReaderWriter, error) {
	if cfg.UseV2SDK {
		return v2.New(cfg, true)
	}
	return v1.New(cfg, true)
}

func readError(err error) error {
	var storageError blob.StorageError
	errors.As(err, &storageError)

	if storageError == nil {
		return errors.Wrap(err, "reading storage container")
	}
	if storageError.ServiceCode() == blob.ServiceCodeBlobNotFound {
		return backend.ErrDoesNotExist
	}
	return errors.Wrap(storageError, "reading Azure blob container")
}
