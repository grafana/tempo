package ingest_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/flagext"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/grafana/tempo/pkg/ingest"
)

// TestKafkaClient_MTLS_SCRAM_RoundTrip is an end-to-end test that exercises the
// Kafka client authentication paths against a real (fake) broker configured for
// mutual TLS and SASL/SCRAM-SHA-256. It produces a record with NewWriterClient
// and consumes it back with NewReaderClient, proving that both the TLS transport
// (including client-certificate mTLS) and the SCRAM mechanism are wired up
// correctly.
func TestKafkaClient_MTLS_SCRAM_RoundTrip(t *testing.T) {
	const (
		topic    = "mtls-scram-topic"
		saslUser = "tempo"
		saslPass = "tempo-secret"
	)

	certs := generateTestCerts(t)

	// Server TLS config: present the server cert and require a client cert
	// signed by our CA (mTLS).
	serverTLS := &tls.Config{
		Certificates: []tls.Certificate{certs.serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certs.caPool,
		MinVersion:   tls.VersionTLS12,
	}

	fake, err := kfake.NewCluster(
		kfake.NumBrokers(1),
		kfake.SeedTopics(1, topic),
		kfake.TLS(serverTLS),
		kfake.EnableSASL(),
		kfake.Superuser("SCRAM-SHA-256", saslUser, saslPass),
	)
	require.NoError(t, err)
	t.Cleanup(fake.Close)

	address := fake.ListenAddrs()[0]

	cfg := mtlsScramKafkaConfig(t, address, topic, certs, saslUser, saslPass)
	require.NoError(t, cfg.Validate())

	logger := log.NewNopLogger()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Produce a record over mTLS + SCRAM.
	writer, err := ingest.NewWriterClient(cfg, 1, logger, prometheus.NewPedanticRegistry())
	require.NoError(t, err)
	t.Cleanup(writer.Close)

	want := []byte("hello over mtls and scram")
	res := writer.ProduceSync(ctx, &kgo.Record{
		Topic:     topic,
		Partition: 0,
		Value:     want,
	})
	require.NoError(t, res.FirstErr(), "producing over mTLS+SCRAM should succeed")

	// Consume the record back with a fresh client that must independently
	// authenticate over mTLS + SCRAM.
	reader, err := ingest.NewReaderClient(
		cfg,
		ingest.NewReaderClientMetrics("test", prometheus.NewPedanticRegistry()),
		logger,
		kgo.ConsumeTopics(topic),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	)
	require.NoError(t, err)
	t.Cleanup(reader.Close)

	var got []byte
	require.Eventually(t, func() bool {
		fetches := reader.PollFetches(ctx)
		if err := fetches.Err(); err != nil {
			return false
		}
		iter := fetches.RecordIter()
		if iter.Done() {
			return false
		}
		got = iter.Next().Value
		return true
	}, 20*time.Second, 100*time.Millisecond, "should consume the produced record")

	require.Equal(t, want, got)
}

// TestKafkaClient_MTLS_MissingClientCert verifies that the broker rejects a
// client that does not present a certificate when mTLS is required. This proves
// the mTLS requirement is actually enforced end-to-end rather than TLS being a
// no-op.
func TestKafkaClient_MTLS_MissingClientCert(t *testing.T) {
	const (
		topic    = "mtls-required-topic"
		saslUser = "tempo"
		saslPass = "tempo-secret"
	)

	certs := generateTestCerts(t)

	serverTLS := &tls.Config{
		Certificates: []tls.Certificate{certs.serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certs.caPool,
		MinVersion:   tls.VersionTLS12,
	}

	fake, err := kfake.NewCluster(
		kfake.NumBrokers(1),
		kfake.SeedTopics(1, topic),
		kfake.TLS(serverTLS),
		kfake.EnableSASL(),
		kfake.Superuser("SCRAM-SHA-256", saslUser, saslPass),
	)
	require.NoError(t, err)
	t.Cleanup(fake.Close)

	address := fake.ListenAddrs()[0]

	// Configure a client that trusts the CA and has SCRAM credentials but does
	// NOT present a client certificate.
	cfg := mtlsScramKafkaConfig(t, address, topic, certs, saslUser, saslPass)
	cfg.TLS.CertPath = ""
	cfg.TLS.KeyPath = ""
	// Fail fast: a rejected mTLS handshake should surface quickly rather than
	// retrying up to the default write timeout.
	cfg.WriteTimeout = 3 * time.Second
	require.NoError(t, cfg.Validate())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	writer, err := ingest.NewWriterClient(cfg, 1, log.NewNopLogger(), prometheus.NewPedanticRegistry())
	require.NoError(t, err)
	t.Cleanup(writer.Close)

	res := writer.ProduceSync(ctx, &kgo.Record{Topic: topic, Partition: 0, Value: []byte("nope")})
	require.Error(t, res.FirstErr(), "producing without a client certificate must fail when mTLS is required")
}

func mtlsScramKafkaConfig(t *testing.T, address, topic string, certs testCerts, user, pass string) ingest.KafkaConfig {
	t.Helper()

	cfg := ingest.KafkaConfig{}
	flagext.DefaultValues(&cfg)
	cfg.Address = address
	cfg.Topic = topic
	cfg.WriteTimeout = 10 * time.Second

	// SASL/SCRAM-SHA-256.
	cfg.SASL.Mechanism = ingest.SASLMechanismScramSHA256
	cfg.SASL.Username = user
	cfg.SASL.Password = flagext.SecretWithValue(pass)

	// mTLS: present our client cert and trust the CA that signed the broker.
	cfg.TLSEnabled = true
	cfg.TLS.CertPath = certs.clientCertPath
	cfg.TLS.KeyPath = certs.clientKeyPath
	cfg.TLS.CAPath = certs.caPath
	cfg.TLS.ServerName = "localhost"

	return cfg
}

type testCerts struct {
	caPool *x509.CertPool
	caPath string

	serverCert tls.Certificate

	clientCertPath string
	clientKeyPath  string
}

// generateTestCerts creates a self-signed CA and issues a server and a client
// certificate from it. The CA and client cert/key are written to files (as the
// dskit TLS config loads them from disk); the server cert is returned in-memory
// for the kfake broker.
func generateTestCerts(t *testing.T) testCerts {
	t.Helper()

	dir := t.TempDir()

	// Certificate authority.
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "tempo-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	require.NoError(t, err)
	caCert, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)

	caPool := x509.NewCertPool()
	caPool.AddCert(caCert)

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	caPath := filepath.Join(dir, "ca.pem")
	require.NoError(t, os.WriteFile(caPath, caPEM, 0o600))

	// issueCert signs a leaf certificate with the CA.
	issueCert := func(cn string, isServer bool) (certPEM, keyPEM []byte) {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		tmpl := &x509.Certificate{
			SerialNumber: big.NewInt(time.Now().UnixNano()),
			Subject:      pkix.Name{CommonName: cn},
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(24 * time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		}
		if isServer {
			tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
			tmpl.DNSNames = []string{"localhost"}
			tmpl.IPAddresses = []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}
		} else {
			tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
		}

		der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
		require.NoError(t, err)

		certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
		keyDER, err := x509.MarshalPKCS8PrivateKey(key)
		require.NoError(t, err)
		keyPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
		return certPEM, keyPEM
	}

	// Server certificate (kept in-memory for the broker).
	serverCertPEM, serverKeyPEM := issueCert("localhost", true)
	serverCert, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	require.NoError(t, err)

	// Client certificate (written to disk for the dskit TLS config).
	clientCertPEM, clientKeyPEM := issueCert("tempo-client", false)
	clientCertPath := filepath.Join(dir, "client.pem")
	clientKeyPath := filepath.Join(dir, "client-key.pem")
	require.NoError(t, os.WriteFile(clientCertPath, clientCertPEM, 0o600))
	require.NoError(t, os.WriteFile(clientKeyPath, clientKeyPEM, 0o600))

	return testCerts{
		caPool:         caPool,
		caPath:         caPath,
		serverCert:     serverCert,
		clientCertPath: clientCertPath,
		clientKeyPath:  clientKeyPath,
	}
}
