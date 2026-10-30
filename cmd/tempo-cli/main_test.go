package main

import (
	"flag"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/cmd/tempo/app"
	"github.com/grafana/tempo/v3/tempodb"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/s3"
)

func TestBackendOptionsApply(t *testing.T) {
	newTraceConfig := func() *tempodb.Config {
		cfg := app.Config{}
		cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})
		return &cfg.StorageConfig.Trace
	}

	t.Run("options override the config", func(t *testing.T) {
		trace := newTraceConfig()
		opts := &backendOptions{
			Backend:            backend.S3Express,
			Bucket:             "tempo--euc1-ist1-az1--x-s3",
			S3Endpoint:         "s3.example.com",
			S3User:             "user",
			S3Pass:             "pass",
			InsecureSkipVerify: true,
		}
		opts.apply(trace)

		assert.Equal(t, backend.S3Express, trace.Backend)
		for _, cfg := range []*s3.Config{trace.S3, &trace.S3Express.Config} {
			assert.Equal(t, "tempo--euc1-ist1-az1--x-s3", cfg.Bucket)
			assert.Equal(t, "s3.example.com", cfg.Endpoint)
			assert.Equal(t, "user", cfg.AccessKey)
			assert.Equal(t, "pass", cfg.SecretKey.String())
			assert.True(t, cfg.InsecureSkipVerify)
		}
		assert.Equal(t, "tempo--euc1-ist1-az1--x-s3", trace.Local.Path)
		assert.Equal(t, "tempo--euc1-ist1-az1--x-s3", trace.GCS.BucketName)
		assert.Equal(t, "tempo--euc1-ist1-az1--x-s3", trace.Azure.ContainerName)
		assert.True(t, trace.GCS.Insecure)
	})

	t.Run("unset options keep the config", func(t *testing.T) {
		trace := newTraceConfig()
		trace.Backend = backend.S3Express
		trace.S3Express.Bucket = "tempo--euc1-ist1-az1--x-s3"
		trace.S3Express.Endpoint = "s3express-euc1-ist1-az1.eu-central-1.amazonaws.com"
		(&backendOptions{}).apply(trace)

		assert.Equal(t, backend.S3Express, trace.Backend)
		assert.Equal(t, "tempo--euc1-ist1-az1--x-s3", trace.S3Express.Bucket)
		assert.Equal(t, "s3express-euc1-ist1-az1.eu-central-1.amazonaws.com", trace.S3Express.Endpoint)
	})
}

func TestBackendFlag(t *testing.T) {
	for _, value := range []string{backend.Local, backend.GCS, backend.S3, backend.S3Express, backend.Azure} {
		t.Run(value, func(t *testing.T) {
			var opts struct {
				backendOptions
			}
			parser, err := kong.New(&opts)
			require.NoError(t, err)

			_, err = parser.Parse([]string{"--backend", value})
			require.NoError(t, err)
			assert.Equal(t, value, opts.Backend)
		})
	}
}
