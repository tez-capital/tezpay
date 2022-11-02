package tezos

import (
	"blockwatch.cc/tzgo/tezos"
)

type Delegator struct {
	Address tezos.Address
	Balance tezos.Z
	Emptied bool
}

type BakersCycleData struct {
	StakingBalance     tezos.Z
	DelegatedBalance   tezos.Z
	BlockRewards       tezos.Z
	EndorsementRewards tezos.Z
	FrozenDeposit      tezos.Z
	NumDelegators      int32
	BlockFees          tezos.Z
	Delegators         []Delegator
}

type ShareInfo struct {
	Baker      tezos.Z
	Delegators map[string]tezos.Z
}

func (cycleData *BakersCycleData) GetTotalRewards() tezos.Z {
	return cycleData.BlockFees.Add(cycleData.BlockRewards).Add(cycleData.EndorsementRewards)
}

func (cycleData *BakersCycleData) GetBakerBalance() tezos.Z {
	return cycleData.StakingBalance.Sub(cycleData.DelegatedBalance)
}

type OperationLimits struct {
	HardGasLimitPerOperation     int64
	HardStorageLimitPerOperation int64
	MaxOperationDataLength       int
}
