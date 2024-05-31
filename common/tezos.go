package common

import (
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/trilitech/tzgo/tezos"
)

type Delegator struct {
	Address          tezos.Address
	DelegatedBalance tezos.Z
	StakedBalance    tezos.Z
	Emptied          bool
}

type BakersCycleData struct {
	OwnDelegatedBalance              tezos.Z
	ExternalDelegatedBalance         tezos.Z
	BlockDelegatedRewards            tezos.Z
	IdealBlockDelegatedRewards       tezos.Z
	EndorsementDelegatedRewards      tezos.Z
	IdealEndorsementDelegatedRewards tezos.Z
	BlockDelegatedFees               tezos.Z
	DelegatorsCount                  int32

	OwnStakingBalance             tezos.Z
	ExternalStakingBalance        tezos.Z
	BlockStakingRewardsEdge       tezos.Z
	EndorsementStakingRewardsEdge tezos.Z
	BlockStakingFees              tezos.Z
	StakersCount                  int32

	FrozenDepositLimit tezos.Z
	Delegators         []Delegator
}

type ShareInfo struct {
	Baker      tezos.Z
	Delegators map[string]tezos.Z
}

func (cycleData *BakersCycleData) getTotalRewards() tezos.Z {
	return cycleData.BlockDelegatedFees.Add(cycleData.BlockDelegatedRewards).Add(cycleData.EndorsementDelegatedRewards)
}

func (cycleData *BakersCycleData) getIdealRewards() tezos.Z {
	return cycleData.IdealBlockDelegatedRewards.Add(cycleData.IdealEndorsementDelegatedRewards).Add(cycleData.BlockDelegatedFees)
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
