package ingest

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/grafana/dskit/flagext"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSASLMechanism(t *testing.T) {
	tests := []struct {
		name         string
		username     string
		password     string
		mechanism    string
		expectedName string // empty string means we expect a nil mechanism
		expectedErr  error
	}{
		{
			name:         "no credentials disables SASL",
			username:     "",
			password:     "",
			mechanism:    SASLMechanismPlain,
			expectedName: "",
		},
		{
			name:         "empty mechanism defaults to plain",
			username:     "user",
			password:     "pass",
			mechanism:    "",
			expectedName: SASLMechanismPlain,
		},
		{
			name:         "plain",
			username:     "user",
			password:     "pass",
			mechanism:    SASLMechanismPlain,
			expectedName: SASLMechanismPlain,
		},
		{
			name:         "scram-sha-256",
			username:     "user",
			password:     "pass",
			mechanism:    SASLMechanismScramSHA256,
			expectedName: SASLMechanismScramSHA256,
		},
		{
			name:         "scram-sha-512",
			username:     "user",
			password:     "pass",
			mechanism:    SASLMechanismScramSHA512,
			expectedName: SASLMechanismScramSHA512,
		},
		{
			name:        "unsupported mechanism returns an error",
			username:    "user",
			password:    "pass",
			mechanism:   "SCRAM-SHA-1",
			expectedErr: ErrUnsupportedSASLMechanism,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := KafkaConfig{
				SASLUsername:  tt.username,
				SASLPassword:  flagext.SecretWithValue(tt.password),
				SASLMechanism: tt.mechanism,
			}

			mechanism, err := saslMechanism(cfg)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				require.Nil(t, mechanism)
				return
			}

			require.NoError(t, err)
			if tt.expectedName == "" {
				require.Nil(t, mechanism)
				return
			}

			require.NotNil(t, mechanism)
			require.Equal(t, tt.expectedName, mechanism.Name())
		})
	}
}

func TestStatusFromProduceErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"nil", nil, codes.OK},
		{"record timeout", kgo.ErrRecordTimeout, codes.Unavailable},
		{"context deadline", context.DeadlineExceeded, codes.Unavailable},
		{"context canceled", context.Canceled, codes.Canceled},
		{"buffer full", kgo.ErrMaxBuffered, codes.Unavailable},
		{"generic", errors.New("boom"), codes.Unavailable},
		{"message too large", kerr.MessageTooLarge, codes.InvalidArgument},
		{"wrapped message too large", fmt.Errorf("failed to produce: %w", kerr.MessageTooLarge), codes.InvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StatusFromProduceErr(tt.err)
			require.Equal(t, tt.want, status.Code(got))
			if tt.err == nil {
				require.NoError(t, got)
				return
			}
			require.Contains(t, got.Error(), tt.err.Error())
		})
	}
}
