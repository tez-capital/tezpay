package main

import (
	"encoding/json"
	"fmt"
	"os"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/core/generate"
)

func GenerateHookSampleData() {
	acg := generate.AfterCandidateGeneratedHookData{
		generate.PayoutCandidate{
			Source:                       tezos.InvalidKey.Address(),
			Recipient:                    tezos.InvalidKey.Address(),
			FeeRate:                      5.0,
			Balance:                      tezos.NewZ(1000000000),
			IsInvalid:                    true,
			IsEmptied:                    true,
			IsBakerPayingTxFee:           true,
			IsBakerPayingAllocationTxFee: true,
			InvalidBecause:               "reason",
		},
	}

	abd := generate.AfterBondsDistributedHookData{
		generate.PayoutCandidateWithBondAmount{
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
			TxKind:      "fa2",
			FATokenId:   tezos.NewZ(10),
			FAContract:  tezos.ZeroContract,
		},
	}
	acb := generate.CheckBalanceHookData{
		SkipTezCheck: true,
		Message:      "This message is used to carry errors from hook to the caller.",
		IsSufficient: true,
		Payouts: []generate.PayoutCandidateWithBondAmount{
			{
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
				TxKind:      "tez",
				FATokenId:   tezos.NewZ(10),
				FAContract:  tezos.ZeroContract,
			},
		},
	}
	ofc := generate.OnFeesCollectionHookData{
		generate.PayoutCandidateWithBondAmountAndFee{
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

	// write to docs/extensions/hooks.md
	os.WriteFile("docs/extensions/hooks.md", []byte(result), 0644)
}
