package main

import (
	"context"
	"fmt"

	"github.com/trilitech/tzgo/contract"
	"github.com/trilitech/tzgo/rpc"
	"github.com/trilitech/tzgo/tezos"
)

type Contract interface {
	GetBalance(ctx context.Context) (tezos.Z, error)
}

type FaContract struct {
	con       *contract.Contract
	token     TokenConfiguration
	payoutPKH string
}

func (f *FaContract) GetBalance(ctx context.Context) (tezos.Z, error) {
	switch f.token.Kind {
	case TokenKindFA1_2:
		return f.con.AsFA1().GetBalance(ctx, tezos.MustParseAddress(f.payoutPKH))
	case TokenKindFA2:
		return f.con.AsFA2(f.token.Id).GetBalance(ctx, tezos.MustParseAddress(f.payoutPKH))
	default:
		return tezos.Zero, fmt.Errorf("Unsupported token kind")
	}
}

func NewContract(ctx context.Context, rpcs []*rpc.Client, PayoutPKH string, token TokenConfiguration) (Contract, error) {
	a, err := tezos.ParseAddress(token.Contract)
	if err != nil {
		return nil, err
	}

	if a.Type() != tezos.AddressTypeContract {
		return nil, fmt.Errorf("Invalid contract address")
	}

	con, _ := AttemptWithRpcClients(ctx, rpcs, func(client *rpc.Client) (*contract.Contract, error) {
		contract := contract.NewContract(a, client)
		if err := contract.Resolve(ctx); err != nil {
			return nil, err
		}
		return contract, nil
	})

	return &FaContract{
		con:       con,
		token:     token,
		payoutPKH: PayoutPKH,
	}, nil
}
