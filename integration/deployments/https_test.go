package deployments // jpe - rename to operations?

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/grafana/e2e"
	e2edb "github.com/grafana/e2e/db"
	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/httpclient"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"
)

const (
	configHTTPS = "config-https.yaml"

	tempoPort = 3200
)

// jpe - currently just running the single binary directly. should we make a special TestHarness for this? or an option to do this?
// can we add an option to query https metrics to the e2e framework?
func TestHTTPS(t *testing.T) {
	km := util.SetupCertificates(t)

	s, err := e2e.NewScenario("tempo_e2e_test_https")
	require.NoError(t, err)
	defer s.Close()

	kafka := e2edb.NewKafka()
	require.NoError(t, s.StartAndWaitReady(kafka), "failed to start Kafka")

	// copy in certs
	require.NoError(t, util.CopyFileToSharedDir(s, km.ServerCertFile, "tls.crt"))
	require.NoError(t, util.CopyFileToSharedDir(s, km.ServerKeyFile, "tls.key"))
	require.NoError(t, util.CopyFileToSharedDir(s, km.CaCertFile, "ca.crt"))

	require.NoError(t, util.CopyFileToSharedDir(s, configHTTPS, "config.yaml"))
	tempo := util.NewTempoAllInOneWithReadinessProbe(e2e.NewHTTPReadinessProbe(3201, "/ready", 200, 299))
	require.NoError(t, s.StartAndWaitReady(tempo))

	c, err := util.NewJaegerToOTLPExporter(tempo.Endpoint(4317))
	require.NoError(t, err)
	require.NotNil(t, c)

	// the test harness queries a metric to determine when the partition ring is
	// ready for writes, but there's no convenient way in the e2e framework to do this over
	// HTTPS, so instead we just try for a bit until a write works - jpe - add a simpler helper method in this file to do this and check for the right things
	var info *tempoUtil.TraceInfo
	require.Eventually(t, func() bool {
		info = tempoUtil.NewTraceInfo(time.Now(), "")
		err := info.EmitAllBatches(c)
		if err != nil {
			return false
		}
		return true
	}, time.Minute, 5*time.Second, "could not write trace to tempo")

	apiClient := httpclient.New("https://"+tempo.Endpoint(tempoPort), "")

	// trust bad certs
	defaultTransport := http.DefaultTransport.(*http.Transport).Clone()
	defaultTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	apiClient.WithTransport(defaultTransport)

	echoReq, err := http.NewRequest("GET", "https://"+tempo.Endpoint(tempoPort)+"/api/echo", nil)
	require.NoError(t, err)
	resp, err := apiClient.Do(echoReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// query an in-memory trace
	util.QueryAndAssertTrace(t, apiClient, info)

	// wait for the trace to be flushed to a wal block for querying. jpe - similar to above. can't use a metric :(
	time.Sleep(30 * time.Second)

	util.SearchTraceQLAndAssertTrace(t, apiClient, info)

	creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
	grpcClient, err := util.NewSearchGRPCClientWithCredentials(context.Background(), tempo.Endpoint(tempoPort), creds)
	require.NoError(t, err)

	now := time.Now()
	util.SearchStreamAndAssertTrace(t, context.Background(), grpcClient, info, now.Add(-time.Hour).Unix(), now.Unix()) // jpe - add grpc client to harness
}
