package livestore

import (
	"flag"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name         string
		modifyConfig func(*Config)
		expectedErr  string
	}{
		{
			name: "valid config",
			modifyConfig: func(_ *Config) {
				// Default valid config, no modifications needed
			},
			expectedErr: "",
		},
		{
			name: "negative complete block timeout",
			modifyConfig: func(cfg *Config) {
				cfg.CompleteBlockTimeout = -1 * time.Second
			},
			expectedErr: "complete_block_timeout must be greater than 0",
		},
		{
			name: "zero query block concurrency",
			modifyConfig: func(cfg *Config) {
				cfg.QueryBlockConcurrency = 0
			},
			expectedErr: "query_blocks must be greater than 0",
		},
		{
			name: "negative complete block concurrency",
			modifyConfig: func(cfg *Config) {
				cfg.CompleteBlockConcurrency = -1
			},
			expectedErr: "complete_block_concurrency must be greater than 0",
		},
		{
			name: "negative instance flush period",
			modifyConfig: func(cfg *Config) {
				cfg.InstanceFlushPeriod = -1 * time.Second
			},
			expectedErr: "flush_check_period must be greater than 0",
		},
		{
			name: "zero instance flush period",
			modifyConfig: func(cfg *Config) {
				cfg.InstanceFlushPeriod = 0
			},
			expectedErr: "flush_check_period must be greater than 0",
		},
		{
			name: "negative instance cleanup period",
			modifyConfig: func(cfg *Config) {
				cfg.InstanceCleanupPeriod = -1 * time.Second
			},
			expectedErr: "flush_op_timeout must be greater than 0",
		},
		{
			name: "zero instance cleanup period",
			modifyConfig: func(cfg *Config) {
				cfg.InstanceCleanupPeriod = 0
			},
			expectedErr: "flush_op_timeout must be greater than 0",
		},
		{
			name: "negative max trace live",
			modifyConfig: func(cfg *Config) {
				cfg.MaxTraceLive = -1 * time.Second
			},
			expectedErr: "max_trace_live must be greater than 0",
		},
		{
			name: "zero max trace live",
			modifyConfig: func(cfg *Config) {
				cfg.MaxTraceLive = 0
			},
			expectedErr: "max_trace_live must be greater than 0",
		},
		{
			name: "negative max trace idle",
			modifyConfig: func(cfg *Config) {
				cfg.MaxTraceIdle = -1 * time.Second
			},
			expectedErr: "max_trace_idle must be greater than 0",
		},
		{
			name: "zero max trace idle",
			modifyConfig: func(cfg *Config) {
				cfg.MaxTraceIdle = 0
			},
			expectedErr: "max_trace_idle must be greater than 0",
		},
		{
			name: "negative max block duration",
			modifyConfig: func(cfg *Config) {
				cfg.MaxBlockDuration = -1 * time.Second
			},
			expectedErr: "max_block_duration must be greater than 0",
		},
		{
			name: "zero max block duration",
			modifyConfig: func(cfg *Config) {
				cfg.MaxBlockDuration = 0
			},
			expectedErr: "max_block_duration must be greater than 0",
		},
		{
			name: "zero max block bytes",
			modifyConfig: func(cfg *Config) {
				cfg.MaxBlockBytes = 0
			},
			expectedErr: "max_block_bytes must be greater than 0",
		},
		{
			name: "max trace idle greater than max trace live",
			modifyConfig: func(cfg *Config) {
				cfg.MaxTraceLive = 10 * time.Second
				cfg.MaxTraceIdle = 20 * time.Second
			},
			expectedErr: "max_trace_idle (20s) cannot be greater than max_trace_live (10s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// default config
			cfg := Config{}
			cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
			tt.modifyConfig(&cfg)

			err := cfg.Validate()

			if tt.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			}
		})
	}
}
