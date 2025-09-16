package x509

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	encoding_asn1 "encoding/asn1"
	"errors"
	"fmt"

	"github.com/ecadlabs/gotez/v2/crypt"
	"golang.org/x/crypto/cryptobyte"
	"golang.org/x/crypto/cryptobyte/asn1"
)

var (
	oidPublicKeyECDSA   = encoding_asn1.ObjectIdentifier{1, 2, 840, 10045, 2, 1}
	oidPublicKeyEd25519 = encoding_asn1.ObjectIdentifier{1, 3, 101, 112}

	oidNamedCurveP224 = encoding_asn1.ObjectIdentifier{1, 3, 132, 0, 33}
	oidNamedCurveP256 = encoding_asn1.ObjectIdentifier{1, 2, 840, 10045, 3, 1, 7}
	oidNamedCurveP384 = encoding_asn1.ObjectIdentifier{1, 3, 132, 0, 34}
	oidNamedCurveP521 = encoding_asn1.ObjectIdentifier{1, 3, 132, 0, 35}
	oidNamedCurveS256 = encoding_asn1.ObjectIdentifier{1, 3, 132, 0, 10} // http://www.secg.org/sec2-v2.pdf
)

func namedCurveFromOID(oid encoding_asn1.ObjectIdentifier) elliptic.Curve {
	switch {
	case oid.Equal(oidNamedCurveP224):
		return elliptic.P224()
	case oid.Equal(oidNamedCurveP256):
		return elliptic.P256()
	case oid.Equal(oidNamedCurveP384):
		return elliptic.P384()
	case oid.Equal(oidNamedCurveP521):
		return elliptic.P521()
	case oid.Equal(oidNamedCurveS256):
		return crypt.S256()
	}
	return nil
}

func ParsePKIXPublicKey(der []byte) (pub any, err error) {
	src := cryptobyte.String(der)
	var (
		obj, algo cryptobyte.String
		algoOid   encoding_asn1.ObjectIdentifier
		keyData   encoding_asn1.BitString
	)

	if !src.ReadASN1(&obj, asn1.SEQUENCE) ||
		!obj.ReadASN1(&algo, asn1.SEQUENCE) ||
		!algo.ReadASN1ObjectIdentifier(&algoOid) ||
		!obj.ReadASN1BitString(&keyData) {
		return nil, errors.New("x509: failed to parse PKIX public key")
	}

	keyBytes := keyData.RightAlign()
	switch {
	case algoOid.Equal(oidPublicKeyECDSA):
		var curveOid encoding_asn1.ObjectIdentifier
		if algo.PeekASN1Tag(asn1.OBJECT_IDENTIFIER) {
			if !algo.ReadASN1ObjectIdentifier(&curveOid) {
				return nil, errors.New("x509: failed to parse EC OID")
			}
		}
		curve := namedCurveFromOID(curveOid)
		if curve == nil {
			return nil, fmt.Errorf("x509: unknown curve: %v", curveOid)
		}
		x, y := elliptic.Unmarshal(curve, keyBytes)
		if x == nil {
			return nil, errors.New("x509: invalid EC point")
		}
		return &ecdsa.PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		}, nil

	case algoOid.Equal(oidPublicKeyEd25519):
		if len(keyBytes) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("x509: invalid Ed25519 public key length: %d", len(keyBytes))
		}
		return ed25519.PublicKey(keyBytes), nil

	default:
		return nil, fmt.Errorf("x509: unsupported algorithm: %v", algo)
	}
}
