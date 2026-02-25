package keyfile

import (
	encasn1 "encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/google/go-tpm/tpm2"
	"golang.org/x/crypto/cryptobyte"
	"golang.org/x/crypto/cryptobyte/asn1"
)

var (
	// id-tpmkey OBJECT IDENTIFIER ::=
	//   {joint-iso-itu-t(2) international-organizations(23) 133 10 1}
	// Probably not used, but here for reference
	oidTPMKey = encasn1.ObjectIdentifier{2, 23, 133, 10, 2}

	OIDOldLoadableKey = encasn1.ObjectIdentifier{2, 23, 133, 10, 2}

	// id-loadablekey OBJECT IDENTIFIER ::=  {id-tpmkey 3}
	OIDLoadableKey = encasn1.ObjectIdentifier{2, 23, 133, 10, 1, 3}

	// id-importablekey OBJECT IDENTIFIER ::=  {id-tpmkey 4}
	OIDImportableKey = encasn1.ObjectIdentifier{2, 23, 133, 10, 1, 4}

	// id-sealedkey OBJECT IDENTIFIER ::= {id-tpmkey 5}
	OIDSealedKey = encasn1.ObjectIdentifier{2, 23, 133, 10, 1, 5}
)

func readOptional(der *cryptobyte.String, tag int) (out cryptobyte.String, ok bool) {
	der.ReadOptionalASN1(&out, &ok, asn1.Tag(tag).ContextSpecific().Constructed())
	return
}

func parseTPMPolicy(der *cryptobyte.String) ([]*TPMPolicy, error) {
	var tpmpolicies []*TPMPolicy

	//   policy      [1] EXPLICIT SEQUENCE OF TPMPolicy OPTIONAL,
	policy, ok := readOptional(der, 1)
	if !ok {
		return []*TPMPolicy{}, nil
	}

	// read outer sequence
	var policySequence cryptobyte.String
	if !policy.ReadASN1(&policySequence, asn1.SEQUENCE) {
		return nil, errors.New("malformed policy sequence")
	}

	for !policySequence.Empty() {
		// TPMPolicy ::= SEQUENCE
		var policyBytes cryptobyte.String
		if !policySequence.ReadASN1(&policyBytes, asn1.SEQUENCE) {
			return nil, errors.New("malformed policy sequence")
		}

		//   commandCode   [0] EXPLICIT INTEGER,
		var ccBytes cryptobyte.String
		if !policyBytes.ReadASN1(&ccBytes, asn1.Tag(0).ContextSpecific().Constructed()) {
			return nil, errors.New("strip tag from commandCode")
		}

		var tpmpolicy TPMPolicy
		if !ccBytes.ReadASN1Integer(&tpmpolicy.CommandCode) {
			return nil, errors.New("malformed policy commandCode")
		}

		//   commandPolicy [1] EXPLICIT OCTET STRING
		var cpBytes cryptobyte.String
		if !policyBytes.ReadASN1(&cpBytes, asn1.Tag(1).ContextSpecific().Constructed()) {
			return nil, errors.New("strip tag from commandPolicy")
		}

		if !cpBytes.ReadASN1Bytes(&tpmpolicy.CommandPolicy, asn1.OCTET_STRING) {
			return nil, errors.New("malformed policy commandPolicy")
		}
		tpmpolicies = append(tpmpolicies, &tpmpolicy)
	}
	return tpmpolicies, nil
}

func parseTPMAuthPolicy(der *cryptobyte.String) ([]*TPMAuthPolicy, error) {
	var authPolicy []*TPMAuthPolicy
	var sAuthPolicy cryptobyte.String

	//   authPolicy  [3] EXPLICIT SEQUENCE OF TPMAuthPolicy OPTIONAL,
	sAuthPolicy, ok := readOptional(der, 3)
	if !ok {
		return authPolicy, nil
	}

	// read outer sequence
	var authPolicySequence cryptobyte.String
	if !sAuthPolicy.ReadASN1(&authPolicySequence, asn1.SEQUENCE) {
		return nil, errors.New("malformed auth policy sequence")
	}

	for !authPolicySequence.Empty() {
		// TPMAuthPolicy ::= SEQUENCE
		var authPolicyBytes cryptobyte.String
		if !authPolicySequence.ReadASN1(&authPolicyBytes, asn1.SEQUENCE) {
			return nil, errors.New("malformed auth policy sequence")
		}

		var tpmAuthPolicy TPMAuthPolicy

		//   name    [0] EXPLICIT UTF8String OPTIONAL,
		nameBytes, ok := readOptional(&authPolicyBytes, 0)
		if ok {
			var nameB cryptobyte.String
			if !nameBytes.ReadASN1(&nameB, asn1.UTF8String) {
				return nil, errors.New("malformed utf8string in auth policy name")
			}
			if !utf8.Valid(nameB) {
				return nil, errors.New("invalid utf8 bytes in name of auth policy")
			}
			tpmAuthPolicy.Name = string(nameB)
		}

		//   policy  [1] EXPLICIT SEQUENCE OF TPMPolicy
		tpmpolicies, err := parseTPMPolicy(&authPolicyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed parsing tpm policies in auth policy: %v", err)
		}
		if len(tpmpolicies) == 0 {
			return nil, errors.New("tpm policies in auth policy is empty")
		}
		tpmAuthPolicy.Policy = tpmpolicies

		authPolicy = append(authPolicy, &tpmAuthPolicy)
	}

	return authPolicy, nil
}

func Parse(b []byte) (*TPMKey, error) {
	var tkey TPMKey
	var err error

	// TPMKey ::= SEQUENCE
	s := cryptobyte.String(b)
	if !s.ReadASN1(&s, asn1.SEQUENCE) {
		return nil, errors.New("no sequence")
	}

	//   type        TPMKeyType,
	var oid encasn1.ObjectIdentifier
	if !s.ReadASN1ObjectIdentifier(&oid) {
		return nil, errors.New("no contentinfo oid")
	}

	// Check if we know the keytype
	// TPMKeyType ::= OBJECT IDENTIFIER (
	//   id-loadablekey |
	//   id-importablekey |
	//   id-sealedkey
	// )
	switch {
	case oid.Equal(OIDLoadableKey):
		fallthrough
	case oid.Equal(OIDImportableKey):
		fallthrough
	case oid.Equal(OIDSealedKey):
		fallthrough
	case oid.Equal(OIDOldLoadableKey):
		tkey.Keytype = oid
	default:
		return nil, errors.New("unknown key type")
	}

	//   emptyAuth   [0] EXPLICIT BOOLEAN OPTIONAL,
	if emptyAuthbytes, ok := readOptional(&s, 0); ok {
		var auth bool
		var bytes cryptobyte.String
		if !emptyAuthbytes.ReadASN1(&bytes, asn1.BOOLEAN) || len(bytes) != 1 {
			return nil, errors.New("no emptyAuth bool")
		}

		switch bytes[0] {
		case 0:
			auth = false
		case 1:
			auth = true
		case 0xff:
			auth = true
		default:
			auth = false
		}
		tkey.EmptyAuth = auth
	}

	policy, err := parseTPMPolicy(&s)
	if err != nil {
		return nil, fmt.Errorf("failed reading TPMPolicy: %v", err)
	}
	tkey.Policy = policy

	//   secret      [2] EXPLICIT OCTET STRING OPTIONAL,
	if secretbytes, ok := readOptional(&s, 2); ok {
		var secretb cryptobyte.String
		if !secretbytes.ReadASN1(&secretb, asn1.OCTET_STRING) {
			return nil, errors.New("could not parse secret")
		}
		secret, err := tpm2.Unmarshal[tpm2.TPM2BEncryptedSecret](secretb)
		if err != nil {
			return nil, errors.New("could not parse public section of key 1")
		}
		tkey.Secret = *secret
	}

	//   authPolicy  [3] EXPLICIT SEQUENCE OF TPMAuthPolicy OPTIONAL,
	authPolicy, err := parseTPMAuthPolicy(&s)
	if err != nil {
		return nil, fmt.Errorf("failed reading TPMAuthPolicy: %v", err)
	}
	tkey.AuthPolicy = authPolicy

	//   description  [4] EXPLICIT OCTET STRING OPTIONAL,
	if descriptionBytes, ok := readOptional(&s, 4); ok {
		var description cryptobyte.String
		if !descriptionBytes.ReadASN1(&description, asn1.UTF8String) {
			return nil, errors.New("could not parse description bytes")
		}
		if !utf8.Valid(description) {
			return nil, errors.New("description is not a valid UTF8 string")
		}
		tkey.Description = string(description)
	}

	// Consume optional directives we don't support
	// TODO: Bump this number when we support more things
	i := 5
	for {
		if _, ok := readOptional(&s, i); !ok {
			break
		}
		i++
	}

	//   parent      INTEGER,
	var parent uint32
	if !s.ReadASN1Integer(&parent) {
		return nil, errors.New("failed reading parent")
	}
	tkey.Parent = tpm2.TPMHandle(parent)

	//   pubkey      OCTET STRING,
	var pubkey cryptobyte.String
	if !s.ReadASN1(&pubkey, asn1.OCTET_STRING) {
		return nil, errors.New("could not parse pubkey")
	}

	public, err := tpm2.Unmarshal[tpm2.TPM2BPublic](pubkey)
	if err != nil {
		return nil, errors.New("could not parse public section of key 1")
	}
	tkey.Pubkey = *public

	//   privkey     OCTET STRING
	var privkey cryptobyte.String
	if !s.ReadASN1(&privkey, asn1.OCTET_STRING) {
		return nil, errors.New("could not parse privkey")
	}
	private, err := tpm2.Unmarshal[tpm2.TPM2BPrivate](privkey)
	if err != nil {
		return nil, errors.New("could not parse public section of key")
	}
	tkey.Privkey = *private

	return &tkey, nil
}

func Marshal(key *TPMKey) []byte {
	var b cryptobyte.Builder

	b.AddASN1(asn1.SEQUENCE, func(b *cryptobyte.Builder) {

		b.AddASN1ObjectIdentifier(key.Keytype)

		if key.EmptyAuth {
			b.AddASN1(asn1.Tag(0).ContextSpecific().Constructed(), func(b *cryptobyte.Builder) {
				b.AddASN1Boolean(true)
			})
		}

		if len(key.Policy) != 0 {
			b.AddASN1(asn1.Tag(1).ContextSpecific().Constructed(), func(b *cryptobyte.Builder) {
				b.AddASN1(asn1.SEQUENCE, func(b *cryptobyte.Builder) {
					for _, policy := range key.Policy {
						b.AddASN1(asn1.SEQUENCE, func(b *cryptobyte.Builder) {
							b.AddASN1(asn1.Tag(0).ContextSpecific().Constructed(), func(b *cryptobyte.Builder) {
								b.AddASN1Int64(int64(policy.CommandCode))
							})
							b.AddASN1(asn1.Tag(1).ContextSpecific().Constructed(), func(b *cryptobyte.Builder) {
								b.AddASN1OctetString(policy.CommandPolicy)
							})
						})
					}
				})
			})
		}

		if len(key.Secret.Buffer) != 0 {
			b.AddASN1(asn1.Tag(2).ContextSpecific().Constructed(), func(b *cryptobyte.Builder) {
				b.AddASN1OctetString(tpm2.Marshal(key.Secret))
			})
		}

		if len(key.AuthPolicy) != 0 {
			b.AddASN1(asn1.Tag(3).ContextSpecific().Constructed(), func(b *cryptobyte.Builder) {
				b.AddASN1(asn1.SEQUENCE, func(b *cryptobyte.Builder) {
					for _, authpolicy := range key.AuthPolicy {
						b.AddASN1(asn1.SEQUENCE, func(b *cryptobyte.Builder) {
							if authpolicy.Name != "" {
								b.AddASN1(asn1.Tag(0).ContextSpecific().Constructed(), func(b *cryptobyte.Builder) {
									b.AddASN1(asn1.UTF8String, func(b *cryptobyte.Builder) {
										// TODO: Is this correct?
										b.AddBytes([]byte(authpolicy.Name))
									})
								})
							}
							// Copy of the policy writing
							b.AddASN1(asn1.Tag(1).ContextSpecific().Constructed(), func(b *cryptobyte.Builder) {
								b.AddASN1(asn1.SEQUENCE, func(b *cryptobyte.Builder) {
									for _, policy := range authpolicy.Policy {
										b.AddASN1(asn1.SEQUENCE, func(b *cryptobyte.Builder) {
											b.AddASN1(asn1.Tag(0).ContextSpecific().Constructed(), func(b *cryptobyte.Builder) {
												b.AddASN1Int64(int64(policy.CommandCode))
											})
											b.AddASN1(asn1.Tag(1).ContextSpecific().Constructed(), func(b *cryptobyte.Builder) {
												b.AddASN1OctetString(policy.CommandPolicy)
											})
										})
									}
								})
							})
						})
					}
				})
			})
		}

		if len(key.Description) != 0 {
			b.AddASN1(asn1.Tag(4).ContextSpecific().Constructed(), func(b *cryptobyte.Builder) {
				b.AddASN1(asn1.UTF8String, func(b *cryptobyte.Builder) {
					b.AddBytes([]byte(key.Description))
				})
			})
		}

		b.AddASN1Int64(int64(key.Parent))
		b.AddASN1OctetString(tpm2.Marshal(key.Pubkey))
		b.AddASN1OctetString(tpm2.Marshal(key.Privkey))
	})

	return b.BytesOrPanic()
}

// Errors
var (
	ErrNotTPMKey = errors.New("not a TSS2 PRIVATE KEY")
)

var (
	pemType = "TSS2 PRIVATE KEY"
)

func Encode(out io.Writer, key *TPMKey) error {
	return pem.Encode(out, &pem.Block{
		Type:  pemType,
		Bytes: Marshal(key),
	})
}

func Decode(b []byte) (*TPMKey, error) {
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, fmt.Errorf("not an armored key")
	}
	switch block.Type {
	case pemType:
		return Parse(block.Bytes)
	default:
		return nil, ErrNotTPMKey
	}
}
