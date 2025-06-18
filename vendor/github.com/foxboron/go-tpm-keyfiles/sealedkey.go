package keyfile

import (
	"fmt"

	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpm2/transport"
)

func NewSealedData(tpm transport.TPMCloser, data []byte, ownerauth []byte, fn ...TPMKeyOption) (*TPMKey, error) {
	sess := NewTPMSession(tpm)
	key := NewTPMKey(OIDSealedKey, tpm2.TPM2BPublic{}, tpm2.TPM2BPrivate{}, fn...)
	parenthandle, err := GetParentHandle(sess, key.Parent, ownerauth)
	if err != nil {
		return nil, err
	}

	sealBlobCmd := tpm2.Create{
		ParentHandle: parenthandle,
		InSensitive: tpm2.TPM2BSensitiveCreate{
			Sensitive: &tpm2.TPMSSensitiveCreate{
				UserAuth: tpm2.TPM2BAuth{
					Buffer: []byte(nil),
				},
				Data: tpm2.NewTPMUSensitiveCreate(&tpm2.TPM2BSensitiveData{
					Buffer: data,
				}),
			},
		},
		InPublic: tpm2.New2B(tpm2.TPMTPublic{
			Type:    tpm2.TPMAlgKeyedHash,
			NameAlg: tpm2.TPMAlgSHA256,
			ObjectAttributes: tpm2.TPMAObject{
				FixedTPM:     true,
				FixedParent:  true,
				UserWithAuth: true,
				NoDA:         true,
			},
		}),
	}

	rsp, err := sealBlobCmd.Execute(sess.GetTPM(), sess.GetHMACIn())
	if err != nil {
		return nil, err
	}

	key.AddOptions(
		WithPubkey(rsp.OutPublic),
		WithPrivkey(rsp.OutPrivate),
	)

	return key, nil
}

func UnsealData(tpm transport.TPMCloser, key *TPMKey, ownerauth []byte) ([]byte, error) {
	sess := NewTPMSession(tpm)
	handle, _, err := LoadKey(sess, key, ownerauth)
	if err != nil {
		return nil, err
	}

	rsp, err := tpm2.Unseal{
		ItemHandle: handle,
	}.Execute(sess.GetTPM(), sess.GetHMACOut())
	if err != nil {
		return nil, fmt.Errorf("failed tpm2_unseal: %v", err)
	}

	return rsp.OutData.Buffer, nil
}

// TODO: Do we define sealed key stuff on top of the data?
// func NewSealedKey(pk any) (*TPMKey, error) {
// 	return nil, nil
// }

// func UnsealKey() ([]byte, error) {
// 	return nil, nil
// }
