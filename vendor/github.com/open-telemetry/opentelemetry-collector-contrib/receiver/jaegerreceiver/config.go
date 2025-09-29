// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package jaegerreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/jaegerreceiver"

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configoptional"
)

const (
	// Default UDP server options
	defaultQueueSize        = 1_000
	defaultMaxPacketSize    = 65_000
	defaultServerWorkers    = 10
	defaultSocketBufferSize = 0
)

// RemoteSamplingConfig defines config key for remote sampling fetch endpoint
type RemoteSamplingConfig struct {
	HostEndpoint               string        `mapstructure:"host_endpoint"`
	StrategyFile               string        `mapstructure:"strategy_file"`
	StrategyFileReloadInterval time.Duration `mapstructure:"strategy_file_reload_interval"`
	configgrpc.ClientConfig    `mapstructure:",squash"`

	// prevent unkeyed literal initialization
	_ struct{}
}

// Protocols is the configuration for the supported protocols.
type Protocols struct {
	GRPC             configoptional.Optional[configgrpc.ServerConfig] `mapstructure:"grpc"`
	ThriftHTTP       configoptional.Optional[confighttp.ServerConfig] `mapstructure:"thrift_http"`
	ThriftBinaryUDP  configoptional.Optional[ProtocolUDP]             `mapstructure:"thrift_binary"`
	ThriftCompactUDP configoptional.Optional[ProtocolUDP]             `mapstructure:"thrift_compact"`

	// prevent unkeyed literal initialization
	_ struct{}
}

// ProtocolUDP is the configuration for a UDP protocol.
type ProtocolUDP struct {
	Endpoint        string `mapstructure:"endpoint"`
	ServerConfigUDP `mapstructure:",squash"`

	// prevent unkeyed literal initialization
	_ struct{}
}

// ServerConfigUDP is the server configuration for a UDP protocol.
type ServerConfigUDP struct {
	QueueSize        int `mapstructure:"queue_size"`
	MaxPacketSize    int `mapstructure:"max_packet_size"`
	Workers          int `mapstructure:"workers"`
	SocketBufferSize int `mapstructure:"socket_buffer_size"`

	// prevent unkeyed literal initialization
	_ struct{}
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

	// prevent unkeyed literal initialization
	_ struct{}
}

var _ component.Config = (*Config)(nil)

// Validate checks the receiver configuration is valid
func (cfg *Config) Validate() error {
	if !cfg.GRPC.HasValue() &&
		!cfg.ThriftHTTP.HasValue() &&
		!cfg.ThriftBinaryUDP.HasValue() &&
		!cfg.ThriftCompactUDP.HasValue() {
		return errors.New("must specify at least one protocol when using the Jaeger receiver")
	}

	if cfg.GRPC.HasValue() {
		grpcConfig := cfg.GRPC.Get()
		if err := checkPortFromEndpoint(grpcConfig.NetAddr.Endpoint); err != nil {
			return fmt.Errorf("invalid port number for the gRPC endpoint: %w", err)
		}
	}

	if cfg.ThriftHTTP.HasValue() {
		httpConfig := cfg.ThriftHTTP.Get()
		if err := checkPortFromEndpoint(httpConfig.Endpoint); err != nil {
			return fmt.Errorf("invalid port number for the Thrift HTTP endpoint: %w", err)
		}
	}

	if cfg.ThriftBinaryUDP.HasValue() {
		binaryUDPConfig := cfg.ThriftBinaryUDP.Get()
		if err := checkPortFromEndpoint(binaryUDPConfig.Endpoint); err != nil {
			return fmt.Errorf("invalid port number for the Thrift UDP Binary endpoint: %w", err)
		}
	}

	if cfg.ThriftCompactUDP.HasValue() {
		compactUDPConfig := cfg.ThriftCompactUDP.Get()
		if err := checkPortFromEndpoint(compactUDPConfig.Endpoint); err != nil {
			return fmt.Errorf("invalid port number for the Thrift UDP Compact endpoint: %w", err)
		}
	}

	if cfg.RemoteSampling != nil {
		if disableJaegerReceiverRemoteSampling.IsEnabled() {
			return errors.New("remote sampling config detected in the Jaeger receiver; use the `jaegerremotesampling` extension instead")
		}
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
		return errors.New("port number must be between 1 and 65535")
	}
	return nil
}
