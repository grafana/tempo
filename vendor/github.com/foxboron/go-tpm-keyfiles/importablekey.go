package keyfile

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"hash"
	"math/big"

	"github.com/foxboron/go-tpm-keyfiles/template"
	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpm2/transport"
)

// ECC coordinates need to maintain a specific size based on the curve, so we pad the front with zeros.
// This is particularly an issue for NIST-P521 coordinates, as they are frequently missing their first byte.
func eccIntToBytes(curve elliptic.Curve, i *big.Int) []byte {
	bytes := i.Bytes()
	curveBytes := (curve.Params().BitSize + 7) / 8
	return append(make([]byte, curveBytes-len(bytes)), bytes...)
}

func createECCSeed(pub *tpm2.TPMTPublic) (seed, encryptedSeed []byte, err error) {
	curve := elliptic.P256()

	// We need access to the values so we don't use ecdh to generate the key
	priv, x, y, err := elliptic.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	privKey := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int).SetBytes(x.Bytes()),
			Y:     new(big.Int).SetBytes(y.Bytes()),
		},
		D: new(big.Int).SetBytes(priv),
	}
	privKeyECDH, err := privKey.ECDH()
	if err != nil {
		return nil, nil, fmt.Errorf("failed creating ecdh key: %v", err)
	}

	ecc, err := pub.Unique.ECC()
	if err != nil {
		return nil, nil, fmt.Errorf("failed getting ECC values from public: %v", err)
	}

	if len(ecc.X.Buffer) == 0 || len(ecc.Y.Buffer) == 0 {
		return nil, nil, fmt.Errorf("public TPM2TPublic does not have a valid ECC public key")
	}

	pubKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(ecc.X.Buffer),
		Y:     new(big.Int).SetBytes(ecc.Y.Buffer),
	}

	pubkeyECDH, err := pubKey.ECDH()
	if err != nil {
		return nil, nil, fmt.Errorf("failed getting ECDH from produced public key: %v", err)
	}

	z, _ := privKeyECDH.ECDH(pubkeyECDH)
	xBytes := eccIntToBytes(curve, x)
	seed = tpm2.KDFe(
		crypto.SHA256,
		z,
		"DUPLICATE",
		xBytes,
		eccIntToBytes(curve, pubKey.X),
		crypto.SHA256.Size()*8)

	encryptedSeed = tpm2.Marshal(tpm2.TPMSECCPoint{
		X: tpm2.TPM2BECCParameter{Buffer: x.FillBytes(make([]byte, len(x.Bytes())))},
		Y: tpm2.TPM2BECCParameter{Buffer: y.FillBytes(make([]byte, len(y.Bytes())))},
	})

	return seed, encryptedSeed, err
}

func createWrap(pub *tpm2.TPMTPublic, pk any, userauth []byte) (tpm2.TPM2BPublic, []byte, []byte, error) {
	var err error
	var seed []byte
	var encryptedSeed []byte

	switch pub.Type {
	case tpm2.TPMAlgECC:
		seed, encryptedSeed, err = createECCSeed(pub)
		if err != nil {
			return tpm2.TPM2BPublic{}, nil, nil, err
		}
	default:
		return tpm2.TPM2BPublic{}, nil, nil, fmt.Errorf("only support ECC parents for import wrapping: %v", pub.Type)
	}

	var public *tpm2.TPMTPublic
	var sensitive tpm2.TPMTSensitive

	switch pk := pk.(type) {
	case ecdsa.PrivateKey:
		public = template.EcdsaToTPMTPublic(&pk.PublicKey, tpm2.TPMAlgSHA256)
		sensitive = tpm2.TPMTSensitive{
			SensitiveType: tpm2.TPMAlgECC,
			Sensitive: tpm2.NewTPMUSensitiveComposite(
				tpm2.TPMAlgECC,
				&tpm2.TPM2BECCParameter{Buffer: pk.D.FillBytes(make([]byte, len(pk.D.Bytes())))},
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

	sens2B := tpm2.Marshal(sensitive)
	sens2B = tpm2.Marshal(tpm2.TPM2BPrivate{Buffer: sens2B})

	b2name, err := tpm2.ObjectName(public)
	if err != nil {
		return tpm2.TPM2BPublic{}, nil, nil, err
	}

	// AES symm encryption key
	symmetricKey := tpm2.KDFa(
		crypto.SHA256,
		seed,
		"STORAGE",
		b2name.Buffer,
		/*contextV=*/ nil,
		128)

	c, err := aes.NewCipher(symmetricKey)
	if err != nil {
		return tpm2.TPM2BPublic{}, nil, nil, err
	}
	encryptedSecret := make([]byte, len(sens2B))
	// The TPM spec requires an all-zero IV.
	iv := make([]byte, len(symmetricKey))
	cipher.NewCFBEncrypter(c, iv).XORKeyStream(encryptedSecret, sens2B)

	macKey := tpm2.KDFa(
		crypto.SHA256,
		seed,
		"INTEGRITY",
		/*contextU=*/ nil,
		/*contextV=*/ nil,
		crypto.SHA256.Size()*8)

	mac := hmac.New(func() hash.Hash { return crypto.SHA256.New() }, macKey)
	mac.Write(encryptedSecret)
	mac.Write(b2name.Buffer)

	hmacSum := mac.Sum(nil)

	// The duplicate structure is a sized TPM2BPrivate for the Digest
	// and a encrypted secret which is read based off on the size
	// of the encrypted metadata.
	dup := tpm2.Marshal(tpm2.TPM2BPrivate{Buffer: hmacSum})
	dup = append(dup, encryptedSecret...)

	return tpm2.New2B(*public), dup, encryptedSeed, nil
}

func NewImportablekey(rempub *tpm2.TPMTPublic, pk any, fn ...TPMKeyOption) (*TPMKey, error) {
	key := NewTPMKey(OIDImportableKey, tpm2.TPM2BPublic{}, tpm2.TPM2BPrivate{}, fn...)
	pub, dup, encSeed, err := createWrap(rempub, pk, key.userAuth)
	if err != nil {
		return nil, err
	}
	key.AddOptions(
		WithPubkey(pub),
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
