// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package tls provides functionality to translate configuration options into tls.Config.
package tls // import "go.opentelemetry.io/contrib/otelconf/internal/tls"

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
)

// CreateConfig creates a tls.Config from certificate files.
func CreateConfig(caCertFile, clientCertFile, clientKeyFile *string) (*tls.Config, error) {
	tlsConfig := &tls.Config{}
	if caCertFile != nil {
		caText, err := os.ReadFile(*caCertFile)
		if err != nil {
			return nil, err
		}
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caText) {
			return nil, errors.New("could not create certificate authority chain from certificate")
		}
		tlsConfig.RootCAs = certPool
	}
	if clientCertFile != nil || clientKeyFile != nil {
		if clientCertFile == nil {
			return nil, errors.New("client key was provided but no client certificate was provided")
		}
		if clientKeyFile == nil {
			return nil, errors.New("client certificate was provided but no client key was provided")
		}
		clientCert, err := tls.LoadX509KeyPair(*clientCertFile, *clientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("could not use client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}
	return tlsConfig, nil
}
