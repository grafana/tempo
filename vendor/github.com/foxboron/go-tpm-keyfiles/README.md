go-tpm-keyfile
==============

Implements the ASN.1 Specification for TPM 2.0 Key Files.

https://www.hansenpartnership.com/draft-bottomley-tpm2-keys.html


### Implementation Status

- [x] Loadable Keys
- [x] Importable Keys
- [ ] Sealed data


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
		keyfile.OIDOldLoadableKey
		eccKeyResponse.OutPublic,
		eccKeyResponse.OutPrivate,
		keyfile.WithDescription("This is a TPM Key"),
	)

	os.Writefile("key.pem", k.Bytes(), 0640)
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
