package core

import (
	"github.com/alis-is/tezpay/common"
	"github.com/alis-is/tezpay/core/execute"
	"github.com/alis-is/tezpay/core/generate"
	"github.com/alis-is/tezpay/core/prepare"
)

type PayoutContext interface {
	*generate.PayoutGenerationContext | *execute.PayoutExecutionContext | *prepare.PayoutPrepareContext
}

type PayoutOptions interface {
	*common.GeneratePayoutsOptions | *common.ExecutePayoutsOptions | *common.PreparePayoutsOptions
}

type Stage[T PayoutContext, U PayoutOptions] func(ctx T, options U) (T, error)

type WrappedStageResult[T PayoutContext, U PayoutOptions] struct {
	Ctx T
	Err error
}

type WrappedStage[T PayoutContext, U PayoutOptions] func(previous WrappedStageResult[T, U], options U) WrappedStageResult[T, U]

func (result WrappedStageResult[T, U]) ExecuteWrappedStage(options U, stage WrappedStage[T, U]) WrappedStageResult[T, U] {
	return stage(result, options)
}

func (result WrappedStageResult[T, U]) ExecuteWrappedStages(options U, stages ...WrappedStage[T, U]) WrappedStageResult[T, U] {
	for _, stage := range stages {
		result = stage(result, options)
	}
	return result
}

func (result WrappedStageResult[T, U]) ExecuteStage(options U, stage Stage[T, U]) WrappedStageResult[T, U] {
	return WrapStage(stage)(result, options)
}

func (result WrappedStageResult[T, U]) ExecuteStages(options U, stages ...Stage[T, U]) WrappedStageResult[T, U] {
	for _, stage := range stages {
		result = WrapStage(stage)(result, options)
	}
	return result
}

func WrapStage[T PayoutContext, U PayoutOptions](stage Stage[T, U]) WrappedStage[T, U] {
	return func(previous WrappedStageResult[T, U], options U) WrappedStageResult[T, U] {
		if previous.Err != nil {
			return previous
		}
		ctx, err := stage(previous.Ctx, options)
		return WrappedStageResult[T, U]{
			Ctx: ctx,
			Err: err,
		}
	}
}

func WrapContext[T PayoutContext, U PayoutOptions](ctx T) WrappedStageResult[T, U] {
	return WrappedStageResult[T, U]{
		Ctx: ctx,
	}
}

func (result WrappedStageResult[T, U]) Unwrap() (T, error) {
	return result.Ctx, result.Err
}
