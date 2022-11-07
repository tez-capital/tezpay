package clients

import (
	"context"
	"fmt"
	"net/url"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/signer"
	"blockwatch.cc/tzgo/signer/remote"
	"blockwatch.cc/tzgo/tezos"
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
		return nil, fmt.Errorf("invalid remote url '%s' - %s", remoteUrl, err.Error())
	}
	rs, err := remote.New(remoteUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to remote - %s", err.Error())
	}
	addr, err := tezos.ParseAddress(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address '%s' - %s", address, err.Error())
	}

	key, err := rs.GetKey(context.Background(), addr)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key of '%s' - %s", address, err.Error())
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
