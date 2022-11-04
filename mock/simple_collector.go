package mock

import (
	"context"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/rpc"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/core/common"
)

type SimpleColletor struct {
}

var (
	defaultCtx context.Context = context.Background()
)

func InitSimpleColletor() *SimpleColletor {
	return &SimpleColletor{}
}

func (engine *SimpleColletor) GetId() string {
	return "DefaultRpcAndTzktColletor"
}

func (engine *SimpleColletor) GetCurrentCycleNumber() (int64, error) {
	return 501, nil
}

func (engine *SimpleColletor) GetLastCompletedCycle() (int64, error) {
	cycle, err := engine.GetCurrentCycleNumber()
	return cycle - 1, err
}

func (engine *SimpleColletor) GetCycleData(baker tezos.Address, cycle int64) (*common.BakersCycleData, error) {
	return &common.BakersCycleData{
		StakingBalance:     tezos.NewZ(100_000).Mul64(constants.MUTEZ_FACTOR),
		DelegatedBalance:   tezos.NewZ(1_000_000).Mul64(constants.MUTEZ_FACTOR),
		BlockRewards:       tezos.NewZ(100).Mul64(constants.MUTEZ_FACTOR),
		EndorsementRewards: tezos.NewZ(50).Mul64(constants.MUTEZ_FACTOR),
		FrozenDeposit:      tezos.NewZ(50_000).Mul64(constants.MUTEZ_FACTOR),
		NumDelegators:      2,
		BlockFees:          tezos.NewZ(25).Mul64(constants.MUTEZ_FACTOR),
		// TODO:
		Delegators: []common.Delegator{},
	}, nil
}

func (engine *SimpleColletor) WasOperationApplied(op tezos.OpHash) (bool, error) {
	return true, nil
}

func (engine *SimpleColletor) GetBranch(offset int64) (hash tezos.BlockHash, err error) {
	return tezos.ZeroBlockHash, nil
}

func (engine *SimpleColletor) Simulate(o *codec.Op, publicKey tezos.Key) (*rpc.Receipt, error) {
	// TODO:
	return &rpc.Receipt{}, nil
}

func (engine *SimpleColletor) GetBalance(addr tezos.Address) (tezos.Z, error) {
	return tezos.NewZ(100).Mul64(constants.MUTEZ_FACTOR), nil
}
