package test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grafana/dskit/runutil"
	"github.com/stretchr/testify/require"
)

type KeyMaterial struct {
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

func SetupCertificates(t *testing.T) KeyMaterial {
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

	return KeyMaterial{
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

	return writeExclusivePEMFile(path, "CERTIFICATE", 0o644, derBytes)
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

	return writeExclusivePEMFile(certPath, "CERTIFICATE", 0o644, derBytes)
}
