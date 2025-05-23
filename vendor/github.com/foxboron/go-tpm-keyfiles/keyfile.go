package keyfile

import (
	"bytes"
	"crypto"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/rsa"
	encasn1 "encoding/asn1"
	"fmt"

	"github.com/foxboron/go-tpm-keyfiles/template"
	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpm2/transport"
)

type TPMPolicy struct {
	CommandCode   int
	CommandPolicy []byte
}

type TPMAuthPolicy struct {
	Name   string
	Policy []*TPMPolicy
}

type TPMKey struct {
	Keytype     encasn1.ObjectIdentifier
	EmptyAuth   bool
	Policy      []*TPMPolicy
	Secret      tpm2.TPM2BEncryptedSecret
	AuthPolicy  []*TPMAuthPolicy
	Description string
	Parent      tpm2.TPMHandle
	Pubkey      tpm2.TPM2BPublic
	Privkey     tpm2.TPM2BPrivate
	userAuth    []byte // Internal detail
}

func NewTPMKey(oid encasn1.ObjectIdentifier, pubkey tpm2.TPM2BPublic, privkey tpm2.TPM2BPrivate, fn ...TPMKeyOption) *TPMKey {
	var key TPMKey

	// Set defaults
	key.AddOptions(
		WithKeytype(oid),
		// We always start of with assuming this key shouldn't have an auth
		WithUserAuth([]byte(nil)),
		// Start out with setting the Owner as parent
		WithParent(tpm2.TPMRHOwner),
		WithPubkey(pubkey),
		WithPrivkey(privkey),
	)

	key.AddOptions(fn...)
	return &key
}

func (t *TPMKey) AddOptions(fn ...TPMKeyOption) {
	// Run over TPMKeyFn
	for _, f := range fn {
		f(t)
	}
}

// Internal function to deserialize the TPMTPublic
func (t *TPMKey) contents() *tpm2.TPMTPublic {
	pub, err := t.Pubkey.Contents()
	if err != nil {
		// This should just not happen. So panic if we get this
		// Prevents a bunch of error in our code.
		panic(fmt.Sprintf("can't serialize public key: %v", err))
	}
	return pub
}

func (t *TPMKey) HasSigner() bool {
	return t.contents().ObjectAttributes.SignEncrypt
}

func (t *TPMKey) HasAuth() bool {
	return !t.EmptyAuth
}

func (t *TPMKey) KeyAlgo() tpm2.TPMAlgID {
	return t.contents().Type
}

func (t *TPMKey) KeySize() int {
	pubkey, err := t.PublicKey()
	if err != nil {
		return 0
	}
	switch pk := pubkey.(type) {
	case *ecdsa.PublicKey:
		// TODO: IDK yo
		return 0
	case *rsa.PublicKey:
		return pk.Size()
	}
	return 0
}

func (t *TPMKey) Bytes() []byte {
	var b bytes.Buffer
	if err := Encode(&b, t); err != nil {
		return nil
	}
	return b.Bytes()
}

// PublicKey returns the ecdsa.Publickey or rsa.Publickey of the TPMKey
func (t *TPMKey) PublicKey() (crypto.PublicKey, error) {
	return template.FromTPMPublicToPubkey(t.contents())
}

// Wraps TPMSigner with some sane defaults
// Use NewTPMSigner if you need more control of the parameters
func (t *TPMKey) Signer(tpm transport.TPMCloser, ownerAuth, auth []byte) (crypto.Signer, error) {
	if !t.HasSigner() {
		// TODO: Implement support for signing with Decrypt operations
		return nil, fmt.Errorf("does not have sign/encrypt attribute set")
	}
	return NewTPMKeySigner(
		t,
		func() ([]byte, error) { return ownerAuth, nil },
		func() transport.TPMCloser { return tpm },
		func(_ *TPMKey) ([]byte, error) { return auth, nil },
	), nil
}

func (t *TPMKey) Verify(alg crypto.Hash, hashed []byte, sig []byte) (bool, error) {
	pubkey, err := t.PublicKey()
	if err != nil {
		return false, fmt.Errorf("failed getting pubkey: %v", err)
	}
	switch pk := pubkey.(type) {
	case *ecdsa.PublicKey:
		if !ecdsa.VerifyASN1(pk, hashed[:], sig) {
			return false, fmt.Errorf("invalid signature")
		}
	case *rsa.PublicKey:
		if err := rsa.VerifyPKCS1v15(pk, alg, hashed[:], sig); err != nil {
			return false, fmt.Errorf("signature verification failed: %v", err)
		}
	}
	return true, nil
}

func (t *TPMKey) Derive(tpm transport.TPMCloser, sessionkey *ecdh.PublicKey, ownerAuth, auth []byte) ([]byte, error) {
	// TODO: This should only be available for ECC keys
	sess := NewTPMSession(tpm)
	defer sess.FlushHandle()
	return DeriveECDH(sess, t, sessionkey, ownerAuth, auth)
}
