package constants

import "errors"

var (
	ErrUnsupportedExtensionHook     = errors.New("unsupported extension hook")
	ErrUnsupportedExtensionHookMode = errors.New("unsupported extension hook mode")
)
