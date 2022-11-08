package notifications

import (
	"fmt"
	"reflect"
	"strings"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/utils"
)

func PopulateMessageTemplate(messageTempalte string, summary *common.CyclePayoutSummary) string {
	v := reflect.ValueOf(*summary)
	typeOfS := v.Type()

	for i := 0; i < v.NumField(); i++ {
		val := fmt.Sprintf("%v", v.Field(i).Interface())
		if typeOfS.Field(i).Type.Name() == "tezos.Z" {
			val = fmt.Sprintf("%v", utils.MutezToTezS(v.Field(i).Interface().(tezos.Z).Int64()))
		}
		messageTempalte = strings.ReplaceAll(messageTempalte, fmt.Sprintf("<%s>", typeOfS.Field(i).Name), val)
	}

	return messageTempalte
}
