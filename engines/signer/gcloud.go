package signer_engines

import (
	"context"
	"encoding/pem"
	"fmt"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/ecadlabs/gotez/v2/crypt"
	"github.com/tez-capital/tezpay/engines/signer/x509"
	"github.com/trilitech/tzgo/codec"
	"github.com/trilitech/tzgo/signer"
	"github.com/trilitech/tzgo/tezos"
)

type GCSigner struct {
	ctx         context.Context
	cryptPubKey crypt.PublicKey
	key         tezos.Key
	source      string
}

func InitGCSigner(ctx context.Context, kmsKeySource string) (s *GCSigner, err error) {
	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return s, err
	}
	defer client.Close()

	pk, err := client.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{Name: kmsKeySource})
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode([]byte(pk.Pem))
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	publicKey, err := crypt.NewPublicKeyFrom(pub)
	if err != nil {
		return nil, err
	}

	key, err := tezos.ParseKey(publicKey.String())
	if err != nil {
		return nil, err
	}

	return &GCSigner{
		ctx:         ctx,
		source:      kmsKeySource,
		cryptPubKey: publicKey,
		key:         key,
	}, nil
}

func (s *GCSigner) GetId() string {
	return "GCSigner"
}

func (s *GCSigner) GetPKH() tezos.Address {
	return s.key.Address()
}

func (s *GCSigner) GetKey() tezos.Key {
	return s.key
}

func (s *GCSigner) Sign(op *codec.Op) error {
	client, err := kms.NewKeyManagementClient(s.ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	req := kmspb.AsymmetricSignRequest{
		Name: s.source,
		Data: op.Digest(),
	}
	resp, err := client.AsymmetricSign(s.ctx, &req)
	if err != nil {
		return fmt.Errorf("AsymmetricSign: %w", err)
	}

	sig, err := crypt.NewSignatureFromBytes(resp.Signature, s.cryptPubKey)
	if err != nil {
		return fmt.Errorf("NewSignatureFromBytes: %w", err)
	}

	op.Signature, err = tezos.ParseSignature(sig.String())
	if err != nil {
		return fmt.Errorf("ParseSignature: %w", err)
	}

	return nil
}

func (s *GCSigner) GetSigner() signer.Signer {
	panic("GetSigner is not yet implemented")
}
