package otlpgrpc

import (
	"testing"

	"github.com/grafana/dskit/flagext"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	type fields struct {
		Endpoints flagext.StringSlice
		TLS       TLSConfig
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "ReturnsNoErrorForValidInsecureConfig",
			fields: fields{
				Endpoints: nil,
				TLS: TLSConfig{
					Insecure: true,
				},
			},
			wantErr: false,
		},
		{
			name: "ReturnsNoErrorForValidSecureConfig",
			fields: fields{
				Endpoints: nil,
				TLS: TLSConfig{
					Insecure: false,
					CertFile: "/test/path",
				},
			},
			wantErr: false,
		},
		{
			name: "ReturnsErrorWithInsecureFalseAndNoCertFile",
			fields: fields{
				Endpoints: nil,
				TLS: TLSConfig{
					Insecure: false,
					CertFile: "",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Endpoints: tt.fields.Endpoints,
				TLS:       tt.fields.TLS,
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
