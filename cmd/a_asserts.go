package cmd

import (
	"os"

	log "github.com/sirupsen/logrus"
)

func assertRunWithErrFmt(toExecute func() error, exitCode int, errorFormat string) {
	err := toExecute()
	if err != nil {
		log.Errorf(errorFormat, err.Error())
		os.Exit(exitCode)
	}
}

func assertRun(toExecute func() error, exitCode int) {
	assertRunWithErrFmt(toExecute, exitCode, "%s")
}

func assertRunWithParamWithErrFmt[T interface{}](toExecute func(T) error, param T, exitCode int, errorFormat string) {
	err := toExecute(param)
	if err != nil {
		log.Errorf(errorFormat, err.Error())
		os.Exit(exitCode)
	}
}

func assertRunWithParam[T interface{}](toExecute func(T) error, param T, exitCode int) {
	assertRunWithParamWithErrFmt(toExecute, param, exitCode, "%s")
}

func assertRunWithResultAndErrFmt[T interface{}](toExecute func() (T, error), exitCode int, errorFormat string) T {
	result, err := toExecute()
	if err != nil {
		log.Errorf(errorFormat, err.Error())
		os.Exit(exitCode)
	}
	return result
}

func assertRunWithResult[T interface{}](toExecute func() (T, error), exitCode int) T {
	return assertRunWithResultAndErrFmt(toExecute, exitCode, "%s")
}

func warnIfFailedWithErrFmt(toExecute func() error, errorFormat string) bool {
	err := toExecute()
	if err != nil {
		log.Warnf(errorFormat, err.Error())
		return false
	}
	return true
}

func assertRequireConfirmation(msg string) {
	assertRunWithParam(requireConfirmation, msg, EXIT_OPERTION_CANCELED)
}
