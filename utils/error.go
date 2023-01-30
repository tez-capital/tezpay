package utils

import log "github.com/sirupsen/logrus"

func WarnIfFailed(err error, errfmt string) error {
	if err != nil {
		log.Warnf(errfmt, err)
	}
	return err
}

// returns true if all errors are nil
func HasNoError(errs []error) bool {
	for _, err := range errs {
		if err != nil {
			return false
		}
	}
	return true
}
