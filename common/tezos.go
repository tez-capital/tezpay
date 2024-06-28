package common

import (
	"github.com/tez-capital/tezpay/constants/enums"
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

	OwnStakedBalance              tezos.Z
	ExternalStakedBalance         tezos.Z
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

func (cycleData *BakersCycleData) getActualDelegatedRewards() tezos.Z {
	return cycleData.BlockDelegatedFees.Add(cycleData.BlockDelegatedRewards).Add(cycleData.EndorsementDelegatedRewards)
}

func (cycleData *BakersCycleData) getIdealDelegatedRewards() tezos.Z {
	return cycleData.IdealBlockDelegatedRewards.Add(cycleData.IdealEndorsementDelegatedRewards).Add(cycleData.BlockDelegatedFees)
}

// GetTotalDelegatedRewards returns the total rewards for the cycle based on payout mode
func (cycleData *BakersCycleData) GetTotalDelegatedRewards(payoutMode enums.EPayoutMode) tezos.Z {
	switch payoutMode {
	case enums.PAYOUT_MODE_IDEAL:
		return cycleData.getIdealDelegatedRewards()
	default:
		return cycleData.getActualDelegatedRewards()
	}
}

func (cycleData *BakersCycleData) GetBakerDelegatedBalance() tezos.Z {
	return cycleData.OwnDelegatedBalance
}

func (cycleData *BakersCycleData) GetBakerStakedBalance() tezos.Z {
	return cycleData.OwnStakedBalance
}

type OperationLimits struct {
	HardGasLimitPerOperation     int64
	HardStorageLimitPerOperation int64
	MaxOperationDataLength       int
}
