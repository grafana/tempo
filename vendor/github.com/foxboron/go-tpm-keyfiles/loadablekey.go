package keyfile

import (
	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpm2/transport"
)

// TODO: Do we want a new struct to represent these?
// type LoadableTPMKey struct {
// 	*LoadableTPMKey
// }

// NewLoadableKey creates a new LoadableKey
func NewLoadableKey(tpm transport.TPMCloser, alg tpm2.TPMAlgID, bits int, ownerauth []byte, fn ...TPMKeyOption) (*TPMKey, error) {
	tpmkey, _, err := NewLoadableKeyWithResponse(tpm, alg, bits, ownerauth, fn...)
	return tpmkey, err
}

// NewLoadableKeyWithResponse creates a new LoadableKey and returns the tpm2.CreateResponse
func NewLoadableKeyWithResponse(tpm transport.TPMCloser, alg tpm2.TPMAlgID, bits int, ownerauth []byte, fn ...TPMKeyOption) (*TPMKey, *tpm2.CreateResponse, error) {
	sess := NewTPMSession(tpm)
	key := NewTPMKey(OIDLoadableKey, tpm2.TPM2BPublic{}, tpm2.TPM2BPrivate{}, fn...)

	parenthandle, err := GetParentHandle(sess, key.Parent, ownerauth)
	if err != nil {
		return nil, nil, err
	}

	defer sess.FlushHandle()

	rsp, err := createKeyWithHandle(sess, *parenthandle, alg, bits, ownerauth, key.userAuth)
	if err != nil {
		return nil, nil, err
	}

	// Add the remaining options to complete the key
	key.AddOptions(
		WithPubkey(rsp.OutPublic),
		WithPrivkey(rsp.OutPrivate),
	)
	return key, rsp, nil
}
