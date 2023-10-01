package cmd

import (
	"github.com/alis-is/tezpay/common"
	log "github.com/sirupsen/logrus"
)

func assertRunWithErrFmt(toExecute func() error, exitCode int, errorFormat string) {
	err := toExecute()
	if err != nil {
		log.Errorf(errorFormat, err.Error())
		panic(common.PanicStatus{
			ExitCode: exitCode,
			Error:    err,
		})
	}
}

func assertRun(toExecute func() error, exitCode int) {
	assertRunWithErrFmt(toExecute, exitCode, "%s")
}

func assertRunWithParamWithErrFmt[T interface{}](toExecute func(T) error, param T, exitCode int, errorFormat string) {
	err := toExecute(param)
	if err != nil {
		log.Errorf(errorFormat, err.Error())
		panic(common.PanicStatus{
			ExitCode: exitCode,
			Error:    err,
		})
	}
}

func assertRunWithParam[T interface{}](toExecute func(T) error, param T, exitCode int) {
	assertRunWithParamWithErrFmt(toExecute, param, exitCode, "%s")
}

func assertRunWithResultAndErrFmt[T interface{}](toExecute func() (T, error), exitCode int, errorFormat string) T {
	result, err := toExecute()
	if err != nil {
		log.Errorf(errorFormat, err.Error())
		panic(common.PanicStatus{
			ExitCode: exitCode,
			Error:    err,
		})
	}
	return result
}

func assertRunWithResult[T interface{}](toExecute func() (T, error), exitCode int) T {
	return assertRunWithResultAndErrFmt(toExecute, exitCode, "%s")
}
