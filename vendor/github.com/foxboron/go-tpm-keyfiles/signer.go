package keyfile

import (
	"crypto"
	"crypto/rsa"
	"fmt"
	"io"
	"sync"

	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpm2/transport"
)

// Ideally, this would be per-TPM, but e.g. go-tpm-tools just uses a global mutex.
var signerMutex sync.Mutex

// TPMKeySigner implements the crypto.Signer interface for TPMKey
// It allows passing callbacks for TPM, ownerAuth and user auth.
type TPMKeySigner struct {
	key       *TPMKey
	ownerAuth func() ([]byte, error)
	tpm       func() transport.TPMCloser
	auth      func(*TPMKey) ([]byte, error)
}

var _ crypto.Signer = &TPMKeySigner{}

// Returns the crypto.PublicKey
func (t *TPMKeySigner) Public() crypto.PublicKey {
	pk, err := t.key.PublicKey()
	// This shouldn't happen!
	if err != nil {
		panic(fmt.Errorf("failed producing public: %v", err))
	}
	return pk
}

// Sign implementation
func (t *TPMKeySigner) Sign(_ io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	var digestalg, signalg tpm2.TPMAlgID

	signerMutex.Lock()
	defer signerMutex.Unlock()

	auth := []byte("")
	if t.key.HasAuth() {
		p, err := t.auth(t.key)
		if err != nil {
			return nil, err
		}
		auth = p
	}

	switch opts.HashFunc() {
	case crypto.SHA256:
		digestalg = tpm2.TPMAlgSHA256
	case crypto.SHA384:
		digestalg = tpm2.TPMAlgSHA384
	case crypto.SHA512:
		digestalg = tpm2.TPMAlgSHA512
	default:
		return nil, fmt.Errorf("%s is not a supported hashing algorithm", opts.HashFunc())
	}

	signalg = t.key.KeyAlgo()
	if _, ok := opts.(*rsa.PSSOptions); ok {
		if signalg != tpm2.TPMAlgRSA {
			return nil, fmt.Errorf("Attempting to use PSSOptions with non-RSA (alg %x) key", signalg)
		}
		signalg = tpm2.TPMAlgRSAPSS
	}

	ownerauth, err := t.ownerAuth()
	if err != nil {
		return nil, err
	}

	sess := NewTPMSession(t.tpm())
	sess.SetTPM(t.tpm())

	return SignASN1(sess, t.key, ownerauth, auth, digest, digestalg, signalg)
}

func NewTPMKeySigner(k *TPMKey, ownerAuth func() ([]byte, error), tpm func() transport.TPMCloser, auth func(*TPMKey) ([]byte, error)) *TPMKeySigner {
	return &TPMKeySigner{
		key:       k,
		ownerAuth: ownerAuth,
		tpm:       tpm,
		auth:      auth,
	}
}

// TPMHandleSigner implements the crypto.Signer interface for an already-loaded TPMKey
type TPMHandleSigner struct {
	tpm     transport.TPMCloser
	pubKey  crypto.PublicKey
	keyAlgo tpm2.TPMAlgID
	keySize int
	handle  handle
}

func (t *TPMHandleSigner) Public() crypto.PublicKey {
	return t.pubKey
}

func (t *TPMHandleSigner) Sign(_ io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	var digestalg, signalg tpm2.TPMAlgID
	var digestlength int

	switch opts.HashFunc() {
	case crypto.SHA256:
		digestalg = tpm2.TPMAlgSHA256
		digestlength = 32
	case crypto.SHA384:
		digestalg = tpm2.TPMAlgSHA384
		digestlength = 48
	case crypto.SHA512:
		digestalg = tpm2.TPMAlgSHA512
		digestlength = 64
	default:
		return nil, fmt.Errorf("%s is not a supported hashing algorithm", opts.HashFunc())
	}

	if len(digest) != digestlength {
		return nil, fmt.Errorf("incorrect checksum length. expected %v got %v", digestlength, len(digest))
	}

	signalg = t.keyAlgo
	if _, ok := opts.(*rsa.PSSOptions); ok {
		if signalg != tpm2.TPMAlgRSA {
			return nil, fmt.Errorf("Attempting to use PSSOptions with non-RSA (alg %x) key", signalg)
		}
		signalg = tpm2.TPMAlgRSAPSS
	}

	signerMutex.Lock()
	defer signerMutex.Unlock()

	rsp, err := TPMSign(t.tpm, t.handle, digest, digestalg, t.keySize, signalg)
	if err != nil {
		return nil, err
	}
	return EncodeSignatureASN1(rsp)
}

func NewTPMHandleSigner(
	tpm transport.TPMCloser,
	pubKey crypto.PublicKey,
	keyAlgo tpm2.TPMAlgID,
	keySize int,
	handle handle,
) TPMHandleSigner {
	return TPMHandleSigner{tpm, pubKey, keyAlgo, keySize, handle}
}

func NewTPMHandleSignerFromKey(
	tpm transport.TPMCloser,
	key *TPMKey,
	handle handle,
) TPMHandleSigner {
	pk, err := key.PublicKey()
	// This shouldn't happen!
	if err != nil {
		panic(fmt.Errorf("failed producing public: %v", err))
	}

	return TPMHandleSigner{tpm, pk, key.KeyAlgo(), key.KeySize(), handle}
}
