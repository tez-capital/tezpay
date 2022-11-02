package utils

import (
	"fmt"

	"github.com/alis-is/tezpay/constants"
)

func MutezToTezS(amount int64) string {
	if amount == 0 {
		return "-"
	}
	tez := float64(amount) / constants.MUTEZ_FACTOR
	return fmt.Sprintf("%f TEZ", tez)
}
