package common

import (
	"fmt"
	"time"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/constants/enums"
)

type OpLimits struct {
	TransactionFee int64 `json:"transaction_fee,omitempty"`
	StorageLimit   int64 `json:"storage_limit,omitempty"`
	GasLimit       int64 `json:"gas_limit,omitempty"`
}

type PayoutRecipe struct {
	Baker            tezos.Address     `json:"baker"`
	Delegator        tezos.Address     `json:"delegator,omitempty"`
	Cycle            int64             `json:"cycle,omitempty"`
	Recipient        tezos.Address     `json:"recipient,omitempty"`
	Kind             enums.EPayoutKind `json:"kind,omitempty"`
	DelegatedBalance tezos.Z           `json:"delegator_balance,omitempty"`
	Amount           tezos.Z           `json:"amount,omitempty"`
	FeeRate          float64           `json:"fee_rate,omitempty"`
	Fee              tezos.Z           `json:"fee,omitempty"`
	OpLimits         *OpLimits         `json:"op_limits,omitempty"`
	Note             string            `json:"note,omitempty"`
	IsValid          bool              `json:"valid,omitempty"`
}

func (pr *PayoutRecipe) PayoutRecipeToPayoutReport() PayoutReport {
	txFee := int64(0)
	if pr.OpLimits != nil {
		txFee = pr.OpLimits.TransactionFee
	}

	return PayoutReport{
		Baker:            pr.Baker,
		Timestamp:        time.Now(),
		Cycle:            pr.Cycle,
		Kind:             pr.Kind,
		Delegator:        pr.Delegator,
		DelegatedBalance: pr.DelegatedBalance,
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

type CyclePayoutSummary struct {
	Cycle              int64     `json:"cycle"`
	Delegators         int       `json:"delegators"`
	PaidDelegators     int       `json:"paid_delegators"`
	StakingBalance     tezos.Z   `json:"staking_balance"`
	EarnedFees         tezos.Z   `json:"cycle_fees"`
	EarnedRewards      tezos.Z   `json:"cycle_rewards"`
	DistributedRewards tezos.Z   `json:"distributed_rewards"`
	BondIncome         tezos.Z   `json:"bond_income"`
	FeeIncome          tezos.Z   `json:"fee_income"`
	IncomeTotal        tezos.Z   `json:"total_income"`
	DonatedBonds       tezos.Z   `json:"donated_bonds"`
	DonatedFees        tezos.Z   `json:"donated_fees"`
	DonatedTotal       tezos.Z   `json:"donated_total"`
	Timestamp          time.Time `json:"timestamp"`
}

func (summary *CyclePayoutSummary) Add(another *CyclePayoutSummary) *CyclePayoutSummary {
	return &CyclePayoutSummary{
		StakingBalance:     summary.StakingBalance.Add(another.StakingBalance),
		EarnedFees:         summary.EarnedFees.Add(another.EarnedFees),
		EarnedRewards:      summary.EarnedRewards.Add(another.EarnedRewards),
		DistributedRewards: summary.DistributedRewards.Add(another.DistributedRewards),
		BondIncome:         summary.BondIncome.Add(another.BondIncome),
		FeeIncome:          summary.FeeIncome.Add(another.FeeIncome),
		IncomeTotal:        summary.IncomeTotal.Add(another.IncomeTotal),
		DonatedBonds:       summary.DonatedBonds.Add(another.DonatedBonds),
		DonatedFees:        summary.DonatedFees.Add(another.DonatedFees),
		DonatedTotal:       summary.DonatedTotal.Add(another.DonatedTotal),
	}
}

type CyclePayoutBlueprint struct {
	Cycle   int64              `json:"cycle,omitempty"`
	Payouts []PayoutRecipe     `json:"payouts,omitempty"`
	Summary CyclePayoutSummary `json:"summary,omitempty"`
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
		return fmt.Errorf("signer engine is not set")
	}
	if engines.collector == nil {
		return fmt.Errorf("collector engine is not set")
	}
	return nil
}

type GeneratePayoutsOptions struct {
	Cycle                    int64 `json:"cycle,omitempty"`
	SkipBalanceCheck         bool  `json:"skip_balance_check,omitempty"`
	WaitForSufficientBalance bool  `json:"wait_for_sufficient_balance,omitempty"`
}

type GeneratePayoutsResult = CyclePayoutBlueprint

type PreparePayoutsEngineContext struct {
	collector   CollectorEngine
	reporter    ReporterEngine
	adminNotify func(msg string)
}

func NewPreparePayoutsEngineContext(collector CollectorEngine, reporter ReporterEngine, adminNotify func(msg string)) *PreparePayoutsEngineContext {
	return &PreparePayoutsEngineContext{
		collector:   collector,
		adminNotify: adminNotify,
		reporter:    reporter,
	}
}

func (engines *PreparePayoutsEngineContext) GetCollector() CollectorEngine {
	return engines.collector
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
		return fmt.Errorf("collector engine is not set")
	}
	if engines.reporter == nil {
		return fmt.Errorf("reporter engine is not set")
	}
	return nil
}

type PreparePayoutsOptions struct {
}

type PreparePayoutsResult struct {
	Blueprint                     *CyclePayoutBlueprint `json:"blueprint,omitempty"`
	Payouts                       []PayoutRecipe        `json:"payouts,omitempty"`
	ReportsOfPastSuccesfulPayouts []PayoutReport        `json:"reports_of_past_succesful_payouts,omitempty"`
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
		return fmt.Errorf("signer engine is not set")
	}
	if engines.transactor == nil {
		return fmt.Errorf("transactor engine is not set")
	}
	if engines.reporter == nil {
		return fmt.Errorf("reporter engine is not set")
	}
	return nil
}

type ExecutePayoutsOptions struct {
	MixInContractCalls bool `json:"mix_in_contract_calls,omitempty"`
}

type ExecutePayoutsResult = BatchResults
