package azure

import (
	"time"

	"github.com/cortexproject/cortex/pkg/util/flagext"
)

type Config struct {
	StorageAccountName flagext.Secret `yaml:"storage-account-name"`
	StorageAccountKey  flagext.Secret `yaml:"storage-account-key"`
	ContainerName      string         `yaml:"container-name"`
	Endpoint           string         `yaml:"endpoint-suffix"`
	MaxBuffers         int            `yaml:"max-buffers"`
	BufferSize         int            `yaml:"buffer-size"`
	HedgeRequestsAt    time.Duration  `yaml:"hedge-requests-at"`
}
