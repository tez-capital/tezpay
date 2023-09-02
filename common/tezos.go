package common

import (
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants/enums"
)

type Delegator struct {
	Address tezos.Address `json:"address"`
	Balance tezos.Z       `json:"balance"`
	Emptied bool          `json:"emptied,omitempty"`
}

type BakersCycleData struct {
	StakingBalance          tezos.Z     `json:"stakingBalance"`
	DelegatedBalance        tezos.Z     `json:"delegatedBalance"`
	BlockRewards            tezos.Z     `json:"blockRewards"`
	IdealBlockRewards       tezos.Z     `json:"idealBlockRewards"`
	EndorsementRewards      tezos.Z     `json:"endorsementRewards"`
	IdealEndorsementRewards tezos.Z     `json:"idealEndorsementRewards"`
	FrozenDepositLimit      tezos.Z     `json:"frozenDepositLimit"`
	NumDelegators           int32       `json:"numDelegators"`
	BlockFees               tezos.Z     `json:"blockFees"`
	Delegators              []Delegator `json:"delegators"`
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
	return cycleData.StakingBalance.Sub(cycleData.DelegatedBalance)
}

type OperationLimits struct {
	HardGasLimitPerOperation     int64 `json:"hard_gas_limit_per_operation"`
	HardStorageLimitPerOperation int64 `json:"hard_storage_limit_per_operation"`
	MaxOperationDataLength       int   `json:"max_operation_data_length"`
}
