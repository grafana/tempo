go-tpm-keyfile
==============

Implements the ASN.1 Specification for TPM 2.0 Key Files.

https://www.hansenpartnership.com/draft-bottomley-tpm2-keys.html


# Loadable Keys

## With NewLoadableKey

```go
package main

import (
	"os"

	keyfile "github.com/foxboron/go-tpm-keyfiles"

	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpm2/transport/simulator"
)

func main() {
	tpm, _ := simulator.OpenSimulator()
	defer tpm.Close()
	k, _ := keyfile.NewLoadableKey(tpm, tpm2.TPMAlgECC, 256, []byte{},
		keyfile.WithDescription("TPM Key"),
	)
	os.Writefile("key.pem", k.Bytes(), 0640)
}
```

## With NewTPMKey

```go
package main

import (
	"os"

	keyfile "github.com/foxboron/go-tpm-keyfiles"

	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpm2/transport/simulator"
)

func main(){
	tpm, _ := simulator.OpenSimulator()
	defer tpm.Close()

	primary, _ := tpm2.CreatePrimary{
		PrimaryHandle: tpm2.TPMRHOwner,
		InPublic:      tpm2.New2B(tpm2.ECCSRKTemplate),
	}.Execute(tpm)

	eccTemplate := tpm2.TPMTPublic{
		Type:    tpm2.TPMAlgECC,
		NameAlg: sha,
		ObjectAttributes: tpm2.TPMAObject{
			SignEncrypt:         true,
			FixedTPM:            true,
			FixedParent:         true,
			SensitiveDataOrigin: true,
			UserWithAuth:        true,
		},
		Parameters: tpm2.NewTPMUPublicParms(
			tpm2.TPMAlgECC,
			&tpm2.TPMSECCParms{
				CurveID: ecc,
				Scheme: tpm2.TPMTECCScheme{
					Scheme: tpm2.TPMAlgNull,
				},
			},
		),
	}

	eccKeyResponse, := tpm2.CreateLoaded{
		ParentHandle: tpm2.AuthHandle{
			Handle: primary.ObjectHandle,
			Name:   primary.Name,
			Auth:   tpm2.PasswordAuth([]byte(nil)),
		},
		InPublic: tpm2.New2BTemplate(&eccTemplate),
	}.Execute(tpm)

	k := keyfile.NewTPMKey(
		keyfile.OIDLoadableKey
		eccKeyResponse.OutPublic,
		eccKeyResponse.OutPrivate,
		keyfile.WithDescription("This is a TPM Key"),
	)

	os.Writefile("key.pem", k.Bytes(), 0640)
}
```

# Importable Key

```go
package main

import (
	"os"

	keyfile "github.com/foxboron/go-tpm-keyfiles"

	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpm2/transport/simulator"
)

var ECCSRK_H2_Template = tpm2.TPMTPublic{
		Type:    tpm2.TPMAlgECC,
		NameAlg: tpm2.TPMAlgSHA256,
		ObjectAttributes: tpm2.TPMAObject{
			FixedTPM:            true,
			FixedParent:         true,
			SensitiveDataOrigin: true,
			UserWithAuth:        true,
			NoDA:                true,
			Restricted:          true,
			Decrypt:             true,
		},
		Parameters: tpm2.NewTPMUPublicParms(
			tpm2.TPMAlgECC,
			&tpm2.TPMSECCParms{
				Symmetric: tpm2.TPMTSymDefObject{
					Algorithm: tpm2.TPMAlgAES,
					KeyBits: tpm2.NewTPMUSymKeyBits(
						tpm2.TPMAlgAES,
						tpm2.TPMKeyBits(128),
					),
					Mode: tpm2.NewTPMUSymMode(
						tpm2.TPMAlgAES,
						tpm2.TPMAlgCFB,
					),
				},
				CurveID: tpm2.TPMECCNistP256,
			},
		),
		Unique: tpm2.NewTPMUPublicID(
			tpm2.TPMAlgECC,
			&tpm2.TPMSECCPoint{
				X: tpm2.TPM2BECCParameter{
					Buffer: make([]byte, 0),
				},
				Y: tpm2.TPM2BECCParameter{
					Buffer: make([]byte, 0),
				},
			},
		),
	}
)

func main() {
	tpm, _ := simulator.OpenSimulator()
	defer tpm.Close()

	pk, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	srk := tpm2.CreatePrimary{
		PrimaryHandle: tpm2.AuthHandle{
			Handle: hier,
			Auth:   tpm2.PasswordAuth(ownerAuth),
		},
		InSensitive: tpm2.TPM2BSensitiveCreate{
			Sensitive: &tpm2.TPMSSensitiveCreate{
				UserAuth: tpm2.TPM2BAuth{
					Buffer: []byte(nil),
				},
			},
		},
		InPublic: tpm2.New2B(ECCSRK_H2_Template),
	}

	var rsp *tpm2.CreatePrimaryResponse
	rsp, _ := srk.Execute(tpm)
	srkPublic, _ := rsp.OutPublic.Contents()

	key, _ := keyfile.NewImportablekey(srkpub, ecc)

	os.Writefile("key.pem", k.Bytes(), 0640)

	// To import the key
	k, _ := keyfile.ImportTPMKey(tpm, key, []byte(nil))
}
```

# Sealed Data

```go
package main

import (
	"os"

	keyfile "github.com/foxboron/go-tpm-keyfiles"

	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpm2/transport/simulator"
)

func main() {
	tpm, _ := simulator.OpenSimulator()
	defer tpm.Close()

	msg := []byte("message")

	k, _ := keyfile.NewSealedData(tpm, msg, []byte(nil))

	data, _ := keyfile.UnsealData(tpm, k, []byte(nil))

	if bytes.Equal(data, msg) {
		fmt.Println("same message")
	}
}
```

# TPMSigner

`go-tpm-keyfile` implements a `crypto.Signer` interface to be used with the keys
for easy signature creation and verification.

The `TPMKeySigner` struct implements a callback-style approach for user auth,
owner auth and for TPM fetching for easier implementation towards things that
require user-input.

```go
package main

import (
	"crypto"
	"os"

	keyfile "github.com/foxboron/go-tpm-keyfiles"

	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpm2/transport/simulator"
)

func main() {
	tpm, _ := simulator.OpenSimulator()
	defer tpm.Close()

	k, _ := NewLoadableKey(tpm, tpm2.TPMAlgECC, 256, []byte(""))

	signer, _ := k.Signer(tpm, []byte(""), []byte(""))

	h := crypto.SHA256.New()
	h.Write([]byte("message"))
	b := h.Sum(nil)

	sig, _ := signer.Sign((io.Reader)(nil), b[:], crypto.SHA256)

	ok, err := k.Verify(crypto.SHA256, b[:], sig)
	if !ok || err != nil {
		log.Fatalf("invalid signature")
	}
}
```
