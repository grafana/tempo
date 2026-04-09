package forwarder

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v2/modules/distributor/forwarder/otlpgrpc"
)

func TestConfig_Validate(t *testing.T) {
	type fields struct {
		Name     string
		Backend  string
		OTLPGRPC otlpgrpc.Config
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "ReturnsNoErrorWithValidArguments",
			fields: fields{
				Name:    "test",
				Backend: OTLPGRPCBackend,
				OTLPGRPC: otlpgrpc.Config{
					Endpoints: nil,
					TLS: otlpgrpc.TLSConfig{
						Insecure: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "ReturnsErrorWithEmptyName",
			fields: fields{
				Name:    "",
				Backend: OTLPGRPCBackend,
				OTLPGRPC: otlpgrpc.Config{
					Endpoints: nil,
					TLS: otlpgrpc.TLSConfig{
						Insecure: false,
						CertFile: "",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "ReturnsErrorWithUnsupportedBackendName",
			fields: fields{
				Name:    "test",
				Backend: "unsupported",
				OTLPGRPC: otlpgrpc.Config{
					Endpoints: nil,
					TLS: otlpgrpc.TLSConfig{
						Insecure: true,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Name:     tt.fields.Name,
				Backend:  tt.fields.Backend,
				OTLPGRPC: tt.fields.OTLPGRPC,
			}

			err := cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
