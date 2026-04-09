package azure

import (
	"flag"
	"time"

	"github.com/grafana/dskit/flagext"

	"github.com/grafana/tempo/v2/pkg/util"
)

type Config struct {
	StorageAccountName string         `yaml:"storage_account_name"`
	StorageAccountKey  flagext.Secret `yaml:"storage_account_key"`
	UseManagedIdentity bool           `yaml:"use_managed_identity"`
	UseFederatedToken  bool           `yaml:"use_federated_token"`
	UserAssignedID     string         `yaml:"user_assigned_id"`
	ContainerName      string         `yaml:"container_name"`
	Prefix             string         `yaml:"prefix"`
	Endpoint           string         `yaml:"endpoint_suffix"`
	MaxBuffers         int            `yaml:"max_buffers"`
	BufferSize         int            `yaml:"buffer_size"`
	HedgeRequestsAt    time.Duration  `yaml:"hedge_requests_at"`
	HedgeRequestsUpTo  int            `yaml:"hedge_requests_up_to"`
}

func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	f.StringVar(&cfg.StorageAccountName, util.PrefixConfig(prefix, "azure.storage_account_name"), "", "Azure storage account name.")
	f.Var(&cfg.StorageAccountKey, util.PrefixConfig(prefix, "azure.storage_account_key"), "Azure storage access key.")
	f.StringVar(&cfg.ContainerName, util.PrefixConfig(prefix, "azure.container_name"), "", "Azure container name to store blocks in.")
	f.StringVar(&cfg.Prefix, util.PrefixConfig(prefix, "azure.prefix"), "", "Azure container prefix to store blocks in.")
	f.StringVar(&cfg.Endpoint, util.PrefixConfig(prefix, "azure.endpoint"), "blob.core.windows.net", "Azure endpoint to push blocks to.")
	f.IntVar(&cfg.MaxBuffers, util.PrefixConfig(prefix, "azure.max_buffers"), 4, "Number of simultaneous uploads.")
	cfg.BufferSize = 3 * 1024 * 1024
	cfg.HedgeRequestsUpTo = 2
}

func (cfg *Config) PathMatches(other *Config) bool {
	return cfg.StorageAccountName == other.StorageAccountName && cfg.ContainerName == other.ContainerName && cfg.Prefix == other.Prefix
}
