// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jaegerreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/jaegerreceiver"

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
)

const (
	// The config field id to load the protocol map from
	protocolsFieldName = "protocols"

	// Default UDP server options
	defaultQueueSize        = 1_000
	defaultMaxPacketSize    = 65_000
	defaultServerWorkers    = 10
	defaultSocketBufferSize = 0
)

// RemoteSamplingConfig defines config key for remote sampling fetch endpoint
type RemoteSamplingConfig struct {
	HostEndpoint                  string        `mapstructure:"host_endpoint"`
	StrategyFile                  string        `mapstructure:"strategy_file"`
	StrategyFileReloadInterval    time.Duration `mapstructure:"strategy_file_reload_interval"`
	configgrpc.GRPCClientSettings `mapstructure:",squash"`
}

// Protocols is the configuration for the supported protocols.
type Protocols struct {
	GRPC          *configgrpc.GRPCServerSettings `mapstructure:"grpc"`
	ThriftHTTP    *confighttp.HTTPServerSettings `mapstructure:"thrift_http"`
	ThriftBinary  *ProtocolUDP                   `mapstructure:"thrift_binary"`
	ThriftCompact *ProtocolUDP                   `mapstructure:"thrift_compact"`
}

// ProtocolUDP is the configuration for a UDP protocol.
type ProtocolUDP struct {
	Endpoint        string `mapstructure:"endpoint"`
	ServerConfigUDP `mapstructure:",squash"`
}

// ServerConfigUDP is the server configuration for a UDP protocol.
type ServerConfigUDP struct {
	QueueSize        int `mapstructure:"queue_size"`
	MaxPacketSize    int `mapstructure:"max_packet_size"`
	Workers          int `mapstructure:"workers"`
	SocketBufferSize int `mapstructure:"socket_buffer_size"`
}

// DefaultServerConfigUDP creates the default ServerConfigUDP.
func DefaultServerConfigUDP() ServerConfigUDP {
	return ServerConfigUDP{
		QueueSize:        defaultQueueSize,
		MaxPacketSize:    defaultMaxPacketSize,
		Workers:          defaultServerWorkers,
		SocketBufferSize: defaultSocketBufferSize,
	}
}

// Config defines configuration for Jaeger receiver.
type Config struct {
	config.ReceiverSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	Protocols               `mapstructure:"protocols"`
	RemoteSampling          *RemoteSamplingConfig `mapstructure:"remote_sampling"`
}

var _ config.Receiver = (*Config)(nil)
var _ config.Unmarshallable = (*Config)(nil)

// Validate checks the receiver configuration is valid
func (cfg *Config) Validate() error {
	if cfg.GRPC == nil &&
		cfg.ThriftHTTP == nil &&
		cfg.ThriftBinary == nil &&
		cfg.ThriftCompact == nil {
		return fmt.Errorf("must specify at least one protocol when using the Jaeger receiver")
	}

	var grpcPort int
	if cfg.GRPC != nil {
		var err error
		if grpcPort, err = extractPortFromEndpoint(cfg.GRPC.NetAddr.Endpoint); err != nil {
			return fmt.Errorf("unable to extract port for the gRPC endpoint: %w", err)
		}
	}

	if cfg.ThriftHTTP != nil {
		if _, err := extractPortFromEndpoint(cfg.ThriftHTTP.Endpoint); err != nil {
			return fmt.Errorf("unable to extract port for the Thrift HTTP endpoint: %w", err)
		}
	}

	if cfg.ThriftBinary != nil {
		if _, err := extractPortFromEndpoint(cfg.ThriftBinary.Endpoint); err != nil {
			return fmt.Errorf("unable to extract port for the Thrift UDP Binary endpoint: %w", err)
		}
	}

	if cfg.ThriftCompact != nil {
		if _, err := extractPortFromEndpoint(cfg.ThriftCompact.Endpoint); err != nil {
			return fmt.Errorf("unable to extract port for the Thrift UDP Compact endpoint: %w", err)
		}
	}

	if cfg.RemoteSampling != nil {
		if _, err := extractPortFromEndpoint(cfg.RemoteSampling.HostEndpoint); err != nil {
			return fmt.Errorf("unable to extract port for the Remote Sampling endpoint: %w", err)
		}

		if len(cfg.RemoteSampling.StrategyFile) != 0 && grpcPort == 0 {
			return fmt.Errorf("strategy file requires the gRPC protocol to be enabled")
		}

		if cfg.RemoteSampling.StrategyFileReloadInterval < 0 {
			return fmt.Errorf("strategy file reload interval should be great or equal zero")
		}
	}

	return nil
}

// Unmarshal a config.Parser into the config struct.
func (cfg *Config) Unmarshal(componentParser *config.Map) error {
	if componentParser == nil || len(componentParser.AllKeys()) == 0 {
		return fmt.Errorf("empty config for Jaeger receiver")
	}

	// UnmarshalExact will not set struct properties to nil even if no key is provided,
	// so set the protocol structs to nil where the keys were omitted.
	err := componentParser.UnmarshalExact(cfg)
	if err != nil {
		return err
	}

	protocols, err := componentParser.Sub(protocolsFieldName)
	if err != nil {
		return err
	}

	if !protocols.IsSet(protoGRPC) {
		cfg.GRPC = nil
	}
	if !protocols.IsSet(protoThriftHTTP) {
		cfg.ThriftHTTP = nil
	}
	if !protocols.IsSet(protoThriftBinary) {
		cfg.ThriftBinary = nil
	}
	if !protocols.IsSet(protoThriftCompact) {
		cfg.ThriftCompact = nil
	}

	return nil
}
