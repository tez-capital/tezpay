package common

import (
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants/enums"
)

type Delegator struct {
	Address tezos.Address
	Balance tezos.Z
	Emptied bool
}

type BakersCycleData struct {
	StakingBalance          tezos.Z
	DelegatedBalance        tezos.Z
	BlockRewards            tezos.Z
	IdealBlockRewards       tezos.Z
	EndorsementRewards      tezos.Z
	IdealEndorsementRewards tezos.Z
	FrozenDepositLimit      tezos.Z
	NumDelegators           int32
	BlockFees               tezos.Z
	Delegators              []Delegator
}

type ShareInfo struct {
	Baker      tezos.Z
	Delegators map[string]tezos.Z
}

func (cycleData *BakersCycleData) getTotalRewards() tezos.Z {
	return cycleData.BlockFees.Add(cycleData.BlockRewards).Add(cycleData.EndorsementRewards)
}

func (cycleData *BakersCycleData) getIdealRewards() tezos.Z {
	return cycleData.IdealBlockRewards.Add(cycleData.IdealEndorsementRewards)
}

// GetTotalRewards returns the total rewards for the cycle based on payout mode
func (cycleData *BakersCycleData) GetTotalRewards(payoutMode enums.EPayoutMode) tezos.Z {
	switch payoutMode {
	case enums.PAYOUT_MODE_IDEAL:
		return cycleData.getIdealRewards()
	case enums.PAYOUT_MODE_BEST:
		return tezos.MaxZ(cycleData.getTotalRewards(), cycleData.getIdealRewards())
	default:
		return cycleData.getTotalRewards()
	}
}

func (cycleData *BakersCycleData) GetBakerBalance() tezos.Z {
	return cycleData.StakingBalance.Sub(cycleData.DelegatedBalance)
}

type OperationLimits struct {
	HardGasLimitPerOperation     int64
	HardStorageLimitPerOperation int64
	MaxOperationDataLength       int
}
