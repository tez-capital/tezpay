package signer_engines

import (
	"context"
	"errors"
	"net/url"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/signer"
	"blockwatch.cc/tzgo/signer/remote"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants"
)

type RemoteSignerSpecs struct {
	Pkh string `json:"pkh"`
	Url string `json:"url"`
}

type RemoteSigner struct {
	Address tezos.Address
	Remote  *remote.RemoteSigner
	Key     tezos.Key
}

func InitRemoteSignerFromSpecs(specs RemoteSignerSpecs) (*RemoteSigner, error) {
	return InitRemoteSigner(specs.Pkh, specs.Url)
}

func InitRemoteSigner(address string, remoteUrl string) (*RemoteSigner, error) {
	if _, err := url.Parse(remoteUrl); err != nil {
		return nil, errors.Join(constants.ErrSignerLoadFailed, err)
	}
	rs, err := remote.New(remoteUrl, nil)
	if err != nil {
		return nil, errors.Join(constants.ErrSignerLoadFailed, err)
	}
	addr, err := tezos.ParseAddress(address)
	if err != nil {
		return nil, errors.Join(constants.ErrSignerLoadFailed, err)
	}

	key, err := rs.GetKey(context.Background(), addr)
	if err != nil {
		return nil, errors.Join(constants.ErrSignerLoadFailed, err)
	}

	return &RemoteSigner{
		Address: addr,
		Remote:  rs,
		Key:     key,
	}, nil
}

func (remoteSigner *RemoteSigner) GetId() string {
	return "RemoteSigner"
}

func (remoteSigner *RemoteSigner) GetPKH() tezos.Address {
	return remoteSigner.Address
}

func (remoteSigner *RemoteSigner) GetKey() tezos.Key {
	return remoteSigner.Key
}

func (remoteSigner *RemoteSigner) GetSigner() signer.Signer {
	return remoteSigner.Remote
}

func (remoteSigner *RemoteSigner) Sign(op *codec.Op) error {
	sig, err := remoteSigner.Remote.SignOperation(context.Background(), remoteSigner.Address, op)
	if err != nil {
		return err
	}
	op.WithSignature(sig)
	return nil
}
