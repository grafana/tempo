package keyfile

import (
	"bytes"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"fmt"

	"github.com/foxboron/go-tpm-keyfiles/template"
	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpm2/transport"
)

func makeSealedBlob(pk any, userauth []byte) (*tpm2.TPMTPublic, []byte, error) {
	var public *tpm2.TPMTPublic
	var sensitive tpm2.TPMTSensitive

	switch pk := pk.(type) {
	case ecdsa.PrivateKey:
		public = template.EcdsaToTPMTPublic(&pk.PublicKey, tpm2.TPMAlgSHA256)
		bb, err := pk.Bytes()
		if err != nil {
			return nil, nil, err
		}
		sensitive = tpm2.TPMTSensitive{
			SensitiveType: tpm2.TPMAlgECC,
			Sensitive: tpm2.NewTPMUSensitiveComposite(
				tpm2.TPMAlgECC,
				&tpm2.TPM2BECCParameter{Buffer: bb},
			),
		}
	case ecdh.PrivateKey:
		public = template.ECDHToTPMTPublic(pk.PublicKey(), tpm2.TPMAlgSHA256)
		sensitive = tpm2.TPMTSensitive{
			SensitiveType: tpm2.TPMAlgECC,
			Sensitive: tpm2.NewTPMUSensitiveComposite(
				tpm2.TPMAlgECC,
				&tpm2.TPM2BECCParameter{Buffer: pk.Bytes()},
			),
		}
	case rsa.PrivateKey:
		// TODO: We only really support 2048 bits
		public = template.RSAToTPMTPublic(&pk.PublicKey, 2048)
		sensitive = tpm2.TPMTSensitive{
			SensitiveType: tpm2.TPMAlgRSA,
			Sensitive: tpm2.NewTPMUSensitiveComposite(
				tpm2.TPMAlgRSA,
				&tpm2.TPM2BPrivateKeyRSA{Buffer: pk.Primes[0].Bytes()},
			),
		}
	}

	// Add user auth
	if !bytes.Equal(userauth, []byte("")) {
		sensitive.AuthValue = tpm2.TPM2BAuth{
			Buffer: userauth,
		}
	}

	return public, tpm2.Marshal(sensitive), nil
}

func NewImportablekey(rempub *tpm2.TPMTPublic, pk any, fn ...TPMKeyOption) (*TPMKey, error) {
	key := NewTPMKey(OIDImportableKey, tpm2.TPM2BPublic{}, tpm2.TPM2BPrivate{}, fn...)

	pub, sensitive, err := makeSealedBlob(pk, key.userAuth)
	if err != nil {
		return nil, err
	}

	kem, err := tpm2.ImportEncapsulationKey(rempub)
	if err != nil {
		return nil, err
	}

	name, err := tpm2.ObjectName(pub)
	if err != nil {
		return nil, err
	}

	dup, encSeed, err := tpm2.CreateDuplicate(rand.Reader, kem, name.Buffer, sensitive)
	if err != nil {
		return nil, err
	}

	key.AddOptions(
		WithPubkey(tpm2.New2B(*pub)),
		WithPrivkey(tpm2.TPM2BPrivate{Buffer: dup}),
		WithSecret(tpm2.TPM2BEncryptedSecret{Buffer: encSeed}),
	)
	return key, nil
}

// Returns a loadable key
func ImportTPMKey(tpm transport.TPMCloser, key *TPMKey, ownerauth []byte) (*TPMKey, error) {
	var sess TPMSession
	if !key.Keytype.Equal(OIDImportableKey) && !key.Keytype.Equal(OIDSealedKey) {
		return nil, fmt.Errorf("need importable key OID")
	}

	sess.SetTPM(tpm)
	parenthandle, err := GetParentHandle(&sess, key.Parent, ownerauth)
	if err != nil {
		return nil, err
	}
	defer sess.FlushHandle()

	importRsp, err := tpm2.Import{
		ParentHandle: parenthandle,
		ObjectPublic: key.Pubkey,
		Duplicate:    key.Privkey,
		InSymSeed:    key.Secret,
	}.Execute(tpm, sess.GetHMAC())
	if err != nil {
		return nil, err
	}

	// copy key
	lkey := *key

	lkey.Secret = tpm2.TPM2BEncryptedSecret{}
	lkey.AddOptions(
		WithKeytype(OIDLoadableKey),
		WithPrivkey(importRsp.OutPrivate),
	)

	return &lkey, nil
}
