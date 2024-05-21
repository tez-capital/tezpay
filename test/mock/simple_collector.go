package mock

import (
	"errors"
	"time"

	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/rpc"
	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/constants"
	"github.com/alis-is/tezpay/utils"
	"github.com/samber/lo"
)

type SimpleColletor struct {
	opts *SimpleCollectorOpts
}

type SimpleCollectorOpts struct {
	StorageBurn           int64
	AllocationBurn        int64
	UsedMilliGas          int64
	SingleOnly            bool
	FailWithError         error
	FailWithReceiptError  error
	ReturnOnlyNCosts      int
	SerializationGasLimit int64
}

func InitSimpleColletor() *SimpleColletor {
	return &SimpleColletor{
		opts: &SimpleCollectorOpts{
			AllocationBurn:        1000,
			StorageBurn:           1000,
			UsedMilliGas:          1000000,
			SerializationGasLimit: 100,
		},
	}
}

func (engine *SimpleColletor) GetId() string {
	return "DefaultRpcAndTzktColletor"
}

func (engine *SimpleColletor) RefreshParams() error {
	return nil
}

func (engine *SimpleColletor) SetOpts(opts *SimpleCollectorOpts) {
	engine.opts = opts
}

func (engine *SimpleColletor) IsRevealed(address tezos.Address) (bool, error) {
	return true, nil
}

func (engine *SimpleColletor) GetOpts() *SimpleCollectorOpts {
	return engine.opts
}

func (engine *SimpleColletor) GetCurrentCycleNumber() (int64, error) {
	return 501, nil
}

func (engine *SimpleColletor) GetLastCompletedCycle() (int64, error) {
	cycle, err := engine.GetCurrentCycleNumber()
	return cycle - 1, err
}

func (engine *SimpleColletor) GetCycleStakingData(baker tezos.Address, cycle int64) (*common.BakersCycleData, error) {
	return &common.BakersCycleData{
		OwnStakingBalance:           tezos.NewZ(50_000).Mul64(constants.MUTEZ_FACTOR),
		OwnDelegatedBalance:         tezos.NewZ(50_000).Mul64(constants.MUTEZ_FACTOR),
		ExternalDelegatedBalance:    tezos.NewZ(1_000_000).Mul64(constants.MUTEZ_FACTOR),
		BlockDelegatedRewards:       tezos.NewZ(100).Mul64(constants.MUTEZ_FACTOR),
		EndorsementDelegatedRewards: tezos.NewZ(50).Mul64(constants.MUTEZ_FACTOR),
		FrozenDepositLimit:          tezos.NewZ(50_000).Mul64(constants.MUTEZ_FACTOR),
		DelegatorsCount:             2,
		BlockDelegatedFees:          tezos.NewZ(25).Mul64(constants.MUTEZ_FACTOR),
		// TODO:
		Delegators: []common.Delegator{},
	}, nil
}

func (engine *SimpleColletor) GetCyclesInDateRange(startDate time.Time, endDate time.Time) ([]int64, error) {
	return []int64{500, 501}, nil
}

func (engine *SimpleColletor) WasOperationApplied(op tezos.OpHash) (common.OperationStatus, error) {
	return common.OPERATION_STATUS_APPLIED, nil
}

func (engine *SimpleColletor) CreateCycleMonitor(options common.CycleMonitorOptions) (common.CycleMonitor, error) {
	return nil, constants.ErrNotImplemented
}

func (engine *SimpleColletor) GetBranch(offset int64) (hash tezos.BlockHash, err error) {
	return tezos.ZeroBlockHash, nil
}

func (engine *SimpleColletor) GetExpectedTxCosts() int64 {
	op := codec.NewOp().WithSource(GetRandomAddress())
	op.WithTTL(constants.MAX_OPERATION_TTL)
	op.WithTransfer(GetRandomAddress(), 100000)
	gasUsed := engine.opts.UsedMilliGas / 1000
	op.Contents[len(op.Contents)-1].WithLimits(tezos.Limits{
		GasLimit:     gasUsed,
		StorageLimit: engine.opts.StorageBurn + engine.opts.AllocationBurn,
	})

	txFee := utils.EstimateTransactionFee(op, []int64{gasUsed + engine.opts.SerializationGasLimit + constants.DEFAULT_TX_DESERIALIZATION_GAS_BUFFER + constants.DEFAULT_TX_GAS_LIMIT_BUFFER}, constants.DEFAULT_TX_FEE_BUFFER)
	return txFee + engine.opts.AllocationBurn + engine.opts.StorageBurn
}

func (engine *SimpleColletor) Simulate(o *codec.Op, publicKey tezos.Key) (*rpc.Receipt, error) {
	if engine.opts.SingleOnly && len(o.Contents) > 3 {
		return nil, errors.New("failed to batch estimate")
	}
	if engine.opts.FailWithError != nil {
		return nil, engine.opts.FailWithError
	}
	returnCostsCount := len(o.Contents)
	if engine.opts.ReturnOnlyNCosts > 0 {
		returnCostsCount = engine.opts.ReturnOnlyNCosts
	}

	opList := append(rpc.OperationList{},
		lo.Slice(lo.Map(o.Contents, func(content codec.Operation, _ int) rpc.TypedOperation {
			return rpc.Transaction{
				Manager: rpc.Manager{
					Fee: 500,
					Generic: rpc.Generic{
						Metadata: rpc.OperationMetadata{
							Result: rpc.OperationResult{
								ConsumedGas:      0,
								ConsumedMilliGas: engine.opts.UsedMilliGas,
								Allocated:        true,
								BalanceUpdates: rpc.BalanceUpdates{
									rpc.BalanceUpdate{
										Kind:   "contract",
										Change: -engine.opts.AllocationBurn,
									},
									rpc.BalanceUpdate{
										Kind:   "contract",
										Change: -engine.opts.StorageBurn,
									},
								},
								PaidStorageSizeDiff: 0,
								Status:              tezos.OpStatusApplied,
							},
						},
					},
				},
			}
		}), 0, returnCostsCount)...)

	// TODO: likely move to test package as util function
	if engine.opts.FailWithReceiptError != nil {
		return &rpc.Receipt{
			Op: &rpc.Operation{
				Contents: []rpc.TypedOperation{
					rpc.Transaction{
						Manager: rpc.Manager{
							Generic: rpc.Generic{
								Metadata: rpc.OperationMetadata{
									Result: rpc.OperationResult{
										Status: tezos.OpStatusFailed,
										Errors: []rpc.OperationError{
											{
												GenericError: rpc.GenericError{
													Kind: engine.opts.FailWithReceiptError.Error(),
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}, nil
	}

	return &rpc.Receipt{
		Block: tezos.ZeroBlockHash,
		List:  0,
		Pos:   0,
		Op: &rpc.Operation{
			Contents: opList,
		},
	}, nil
}

func (engine *SimpleColletor) GetBalance(addr tezos.Address) (tezos.Z, error) {
	return tezos.NewZ(100).Mul64(constants.MUTEZ_FACTOR), nil
}

func (engine *SimpleColletor) SendAnalytics(bakerId string, version string) {}

func (engine *SimpleColletor) GetCurrentProtocol() (tezos.ProtocolHash, error) {
	return tezos.ZeroProtocolHash, nil
}
