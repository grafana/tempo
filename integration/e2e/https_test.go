package e2e

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/v2/cmd/tempo/app"
	"github.com/grafana/tempo/v2/integration/e2e/backend"
	e2e_ca "github.com/grafana/tempo/v2/integration/e2e/ca"
	"github.com/grafana/tempo/v2/integration/util"
	"github.com/grafana/tempo/v2/pkg/httpclient"
	tempoUtil "github.com/grafana/tempo/v2/pkg/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"
	"gopkg.in/yaml.v2"
)

const (
	configHTTPS = "config-https.yaml"
)

func TestHTTPS(t *testing.T) {
	km := e2e_ca.SetupCertificates(t)

	s, err := e2e.NewScenario("tempo_e2e")
	require.NoError(t, err)
	defer s.Close()

	// set up the backend
	cfg := app.Config{}
	buff, err := os.ReadFile(configHTTPS)
	require.NoError(t, err)
	err = yaml.UnmarshalStrict(buff, &cfg)
	require.NoError(t, err)
	_, err = backend.New(s, cfg)
	require.NoError(t, err)

	// copy in certs
	require.NoError(t, util.CopyFileToSharedDir(s, km.ServerCertFile, "tls.crt"))
	require.NoError(t, util.CopyFileToSharedDir(s, km.ServerKeyFile, "tls.key"))
	require.NoError(t, util.CopyFileToSharedDir(s, km.CaCertFile, "ca.crt"))

	require.NoError(t, util.CopyFileToSharedDir(s, configHTTPS, "config.yaml"))
	tempo := util.NewTempoAllInOneWithReadinessProbe(e2e.NewHTTPReadinessProbe(3201, "/ready", 200, 299))
	require.NoError(t, s.StartAndWaitReady(tempo))

	// Get port for the Jaeger gRPC receiver endpoint
	c, err := util.NewJaegerGRPCClient(tempo.Endpoint(14250))
	require.NoError(t, err)
	require.NotNil(t, c)

	time.Sleep(10 * time.Second)

	info := tempoUtil.NewTraceInfo(time.Now(), "")
	require.NoError(t, info.EmitAllBatches(c))

	apiClient := httpclient.New("https://"+tempo.Endpoint(3200), "")

	// trust bad certs
	defaultTransport := http.DefaultTransport.(*http.Transport).Clone()
	defaultTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	apiClient.WithTransport(defaultTransport)

	echoReq, err := http.NewRequest("GET", "https://"+tempo.Endpoint(3200)+"/api/echo", nil)
	require.NoError(t, err)
	resp, err := apiClient.Do(echoReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// query an in-memory trace
	util.QueryAndAssertTrace(t, apiClient, info)
	util.SearchAndAssertTrace(t, apiClient, info)
	util.SearchTraceQLAndAssertTrace(t, apiClient, info)

	creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
	grpcClient, err := util.NewSearchGRPCClientWithCredentials(context.Background(), tempo.Endpoint(3200), creds)
	require.NoError(t, err)

	now := time.Now()
	util.SearchStreamAndAssertTrace(t, context.Background(), grpcClient, info, now.Add(-time.Hour).Unix(), now.Add(time.Hour).Unix())
}
