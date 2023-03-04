package common

import (
	"testing"

	"github.com/alis-is/tezpay/constants/enums"
	"github.com/stretchr/testify/assert"
)

func TestUnmarshalExtensionHook(t *testing.T) {
	assert := assert.New(t)

	var hook ExtensionHook
	err := hook.UnmarshalJSON([]byte(`"after_candidate_generated:ro"`))
	assert.NoError(err)
	assert.Equal(enums.EXTENSION_HOOK_AFTER_CANDIDATE_GENERATED, hook.Id)
	assert.Equal(enums.EXTENSION_HOOK_MODE_READ_ONLY, hook.Mode)

	err = hook.UnmarshalJSON([]byte(`"after_candidate_generated:rw"`))
	assert.NoError(err)
	assert.Equal(enums.EXTENSION_HOOK_AFTER_CANDIDATE_GENERATED, hook.Id)
	assert.Equal(enums.EXTENSION_HOOK_MODE_READ_WRITE, hook.Mode)

	err = hook.UnmarshalJSON([]byte(`"after_candidate_generated"`))
	assert.NoError(err)
	assert.Equal(enums.EXTENSION_HOOK_AFTER_CANDIDATE_GENERATED, hook.Id)
	assert.Equal(enums.EXTENSION_HOOK_MODE_UNKNOWN, hook.Mode)

	err = hook.UnmarshalJSON([]byte(`{"id": "after_candidate_generated", "mode": "ro"}`))
	assert.NoError(err)
	assert.Equal(enums.EXTENSION_HOOK_AFTER_CANDIDATE_GENERATED, hook.Id)
	assert.Equal(enums.EXTENSION_HOOK_MODE_READ_ONLY, hook.Mode)

	err = hook.UnmarshalJSON([]byte(`{"id": "after_candidate_generated", "mode": "rw"}`))
	assert.NoError(err)
	assert.Equal(enums.EXTENSION_HOOK_AFTER_CANDIDATE_GENERATED, hook.Id)
	assert.Equal(enums.EXTENSION_HOOK_MODE_READ_WRITE, hook.Mode)

	err = hook.UnmarshalJSON([]byte(`{"id": "after_candidate_generated", "mode": "w"}`))
	assert.Nil(err)
	assert.Equal(hook.Id, enums.EXTENSION_HOOK_AFTER_CANDIDATE_GENERATED)
	assert.Equal(hook.Mode, enums.EExtensionHookMode("w"))

	err = hook.UnmarshalJSON([]byte(`{"id": "after_candidate_generated22", "mode": "ro"}`))
	assert.Nil(err)
	assert.Equal(hook.Id, enums.EExtensionHook("after_candidate_generated22"))
	assert.Equal(hook.Mode, enums.EXTENSION_HOOK_MODE_READ_ONLY)

	// fail cases
	err = hook.UnmarshalJSON([]byte(`{"id": "after_candidate_generated", mode": "ro"`))
	assert.Error(err)
}

func TestMarshalExtensionHook(t *testing.T) {
	assert := assert.New(t)

	hook := ExtensionHook{
		Id:   enums.EXTENSION_HOOK_AFTER_CANDIDATE_GENERATED,
		Mode: enums.EXTENSION_HOOK_MODE_READ_ONLY,
	}
	data, err := hook.MarshalJSON()
	assert.NoError(err)
	assert.Equal(`"after_candidate_generated:ro"`, string(data))

	hook = ExtensionHook{
		Id:   enums.EXTENSION_HOOK_AFTER_CANDIDATE_GENERATED,
		Mode: enums.EXTENSION_HOOK_MODE_READ_WRITE,
	}
	data, err = hook.MarshalJSON()
	assert.NoError(err)
	assert.Equal(`"after_candidate_generated:rw"`, string(data))
}
