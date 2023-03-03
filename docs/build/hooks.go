package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/core/generate"
)

func GenerateHookSampleData() {
	payoutCandidate := generate.PayoutCandidateWithBondAmountAndFee{
		PayoutCandidateWithBondAmount: generate.PayoutCandidateWithBondAmount{
			PayoutCandidate: generate.PayoutCandidate{
				Source:                       tezos.ZeroAddress,
				Recipient:                    tezos.ZeroAddress,
				FeeRate:                      5.0,
				Balance:                      tezos.NewZ(1000000000),
				IsInvalid:                    true,
				IsEmptied:                    true,
				IsBakerPayingTxFee:           true,
				IsBakerPayingAllocationTxFee: true,
				InvalidBecause:               "reason",
			},
			BondsAmount: tezos.NewZ(1000000000),
			TxKind:      "fa1",
			FATokenId:   tezos.NewZ(10),
			FAContract:  tezos.ZeroContract,
		},
		Fee: tezos.NewZ(1000000000),
	}

	acg := generate.AfterCandidateGeneratedHookData{
		payoutCandidate.PayoutCandidate,
	}

	abd := generate.AfterBondsDistributedHookData{
		payoutCandidate.PayoutCandidateWithBondAmount,
	}
	acb := generate.CheckBalanceHookData{
		SkipTezCheck: true,
		Message:      "This message is used to carry errors from hook to the caller.",
		IsSufficient: true,
		Payouts: []generate.PayoutCandidateWithBondAmount{
			payoutCandidate.PayoutCandidateWithBondAmount,
		},
	}
	ofc := generate.OnFeesCollectionHookData{
		payoutCandidate,
	}

	simulatedCandidate := generate.PayoutCandidateSimulated{
		PayoutCandidateWithBondAmountAndFee: payoutCandidate,
		AllocationBurn:                      1,
		StorageBurn:                         1,
		OpLimits: &common.OpLimits{
			TransactionFee: 1,
			StorageLimit:   1,
			GasLimit:       1,
		},
	}

	apg := generate.AfterPayoutsBlueprintGeneratedHookData{
		Cycle: 1,
		Payouts: []common.PayoutRecipe{
			simulatedCandidate.ToPayoutRecipe(tezos.ZeroAddress, 1, enums.PAYOUT_KIND_DELEGATOR_REWARD),
		},
		Summary: common.CyclePayoutSummary{
			Cycle:              1,
			Delegators:         2,
			PaidDelegators:     1,
			StakingBalance:     tezos.NewZ(1000000000),
			EarnedFees:         tezos.NewZ(1000000000),
			EarnedRewards:      tezos.NewZ(1000000000),
			DistributedRewards: tezos.NewZ(1000000000),
			BondIncome:         tezos.NewZ(1000000000),
			FeeIncome:          tezos.NewZ(1000000000),
			IncomeTotal:        tezos.NewZ(1000000000),
			DonatedBonds:       tezos.NewZ(1000000000),
			DonatedFees:        tezos.NewZ(1000000000),
			DonatedTotal:       tezos.NewZ(1000000000),
			Timestamp:          time.Now(),
		},
	}

	result := "# Available Hooks\n\n"
	result += "NOTE: *all bellow examples are just sample data to showcase fields used in data passed to hooks.*\n\n"

	result += fmt.Sprintf("## %s\n\n", enums.EXTENSION_HOOK_AFTER_CANDIDATE_GENERATED)
	result += "This hook is capable of mutating data.\n"
	result += "```json\n"
	acgSerialized, _ := json.MarshalIndent(acg, "", "  ")
	result += string(acgSerialized)
	result += "\n```\n\n"

	result += fmt.Sprintf("## %s\n\n", enums.EXTENSION_HOOK_AFTER_BONDS_DISTRIBUTED)
	result += "This hook is capable of mutating data.\n"
	result += "```json\n"
	abdSerialized, _ := json.MarshalIndent(abd, "", "  ")
	result += string(abdSerialized)
	result += "\n```\n\n"

	result += fmt.Sprintf("## %s\n\n", enums.EXTENSION_HOOK_CHECK_BALANCE)
	result += "This hook is NOT capable of mutating data.\n"
	result += "```json\n"
	acbSerialized, _ := json.MarshalIndent(acb, "", "  ")
	result += string(acbSerialized)
	result += "\n```\n\n"

	result += fmt.Sprintf("## %s\n\n", enums.EXTENSION_HOOK_ON_FEES_COLLECTION)
	result += "This hook is capable of mutating data.\n"
	result += "```json\n"
	ofcSerialized, _ := json.MarshalIndent(ofc, "", "  ")
	result += string(ofcSerialized)
	result += "\n```\n\n"

	result += fmt.Sprintf("## %s\n\n", enums.EXTENSION_HOOK_AFTER_PAYOUTS_BLUEPRINT_GENERATED)
	result += "This hook is NOT capable of mutating data *currently*.\n"
	result += "```json\n"
	apgSerialized, _ := json.MarshalIndent(apg, "", "  ")
	result += string(apgSerialized)
	result += "\n```\n\n"

	// write to docs/extensions/hooks.md
	os.WriteFile("docs/extensions/hooks.md", []byte(result), 0644)
}
