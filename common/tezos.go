package common

import (
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants/enums"
)

type Delegator struct {
	Address          tezos.Address
	DelegatedBalance tezos.Z
	StakedBalance    tezos.Z
	Emptied          bool
}

type BakersCycleData struct {
	OwnStakingBalance        tezos.Z
	ExternalStakingBalance   tezos.Z
	OwnDelegatedBalance      tezos.Z
	ExternalDelegatedBalance tezos.Z
	BlockRewards             tezos.Z
	IdealBlockRewards        tezos.Z
	EndorsementRewards       tezos.Z
	IdealEndorsementRewards  tezos.Z
	FrozenDepositLimit       tezos.Z
	DelegatorsCount          int32
	StakersCount             int32
	BlockFees                tezos.Z
	Delegators               []Delegator
}

type ShareInfo struct {
	Baker      tezos.Z
	Delegators map[string]tezos.Z
}

func (cycleData *BakersCycleData) getTotalRewards() tezos.Z {
	return cycleData.BlockFees.Add(cycleData.BlockRewards).Add(cycleData.EndorsementRewards)
}

func (cycleData *BakersCycleData) getIdealRewards() tezos.Z {
	return cycleData.IdealBlockRewards.Add(cycleData.IdealEndorsementRewards).Add(cycleData.BlockFees)
}

// GetTotalRewards returns the total rewards for the cycle based on payout mode
func (cycleData *BakersCycleData) GetTotalRewards(payoutMode enums.EPayoutMode) tezos.Z {
	switch payoutMode {
	case enums.PAYOUT_MODE_IDEAL:
		return cycleData.getIdealRewards()
	default:
		return cycleData.getTotalRewards()
	}
}

func (cycleData *BakersCycleData) GetBakerBalance() tezos.Z {
	return cycleData.OwnStakingBalance.Add(cycleData.OwnDelegatedBalance)
}

type OperationLimits struct {
	HardGasLimitPerOperation     int64
	HardStorageLimitPerOperation int64
	MaxOperationDataLength       int
}
