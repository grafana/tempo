package template

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"fmt"
	"math/big"

	"github.com/google/go-tpm/tpm2"
)

func EcdsaToTPMTPublic(pubkey *ecdsa.PublicKey, sha tpm2.TPMAlgID) *tpm2.TPMTPublic {
	var ecc tpm2.TPMECCCurve
	switch pubkey.Curve {
	case elliptic.P256():
		ecc = tpm2.TPMECCNistP256
	case elliptic.P384():
		ecc = tpm2.TPMECCNistP384
	case elliptic.P521():
		ecc = tpm2.TPMECCNistP521
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
					Buffer: pubkey.X.FillBytes(make([]byte, len(pubkey.X.Bytes()))),
				},
				Y: tpm2.TPM2BECCParameter{
					Buffer: pubkey.Y.FillBytes(make([]byte, len(pubkey.Y.Bytes()))),
				},
			},
		),
	}
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

func fromTPMPublicToECDSA(pub *tpm2.TPMTPublic) (*ecdsa.PublicKey, error) {
	ecc, err := pub.Unique.ECC()
	if err != nil {
		return nil, err
	}

	eccdeets, err := pub.Parameters.ECCDetail()
	if err != nil {
		return nil, err
	}

	var ecdsaKey *ecdsa.PublicKey

	switch eccdeets.CurveID {
	case tpm2.TPMECCNistP256:
		ecdsaKey = &ecdsa.PublicKey{Curve: elliptic.P256(),
			X: big.NewInt(0).SetBytes(ecc.X.Buffer),
			Y: big.NewInt(0).SetBytes(ecc.Y.Buffer),
		}
	case tpm2.TPMECCNistP384:
		ecdsaKey = &ecdsa.PublicKey{Curve: elliptic.P384(),
			X: big.NewInt(0).SetBytes(ecc.X.Buffer),
			Y: big.NewInt(0).SetBytes(ecc.Y.Buffer),
		}
	case tpm2.TPMECCNistP521:
		ecdsaKey = &ecdsa.PublicKey{Curve: elliptic.P521(),
			X: big.NewInt(0).SetBytes(ecc.X.Buffer),
			Y: big.NewInt(0).SetBytes(ecc.Y.Buffer),
		}
	}
	return ecdsaKey, nil
}

func fromTPMPublicToRSA(pub *tpm2.TPMTPublic) (*rsa.PublicKey, error) {
	rsaDetail, err := pub.Parameters.RSADetail()
	if err != nil {
		return nil, fmt.Errorf("failed getting rsa details: %v", err)
	}
	rsaUnique, err := pub.Unique.RSA()
	if err != nil {
		return nil, fmt.Errorf("failed getting unique rsa: %v", err)
	}

	return tpm2.RSAPub(rsaDetail, rsaUnique)
}

// FromTPMPublicToPubkey transform a tpm2.TPMTPublic to crypto.PublicKey
func FromTPMPublicToPubkey(pub *tpm2.TPMTPublic) (crypto.PublicKey, error) {
	switch pub.Type {
	case tpm2.TPMAlgECC:
		return fromTPMPublicToECDSA(pub)
	case tpm2.TPMAlgRSA:
		return fromTPMPublicToRSA(pub)
	}
	return nil, fmt.Errorf("no a supported public key")
}
