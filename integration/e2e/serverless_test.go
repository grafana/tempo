package e2e

import (
	"testing"
	"time"

	cortex_e2e "github.com/cortexproject/cortex/integration/e2e"
	cortex_e2e_db "github.com/cortexproject/cortex/integration/e2e/db"
	util "github.com/grafana/tempo/integration"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
)

func TestServerless(t *testing.T) {
	s, err := cortex_e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	minio := cortex_e2e_db.NewMinio(9000, "tempo")
	require.NotNil(t, minio)
	require.NoError(t, s.StartAndWaitReady(minio))

	require.NoError(t, util.CopyFileToSharedDir(s, configServerless, "config.yaml"))
	tempoIngester1 := util.NewTempoIngester(1)
	tempoIngester2 := util.NewTempoIngester(2)
	tempoIngester3 := util.NewTempoIngester(3)
	tempoDistributor := util.NewTempoDistributor()
	tempoQueryFrontend := util.NewTempoQueryFrontend()
	tempoQuerier := util.NewTempoQuerier()
	tempoServerless := newTempoServerless()
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
	require.NoError(t, tempoDistributor.WaitSumMetricsWithOptions(cortex_e2e.Equals(3), []string{`cortex_ring_members`}, cortex_e2e.WithLabelMatchers(matchers...), cortex_e2e.WaitMissingMetrics))

	// Get port for the Jaeger gRPC receiver endpoint
	c, err := newJaegerGRPCClient(tempoDistributor.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, c)

	info := tempoUtil.NewTraceInfo(time.Now(), "")
	require.NoError(t, info.EmitAllBatches(c))

	// ensure trace is created in ingester (trace_idle_time has passed)
	require.NoError(t, tempoIngester1.WaitSumMetrics(cortex_e2e.Greater(0), "tempo_ingester_traces_created_total"))
	require.NoError(t, tempoIngester2.WaitSumMetrics(cortex_e2e.Greater(0), "tempo_ingester_traces_created_total"))
	require.NoError(t, tempoIngester3.WaitSumMetrics(cortex_e2e.Greater(0), "tempo_ingester_traces_created_total"))

	apiClient := tempoUtil.NewClient("http://"+tempoQueryFrontend.Endpoint(3200), "")

	// flush trace to backend
	res, err := cortex_e2e.GetRequest("http://" + tempoIngester1.Endpoint(3200) + "/flush")
	require.NoError(t, err)
	require.Equal(t, 204, res.StatusCode)

	res, err = cortex_e2e.GetRequest("http://" + tempoIngester2.Endpoint(3200) + "/flush")
	require.NoError(t, err)
	require.Equal(t, 204, res.StatusCode)

	res, err = cortex_e2e.GetRequest("http://" + tempoIngester3.Endpoint(3200) + "/flush")
	require.NoError(t, err)
	require.Equal(t, 204, res.StatusCode)

	// zzz
	time.Sleep(10 * time.Second)

	// search the backend. this works b/c we're passing a start/end AND setting query ingesters within min/max to 0
	now := time.Now()
	searchAndAssertTraceBackend(t, apiClient, info, now.Add(-20*time.Minute).Unix(), now.Unix())
}

func newTempoServerless() *cortex_e2e.HTTPService {
	s := cortex_e2e.NewHTTPService(
		"serverless",
		"tempo-serverless", // created by buildpacks in ./cmd/tempo-serverless
		nil,
		nil,
		8080,
	)

	s.SetEnvVars(map[string]string{
		"TEMPO_S3_BUCKET":     "tempo",
		"TEMPO_S3_ENDPOINT":   "tempo_e2e-minio-9000:9000",
		"TEMPO_S3_ACCESS_KEY": cortex_e2e_db.MinioAccessKey,
		"TEMPO_S3_SECRET_KEY": cortex_e2e_db.MinioSecretKey,
		"TEMPO_S3_INSECURE":   "true",
		"TEMPO_BACKEND":       "s3",
	})

	s.SetBackoff(util.TempoBackoff())

	return s
}
