package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/trilitech/tzgo/contract"
	"github.com/trilitech/tzgo/micheline"
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

type FA2BalanceRequest struct {
	Owner   tezos.Address `json:"owner"`
	TokenId tezos.Z       `json:"token_id"`
}

type FA2BalanceResponse struct {
	Request FA2BalanceRequest `json:"request"`
	Balance tezos.Z           `json:"balance"`
}

func (f *FaContract) GetFa2Balance(ctx context.Context) (tezos.Z, error) {
	req := []contract.FA2BalanceRequest{
		{
			Owner:   tezos.MustParseAddress(f.payoutPKH),
			TokenId: tezos.NewZ(f.token.Id),
		},
	}

	args := micheline.NewSeq()
	for _, r := range req {
		args.Args = append(args.Args, micheline.NewPair(
			micheline.NewBytes(r.Owner.EncodePadded()),
			micheline.NewNat(r.TokenId.Big()),
		))
	}
	prim, err := f.con.RunCallback(ctx, "balance_of", args)
	if err != nil {
		return tezos.Zero, err
	}

	val := micheline.NewValue(
		micheline.NewType(micheline.NewCode(micheline.T_LIST,
			micheline.NewPairType(
				micheline.NewPairType(
					micheline.NewCodeAnno(micheline.T_ADDRESS, "%owner"),
					micheline.NewCodeAnno(micheline.T_NAT, "%token_id"),
					"%request",
				),
				micheline.NewCodeAnno(micheline.T_NAT, "%balance"),
			),
		)),
		prim,
	)
	resp := make([]FA2BalanceResponse, 0)
	err = val.Unmarshal(&resp)
	if err != nil {
		return tezos.Zero, err
	}

	return resp[0].Balance, nil
}

func (f *FaContract) GetBalance(ctx context.Context) (tezos.Z, error) {
	return f.GetBalanceOf(ctx, f.payoutPKH)
}

func (f *FaContract) GetBalanceOf(ctx context.Context, addr string) (tezos.Z, error) {
	switch f.token.Kind {
	case TokenKindFA1_2:
		balance, err := f.con.AsFA1().GetBalance(ctx, tezos.MustParseAddress(addr))
		if err != nil {
			return tezos.Zero, errors.Join(errors.New("failed to get FA1_2 balance"), err)
		}
		return balance, err
	case TokenKindFA2:
		balance, err := f.GetFa2Balance(ctx)
		if err != nil {
			return tezos.Zero, errors.Join(errors.New("failed to get FA2 balance"), err)
		}
		return balance, nil
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
