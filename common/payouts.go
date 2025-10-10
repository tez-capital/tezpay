package common

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/samber/lo"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/trilitech/tzgo/base58"
	"github.com/trilitech/tzgo/tezos"
)

type OpLimits struct {
	TransactionFee          int64 `json:"transaction_fee,omitempty"`
	StorageLimit            int64 `json:"storage_limit,omitempty"`
	GasLimit                int64 `json:"gas_limit,omitempty"`
	DeserializationGasLimit int64 `json:"deserialization_gas_limit,omitempty"`
	AllocationBurn          int64 `json:"allocation_burn,omitempty"`
	StorageBurn             int64 `json:"storage_burn,omitempty"`
}

func (psr *OpLimits) GetOperationTotalFees() int64 {
	return psr.TransactionFee + psr.AllocationBurn + psr.StorageBurn
}

func (psr *OpLimits) GetAllocationFee() int64 {
	return psr.AllocationBurn
}

func (psr *OpLimits) GetOperationFeesWithoutAllocation() int64 {
	return psr.TransactionFee + psr.StorageBurn
}

type PayoutRecipe struct {
	Baker            tezos.Address                `json:"baker"`
	Delegator        tezos.Address                `json:"delegator,omitempty"`
	Cycle            int64                        `json:"cycle,omitempty"`
	Recipient        tezos.Address                `json:"recipient,omitempty"`
	Kind             enums.EPayoutKind            `json:"kind,omitempty"`
	TxKind           enums.EPayoutTransactionKind `json:"tx_kind,omitempty"`
	FATokenId        tezos.Z                      `json:"fa_token_id,omitempty"`
	FAContract       tezos.Address                `json:"fa_contract,omitempty"`
	FAAlias          string                       `json:"fa_alias,omitempty"`
	FADecimals       int                          `json:"fa_decimals,omitempty"`
	DelegatedBalance tezos.Z                      `json:"delegator_balance,omitempty"`
	StakedBalance    tezos.Z                      `json:"staked_balance,omitempty"`
	Amount           tezos.Z                      `json:"amount,omitempty"`
	FeeRate          float64                      `json:"fee_rate,omitempty"`
	Fee              tezos.Z                      `json:"fee,omitempty"`
	OpLimits         *OpLimits                    `json:"op_limits,omitempty"`
	Note             string                       `json:"note,omitempty"`
	IsValid          bool                         `json:"valid,omitempty"`
}

func (candidate *PayoutRecipe) GetDestination() tezos.Address {
	return candidate.Recipient
}

func (candidate *PayoutRecipe) GetTxKind() enums.EPayoutTransactionKind {
	return candidate.TxKind
}

func (candidate *PayoutRecipe) GetFATokenId() tezos.Z {
	return candidate.FATokenId
}

func (candidate *PayoutRecipe) GetFAContract() tezos.Address {
	return candidate.FAContract
}

func (candidate *PayoutRecipe) GetAmount() tezos.Z {
	return candidate.Amount
}

type PayoutRecipeIdentifier struct {
	Delegator  tezos.Address                `json:"delegator,omitempty"`
	Recipient  tezos.Address                `json:"recipient,omitempty"`
	Kind       enums.EPayoutKind            `json:"kind,omitempty"`
	TxKind     enums.EPayoutTransactionKind `json:"tx_kind,omitempty"`
	FATokenId  tezos.Z                      `json:"fa_token_id,omitempty"`
	FAContract tezos.Address                `json:"fa_contract,omitempty"`
	IsValid    bool                         `json:"valid,omitempty"`
}

func (identifier *PayoutRecipeIdentifier) ToJSON() ([]byte, error) {
	return json.Marshal(identifier)
}

func (recipe *PayoutRecipe) GetIdentifier() string {
	identifier := PayoutRecipeIdentifier{
		Delegator:  recipe.Delegator,
		Recipient:  recipe.Recipient,
		Kind:       recipe.Kind,
		TxKind:     recipe.TxKind,
		FATokenId:  recipe.FATokenId,
		FAContract: recipe.FAContract,
		IsValid:    recipe.IsValid,
	}
	k, err := identifier.ToJSON()
	if err != nil {
		return ""
	}
	hashBytes := sha256.Sum256(k)
	return base58.Encode(hashBytes[:])
}

func (recipe *PayoutRecipe) GetShortIdentifier() string {
	return recipe.GetIdentifier()[:16]
}

func (recipe PayoutRecipe) AsAccumulated() *AccumulatedPayoutRecipe {
	return &AccumulatedPayoutRecipe{
		PayoutRecipe: recipe,
		Accumulated:  []*PayoutRecipe{&recipe},
	}
}

func (pr *PayoutRecipe) ToPayoutReport() PayoutReport {
	txFee := int64(0)
	if pr.OpLimits != nil {
		txFee = pr.OpLimits.TransactionFee
	}

	return PayoutReport{
		Id:               pr.GetShortIdentifier(),
		Baker:            pr.Baker,
		Timestamp:        time.Now(),
		Cycle:            pr.Cycle,
		Kind:             pr.Kind,
		TxKind:           pr.TxKind,
		FAContract:       pr.FAContract,
		FATokenId:        pr.FATokenId,
		FAAlias:          pr.FAAlias,
		FADecimals:       pr.FADecimals,
		Delegator:        pr.Delegator,
		DelegatedBalance: pr.DelegatedBalance,
		StakedBalance:    pr.StakedBalance,
		Recipient:        pr.Recipient,
		Amount:           pr.Amount,
		FeeRate:          pr.FeeRate,
		Fee:              pr.Fee,
		TransactionFee:   txFee,
		OpHash:           tezos.ZeroOpHash,
		IsSuccess:        false,
		Note:             pr.Note,
	}
}

func (pr *PayoutRecipe) GetTransactionFee() int64 {
	if pr.OpLimits != nil {
		return pr.OpLimits.TransactionFee
	}
	return 0
}

func (pr *PayoutRecipe) ToTableRowData() []string {
	return []string{
		ShortenAddress(pr.Delegator),
		ShortenAddress(pr.Recipient),
		MutezToTezS(pr.DelegatedBalance.Int64()),
		string(pr.Kind),
		ShortenAddress(pr.FAContract),
		ToStringEmptyIfZero(pr.FATokenId.Int64()),
		FormatTokenAmount(pr.TxKind, pr.Amount.Int64(), pr.FAAlias, pr.FADecimals),
		FloatToPercentage(pr.FeeRate),
		MutezToTezS(pr.Fee.Int64()),
		MutezToTezS(pr.GetTransactionFee()),
		pr.Note,
	}
}

func (pr *PayoutRecipe) GetTableHeaders() []string {
	return []string{
		"Delegator",
		"Recipient",
		"Delegated Balance",
		"Kind",
		"FA Contract",
		"FA Token Id",
		"Amount",
		"Fee Rate",
		"Fee",
		"Tx Fee",
		"Note",
	}
}

func GetRecipesTotals(recipes []PayoutRecipe) []string {
	totalAmount := int64(0)
	totalFee := int64(0)
	totalTx := int64(0)
	for _, recipe := range recipes {
		if recipe.TxKind == enums.PAYOUT_TX_KIND_TEZ {
			totalAmount += recipe.Amount.Int64()
		}
		totalFee += recipe.Fee.Int64()
		totalTx += recipe.GetTransactionFee()
	}
	return []string{
		"",
		"",
		"",
		"",
		"",
		"",
		MutezToTezS(totalAmount),
		"",
		MutezToTezS(totalFee),
		MutezToTezS(totalTx),
		"",
	}
}

// returns totals and number of filtered recipes
func GetRecipesFilteredTotals(recipes []PayoutRecipe, kind enums.EPayoutKind) ([]string, int) {
	r := lo.Filter(recipes, func(recipe PayoutRecipe, _ int) bool {
		return recipe.Kind == kind
	})
	return GetRecipesTotals(r), len(r)
}

type AccumulatedPayoutRecipe struct {
	PayoutRecipe
	Accumulated []*PayoutRecipe `json:"-"`
}

func (recipe *AccumulatedPayoutRecipe) Add(otherRecipe *PayoutRecipe) (*AccumulatedPayoutRecipe, error) {
	if !recipe.Recipient.Equal(otherRecipe.Recipient) {
		return nil, errors.New("cannot add different recipients")
	}
	if !recipe.Delegator.Equal(otherRecipe.Delegator) {
		return nil, errors.New("cannot add different delegators")
	}
	if recipe.Kind != otherRecipe.Kind {
		return nil, errors.New("cannot add different kinds")
	}
	if recipe.TxKind != otherRecipe.TxKind {
		return nil, errors.New("cannot add different tx kinds")
	}
	if !recipe.FATokenId.Equal(otherRecipe.FATokenId) {
		return nil, errors.New("cannot add different FA token ids")
	}
	if !recipe.FAContract.Equal(otherRecipe.FAContract) {
		return nil, errors.New("cannot add different FA contracts")
	}
	if recipe.IsValid != otherRecipe.IsValid {
		return nil, errors.New("cannot add different validities")
	}
	if (recipe.OpLimits == nil || otherRecipe.OpLimits == nil) && recipe.IsValid {
		return nil, errors.New("cannot add valid recipes with missing op limits")
	}

	recipe.DelegatedBalance = recipe.DelegatedBalance.Add(otherRecipe.DelegatedBalance).Div64(2)
	recipe.StakedBalance = recipe.StakedBalance.Add(otherRecipe.StakedBalance).Div64(2)
	recipe.Amount = recipe.Amount.Add(otherRecipe.Amount)
	recipe.Fee = recipe.Fee.Add(otherRecipe.Fee)
	if recipe.IsValid { // only accumulate op limits if valid recipes
		recipe.OpLimits = &OpLimits{
			StorageBurn:             recipe.OpLimits.StorageBurn + otherRecipe.OpLimits.StorageBurn,
			AllocationBurn:          recipe.OpLimits.AllocationBurn + otherRecipe.OpLimits.AllocationBurn,
			TransactionFee:          recipe.OpLimits.TransactionFee + otherRecipe.OpLimits.TransactionFee,
			StorageLimit:            recipe.OpLimits.StorageLimit + otherRecipe.OpLimits.StorageLimit,
			GasLimit:                recipe.OpLimits.GasLimit + otherRecipe.OpLimits.GasLimit,
			DeserializationGasLimit: recipe.OpLimits.DeserializationGasLimit + otherRecipe.OpLimits.DeserializationGasLimit,
		}
	}

	otherRecipe.Kind = enums.PAYOUT_KIND_ACCUMULATED
	otherRecipe.Note = fmt.Sprintf("%s_%d", recipe.GetShortIdentifier(), recipe.Cycle)
	recipe.Accumulated = append(recipe.Accumulated, otherRecipe)
	return recipe, nil
}

func (pr *AccumulatedPayoutRecipe) GetAccumulatedIdentifier() string {
	return fmt.Sprintf("%s #%d", pr.GetShortIdentifier(), pr.Cycle)
}

func (pr *AccumulatedPayoutRecipe) GetAccumulatedPayoutDetails() (wasAccumulated bool, id string, cycle int64) {
	if pr.Kind != enums.PAYOUT_KIND_ACCUMULATED {
		return false, "", 0
	}
	if len(pr.Note) > 0 {
		_, err := fmt.Sscanf(pr.Note, "%s_%d", &id, &cycle)
		if err == nil {
			return true, id, cycle
		}
	}

	return false, "", 0
}

func (pr *AccumulatedPayoutRecipe) ToPayoutReport() PayoutReport {
	report := pr.PayoutRecipe.ToPayoutReport()
	report.Accumulated = lo.Map(pr.Accumulated, func(p *PayoutRecipe, _ int) *PayoutReport {
		accumulated := p.ToPayoutReport()
		return &accumulated
	})
	return report
}

func (pr *AccumulatedPayoutRecipe) DisperseToInvalid() []PayoutRecipe {
	if pr.IsValid {
		panic("THIS SHOULD NEVER HAPPEN: cannot disperse valid accumulated payout")
	}

	return lo.Map(pr.Accumulated, func(r *PayoutRecipe, _ int) PayoutRecipe {
		r.IsValid = false
		r.Note = pr.Note
		r.Fee = r.Fee.Add(pr.Amount) // collect the whole bonds amount as fee if invalid
		r.Amount = tezos.Zero
		return *r
	})
}

// AsRecipe returns the PayoutRecipe representation of the AccumulatedPayoutRecipe.
// This is useful only for printing and reporting purposes. Do not use it for execution.
func (pr *AccumulatedPayoutRecipe) AsRecipe() PayoutRecipe {
	return pr.PayoutRecipe
}

type CyclePayoutSummary struct {
	Delegators               int       `json:"delegators"`
	PaidDelegators           int       `json:"paid_delegators"`
	OwnStakedBalance         tezos.Z   `json:"own_staked_balance"`
	OwnDelegatedBalance      tezos.Z   `json:"own_delegated_balance"`
	ExternalStakedBalance    tezos.Z   `json:"external_staked_balance"`
	ExternalDelegatedBalance tezos.Z   `json:"external_delegated_balance"`
	EarnedFees               tezos.Z   `json:"cycle_fees"`
	EarnedRewards            tezos.Z   `json:"cycle_rewards"`
	DistributedRewards       tezos.Z   `json:"distributed_rewards"`
	BondIncome               tezos.Z   `json:"bond_income"`
	FeeIncome                tezos.Z   `json:"fee_income"`
	IncomeTotal              tezos.Z   `json:"total_income"`
	TransactionFeesPaid      tezos.Z   `json:"transaction_fees_paid"`
	DonatedBonds             tezos.Z   `json:"donated_bonds"`
	DonatedFees              tezos.Z   `json:"donated_fees"`
	DonatedTotal             tezos.Z   `json:"donated_total"`
	Timestamp                time.Time `json:"timestamp"`
}

type PayoutSummary struct {
	CyclePayoutSummary
	Cycles         []int64                      `json:"cycle"`
	CycleSummaries map[int64]CyclePayoutSummary `json:"cycle_summaries,omitempty"`
}

func (summary *PayoutSummary) GetTotalStakedBalance() tezos.Z {
	return summary.OwnStakedBalance.Add(summary.ExternalStakedBalance)
}

func (summary *PayoutSummary) GetTotalDelegatedBalance() tezos.Z {
	return summary.OwnDelegatedBalance.Add(summary.ExternalDelegatedBalance)
}

func (summary *PayoutSummary) AddCycleSummary(cycle int64, another *CyclePayoutSummary) *PayoutSummary {
	if summary.CycleSummaries == nil {
		summary.CycleSummaries = make(map[int64]CyclePayoutSummary)
	}
	if _, ok := summary.CycleSummaries[cycle]; ok {
		panic("cannot add the same cycle summary twice")
	}
	cycles := append(summary.Cycles, cycle)
	cycles = lo.Uniq(cycles)

	cycleSummaries := maps.Clone(summary.CycleSummaries)
	cycleSummaries[cycle] = *another

	return &PayoutSummary{
		Cycles: cycles,
		CyclePayoutSummary: CyclePayoutSummary{
			OwnStakedBalance:         summary.OwnStakedBalance.Add(another.OwnStakedBalance),
			OwnDelegatedBalance:      summary.OwnDelegatedBalance.Add(another.OwnDelegatedBalance),
			ExternalStakedBalance:    summary.ExternalStakedBalance.Add(another.ExternalStakedBalance),
			ExternalDelegatedBalance: summary.ExternalDelegatedBalance.Add(another.ExternalDelegatedBalance),
			EarnedFees:               summary.EarnedFees.Add(another.EarnedFees),
			EarnedRewards:            summary.EarnedRewards.Add(another.EarnedRewards),
			DistributedRewards:       summary.DistributedRewards.Add(another.DistributedRewards),
			BondIncome:               summary.BondIncome.Add(another.BondIncome),
			FeeIncome:                summary.FeeIncome.Add(another.FeeIncome),
			IncomeTotal:              summary.IncomeTotal.Add(another.IncomeTotal),
			TransactionFeesPaid:      summary.TransactionFeesPaid.Add(another.TransactionFeesPaid),
			DonatedBonds:             summary.DonatedBonds.Add(another.DonatedBonds),
			DonatedFees:              summary.DonatedFees.Add(another.DonatedFees),
			DonatedTotal:             summary.DonatedTotal.Add(another.DonatedTotal),
		},
		CycleSummaries: cycleSummaries,
	}
}

type CyclePayoutBlueprint struct {
	Cycle   int64          `json:"cycles,omitempty"`
	Payouts []PayoutRecipe `json:"payouts,omitempty"`

	OwnStakedBalance         tezos.Z `json:"own_staked_balance"`
	OwnDelegatedBalance      tezos.Z `json:"own_delegated_balance"`
	ExternalStakedBalance    tezos.Z `json:"external_staked_balance"`
	ExternalDelegatedBalance tezos.Z `json:"external_delegated_balance"`
	EarnedFees               tezos.Z `json:"cycle_fees"`
	EarnedRewards            tezos.Z `json:"cycle_rewards"`
	BondIncome               tezos.Z `json:"bond_income"`
	DonatedBonds             tezos.Z `json:"donated_bonds"`
	DonatedFees              tezos.Z `json:"donated_fees"`
	DonatedTotal             tezos.Z `json:"donated_total"`
	// Summary                              CyclePayoutSummary `json:"summary,omitempty"`
	BatchMetadataDeserializationGasLimit int64 `json:"batch_metadata_deserialization_gas_limit,omitempty"`
}

type GeneratePayoutsEngineContext struct {
	collector   CollectorEngine
	signer      SignerEngine
	adminNotify func(msg string)
}

func NewGeneratePayoutsEngines(collector CollectorEngine, signer SignerEngine, adminNotify func(msg string)) *GeneratePayoutsEngineContext {
	return &GeneratePayoutsEngineContext{
		collector:   collector,
		signer:      signer,
		adminNotify: adminNotify,
	}
}

func (engines *GeneratePayoutsEngineContext) GetSigner() SignerEngine {
	return engines.signer
}

func (engines *GeneratePayoutsEngineContext) GetCollector() CollectorEngine {
	return engines.collector
}

func (engines *GeneratePayoutsEngineContext) AdminNotify(msg string) {
	if engines.adminNotify != nil {
		engines.adminNotify(msg)
	}
}

func (engines *GeneratePayoutsEngineContext) Validate() error {
	if engines.signer == nil {
		return errors.Join(constants.ErrMissingEngine, constants.ErrMissingSignerEngine)
	}
	if engines.collector == nil {
		return errors.Join(constants.ErrMissingEngine, constants.ErrMissingCollectorEngine)
	}
	return nil
}

type GeneratePayoutsOptions struct {
	Cycle                    int64 `json:"cycle,omitempty"`
	SkipBalanceCheck         bool  `json:"skip_balance_check,omitempty"`
	WaitForSufficientBalance bool  `json:"wait_for_sufficient_balance,omitempty"`
}

type CyclePayoutBlueprints []*CyclePayoutBlueprint

// func (results CyclePayoutBlueprints) GetSummary() *PayoutSummary {
// 	summary := &PayoutSummary{
// 		Cycles: make([]int64, 0, len(results)),
// 	}
// 	delegators := 0
// 	for _, result := range results {
// 		delegators += result.Summary.Delegators
// 		summary = summary.AddCycleSummary(result.Cycle, &result.Summary)
// 	}
// 	summary.Delegators = delegators / len(results) // average
// 	return summary
// }

func (results CyclePayoutBlueprints) GetCycles() []int64 {
	return lo.Reduce(results, func(acc []int64, result *CyclePayoutBlueprint, _ int) []int64 {
		for _, p := range result.Payouts {
			if !slices.Contains(acc, p.Cycle) {
				acc = append(acc, p.Cycle)
			}
		}
		return acc
	}, []int64{})
}

type PreparePayoutsEngineContext struct {
	collector   CollectorEngine
	signer      SignerEngine
	reporter    ReporterEngine
	adminNotify func(msg string)
}

func NewPreparePayoutsEngineContext(collector CollectorEngine, signer SignerEngine, reporter ReporterEngine, adminNotify func(msg string)) *PreparePayoutsEngineContext {
	return &PreparePayoutsEngineContext{
		collector:   collector,
		adminNotify: adminNotify,
		signer:      signer,
		reporter:    reporter,
	}
}

func (engines *PreparePayoutsEngineContext) GetCollector() CollectorEngine {
	return engines.collector
}

func (engines *PreparePayoutsEngineContext) GetSigner() SignerEngine {
	return engines.signer
}

func (engines *PreparePayoutsEngineContext) GetReporter() ReporterEngine {
	return engines.reporter
}

func (engines *PreparePayoutsEngineContext) AdminNotify(msg string) {
	if engines.adminNotify != nil {
		engines.adminNotify(msg)
	}
}

func (engines *PreparePayoutsEngineContext) Validate() error {
	if engines.collector == nil {
		return errors.Join(constants.ErrMissingEngine, constants.ErrMissingCollectorEngine)
	}
	if engines.reporter == nil {
		return errors.Join(constants.ErrMissingEngine, constants.ErrMissingReporterEngine)
	}
	return nil
}

type PreparePayoutsOptions struct {
	Accumulate bool `json:"accumulate,omitempty"`
}

type PreparePayoutsResult struct {
	Blueprints                     []*CyclePayoutBlueprint    `json:"blueprint,omitempty"`
	ValidPayouts                   []*AccumulatedPayoutRecipe `json:"payouts,omitempty"`
	InvalidPayouts                 []PayoutRecipe             `json:"invalid_payouts,omitempty"`
	ReportsOfPastSuccessfulPayouts []PayoutReport             `json:"reports_of_past_successful_payouts,omitempty"`
}

func (result *PreparePayoutsResult) GetCycles() []int64 {
	return lo.Reduce(result.Blueprints, func(acc []int64, blueprint *CyclePayoutBlueprint, _ int) []int64 {
		if !slices.Contains(acc, blueprint.Cycle) {
			acc = append(acc, blueprint.Cycle)
		}
		return acc
	}, []int64{})
}

type ExecutePayoutsEngineContext struct {
	signer      SignerEngine
	transactor  TransactorEngine
	reporter    ReporterEngine
	adminNotify func(msg string)
}

func NewExecutePayoutsEngineContext(signer SignerEngine, transactor TransactorEngine, reporter ReporterEngine, adminNotify func(msg string)) *ExecutePayoutsEngineContext {
	return &ExecutePayoutsEngineContext{
		signer:      signer,
		transactor:  transactor,
		reporter:    reporter,
		adminNotify: adminNotify,
	}
}

func (engines *ExecutePayoutsEngineContext) GetSigner() SignerEngine {
	return engines.signer
}

func (engines *ExecutePayoutsEngineContext) GetTransactor() TransactorEngine {
	return engines.transactor
}

func (engines *ExecutePayoutsEngineContext) GetReporter() ReporterEngine {
	return engines.reporter
}

func (engines *ExecutePayoutsEngineContext) AdminNotify(msg string) {
	if engines.adminNotify != nil {
		engines.adminNotify(msg)
	}
}

func (engines *ExecutePayoutsEngineContext) Validate() error {
	if engines.signer == nil {
		return errors.Join(constants.ErrMissingEngine, constants.ErrMissingSignerEngine)
	}
	if engines.transactor == nil {
		return errors.Join(constants.ErrMissingEngine, constants.ErrMissingTransactorEngine)
	}
	if engines.reporter == nil {
		return errors.Join(constants.ErrMissingEngine, constants.ErrMissingReporterEngine)
	}
	return nil
}

type ExecutePayoutsOptions struct {
	MixInContractCalls bool `json:"mix_in_contract_calls,omitempty"`
	MixInFATransfers   bool `json:"mix_in_fa_transfers,omitempty"`
	DryRun             bool `json:"dry_run,omitempty"`
}

type ExecutePayoutsResult struct {
	BatchResults   BatchResults  `json:"batch_results,omitempty"`
	PaidDelegators int           `json:"paid_delegators,omitempty"`
	Summary        PayoutSummary `json:"cycle_payout_summary,omitempty"`
}
