package stages

import (
	"fmt"
	"testing"

	"blockwatch.cc/tzgo/tezos"
)

func TestCollectTransactionFees(t *testing.T) {
	result := CollectTransactionFees(WrappedStageResult{
		Ctx: Context{
			StageData: StageData{
				PayoutCandidatesWithBondAmount: []PayoutCandidateWithBondAmount{
					{
						Candidate: PayoutCandidate{
							Source:         tezos.InvalidAddress,
							Recipient:      tezos.InvalidAddress,
							FeeRate:        5.5,
							Balance:        tezos.NewZ(1000000000),
							IsInvalid:      false,
							InvalidBecause: "",
						},
						BondsAmount: tezos.NewZ(10000000),
					},
					{
						Candidate: PayoutCandidate{
							Source:         tezos.BurnAddress,
							Recipient:      tezos.BurnAddress,
							FeeRate:        7.5,
							Balance:        tezos.NewZ(2000000000),
							IsInvalid:      false,
							InvalidBecause: "",
						},
						BondsAmount: tezos.NewZ(20000000),
					},
				},
			},
		},
		Err: fmt.Errorf("test"),
	})
	fmt.Println(result)
}
