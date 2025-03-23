package constants

import "errors"

var (
	// miscllaneous

	ErrNotImplemented   = errors.New("not implemented")
	ErrUserNotConfirmed = errors.New("user not confirmed")

	// load

	ErrConfigurationLoadFailed            = errors.New("failed to load configuration")
	ErrConfigurationValidationFailed      = errors.New("failed to validate configuration")
	ErrSignerLoadFailed                   = errors.New("failed to load signer engine")
	ErrTransactorLoadFailed               = errors.New("failed to load transactor engine")
	ErrCollectorLoadFailed                = errors.New("failed to load collector engine")
	ErrExtensionStoreInitializationFailed = errors.New("failed to initialize extension store")

	// consfiguration
	// configuration - import

	ErrInvalidConfigurationImportSource = errors.New("invalid configuration import source")
	ErrInvalidSourceVersionInfo         = errors.New("invalid version info")
	ErrUnsupportedBCVersion             = errors.New("unsupported bc version")
	ErrUnsupportedTRDVersion            = errors.New("unsupported trd version")

	// configuration - migration
	ErrConfigurationMigrationFailed = errors.New("failed to migrate configuration")

	// collector engines

	// baker did not have any rewards in cycle
	ErrNoCycleDataAvailable                = errors.New("no cycle data available")
	ErrCycleDataFetchFailed                = errors.New("failed to fetch cycle data")
	ErrCycleDataProtocolRewardsFetchFailed = errors.New("failed to fetch protocol-rewards cycle data")
	ErrCycleDataProtocolRewardsMismatch    = errors.New("protocol-rewards cycle data mismatch")
	ErrCycleDataUnmarshalFailed            = errors.New("failed to unmarshal cycle data")
	ErrOperationStatusCheckFailed          = errors.New("failed to check operation status")

	// cycle monitor

	ErrMonitoringCanceled = errors.New("monitoring canceled")

	// context validation

	ErrMissingEngine           = errors.New("missing engine")
	ErrMissingSignerEngine     = errors.New("undefined signer engine")
	ErrMissingCollectorEngine  = errors.New("undefined collector engine")
	ErrMissingReporterEngine   = errors.New("undefined reporter engine")
	ErrMissingTransactorEngine = errors.New("undefined transactor engine")
	ErrMissingConfiguration    = errors.New("undefined configuration")
	ErrMissingPayoutBlueprint  = errors.New("undefined payout blueprint")
	ErrMixedRpcs               = errors.New("defined rpcs from different networks")

	// generate payouts

	ErrRevealCheckFailed                     = errors.New("failed to check if address is revealed")
	ErrNotRevealed                           = errors.New("address is not revealed")
	ErrCycleDataCollectionFailed             = errors.New("failed to collect cycle data")
	ErrPayoutsFromFileLoadFailed             = errors.New("failed to load payouts from file")
	ErrPayoutsFromBytesLoadFailed            = errors.New("failed to load payouts from bytes")
	ErrPayoutsFromStdinLoadFailed            = errors.New("failed to load payouts from stdin")
	ErrPayoutsSaveToFileFailed               = errors.New("failed to save payouts to file")
	ErrInsufficientBalance                   = errors.New("insufficient balance")
	ErrFailedToEstimateSerializationGasLimit = errors.New("failed to estimate batch serialization gas limit")

	// execute payouts

	ErrFailedToCompleteOperation    = errors.New("failed to complete operation")
	ErrFailedToSignOperation        = errors.New("failed to sign operation")
	ErrExecutePayoutsUserTerminated = errors.New("user terminated execution")
	ErrGetChainLimitsFailed         = errors.New("failed to get chain limits")

	// notifications

	ErrUnsupportedNotificator          = errors.New("unsupported notificator")
	ErrPayoutDidNotFitTheBatch         = errors.New("payout did not fit the batch")
	ErrInvalidNotificatorConfiguration = errors.New("invalid notificator configuration")

	// operations

	ErrOperationContextCreationFailed  = errors.New("failed to create operation context")
	ErrOperationBroadcastFailed        = errors.New("failed to broadcast operation")
	ErrOperationConfirmationFailed     = errors.New("failed to confirm operation")
	ErrOperationNotDispatched          = errors.New("operation not dispatched")
	ErrOperationInvalidContractAddress = errors.New("invalid contract address")
	ErrOperationInvalidLimits          = errors.New("invalid limits")
	ErrOperationFailed                 = errors.New("operation failed")

	// extensions

	ErrExtensionLoadFailed          = errors.New("failed to load extension")
	ErrUnsupportedExtensionHook     = errors.New("unsupported extension hook")
	ErrUnsupportedExtensionHookMode = errors.New("unsupported extension hook mode")
	ErrUnsupportedExtensionKind     = errors.New("unsupported extension kind")
	ErrExtensionHookMissingData     = errors.New("no data forwarded to hook, cannot execute")
)
