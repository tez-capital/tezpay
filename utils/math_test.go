package utils

import (
	"testing"

	"blockwatch.cc/tzgo/tezos"
	"github.com/stretchr/testify/assert"
)

func TestGetZPortion(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(GetZPortion(tezos.NewZ(2000), 0.1005).Int64(), int64(201))
	assert.Equal(GetZPortion(tezos.NewZ(2000), 1.0).Int64(), int64(2000))
	assert.Equal(GetZPortion(tezos.NewZ(2000), 0.10).Int64(), int64(200))
	assert.Equal(GetZPortion(tezos.NewZ(2000), 0.01).Int64(), int64(20))
	assert.Equal(GetZPortion(tezos.NewZ(2000), float64(0)).Int64(), int64(0))
}

func TestMax(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(Max(2, 1), 2)
	assert.Equal(Max(2, 3), 3)
	assert.Equal(Max(2, 2), 2)
}
