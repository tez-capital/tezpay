package common

import (
	"blockwatch.cc/tzgo/codec"
	"blockwatch.cc/tzgo/rpc"
	"blockwatch.cc/tzgo/tezos"
)

type OperationStatus string

const (
	OPERATION_STATUS_FAILED     OperationStatus = "failed"
	OPERATION_STATUS_APPLIED    OperationStatus = "applied"
	OPERATION_STATUS_NOT_EXISTS OperationStatus = "not exists"
	OPERATION_STATUS_UNKNOWN    OperationStatus = "unknown"
)

type CycleMonitorOptions struct {
	NotificationDelay int64 `json:"notificationDelay"`
	CheckFrequency    int64 `json:"checkFrequency"`
}

type CycleMonitor interface {
	//GetCycleChannel() chan int64
	Cancel()
	//Terminate()
	//CreateBlockHeaderMonitor() error
	WaitForNextCompletedCycle(lastProcessedCycle int64) (int64, error)
}

type CollectorEngine interface {
	GetId() string
	RefreshParams() error
	GetCurrentCycleNumber() (int64, error)
	GetLastCompletedCycle() (int64, error)
	GetCycleData(baker tezos.Address, cycle int64) (*BakersCycleData, error)
	WasOperationApplied(opHash tezos.OpHash) (OperationStatus, error)
	GetBranch(offset int64) (tezos.BlockHash, error)
	Simulate(o *codec.Op, publicKey tezos.Key) (*rpc.Receipt, error)
	GetBalance(pkh tezos.Address) (tezos.Z, error)
	CreateCycleMonitor(options CycleMonitorOptions) (CycleMonitor, error)
	SendAnalytics(bakerId string, version string)
	GetCurrentProtocol() (tezos.ProtocolHash, error)
}

type SignerEngine interface {
	GetId() string
	Sign(op *codec.Op) error
	GetPKH() tezos.Address
	GetKey() tezos.Key
	//GetSigner() signer.Signer
}

type OpResult interface {
	GetOpHash() tezos.OpHash
	WaitForApply() error
}

type DispatchOptions struct {
	TTL           int64 `json:"ttl"`
	Confirmations int64 `json:"confirmations"`
}

type TransactorEngine interface {
	GetId() string
	RefreshParams() error
	Complete(op *codec.Op, key tezos.Key) error
	Dispatch(op *codec.Op, opts *DispatchOptions) (OpResult, error)
	//Broadcast(op *codec.Op) (tezos.OpHash, error)
	//Send(op *codec.Op, opts *rpc.CallOptions) (*rpc.Receipt, error)
	GetLimits() (*OperationLimits, error)
	//WaitOpConfirmation(opHash tezos.OpHash, ttl int64, confirmations int64) (*rpc.Receipt, error)
}

type NotificatorEngine interface {
	PayoutSummaryNotify(summary *CyclePayoutSummary, additionalData map[string]string) error
	AdminNotify(msg string) error
	TestNotify() error
}

type ReporterEngine interface {
	GetExistingReports(cycle int64) ([]PayoutReport, error)
	ReportPayouts(reports []PayoutReport) error
	ReportInvalidPayouts(reports []PayoutRecipe) error
	ReportCycleSummary(summary CyclePayoutSummary) error
	GetExistingCycleSummary(cycle int64) (*CyclePayoutSummary, error)
}
