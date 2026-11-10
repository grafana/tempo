package template

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/google/go-tpm/tpm2"
)

func ECDHToTPMTPublic(pubkey *ecdh.PublicKey, sha tpm2.TPMAlgID) *tpm2.TPMTPublic {
	var ecc tpm2.TPMECCCurve
	switch pubkey.Curve() {
	case ecdh.P256():
		ecc = tpm2.TPMECCNistP256
	case ecdh.P384():
		ecc = tpm2.TPMECCNistP384
	case ecdh.P521():
		ecc = tpm2.TPMECCNistP521
	}
	x, y, err := tpm2.ECCPoint(pubkey)
	if err != nil {
		panic(err)
	}
	return &tpm2.TPMTPublic{
		Type:    tpm2.TPMAlgECC,
		NameAlg: sha,
		ObjectAttributes: tpm2.TPMAObject{
			SignEncrypt:  true,
			UserWithAuth: true,
			Decrypt:      true,
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
		Unique: tpm2.NewTPMUPublicID(
			tpm2.TPMAlgECC,
			&tpm2.TPMSECCPoint{
				X: tpm2.TPM2BECCParameter{
					Buffer: x.Bytes(),
				},
				Y: tpm2.TPM2BECCParameter{
					Buffer: y.Bytes(),
				},
			},
		),
	}
}

func EcdsaToTPMTPublic(pubkey *ecdsa.PublicKey, sha tpm2.TPMAlgID) *tpm2.TPMTPublic {
	pk, err := pubkey.ECDH()
	if err != nil {
		panic(err)
	}
	return ECDHToTPMTPublic(pk, sha)
}

func RSAToTPMTPublic(pubkey *rsa.PublicKey, bits int) *tpm2.TPMTPublic {
	return &tpm2.TPMTPublic{
		Type:    tpm2.TPMAlgRSA,
		NameAlg: tpm2.TPMAlgSHA256,
		ObjectAttributes: tpm2.TPMAObject{
			SignEncrypt:  true,
			UserWithAuth: true,
			Decrypt:      true,
		},
		Parameters: tpm2.NewTPMUPublicParms(
			tpm2.TPMAlgRSA,
			&tpm2.TPMSRSAParms{
				Scheme: tpm2.TPMTRSAScheme{
					Scheme: tpm2.TPMAlgNull,
				},
				KeyBits: tpm2.TPMKeyBits(bits),
			},
		),
		Unique: tpm2.NewTPMUPublicID(
			tpm2.TPMAlgRSA,
			&tpm2.TPM2BPublicKeyRSA{Buffer: pubkey.N.Bytes()},
		),
	}
}

func ECDSAToSRK(pubkey *ecdsa.PublicKey, sha tpm2.TPMAlgID) *tpm2.TPMTPublic {
	pk, err := pubkey.ECDH()
	if err != nil {
		panic(err)
	}
	return ECDHToSRK(pk, sha)
}

func ECDHToSRK(pubkey *ecdh.PublicKey, sha tpm2.TPMAlgID) *tpm2.TPMTPublic {
	x, y, err := tpm2.ECCPoint(pubkey)
	if err != nil {
		panic(err)
	}
	tmpl := tpm2.ECCSRKTemplate
	tmpl.Unique = tpm2.NewTPMUPublicID(
		tpm2.TPMAlgECC,
		&tpm2.TPMSECCPoint{
			X: tpm2.TPM2BECCParameter{
				Buffer: x.Bytes(),
			},
			Y: tpm2.TPM2BECCParameter{
				Buffer: y.Bytes(),
			},
		},
	)
	return &tmpl
}

func RSAToSRK(pubkey *rsa.PublicKey, _ int) *tpm2.TPMTPublic {
	// TODO: Use bits
	tmpl := tpm2.RSASRKTemplate
	tmpl.Unique = tpm2.NewTPMUPublicID(
		tpm2.TPMAlgRSA,
		&tpm2.TPM2BPublicKeyRSA{Buffer: pubkey.N.Bytes()},
	)
	return &tmpl
}

// PublicToSRK turns a simple public key to the assumed SRK public template
func PublicToSRK(pKey []byte) (*tpm2.TPMTPublic, error) {
	block, _ := pem.Decode([]byte(pKey))

	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("failed to decode PEM block containing public key")
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed parsing pem key: %v", err)
	}

	switch p := key.(type) {
	case *ecdsa.PublicKey:
		return ECDSAToSRK(p, tpm2.TPMAlgSHA256), nil
	case *ecdh.PublicKey:
		return ECDHToSRK(p, tpm2.TPMAlgSHA256), nil
	case *rsa.PublicKey:
		// TODO: Support other bit lengths
		return RSAToSRK(p, 2048), nil
	default:
		return nil, fmt.Errorf("unsupported keytype")
	}
}
