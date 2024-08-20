package serverless

import (
	"testing"
	"time"

	"github.com/grafana/tempo/v2/integration/util"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"

	"github.com/grafana/e2e"
	e2e_db "github.com/grafana/e2e/db"

	"github.com/grafana/tempo/v2/pkg/httpclient"
	tempoUtil "github.com/grafana/tempo/v2/pkg/util"
	"github.com/grafana/tempo/v2/tempodb/backend"
)

const (
	configServerlessGCR    = "config-serverless-gcr.yaml"
	configServerlessLambda = "config-serverless-lambda.yaml"
)

func TestServerless(t *testing.T) {
	testClouds := []struct {
		name       string
		serverless *e2e.HTTPService
		config     string
	}{
		{
			name:       "gcr",
			serverless: newTempoServerlessGCR(),
			config:     configServerlessGCR,
		},
		{
			name:       "lambda",
			serverless: newTempoServerlessLambda(),
			config:     configServerlessLambda,
		},
	}

	for _, tc := range testClouds {
		t.Run(tc.name, func(t *testing.T) {
			s, err := e2e.NewScenario("tempo_e2e")
			require.NoError(t, err)
			defer s.Close()

			minio := e2e_db.NewMinio(9000, "tempo")
			require.NotNil(t, minio)
			require.NoError(t, s.StartAndWaitReady(minio))

			require.NoError(t, util.CopyFileToSharedDir(s, tc.config, "config.yaml"))
			tempoIngester1 := util.NewTempoIngester(1)
			tempoIngester2 := util.NewTempoIngester(2)
			tempoIngester3 := util.NewTempoIngester(3)
			tempoDistributor := util.NewTempoDistributor()
			tempoQueryFrontend := util.NewTempoQueryFrontend()
			tempoQuerier := util.NewTempoQuerier()
			tempoServerless := tc.serverless
			require.NoError(t, s.StartAndWaitReady(tempoIngester1, tempoIngester2, tempoIngester3, tempoDistributor, tempoQueryFrontend, tempoQuerier, tempoServerless))

			// wait for 2 active ingesters
			time.Sleep(1 * time.Second)
			matchers := []*labels.Matcher{
				{
					Type:  labels.MatchEqual,
					Name:  "name",
					Value: "ingester",
				},
				{
					Type:  labels.MatchEqual,
					Name:  "state",
					Value: "ACTIVE",
				},
			}
			require.NoError(t, tempoDistributor.WaitSumMetricsWithOptions(e2e.Equals(3), []string{`tempo_ring_members`}, e2e.WithLabelMatchers(matchers...), e2e.WaitMissingMetrics))

			// Get port for the Jaeger gRPC receiver endpoint
			c, err := util.NewJaegerGRPCClient(tempoDistributor.Endpoint(14250))
			require.NoError(t, err)
			require.NotNil(t, c)

			info := tempoUtil.NewTraceInfo(time.Now(), "")
			require.NoError(t, info.EmitAllBatches(c))

			// wait trace_idle_time and ensure trace is created in ingester
			require.NoError(t, tempoIngester1.WaitSumMetricsWithOptions(e2e.Less(3), []string{"tempo_ingester_traces_created_total"}, e2e.WaitMissingMetrics))
			require.NoError(t, tempoIngester2.WaitSumMetricsWithOptions(e2e.Less(3), []string{"tempo_ingester_traces_created_total"}, e2e.WaitMissingMetrics))
			require.NoError(t, tempoIngester3.WaitSumMetricsWithOptions(e2e.Less(3), []string{"tempo_ingester_traces_created_total"}, e2e.WaitMissingMetrics))

			features := []*labels.Matcher{
				{
					Type:  labels.MatchEqual,
					Name:  "feature",
					Value: "search_external_endpoints",
				},
			}
			require.NoError(t, tempoDistributor.WaitSumMetricsWithOptions(e2e.Equals(1), []string{`tempo_feature_enabled`}, e2e.WithLabelMatchers(features...), e2e.WaitMissingMetrics))

			apiClient := httpclient.New("http://"+tempoQueryFrontend.Endpoint(3200), "")

			// flush trace to backend
			res, err := e2e.DoGet("http://" + tempoIngester1.Endpoint(3200) + "/flush")
			require.NoError(t, err)
			require.Equal(t, 204, res.StatusCode)

			res, err = e2e.DoGet("http://" + tempoIngester2.Endpoint(3200) + "/flush")
			require.NoError(t, err)
			require.Equal(t, 204, res.StatusCode)

			res, err = e2e.DoGet("http://" + tempoIngester3.Endpoint(3200) + "/flush")
			require.NoError(t, err)
			require.Equal(t, 204, res.StatusCode)

			// zzz
			time.Sleep(10 * time.Second)

			// search the backend. this works b/c we're passing a start/end AND setting query ingesters within min/max to 0
			now := time.Now()
			util.SearchAndAssertTraceBackend(t, apiClient, info, now.Add(-20*time.Minute).Unix(), now.Unix())
		})
	}
}

func newTempoServerlessGCR() *e2e.HTTPService {
	s := e2e.NewHTTPService(
		"serverless",
		"tempo-serverless", // created by Makefile in /cmd/tempo-serverless
		nil,
		nil,
		8080,
	)

	s.SetEnvVars(map[string]string{
		"TEMPO_S3_BUCKET":     "tempo",
		"TEMPO_S3_ENDPOINT":   "tempo_e2e-minio-9000:9000",
		"TEMPO_S3_ACCESS_KEY": e2e_db.MinioAccessKey,
		"TEMPO_S3_SECRET_KEY": e2e_db.MinioSecretKey,
		"TEMPO_S3_INSECURE":   "true",
		"TEMPO_BACKEND":       backend.S3,
	})

	s.SetBackoff(util.TempoBackoff())

	return s
}

func newTempoServerlessLambda() *e2e.HTTPService {
	s := e2e.NewHTTPService(
		"serverless",
		"tempo-serverless-lambda", // created by build-docker-lambda-test make target
		nil,
		nil,
		9000,
	)

	s.SetEnvVars(map[string]string{
		"TEMPO_S3_BUCKET":     "tempo",
		"TEMPO_S3_ENDPOINT":   "tempo_e2e-minio-9000:9000",
		"TEMPO_S3_ACCESS_KEY": e2e_db.MinioAccessKey,
		"TEMPO_S3_SECRET_KEY": e2e_db.MinioSecretKey,
		"TEMPO_S3_INSECURE":   "true",
		"TEMPO_BACKEND":       backend.S3,
	})

	s.SetBackoff(util.TempoBackoff())

	return s
}
