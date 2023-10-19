// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package jaegerreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/jaegerreceiver"

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/confmap"
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

// defaultServerConfigUDP creates the default ServerConfigUDP.
func defaultServerConfigUDP() ServerConfigUDP {
	return ServerConfigUDP{
		QueueSize:        defaultQueueSize,
		MaxPacketSize:    defaultMaxPacketSize,
		Workers:          defaultServerWorkers,
		SocketBufferSize: defaultSocketBufferSize,
	}
}

// Config defines configuration for Jaeger receiver.
type Config struct {
	Protocols      `mapstructure:"protocols"`
	RemoteSampling *RemoteSamplingConfig `mapstructure:"remote_sampling"`
}

var _ component.Config = (*Config)(nil)
var _ confmap.Unmarshaler = (*Config)(nil)

// Validate checks the receiver configuration is valid
func (cfg *Config) Validate() error {
	if cfg.GRPC == nil &&
		cfg.ThriftHTTP == nil &&
		cfg.ThriftBinary == nil &&
		cfg.ThriftCompact == nil {
		return fmt.Errorf("must specify at least one protocol when using the Jaeger receiver")
	}

	if cfg.GRPC != nil {
		if err := checkPortFromEndpoint(cfg.GRPC.NetAddr.Endpoint); err != nil {
			return fmt.Errorf("invalid port number for the gRPC endpoint: %w", err)
		}
	}

	if cfg.ThriftHTTP != nil {
		if err := checkPortFromEndpoint(cfg.ThriftHTTP.Endpoint); err != nil {
			return fmt.Errorf("invalid port number for the Thrift HTTP endpoint: %w", err)
		}
	}

	if cfg.ThriftBinary != nil {
		if err := checkPortFromEndpoint(cfg.ThriftBinary.Endpoint); err != nil {
			return fmt.Errorf("invalid port number for the Thrift UDP Binary endpoint: %w", err)
		}
	}

	if cfg.ThriftCompact != nil {
		if err := checkPortFromEndpoint(cfg.ThriftCompact.Endpoint); err != nil {
			return fmt.Errorf("invalid port number for the Thrift UDP Compact endpoint: %w", err)
		}
	}

	if cfg.RemoteSampling != nil {
		if disableJaegerReceiverRemoteSampling.IsEnabled() {
			return fmt.Errorf("remote sampling config detected in the Jaeger receiver; use the `jaegerremotesampling` extension instead")
		}
	}

	return nil
}

// Unmarshal a config.Parser into the config struct.
func (cfg *Config) Unmarshal(componentParser *confmap.Conf) error {
	if componentParser == nil || len(componentParser.AllKeys()) == 0 {
		return fmt.Errorf("empty config for Jaeger receiver")
	}

	// UnmarshalExact will not set struct properties to nil even if no key is provided,
	// so set the protocol structs to nil where the keys were omitted.
	err := componentParser.Unmarshal(cfg, confmap.WithErrorUnused())
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

// checkPortFromEndpoint checks that the endpoint string contains a port in the format "address:port". If the
// port number cannot be parsed, returns an error.
func checkPortFromEndpoint(endpoint string) error {
	_, portStr, err := net.SplitHostPort(endpoint)
	if err != nil {
		return fmt.Errorf("endpoint is not formatted correctly: %w", err)
	}
	port, err := strconv.ParseInt(portStr, 10, 0)
	if err != nil {
		return fmt.Errorf("endpoint port is not a number: %w", err)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("port number must be between 1 and 65535")
	}
	return nil
}
