package utils

import (
	"math"

	"blockwatch.cc/tzgo/tezos"
)

type FloatConstraint interface {
	float32 | float64
}

func GetZPortion[T FloatConstraint](val tezos.Z, portion T) tezos.Z {
	if portion == 0 {
		return tezos.Zero
	}
	if portion >= 1 {
		return val
	}
	portionRelativeTo1000000 := int64(math.Floor(float64(portion) * 10000))
	return val.Mul64(portionRelativeTo1000000).Div64(10000)
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
