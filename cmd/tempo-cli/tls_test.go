package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alecthomas/kong"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func TestTLSOptionsValidation(t *testing.T) {
	invalid := filepath.Join(t.TempDir(), "invalid.pem")
	require.NoError(t, os.WriteFile(invalid, []byte("not a certificate"), 0o600))
	missing := filepath.Join(t.TempDir(), "missing.pem")

	tests := []struct {
		name    string
		options tlsOptions
		secure  bool
		wantErr string
	}{
		{name: "plaintext defaults"},
		{name: "TLS defaults", secure: true},
		{name: "certificate without key", secure: true, options: tlsOptions{TLSCert: invalid}, wantErr: "--tls-cert and --tls-key must be provided together"},
		{name: "key without certificate", secure: true, options: tlsOptions{TLSKey: invalid}, wantErr: "--tls-cert and --tls-key must be provided together"},
		{name: "certificate on plaintext", options: tlsOptions{TLSCert: invalid, TLSKey: invalid}, wantErr: "require TLS"},
		{name: "CA on plaintext", options: tlsOptions{TLSCA: invalid}, wantErr: "require TLS"},
		{name: "server name on plaintext", options: tlsOptions{TLSServerName: "tempo.example.com"}, wantErr: "require TLS"},
		{name: "missing CA", secure: true, options: tlsOptions{TLSCA: missing}, wantErr: "reading CA certificate"},
		{name: "invalid CA", secure: true, options: tlsOptions{TLSCA: invalid}, wantErr: "no valid certificates"},
		{name: "missing certificate", secure: true, options: tlsOptions{TLSCert: missing, TLSKey: invalid}, wantErr: "loading client certificate and key"},
		{name: "invalid certificate", secure: true, options: tlsOptions{TLSCert: invalid, TLSKey: invalid}, wantErr: "loading client certificate and key"},
		{name: "server name only", secure: true, options: tlsOptions{TLSServerName: "tempo.example.com"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := tt.options.tlsConfig(tt.secure)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			if !tt.secure {
				require.Nil(t, cfg)
				return
			}
			require.NotNil(t, cfg)
			require.False(t, cfg.InsecureSkipVerify)
			require.Equal(t, tt.options.TLSServerName, cfg.ServerName)
		})
	}
}

type tlsTestPKI struct {
	options tlsOptions
	server  *tls.Config
}

// Generate short-lived certificates at test time so fixtures never expire in the repository.
func newTLSTestPKI(t *testing.T) tlsTestPKI {
	t.Helper()
	dir := t.TempDir()
	issue := func(name string, template, parent *x509.Certificate, signer *ecdsa.PrivateKey) tls.Certificate {
		t.Helper()
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		if parent == nil {
			parent, signer = template, key
		}
		template.NotBefore = time.Now().Add(-time.Hour)
		template.NotAfter = time.Now().Add(time.Hour)
		der, err := x509.CreateCertificate(rand.Reader, template, parent, &key.PublicKey, signer)
		require.NoError(t, err)
		certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
		keyDER, err := x509.MarshalECPrivateKey(key)
		require.NoError(t, err)
		keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
		require.NoError(t, os.WriteFile(filepath.Join(dir, name+".crt"), certPEM, 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, name+".key"), keyPEM, 0o600))
		cert, err := tls.X509KeyPair(certPEM, keyPEM)
		require.NoError(t, err)
		cert.Leaf, err = x509.ParseCertificate(der)
		require.NoError(t, err)
		return cert
	}
	ca := issue("ca", &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "tempo-cli test CA"},
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}, nil, nil)
	caKey := ca.PrivateKey.(*ecdsa.PrivateKey)
	server := issue("server", &x509.Certificate{
		SerialNumber: big.NewInt(2), DNSNames: []string{"tempo.test"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}, ca.Leaf, caKey)
	issue("client", &x509.Certificate{
		SerialNumber: big.NewInt(3), Subject: pkix.Name{CommonName: "tempo-cli test client"},
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}, ca.Leaf, caKey)
	pool := x509.NewCertPool()
	pool.AddCert(ca.Leaf)
	return tlsTestPKI{
		options: tlsOptions{TLSCert: filepath.Join(dir, "client.crt"), TLSKey: filepath.Join(dir, "client.key"), TLSCA: filepath.Join(dir, "ca.crt")},
		server: &tls.Config{
			MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{server},
			ClientCAs: pool, ClientAuth: tls.RequireAndVerifyClientCert,
		},
	}
}

func runTLSQuery(t *testing.T, args ...string) error {
	t.Helper()
	commands := cli
	parser, err := kong.New(&commands, kong.Writers(io.Discard, io.Discard))
	require.NoError(t, err)
	ctx, err := parser.Parse(append([]string{"query", "api"}, args...))
	if err != nil {
		return err
	}
	return ctx.Run(&commands.globalOptions)
}

func TestQueryAPIMTLSHTTP(t *testing.T) {
	pki := newTLSTestPKI(t)
	for _, secure := range []bool{false, true} {
		for _, tc := range []struct {
			name  string
			args  []string
			path  string
			query string
		}{
			{name: "trace-v1", args: []string{"trace-id", "--v1"}, path: "/api/traces/1234"},
			{name: "trace-v2", args: []string{"trace-id", "--q={}"}, path: "/api/v2/traces/1234", query: "{}"},
			{name: "search", args: []string{"search"}, path: "/api/search", query: "{}"},
			{name: "tags", args: []string{"search-tags"}, path: "/api/v2/search/tags"},
			{name: "tag-values", args: []string{"search-tag-values", "--query={}"}, path: "/api/v2/search/tag/resource.service.name/values", query: "{}"},
			{name: "metrics-range", args: []string{"metrics"}, path: "/api/metrics/query_range", query: "{} | rate()"},
			{name: "metrics-instant", args: []string{"metrics", "--instant"}, path: "/api/metrics/query", query: "{} | rate()"},
		} {
			t.Run(httpScheme(secure)+"/"+tc.name, func(t *testing.T) {
				called := make(chan struct{}, 1)
				srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					called <- struct{}{}
					assert.Equal(t, "/tempo"+tc.path, r.URL.Path)
					assert.Equal(t, tc.query, r.URL.Query().Get("q"))
					assert.Equal(t, "test-tenant", r.Header.Get("X-Scope-OrgID"))
					assert.Equal(t, "Bearer token=abc", r.Header.Get("Authorization"))
					if secure {
						assert.Equal(t, "tempo.test", r.TLS.ServerName)
						if assert.NotEmpty(t, r.TLS.VerifiedChains) {
							assert.Equal(t, "tempo-cli test client", r.TLS.PeerCertificates[0].Subject.CommonName)
						}
					}
					if tc.name == "trace-v1" || tc.name == "trace-v2" {
						assert.Equal(t, "application/protobuf", r.Header.Get("Accept"))
						var response proto.Message = &tempopb.Trace{}
						if tc.name == "trace-v2" {
							response = &tempopb.TraceByIDResponse{Trace: &tempopb.Trace{}}
						}
						body, err := proto.Marshal(response)
						assert.NoError(t, err)
						_, err = w.Write(body)
						assert.NoError(t, err)
						return
					}
					_, err := io.WriteString(w, "{}")
					assert.NoError(t, err)
				}))
				if secure {
					srv.TLS = pki.server.Clone()
					srv.StartTLS()
				} else {
					srv.Start()
				}
				defer srv.Close()
				args := append([]string{}, tc.args...)
				if tc.args[0] == "trace-id" {
					args = append(args, srv.URL+"/tempo", "1234")
				} else {
					args = append(args, "--path-prefix=/tempo", srv.Listener.Addr().String())
					switch tc.args[0] {
					case "search", "metrics":
						args = append(args, tc.query)
					case "search-tag-values":
						args = append(args, "resource.service.name")
					}
					args = append(args, "now-1h", "now")
					if secure {
						args = append(args, "--secure")
					}
				}
				args = append(args, "--org-id=test-tenant", "--header=Authorization=Bearer token=abc")
				if secure {
					args = append(args, "--tls-cert="+pki.options.TLSCert, "--tls-key="+pki.options.TLSKey, "--tls-ca="+pki.options.TLSCA, "--tls-server-name=tempo.test")
				}
				require.NoError(t, runTLSQuery(t, args...))
				require.Len(t, called, 1)
			})
		}
	}
}

func TestTLSHandshake(t *testing.T) {
	pki := newTLSTestPKI(t)
	untrusted := newTLSTestPKI(t)
	for _, transport := range []string{"http", "grpc"} {
		for _, tc := range []struct {
			name        string
			options     tlsOptions
			wantErr     bool
			errContains string
		}{
			{name: "valid mTLS", options: pki.options},
			// TLS 1.3 server rejection can reach the client as EOF or a broken pipe instead of a certificate alert.
			{name: "missing client certificate", options: tlsOptions{TLSCA: pki.options.TLSCA}, wantErr: true},
			{name: "untrusted client certificate", options: tlsOptions{TLSCA: pki.options.TLSCA, TLSCert: untrusted.options.TLSCert, TLSKey: untrusted.options.TLSKey}, wantErr: true},
			{name: "untrusted server", options: tlsOptions{TLSCert: pki.options.TLSCert, TLSKey: pki.options.TLSKey}, wantErr: true, errContains: "unknown authority"},
			{name: "wrong CA", options: tlsOptions{TLSCA: untrusted.options.TLSCA, TLSCert: pki.options.TLSCert, TLSKey: pki.options.TLSKey}, wantErr: true, errContains: "unknown authority"},
			{name: "wrong server name", options: tlsOptions{TLSCA: pki.options.TLSCA, TLSCert: pki.options.TLSCert, TLSKey: pki.options.TLSKey, TLSServerName: "wrong.test"}, wantErr: true, errContains: "wrong.test"},
		} {
			t.Run(transport+"/"+tc.name, func(t *testing.T) {
				var endpoint string
				if transport == "http" {
					srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						_, err := io.WriteString(w, "{}")
						assert.NoError(t, err)
					}))
					srv.TLS = pki.server.Clone()
					srv.StartTLS()
					defer srv.Close()
					endpoint = srv.Listener.Addr().String()
				} else {
					endpoint = startTLSQueryGRPCServer(t, pki.server, false)
				}
				cmd := querySearchTagsCmd{tlsOptions: tc.options, HostPort: endpoint, Secure: true, UseGRPC: transport == "grpc", OrgID: "test-tenant"}
				err := cmd.Run(nil)
				if tc.wantErr {
					require.Error(t, err)
					if tc.errContains != "" {
						require.ErrorContains(t, err, tc.errContains)
					}
				} else {
					require.NoError(t, err)
				}
			})
		}
	}
}

func TestTLSKeyValidation(t *testing.T) {
	pki := newTLSTestPKI(t)
	other := newTLSTestPKI(t)
	for _, key := range []string{other.options.TLSKey, filepath.Join(t.TempDir(), "missing.key"), pki.options.TLSCert} {
		options := pki.options
		options.TLSKey = key
		_, err := options.tlsConfig(true)
		require.ErrorContains(t, err, "loading client certificate and key")
	}
}

func TestTLSWithoutClientCertificate(t *testing.T) {
	pki := newTLSTestPKI(t)
	pki.server.ClientAuth = tls.NoClientCert
	for _, transport := range []string{"http", "grpc"} {
		t.Run(transport, func(t *testing.T) {
			var endpoint string
			if transport == "http" {
				srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					_, err := io.WriteString(w, "{}")
					assert.NoError(t, err)
				}))
				srv.TLS = pki.server.Clone()
				srv.StartTLS()
				defer srv.Close()
				endpoint = srv.Listener.Addr().String()
			} else {
				endpoint = startTLSQueryGRPCServer(t, pki.server, false)
			}
			cmd := querySearchTagsCmd{
				tlsOptions: tlsOptions{TLSCA: pki.options.TLSCA, TLSServerName: "tempo.test"},
				HostPort:   endpoint, Secure: true, UseGRPC: transport == "grpc", OrgID: "test-tenant",
			}
			require.NoError(t, cmd.Run(nil))
		})
	}
}

func TestQueryAPITLSRequiresSecure(t *testing.T) {
	for _, transport := range []string{"http", "grpc"} {
		for _, args := range [][]string{
			{"trace-id", "http://localhost:0", "1234"},
			{"search", "localhost:0", "{}", "now-1h", "now"},
			{"search-tags", "localhost:0"},
			{"search-tag-values", "localhost:0", "resource.service.name"},
			{"metrics", "localhost:0", "{} | rate()", "now-1h", "now"},
			{"metrics", "--instant", "localhost:0", "{} | rate()", "now-1h", "now"},
		} {
			if transport == "grpc" && args[0] == "trace-id" {
				continue
			}
			t.Run(transport+"/"+strings.Join(args[:2], " "), func(t *testing.T) {
				args = append(args, "--tls-ca=unused.pem", "--org-id=test-tenant")
				if transport == "grpc" {
					args = append(args, "--use-grpc")
				}
				require.ErrorContains(t, runTLSQuery(t, args...), "require TLS")
			})
		}
	}
}

func TestHTTPTransportIsolation(t *testing.T) {
	pki := newTLSTestPKI(t)
	base := http.DefaultTransport.(*http.Transport)
	before := base.Clone()
	transport, err := pki.options.httpTransport(true)
	require.NoError(t, err)
	defer transport.CloseIdleConnections()
	require.NotSame(t, base, transport)
	require.Equal(t, before.TLSClientConfig, base.TLSClientConfig)
	require.Equal(t, base.ForceAttemptHTTP2, transport.ForceAttemptHTTP2)
	require.NotNil(t, transport.Proxy)
	require.Equal(t, base.TLSHandshakeTimeout, transport.TLSHandshakeTimeout)

	defaults := tlsOptions{}
	other, err := defaults.httpTransport(true)
	require.NoError(t, err)
	defer other.CloseIdleConnections()
	require.Empty(t, other.TLSClientConfig.Certificates)
	require.Nil(t, other.TLSClientConfig.RootCAs)
	require.Empty(t, other.TLSClientConfig.ServerName)
}

type tlsQueryTestServer struct {
	tempopb.UnimplementedStreamingQuerierServer
}

func (*tlsQueryTestServer) Search(_ *tempopb.SearchRequest, stream tempopb.StreamingQuerier_SearchServer) error {
	return stream.Send(&tempopb.SearchResponse{})
}

func (*tlsQueryTestServer) SearchTagsV2(_ *tempopb.SearchTagsRequest, stream tempopb.StreamingQuerier_SearchTagsV2Server) error {
	return stream.Send(&tempopb.SearchTagsV2Response{})
}

func (*tlsQueryTestServer) SearchTagValuesV2(_ *tempopb.SearchTagValuesRequest, stream tempopb.StreamingQuerier_SearchTagValuesV2Server) error {
	return stream.Send(&tempopb.SearchTagValuesV2Response{})
}

func (*tlsQueryTestServer) MetricsQueryRange(_ *tempopb.QueryRangeRequest, stream tempopb.StreamingQuerier_MetricsQueryRangeServer) error {
	return stream.Send(&tempopb.QueryRangeResponse{})
}

func (*tlsQueryTestServer) MetricsQueryInstant(_ *tempopb.QueryInstantRequest, stream tempopb.StreamingQuerier_MetricsQueryInstantServer) error {
	return stream.Send(&tempopb.QueryInstantResponse{})
}

func startTLSQueryGRPCServer(t *testing.T, cfg *tls.Config, checkHeaders bool) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	opts := []grpc.ServerOption{grpc.StreamInterceptor(func(srv any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		md, _ := metadata.FromIncomingContext(stream.Context())
		assert.Equal(t, []string{"test-tenant"}, md.Get("x-scope-orgid"))
		if checkHeaders {
			assert.Equal(t, []string{"Bearer token=abc"}, md.Get("authorization"))
		}
		if cfg != nil {
			p, ok := peer.FromContext(stream.Context())
			if assert.True(t, ok) {
				info, ok := p.AuthInfo.(credentials.TLSInfo)
				if assert.True(t, ok) {
					if cfg.ClientAuth == tls.RequireAndVerifyClientCert {
						assert.NotEmpty(t, info.State.VerifiedChains)
					}
					if checkHeaders {
						assert.Equal(t, "tempo.test", info.State.ServerName)
					}
				}
			}
		}
		return handler(srv, stream)
	})}
	if cfg != nil {
		opts = append(opts, grpc.Creds(credentials.NewTLS(cfg.Clone())))
	}
	srv := grpc.NewServer(opts...)
	tempopb.RegisterStreamingQuerierServer(srv, &tlsQueryTestServer{})
	done := make(chan error, 1)
	go func() { done <- srv.Serve(listener) }()
	t.Cleanup(func() {
		srv.Stop()
		require.NoError(t, <-done)
	})
	return listener.Addr().String()
}

func TestQueryAPIMTLSGRPC(t *testing.T) {
	pki := newTLSTestPKI(t)
	for _, secure := range []bool{false, true} {
		mode := "plaintext"
		if secure {
			mode = "tls"
		}
		for _, args := range [][]string{
			{"search"}, {"search-tags"}, {"search-tag-values"}, {"metrics"}, {"metrics", "--instant"},
		} {
			t.Run(mode+"/"+strings.Join(args, " "), func(t *testing.T) {
				var cfg *tls.Config
				if secure {
					cfg = pki.server
				}
				endpoint := startTLSQueryGRPCServer(t, cfg, true)
				args = append(args, endpoint)
				switch args[0] {
				case "search":
					args = append(args, "{}")
				case "metrics":
					args = append(args, "{} | rate()")
				case "search-tag-values":
					args = append(args, "resource.service.name")
				}
				args = append(args, "now-1h", "now", "--use-grpc", "--org-id=test-tenant", "--header=Authorization=Bearer token=abc")
				if secure {
					args = append(args, "--secure", "--tls-cert="+pki.options.TLSCert, "--tls-key="+pki.options.TLSKey, "--tls-ca="+pki.options.TLSCA, "--tls-server-name=tempo.test")
				}
				require.NoError(t, runTLSQuery(t, args...))
			})
		}
	}
}
