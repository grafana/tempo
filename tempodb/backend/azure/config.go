package azure

import (
	"time"

	"github.com/grafana/dskit/flagext"
)

type Config struct {
	StorageAccountName string         `yaml:"storage_account_name"`
	StorageAccountKey  flagext.Secret `yaml:"storage_account_key"`
	UseManagedIdentity bool           `yaml:"use_managed_identity"`
	UseFederatedToken  bool           `yaml:"use_federated_token"`
	UserAssignedID     string         `yaml:"user_assigned_id"`
	ContainerName      string         `yaml:"container_name"`
	Endpoint           string         `yaml:"endpoint_suffix"`
	MaxBuffers         int            `yaml:"max_buffers"`
	BufferSize         int            `yaml:"buffer_size"`
	HedgeRequestsAt    time.Duration  `yaml:"hedge_requests_at"`
	HedgeRequestsUpTo  int            `yaml:"hedge_requests_up_to"`
}
