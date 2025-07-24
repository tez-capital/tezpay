package utils

import (
	"math"

	"github.com/trilitech/tzgo/tezos"
)

type FloatConstraint interface {
	float32 | float64
}

func getZPortion[T FloatConstraint](val tezos.Z, portion T) tezos.Z {
	// percentage with 4 decimals
	portionFromVal := int64(math.Floor(float64(portion) * 10000))
	return val.Mul64(portionFromVal).Div64(10000)
}

func GetZPortion[T FloatConstraint](val tezos.Z, portion T) tezos.Z {
	if portion <= 0 {
		return tezos.Zero
	}
	if portion >= 1 {
		return val
	}
	result := getZPortion(val, portion)
	if val.IsLessEqual(result) { // make sure we don't return more than the original value
		return val
	}
	return result
}

func IsPortionWithin0n1[T FloatConstraint](portion T) bool {
	total := tezos.NewZ(1000000)
	zPortion := getZPortion(total, portion)
	totalSubZPortion := total.Sub(zPortion)
	return !zPortion.IsNeg() && !totalSubZPortion.IsNeg()
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

func Abs[T NumberConstraint](v T) T {
	if v < 0 {
		return -v
	}
	return v
}
