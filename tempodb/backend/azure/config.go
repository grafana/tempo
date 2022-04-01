package azure

import (
	"time"

	"github.com/grafana/dskit/flagext"
)

type Config struct {
	StorageAccountName string         `yaml:"storage-account-name"`
	StorageAccountKey  flagext.Secret `yaml:"storage-account-key"`
	ContainerName      string         `yaml:"container-name"`
	Endpoint           string         `yaml:"endpoint-suffix"`
	MaxBuffers         int            `yaml:"max-buffers"`
	BufferSize         int            `yaml:"buffer-size"`
	HedgeRequestsAt    time.Duration  `yaml:"hedge-requests-at"`
	HedgeRequestsUpTo  int            `yaml:"hedge-requests-up-to"`
}
