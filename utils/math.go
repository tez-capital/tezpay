package utils

import (
	"math"

	"blockwatch.cc/tzgo/tezos"
)

type FloatConstraint interface {
	float32 | float64
}

func GetZPortion[T FloatConstraint](val tezos.Z, percent T) tezos.Z {
	if percent == 0 {
		return tezos.Zero
	}
	portionRelativeTo10000 := int64(math.Floor(float64(percent) * 100))
	return val.Mul64(portionRelativeTo10000).Div64(10000)
}

type NumberConstraint interface {
	int | int8 | int16 | int32 | int64 | float32 | float64
}

func Max[T NumberConstraint](v1 T, v2 T) T {
	if v1 > v2 {
		return v1
	}
	return v2
}
