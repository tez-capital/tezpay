package utils

import (
	"errors"

	"blockwatch.cc/tzgo/rpc"
	log "github.com/sirupsen/logrus"
)

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

func TryUnwrapRPCError(err error) error {
	if rpcError, ok := err.(rpc.RPCError); ok {
		body := rpcError.Body()
		if len(body) == 0 {
			return err
		}

		return errors.New(string(body))
	}
	return err
}
