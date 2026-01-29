package keyfile

import (
	"bytes"
	"crypto"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpm2/transport"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/cryptobyte"
	"golang.org/x/crypto/cryptobyte/asn1"
	"golang.org/x/crypto/hkdf"
)

var (
	// If a permanent handle (MSO 0x40) is specified then the implementation MUST run
	// TPM2_CreatePrimary on the handle using the TCG specified Elliptic Curve
	// template [TCG-Provision] (section 7.5.1 for the Storage and other seeds or
	// 7.4.1 for the endorsement seed) which refers to the TCG EK Credential Profile
	// [TCG-EK-Profile] . Since there are several possible templates, implementations
	// MUST always use the H template (the one with zero size unique fields). The
	// template used MUST be H-2 (EK Credential Profile section B.4.5) for the NIST
	// P-256 curve if rsaParent is absent or the H-1 (EK Credential Profile section
	// B.4.4) RSA template with a key length of 2048 if rsaParent is present and true
	// and use the primary key so generated as the parent.
	ECCSRK_H2_Template = tpm2.TPMTPublic{
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

// This is a helper to deal with TPM Session encryption.
type TPMSession struct {
	tpm    transport.TPMCloser
	opt    tpm2.AuthOption
	handle tpm2.TPMHandle
}

func NewTPMSession(tpm transport.TPMCloser) *TPMSession {
	var s TPMSession
	s.tpm = tpm
	return &s
}

func (t *TPMSession) SetTPM(tpm transport.TPMCloser) {
	t.tpm = tpm
}

func (t *TPMSession) GetTPM() transport.TPMCloser {
	return t.tpm
}

func (t *TPMSession) SetOpt(opt tpm2.AuthOption) {
	t.opt = opt
}

func (t *TPMSession) SetSalted(handle tpm2.TPMHandle, pub tpm2.TPMTPublic) {
	t.handle = handle
	t.SetOpt(tpm2.Salted(handle, pub))
}

func (t *TPMSession) FlushHandle() {
	FlushHandle(t.tpm, t.handle)
}

func (t *TPMSession) GetHMAC() tpm2.Session {
	// TODO: Do we want a jit encryption or a continous session?
	return tpm2.HMAC(tpm2.TPMAlgSHA256, 16,
		tpm2.AESEncryption(128, tpm2.EncryptInOut),
		t.opt)
}

func (t *TPMSession) GetHMACIn() tpm2.Session {
	// EncryptIn and EncryptInOut are internal to go-tpm so.. this is the solution
	return tpm2.HMAC(tpm2.TPMAlgSHA256, 16,
		tpm2.AESEncryption(128, tpm2.EncryptIn),
		t.opt)
}

func (t *TPMSession) GetHMACOut() tpm2.Session {
	// EncryptIn and EncryptInOut are internal to go-tpm so.. this is the solution
	return tpm2.HMAC(tpm2.TPMAlgSHA256, 16,
		tpm2.AESEncryption(128, tpm2.EncryptOut),
		t.opt)
}

func LoadKeyWithParent(session *TPMSession, parent tpm2.AuthHandle, key *TPMKey) (*tpm2.AuthHandle, error) {
	loadBlobCmd := tpm2.Load{
		ParentHandle: parent,
		InPrivate:    key.Privkey,
		InPublic:     key.Pubkey,
	}
	loadBlobRsp, err := loadBlobCmd.Execute(session.GetTPM(), session.GetHMAC())
	if err != nil {
		return nil, fmt.Errorf("failed getting handle: %v", err)
	}

	// Return a AuthHandle with a nil PasswordAuth
	return &tpm2.AuthHandle{
		Handle: loadBlobRsp.ObjectHandle,
		Name:   loadBlobRsp.Name,
		Auth:   tpm2.PasswordAuth(nil),
	}, nil
}

func LoadKey(sess *TPMSession, key *TPMKey, ownerauth []byte) (keyhandle *tpm2.AuthHandle, parenthandle *tpm2.AuthHandle, err error) {
	if key.Keytype.Equal(OIDImportableKey) {
		key, err = ImportTPMKey(sess.tpm, key, ownerauth)
		if err != nil {
			return nil, nil, fmt.Errorf("failing loading imported key: %v", err)
		}
	} else if !key.Keytype.Equal(OIDLoadableKey) && !key.Keytype.Equal(OIDSealedKey) {
		return nil, nil, fmt.Errorf("not a loadable key")
	}

	parenthandle, err = GetParentHandle(sess, key.Parent, ownerauth)
	if err != nil {
		return nil, nil, err
	}
	keyhandle, err = LoadKeyWithParent(sess, *parenthandle, key)

	return keyhandle, parenthandle, err
}

// Creates a Storage Key, or return the loaded storage key
func CreateSRK(sess *TPMSession, hier tpm2.TPMHandle, ownerAuth []byte) (*tpm2.AuthHandle, *tpm2.TPMTPublic, error) {
	public := tpm2.New2B(ECCSRK_H2_Template)

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
		InPublic: public,
	}

	var rsp *tpm2.CreatePrimaryResponse
	rsp, err := srk.Execute(sess.GetTPM())
	if err != nil {
		return nil, nil, fmt.Errorf("failed creating primary key: %v", err)
	}

	srkPublic, err := rsp.OutPublic.Contents()
	if err != nil {
		return nil, nil, fmt.Errorf("failed getting srk public content: %v", err)
	}

	return &tpm2.AuthHandle{
		Handle: rsp.ObjectHandle,
		Name:   rsp.Name,
		Auth:   tpm2.PasswordAuth(nil),
	}, srkPublic, nil
}

func createECCKey(ecc tpm2.TPMECCCurve, sha tpm2.TPMAlgID) tpm2.TPM2B[tpm2.TPMTPublic, *tpm2.TPMTPublic] {
	return tpm2.New2B(tpm2.TPMTPublic{
		Type:    tpm2.TPMAlgECC,
		NameAlg: sha,
		ObjectAttributes: tpm2.TPMAObject{
			FixedTPM:            true,
			FixedParent:         true,
			SensitiveDataOrigin: true,
			UserWithAuth:        true,
			SignEncrypt:         true,
			Decrypt:             true,
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
	})
}

func createRSAKey(bits tpm2.TPMKeyBits, sha tpm2.TPMAlgID) tpm2.TPM2B[tpm2.TPMTPublic, *tpm2.TPMTPublic] {
	return tpm2.New2B(tpm2.TPMTPublic{
		Type:    tpm2.TPMAlgRSA,
		NameAlg: sha,
		ObjectAttributes: tpm2.TPMAObject{
			FixedTPM:            true,
			FixedParent:         true,
			SensitiveDataOrigin: true,
			UserWithAuth:        true,
			SignEncrypt:         true,
			Decrypt:             true,
		},
		Parameters: tpm2.NewTPMUPublicParms(
			tpm2.TPMAlgRSA,
			&tpm2.TPMSRSAParms{
				Scheme: tpm2.TPMTRSAScheme{
					Scheme: tpm2.TPMAlgNull,
				},
				KeyBits: bits,
			},
		),
	})
}

// from crypto/ecdsa
func addASN1IntBytes(b *cryptobyte.Builder, bytes []byte) {
	for len(bytes) > 0 && bytes[0] == 0 {
		bytes = bytes[1:]
	}
	if len(bytes) == 0 {
		b.SetError(errors.New("invalid integer"))
		return
	}
	b.AddASN1(asn1.INTEGER, func(c *cryptobyte.Builder) {
		if bytes[0]&0x80 != 0 {
			c.AddUint8(0)
		}
		c.AddBytes(bytes)
	})
}

// from crypto/ecdsa
func encodeSignature(r, s []byte) ([]byte, error) {
	var b cryptobyte.Builder
	b.AddASN1(asn1.SEQUENCE, func(b *cryptobyte.Builder) {
		addASN1IntBytes(b, r)
		addASN1IntBytes(b, s)
	})
	return b.Bytes()
}

func newECCSigScheme(digest tpm2.TPMAlgID) tpm2.TPMTSigScheme {
	return tpm2.TPMTSigScheme{
		Scheme: tpm2.TPMAlgECDSA,
		Details: tpm2.NewTPMUSigScheme(
			tpm2.TPMAlgECDSA,
			&tpm2.TPMSSchemeHash{
				HashAlg: digest,
			},
		),
	}
}

func newRSASSASigScheme(digest tpm2.TPMAlgID) tpm2.TPMTSigScheme {
	return tpm2.TPMTSigScheme{
		Scheme: tpm2.TPMAlgRSASSA,
		Details: tpm2.NewTPMUSigScheme(
			tpm2.TPMAlgRSASSA,
			&tpm2.TPMSSchemeHash{
				HashAlg: digest,
			},
		),
	}
}

func newRSAPSSSigScheme(digest tpm2.TPMAlgID) tpm2.TPMTSigScheme {
	return tpm2.TPMTSigScheme{
		Scheme: tpm2.TPMAlgRSAPSS,
		Details: tpm2.NewTPMUSigScheme(
			tpm2.TPMAlgRSAPSS,
			&tpm2.TPMSSchemeHash{
				HashAlg: digest,
			},
		),
	}
}

func Sign(sess *TPMSession, key *TPMKey, ownerauth, auth, digest []byte, digestalgo, signalgo tpm2.TPMAlgID) (*tpm2.TPMTSignature, error) {
	var digestlength int
	var err error

	switch digestalgo {
	case tpm2.TPMAlgSHA256:
		digestlength = 32
	case tpm2.TPMAlgSHA384:
		digestlength = 48
	case tpm2.TPMAlgSHA512:
		digestlength = 64
	default:
		return nil, fmt.Errorf("%v is not a supported hashing algorithm", digestalgo)
	}

	if len(digest) != digestlength {
		return nil, fmt.Errorf("incorrect checksum length. expected %v got %v", digestlength, len(digest))
	}

	if key.Keytype.Equal(OIDImportableKey) {
		key, err = ImportTPMKey(sess.tpm, key, ownerauth)
		if err != nil {
			return nil, fmt.Errorf("failing loading imported key for signing: %v", err)
		}
	} else if !key.Keytype.Equal(OIDLoadableKey) {
		return nil, fmt.Errorf("not a loadable key")
	}

	if !key.HasSigner() {
		return nil, fmt.Errorf("key does not have a signer")
	}

	parenthandle, err := GetParentHandle(sess, key.Parent, ownerauth)
	if err != nil {
		return nil, err
	}
	defer sess.FlushHandle()

	handle, err := LoadKeyWithParent(sess, *parenthandle, key)
	if err != nil {
		return nil, err
	}
	defer FlushHandle(sess.GetTPM(), handle)

	if len(auth) != 0 {
		handle.Auth = tpm2.PasswordAuth(auth)
	}

	return TPMSign(sess.GetTPM(), *handle, digest, digestalgo, key.KeySize(), signalgo, sess.GetHMACIn())
}

func TPMSign(tpm transport.TPMCloser, handle handle, digest []byte, digestalgo tpm2.TPMAlgID, keysize int, keyalgo tpm2.TPMAlgID, sess ...tpm2.Session) (*tpm2.TPMTSignature, error) {

	// Seperate function to include our own sigscheme?
	var sigscheme tpm2.TPMTSigScheme
	switch keyalgo {
	case tpm2.TPMAlgECC:
		sigscheme = newECCSigScheme(digestalgo)
	case tpm2.TPMAlgRSA, tpm2.TPMAlgRSASSA:
		sigscheme = newRSASSASigScheme(digestalgo)
	case tpm2.TPMAlgRSAPSS:
		sigscheme = newRSAPSSSigScheme(digestalgo)
	default:
		return nil, fmt.Errorf("Unexpected key algorithm 0x%x", keyalgo)
	}

	// If we encounter RSA with SHA512 keys we use TPM_Decrypt to sign
	// This implements
	if digestalgo == tpm2.TPMAlgSHA512 && (keyalgo == tpm2.TPMAlgRSA || keyalgo == tpm2.TPMAlgRSASSA) {
		// TODO: Refactor this part
		// Taken from crypto/rsa
		pkcsPadding := func(hashed []byte, privkeySize int, h crypto.Hash) []byte {
			var hashPrefixes = map[crypto.Hash][]byte{
				crypto.SHA256: {0x30, 0x31, 0x30, 0x0d, 0x06, 0x09, 0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x01, 0x05, 0x00, 0x04, 0x20},
				crypto.SHA384: {0x30, 0x41, 0x30, 0x0d, 0x06, 0x09, 0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x02, 0x05, 0x00, 0x04, 0x30},
				crypto.SHA512: {0x30, 0x51, 0x30, 0x0d, 0x06, 0x09, 0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x03, 0x05, 0x00, 0x04, 0x40},
			}
			hashLen := h.Size()

			prefix := hashPrefixes[h]
			tLen := len(hashPrefixes[h]) + hashLen
			em := make([]byte, privkeySize)
			em[1] = 1
			for i := 2; i < privkeySize-tLen-1; i++ {
				em[i] = 0xff
			}
			copy(em[privkeySize-tLen:privkeySize-hashLen], prefix)
			copy(em[privkeySize-hashLen:privkeySize], hashed)
			return em
		}
		paddedDigest := pkcsPadding(digest, keysize, crypto.SHA512)
		decryptRsp, err := tpm2.RSADecrypt{
			KeyHandle:  handle,
			CipherText: tpm2.TPM2BPublicKeyRSA{Buffer: paddedDigest[:]},
			InScheme: tpm2.TPMTRSADecrypt{
				Scheme: tpm2.TPMAlgNull,
			},
		}.Execute(tpm, sess...)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt+sign: %w", err)
		}
		return &tpm2.TPMTSignature{
			SigAlg: tpm2.TPMAlgRSASSA,
			Signature: tpm2.NewTPMUSignature(
				tpm2.TPMAlgRSASSA,
				&tpm2.TPMSSignatureRSA{
					Hash: tpm2.TPMAlgSHA512,
					Sig:  tpm2.TPM2BPublicKeyRSA{Buffer: decryptRsp.Message.Buffer},
				},
			),
		}, nil
	} else {
		sign := tpm2.Sign{
			KeyHandle: handle,
			Digest:    tpm2.TPM2BDigest{Buffer: digest[:]},
			InScheme:  sigscheme,
			Validation: tpm2.TPMTTKHashCheck{
				Tag: tpm2.TPMSTHashCheck,
			},
		}
		rspSign, err := sign.Execute(tpm, sess...)
		if err != nil {
			return nil, fmt.Errorf("failed to sign: %w", err)
		}
		return &rspSign.Signature, nil
	}
}

func SignASN1(sess *TPMSession, key *TPMKey, ownerauth, auth, digest []byte, digestalgo, signalgo tpm2.TPMAlgID) ([]byte, error) {
	rsp, err := Sign(sess, key, ownerauth, auth, digest, digestalgo, signalgo)
	if err != nil {
		return nil, err
	}
	return EncodeSignatureASN1(rsp)
}

func EncodeSignatureASN1(rsp *tpm2.TPMTSignature) ([]byte, error) {
	switch rsp.SigAlg {
	case tpm2.TPMAlgECDSA:
		eccsig, err := rsp.Signature.ECDSA()
		if err != nil {
			return nil, fmt.Errorf("failed getting signature: %v", err)
		}
		return encodeSignature(eccsig.SignatureR.Buffer, eccsig.SignatureS.Buffer)
	case tpm2.TPMAlgRSASSA:
		rsassa, err := rsp.Signature.RSASSA()
		if err != nil {
			return nil, fmt.Errorf("failed getting rsassa signature")
		}
		return rsassa.Sig.Buffer, nil
	case tpm2.TPMAlgRSAPSS:
		rsapss, err := rsp.Signature.RSAPSS()
		if err != nil {
			return nil, fmt.Errorf("failed getting rsapss signature")
		}
		return rsapss.Sig.Buffer, nil
	}
	return nil, fmt.Errorf("failed returning signature")
}

// shadow the unexported interface from go-tpm
type handle interface {
	HandleValue() uint32
	KnownName() *tpm2.TPM2BName
}

// Helper to flush handles
func FlushHandle(tpm transport.TPM, h handle) {
	//TODO: We should probably handle the error here
	flushSrk := tpm2.FlushContext{FlushHandle: h}
	flushSrk.Execute(tpm)
}

func SupportedECCAlgorithms(tpm transport.TPMCloser) []int {
	var getCapRsp *tpm2.GetCapabilityResponse
	var supportedBitsizes []int

	eccCapCmd := tpm2.GetCapability{
		Capability:    tpm2.TPMCapECCCurves,
		PropertyCount: 100,
	}
	getCapRsp, err := eccCapCmd.Execute(tpm)
	if err != nil {
		return []int{}
	}
	curves, err := getCapRsp.CapabilityData.Data.ECCCurves()
	if err != nil {
		return []int{}
	}
	for _, curve := range curves.ECCCurves {
		c, err := curve.Curve()
		// if we fail here we are dealing with an unsupported curve
		if err != nil {
			continue
		}
		supportedBitsizes = append(supportedBitsizes, c.Params().BitSize)
	}
	return supportedBitsizes
}

func createKeyWithHandle(sess *TPMSession, parent tpm2.AuthHandle, keytype tpm2.TPMAlgID, bits int, ownerAuth []byte, auth []byte) (*tpm2.CreateResponse, error) {
	rsaBits := []int{2048}
	ecdsaBits := []int{256, 384, 521}

	supportedECCBitsizes := SupportedECCAlgorithms(sess.GetTPM())

	switch keytype {
	case tpm2.TPMAlgECC:
		if bits == 0 {
			bits = ecdsaBits[0]
		}
		if !slices.Contains(ecdsaBits, bits) {
			return nil, errors.New("invalid ecdsa key length: valid length are 256, 384 or 512 bits")
		}
		if !slices.Contains(supportedECCBitsizes, bits) {
			return nil, fmt.Errorf("invalid ecdsa key length: TPM does not support %v bits", bits)
		}
	case tpm2.TPMAlgRSA:
		if bits == 0 {
			bits = rsaBits[0]
		}
		if !slices.Contains(rsaBits, bits) {
			return nil, errors.New("invalid rsa key length: only 2048 is supported")
		}
	default:
		return nil, fmt.Errorf("unsupported key type")
	}

	var keyPublic tpm2.TPM2BPublic
	switch {
	case keytype == tpm2.TPMAlgECC && bits == 256:
		keyPublic = createECCKey(tpm2.TPMECCNistP256, tpm2.TPMAlgSHA256)
	case keytype == tpm2.TPMAlgECC && bits == 384:
		keyPublic = createECCKey(tpm2.TPMECCNistP384, tpm2.TPMAlgSHA256)
	case keytype == tpm2.TPMAlgECC && bits == 521:
		keyPublic = createECCKey(tpm2.TPMECCNistP521, tpm2.TPMAlgSHA256)
	case keytype == tpm2.TPMAlgRSA:
		keyPublic = createRSAKey(2048, tpm2.TPMAlgSHA256)
	}

	// Template for en ECC key for signing
	createKey := tpm2.Create{
		ParentHandle: parent,
		InPublic:     keyPublic,
	}

	if !bytes.Equal(auth, []byte("")) {
		createKey.InSensitive = tpm2.TPM2BSensitiveCreate{
			Sensitive: &tpm2.TPMSSensitiveCreate{
				UserAuth: tpm2.TPM2BAuth{
					Buffer: auth,
				},
			},
		}
	}

	createRsp, err := createKey.Execute(sess.GetTPM(), sess.GetHMAC())
	if err != nil {
		return nil, fmt.Errorf("failed creating TPM key: %v", err)
	}

	return createRsp, nil
}

// TODO: Private until I'm confident of the API
func CreateKey(sess *TPMSession, keytype tpm2.TPMAlgID, bits int, ownerAuth []byte, auth []byte) (tpm2.TPM2BPublic, tpm2.TPM2BPrivate, error) {
	srkHandle, pub, err := CreateSRK(sess, tpm2.TPMRHOwner, ownerAuth)
	if err != nil {
		return tpm2.TPM2BPublic{}, tpm2.TPM2BPrivate{}, err
	}
	sess.SetSalted(srkHandle.Handle, *pub)
	defer FlushHandle(sess.GetTPM(), srkHandle)
	rsp, err := createKeyWithHandle(sess, *srkHandle, keytype, bits, ownerAuth, auth)
	return rsp.OutPublic, rsp.OutPrivate, err
}

func ReadPublic(tpm transport.TPMCloser, handle tpm2.TPMHandle) (*tpm2.AuthHandle, *tpm2.TPMTPublic, error) {
	rsp, err := tpm2.ReadPublic{
		ObjectHandle: handle,
	}.Execute(tpm)
	if err != nil {
		return nil, nil, err
	}
	pub, err := rsp.OutPublic.Contents()
	if err != nil {
		return nil, nil, err
	}
	return &tpm2.AuthHandle{
		Handle: handle,
		Name:   rsp.Name,
	}, pub, nil
}

// This looks at the passed parent defined in a TPMKey and gives back the
// appropriate handle to load our TPM key under.
// With a PERMANENT handle it will load an transient SRK under the parent heier,
// and give back the handle.
// With a TRANSIENT handle it will load a transient SRK under the Owner hier,
// and hand back the handle.
// With a PERSISTENT handle it will try to read the public portion of the key.
//
// This function will also set the appropriate bound HMAC session to the
// returned keys.
func GetParentHandle(sess *TPMSession, parent tpm2.TPMHandle, ownerauth []byte) (*tpm2.AuthHandle, error) {
	var parenthandle tpm2.AuthHandle

	if IsMSO(parent, TPM_HT_PERMANENT) {
		srkHandle, pub, err := CreateSRK(sess, parent, ownerauth)
		if err != nil {
			return nil, err
		}
		sess.SetSalted(srkHandle.Handle, *pub)
		parenthandle = *srkHandle
	} else if IsMSO(parent, TPM_HT_TRANSIENT) {
		// Parent should never be transient, but we might have keys that use the
		// wrong handle lets try to load this under the owner hier
		srkHandle, pub, err := CreateSRK(sess, tpm2.TPMRHOwner, ownerauth)
		if err != nil {
			return nil, err
		}
		sess.SetSalted(srkHandle.Handle, *pub)
		parenthandle = *srkHandle
	} else if IsMSO(parent, TPM_HT_PERSISTENT) {
		handle, pub, err := ReadPublic(sess.GetTPM(), parent)
		if err != nil {
			return nil, err
		}
		parenthandle = *handle
		parenthandle.Auth = tpm2.PasswordAuth(ownerauth)

		// TODO: Unclear to me if we just load the EK and use that, instead of the key.
		sess.SetSalted(parent, *pub)
	}
	return &parenthandle, nil
}

// ChangeAuth changes the object authn header to something else
// notice this changes the private blob inside the key in-place.
func ChangeAuth(tpm transport.TPMCloser, ownerauth []byte, key *TPMKey, oldpin, newpin []byte) error {
	// TODO: For imported keys I assume we need to do the entire encryption dance again?
	if !key.Keytype.Equal(OIDLoadableKey) {
		return fmt.Errorf("can only be used on loadable keys")
	}

	var err error

	sess := NewTPMSession(tpm)
	defer sess.FlushHandle()

	handle, parenthandle, err := LoadKey(sess, key, ownerauth)
	if err != nil {
		return err
	}
	defer FlushHandle(tpm, handle)

	if len(oldpin) != 0 {
		handle.Auth = tpm2.PasswordAuth(oldpin)
	}

	oca := tpm2.ObjectChangeAuth{
		ParentHandle: parenthandle,
		ObjectHandle: *handle,
		NewAuth: tpm2.TPM2BAuth{
			Buffer: newpin,
		},
	}
	rsp, err := oca.Execute(tpm, sess.GetHMAC())
	if err != nil {
		return fmt.Errorf("ObjectChangeAuth failed: %v", err)
	}

	key.AddOptions(
		WithPrivkey(rsp.OutPrivate),
		WithUserAuth(newpin),
	)

	return nil
}

const p256Label = "github.com/foxboron/go-tpm-keyfile/v1/p256"

func kdf(sharedKey, publicKey *ecdh.PublicKey, shared []byte) ([]byte, error) {
	// NOTE:
	// This should probably be compatible with whatever openssl is doing,
	// but I have no clue. So this is just what age is doing for figuring out
	// shared tokens
	sharedKeyB := sharedKey.Bytes()
	publicKeyB := publicKey.Bytes()

	// We use the concatinated bytes of the shared key and the public key for the
	// key derivative functions.
	salt := make([]byte, 0, len(sharedKeyB)+len(publicKeyB))
	salt = append(salt, sharedKeyB...)
	salt = append(salt, publicKeyB...)

	h := hkdf.New(sha256.New, shared, salt, []byte(p256Label))
	wrappingKey := make([]byte, chacha20poly1305.KeySize)
	if _, err := io.ReadFull(h, wrappingKey); err != nil {
		return nil, err
	}
	return wrappingKey, nil
}

func DeriveECDH(sess *TPMSession, key *TPMKey, sessionkey *ecdh.PublicKey, ownerauth, auth []byte) ([]byte, error) {
	var publickey *ecdh.PublicKey
	pubkey, err := key.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed getting pubkey: %v", err)
	}
	switch pk := pubkey.(type) {
	case *ecdsa.PublicKey:
		publickey, err = pk.ECDH()
		if err != nil {
			return nil, fmt.Errorf("can't get ecdh key")
		}
	case *rsa.PublicKey:
		return nil, fmt.Errorf("only ecdh key scan use DeriveECDH")
	}

	parenthandle, err := GetParentHandle(sess, key.Parent, ownerauth)
	if err != nil {
		return nil, err
	}
	defer sess.FlushHandle()

	handle, err := LoadKeyWithParent(sess, *parenthandle, key)
	if err != nil {
		return nil, err
	}
	defer FlushHandle(sess.GetTPM(), handle)

	if len(auth) != 0 {
		handle.Auth = tpm2.PasswordAuth(auth)
	}

	x, y := elliptic.Unmarshal(elliptic.P256(), sessionkey.Bytes())

	// ECDHZGen command for the TPM, turns the sesion key into something we understand.
	ecdhRsp, err := tpm2.ECDHZGen{
		KeyHandle: *handle,
		InPoint: tpm2.New2B(
			tpm2.TPMSECCPoint{
				X: tpm2.TPM2BECCParameter{Buffer: x.FillBytes(make([]byte, 32))},
				Y: tpm2.TPM2BECCParameter{Buffer: y.FillBytes(make([]byte, 32))},
			},
		),
	}.Execute(sess.GetTPM(), sess.GetHMAC())
	if err != nil {
		fmt.Println("here")
		return nil, err
	}

	shared, err := ecdhRsp.OutPoint.Contents()
	if err != nil {
		return nil, fmt.Errorf("failed getting ecdh point: %v", err)
	}

	return kdf(sessionkey, publickey, shared.X.Buffer)
}
