package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type tlsOptions struct {
	TLSCert       string `name:"tls-cert" type:"path" help:"PEM client certificate file for mTLS; requires --tls-key and TLS"`
	TLSKey        string `name:"tls-key" type:"path" help:"PEM private key file for mTLS; requires --tls-cert and TLS"`
	TLSCA         string `name:"tls-ca" type:"path" help:"PEM CA certificate bundle to add to system trust roots; requires TLS"`
	TLSServerName string `name:"tls-server-name" help:"override the TLS server name for certificate verification and SNI; requires TLS"`
}

func (o *tlsOptions) tlsConfig(secure bool) (*tls.Config, error) {
	if !secure {
		if o.TLSCert != "" || o.TLSKey != "" || o.TLSCA != "" || o.TLSServerName != "" {
			return nil, fmt.Errorf("TLS options require TLS: use an https:// URL for trace-id, or --secure for other query api commands")
		}
		return nil, nil
	}
	if (o.TLSCert == "") != (o.TLSKey == "") {
		return nil, fmt.Errorf("--tls-cert and --tls-key must be provided together")
	}

	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: o.TLSServerName,
	}
	if o.TLSCA != "" {
		pool, err := x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("loading system certificate pool: %w", err)
		}
		if pool == nil {
			pool = x509.NewCertPool()
		}
		pem, err := os.ReadFile(o.TLSCA)
		if err != nil {
			return nil, fmt.Errorf("reading CA certificate %q: %w", o.TLSCA, err)
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("no valid certificates found in CA bundle %q", o.TLSCA)
		}
		cfg.RootCAs = pool
	}
	if o.TLSCert != "" {
		cert, err := tls.LoadX509KeyPair(o.TLSCert, o.TLSKey)
		if err != nil {
			return nil, fmt.Errorf("loading client certificate and key %q, %q: %w", o.TLSCert, o.TLSKey, err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}
	return cfg, nil
}

func (o *tlsOptions) httpTransport(secure bool) (*http.Transport, error) {
	cfg, err := o.tlsConfig(secure)
	if err != nil {
		return nil, err
	}
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("unexpected default HTTP transport type %T", http.DefaultTransport)
	}
	transport := base.Clone()
	transport.TLSClientConfig = cfg
	return transport, nil
}

func (o *tlsOptions) grpcTransportCredentials(secure bool) (grpc.DialOption, error) {
	cfg, err := o.tlsConfig(secure)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return grpc.WithTransportCredentials(insecure.NewCredentials()), nil
	}
	return grpc.WithTransportCredentials(credentials.NewTLS(cfg)), nil
}
