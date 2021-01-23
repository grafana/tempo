package azure

import "github.com/cortexproject/cortex/pkg/util/flagext"

type Config struct {
	StorageAccountName flagext.Secret `yaml:"storage-account-name"`
	StorageAccountKey  flagext.Secret `yaml:"storage-account-key"`
	ContainerName      string         `yaml:"container-name"`
	Endpoint           string         `yaml:"endpoint-suffix"`
	MaxBuffers         int            `yaml:"max-buffers"`
	BufferSize         int            `yaml:"buffer-size"`
	UseManagedIdentity bool           `yaml:"use-managed-identity"`
	ClientID           flagext.Secret `yaml:"client-id"`
	ClientSecret       flagext.Secret `yaml:"client-secret"`
	TenantID           flagext.Secret `yaml:"tenant-id"`
	AzureEnvironment   string         `yaml:"azure-environment"`
}
