package cmd

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trilitech/tzgo/tezos"
)

func makeProtocolHash(seed byte) tezos.ProtocolHash {
	var hash tezos.ProtocolHash
	hash[0] = seed
	return hash
}

func TestEvaluateProtocolChange_NoChange(t *testing.T) {
	assert := assert.New(t)
	protoA := makeProtocolHash(1)
	protoB := makeProtocolHash(2)

	decision := evaluateProtocolChange(protoA, protoA, fmt.Sprintf("%s->%s", protoA, protoB), false)

	assert.False(decision.HasProtocolChanged)
	assert.False(decision.ShouldNotify)
	assert.False(decision.ShouldSkipPayouts)
	assert.Equal(protoA, decision.UpdatedExpectedProtocol)
	assert.Equal(fmt.Sprintf("%s->%s", protoA, protoB), decision.UpdatedLastNotifiedProtocolPair)
	assert.Equal("", decision.NotificationMessage)
}

func TestEvaluateProtocolChange_SafeModeChange_NotifyAndSkip(t *testing.T) {
	assert := assert.New(t)
	protoA := makeProtocolHash(1)
	protoB := makeProtocolHash(2)

	decision := evaluateProtocolChange(protoA, protoB, "", false)

	assert.True(decision.HasProtocolChanged)
	assert.True(decision.ShouldNotify)
	assert.True(decision.ShouldSkipPayouts)
	assert.Equal(fmt.Sprintf("Protocol changed from %s to %s.", protoA, protoB), decision.NotificationMessage)
	assert.Equal(protoA, decision.UpdatedExpectedProtocol)
	assert.Equal(fmt.Sprintf("%s->%s", protoA, protoB), decision.UpdatedLastNotifiedProtocolPair)
}

func TestEvaluateProtocolChange_SafeModeChange_DedupStillSkips(t *testing.T) {
	assert := assert.New(t)
	protoA := makeProtocolHash(1)
	protoB := makeProtocolHash(2)

	decision := evaluateProtocolChange(protoA, protoB, fmt.Sprintf("%s->%s", protoA, protoB), false)

	assert.True(decision.HasProtocolChanged)
	assert.False(decision.ShouldNotify)
	assert.True(decision.ShouldSkipPayouts)
	assert.Equal(protoA, decision.UpdatedExpectedProtocol)
	assert.Equal(fmt.Sprintf("%s->%s", protoA, protoB), decision.UpdatedLastNotifiedProtocolPair)
	assert.Equal("", decision.NotificationMessage)
}

func TestEvaluateProtocolChange_IgnoreModeChange_NotifyAndContinue(t *testing.T) {
	assert := assert.New(t)
	protoA := makeProtocolHash(1)
	protoB := makeProtocolHash(2)

	decision := evaluateProtocolChange(protoA, protoB, "", true)

	assert.True(decision.HasProtocolChanged)
	assert.True(decision.ShouldNotify)
	assert.False(decision.ShouldSkipPayouts)
	assert.Equal(fmt.Sprintf("Protocol changed from %s to %s.", protoA, protoB), decision.NotificationMessage)
	assert.Equal(protoB, decision.UpdatedExpectedProtocol)
	assert.Equal(fmt.Sprintf("%s->%s", protoA, protoB), decision.UpdatedLastNotifiedProtocolPair)
}

func TestEvaluateProtocolChange_IgnoreModeSecondTransition_NewPairNotifies(t *testing.T) {
	assert := assert.New(t)
	protoA := makeProtocolHash(1)
	protoB := makeProtocolHash(2)
	protoC := makeProtocolHash(3)

	decision := evaluateProtocolChange(protoB, protoC, fmt.Sprintf("%s->%s", protoA, protoB), true)

	assert.True(decision.HasProtocolChanged)
	assert.True(decision.ShouldNotify)
	assert.False(decision.ShouldSkipPayouts)
	assert.Equal(fmt.Sprintf("Protocol changed from %s to %s.", protoB, protoC), decision.NotificationMessage)
	assert.Equal(protoC, decision.UpdatedExpectedProtocol)
	assert.Equal(fmt.Sprintf("%s->%s", protoB, protoC), decision.UpdatedLastNotifiedProtocolPair)
}
