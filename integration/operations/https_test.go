package deployments

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grafana/dskit/runutil"
	"github.com/grafana/e2e"
	"github.com/grafana/tempo/integration/util"
	"github.com/grafana/tempo/pkg/httpclient"
	tempoUtil "github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"
)

const (
	configHTTPS = "config-https.yaml"

	tempoPort = 3200 // magically matches the port in config-base.yaml
)

// TestHTTPS tests the use of unsigned certs with Tempo. Due to this we run the special "internal server"
// on port 3201 which requires us to pass custome readiness probe. Additionally we have to create custom
// a custom API client that uses https, but doesn't validate the certs.
// Finally note that we actually push over an unencrypted connection, using the default harness functions.
// This works b/c the TLS configuration for ingestion is configured through the OTEL receiver config.
func TestHTTPS(t *testing.T) {
	km := setupCertificates(t)

	util.WithTempoHarness(t, util.TestHarnessConfig{
		ConfigOverlay:  configHTTPS,
		ReadinessProbe: e2e.NewHTTPReadinessProbe(3201, "/ready", 200, 299), // this works b/c the service creation code in ../util/services.go adds a 3201 port to the services. we could also use a custom readiness probe.
		PreStartHook: func(s *e2e.Scenario, _ map[string]any) error {
			require.NoError(t, util.CopyFileToSharedDir(s, km.ServerCertFile, "tls.crt"))
			require.NoError(t, util.CopyFileToSharedDir(s, km.ServerKeyFile, "tls.key"))
			require.NoError(t, util.CopyFileToSharedDir(s, km.CaCertFile, "ca.crt"))

			return nil
		},
	}, func(h *util.TempoHarness) {
		// wait for traces to be writable
		require.True(t, scrapeMetrics(t, h.Services[util.ServiceDistributor], tempoPort, "tempo_partition_ring_partitions{name=\"livestore-partitions\",state=\"Active\"} 1"))

		// write a trace
		info := tempoUtil.NewTraceInfo(time.Now(), "")
		require.NoError(t, h.WriteTraceInfo(info, ""))

		queryFrontend := h.Services[util.ServiceQueryFrontend]
		apiClient := httpclient.New("https://"+queryFrontend.Endpoint(tempoPort), "")

		// trust bad certs
		defaultTransport := http.DefaultTransport.(*http.Transport).Clone()
		defaultTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		apiClient.WithTransport(defaultTransport)

		util.QueryAndAssertTrace(t, apiClient, info)

		// wait for the traces to be queryable
		require.True(t, scrapeMetrics(t, h.Services[util.ServiceLiveStoreZoneA], tempoPort, "tempo_live_store_traces_created_total{tenant=\"single-tenant\"} 1"))
		require.True(t, scrapeMetrics(t, h.Services[util.ServiceLiveStoreZoneB], tempoPort, "tempo_live_store_traces_created_total{tenant=\"single-tenant\"} 1"))

		util.SearchTraceQLAndAssertTrace(t, apiClient, info)

		creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
		grpcClient, err := util.NewSearchGRPCClient(context.Background(), queryFrontend.Endpoint(tempoPort), creds)
		require.NoError(t, err)

		now := time.Now()
		util.SearchStreamAndAssertTrace(t, context.Background(), grpcClient, info, now.Add(-time.Hour).Unix(), now.Unix())
	})
}

type keyMaterial struct {
	CaCertFile                string
	ServerCertFile            string
	ServerKeyFile             string
	ServerNoLocalhostCertFile string
	ServerNoLocalhostKeyFile  string
	ClientCA1CertFile         string
	ClientCABothCertFile      string
	Client1CertFile           string
	Client1KeyFile            string
	Client2CertFile           string
	Client2KeyFile            string
}

func setupCertificates(t *testing.T) keyMaterial {
	testCADir := t.TempDir()

	// create server side CA

	testCA := newCA("Test")
	caCertFile := filepath.Join(testCADir, "ca.crt")
	require.NoError(t, testCA.writeCACertificate(caCertFile))

	serverCertFile := filepath.Join(testCADir, "server.crt")
	serverKeyFile := filepath.Join(testCADir, "server.key")
	require.NoError(t, testCA.writeCertificate(
		&x509.Certificate{
			Subject:     pkix.Name{CommonName: "server"},
			DNSNames:    []string{"localhost", "my-other-name"},
			ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		},
		serverCertFile,
		serverKeyFile,
	))

	serverNoLocalhostCertFile := filepath.Join(testCADir, "server-no-localhost.crt")
	serverNoLocalhostKeyFile := filepath.Join(testCADir, "server-no-localhost.key")
	require.NoError(t, testCA.writeCertificate(
		&x509.Certificate{
			Subject:     pkix.Name{CommonName: "server-no-localhost"},
			DNSNames:    []string{"my-other-name"},
			ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		},
		serverNoLocalhostCertFile,
		serverNoLocalhostKeyFile,
	))

	// create client CAs
	testClientCA1 := newCA("Test Client CA 1")
	testClientCA2 := newCA("Test Client CA 2")

	clientCA1CertFile := filepath.Join(testCADir, "ca-client-1.crt")
	require.NoError(t, testClientCA1.writeCACertificate(clientCA1CertFile))
	clientCA2CertFile := filepath.Join(testCADir, "ca-client-2.crt")
	require.NoError(t, testClientCA2.writeCACertificate(clientCA2CertFile))

	// create a ca file with both certs
	clientCABothCertFile := filepath.Join(testCADir, "ca-client-both.crt")
	func() {
		src1, err := os.Open(clientCA1CertFile)
		require.NoError(t, err)
		defer src1.Close()
		src2, err := os.Open(clientCA2CertFile)
		require.NoError(t, err)
		defer src2.Close()

		dst, err := os.Create(clientCABothCertFile)
		require.NoError(t, err)
		defer dst.Close()

		_, err = io.Copy(dst, src1)
		require.NoError(t, err)
		_, err = io.Copy(dst, src2)
		require.NoError(t, err)
	}()

	client1CertFile := filepath.Join(testCADir, "client-1.crt")
	client1KeyFile := filepath.Join(testCADir, "client-1.key")
	require.NoError(t, testClientCA1.writeCertificate(
		&x509.Certificate{
			Subject:     pkix.Name{CommonName: "client-1"},
			ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		},
		client1CertFile,
		client1KeyFile,
	))

	client2CertFile := filepath.Join(testCADir, "client-2.crt")
	client2KeyFile := filepath.Join(testCADir, "client-2.key")
	require.NoError(t, testClientCA2.writeCertificate(
		&x509.Certificate{
			Subject:     pkix.Name{CommonName: "client-2"},
			ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		},
		client2CertFile,
		client2KeyFile,
	))

	return keyMaterial{
		CaCertFile:                caCertFile,
		ServerCertFile:            serverCertFile,
		ServerKeyFile:             serverKeyFile,
		ServerNoLocalhostCertFile: serverNoLocalhostCertFile,
		ServerNoLocalhostKeyFile:  serverNoLocalhostKeyFile,
		ClientCA1CertFile:         clientCA1CertFile,
		ClientCABothCertFile:      clientCABothCertFile,
		Client1CertFile:           client1CertFile,
		Client1KeyFile:            client1KeyFile,
		Client2CertFile:           client2CertFile,
		Client2KeyFile:            client2KeyFile,
	}
}

type ca struct {
	key    *ecdsa.PrivateKey
	cert   *x509.Certificate
	serial *big.Int
}

func newCA(name string) *ca {
	key, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		panic(err)
	}

	return &ca{
		key: key,
		cert: &x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject: pkix.Name{
				Organization: []string{name},
			},
			NotBefore: time.Now(),
			NotAfter:  time.Now().Add(time.Hour * 24 * 180),

			KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
			BasicConstraintsValid: true,
			IsCA:                  true,
		},
		serial: big.NewInt(2),
	}
}

func writeExclusivePEMFile(path, marker string, mode os.FileMode, data []byte) (err error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	defer runutil.CloseWithErrCapture(&err, f, "write pem file")

	return pem.Encode(f, &pem.Block{Type: marker, Bytes: data})
}

func (ca *ca) writeCACertificate(path string) error {
	derBytes, err := x509.CreateCertificate(rand.Reader, ca.cert, ca.cert, ca.key.Public(), ca.key)
	if err != nil {
		return err
	}

	return writeExclusivePEMFile(path, "CERTIFICATE", 0o600, derBytes)
}

func (ca *ca) writeCertificate(template *x509.Certificate, certPath string, keyPath string) error {
	key, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		return err
	}

	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}

	if err := writeExclusivePEMFile(keyPath, "PRIVATE KEY", 0o600, keyBytes); err != nil {
		return err
	}

	template.IsCA = false
	template.NotBefore = time.Now()
	if template.NotAfter.IsZero() {
		template.NotAfter = time.Now().Add(time.Hour * 24 * 180)
	}
	template.SerialNumber = ca.serial.Add(ca.serial, big.NewInt(1))

	derBytes, err := x509.CreateCertificate(rand.Reader, template, ca.cert, key.Public(), ca.key)
	if err != nil {
		return err
	}

	return writeExclusivePEMFile(certPath, "CERTIFICATE", 0o600, derBytes)
}

func scrapeMetrics(t *testing.T, service *e2e.HTTPService, port int, searchString string) bool {
	t.Helper()

	// create HTTPS client with insecure skip verify
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: util.MetricsTimeout,
	}

	found := false

	require.Eventually(t, func() bool {
		url := "https://" + service.Endpoint(port) + "/metrics"
		resp, err := client.Get(url)
		if err != nil {
			t.Logf("failed to scrape metrics: %v", err)
			return false
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Logf("unexpected status code: %d", resp.StatusCode)
			return false
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Logf("failed to read response body: %v", err)
			return false
		}

		found = bytes.Contains(body, []byte(searchString))
		return found
	}, time.Minute, time.Second, "could not write trace to tempo")

	return found
}
