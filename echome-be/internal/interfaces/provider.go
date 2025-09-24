package interfaces

import (
	"github.com/google/wire"
)

// Provider sets for Wire
var (
	// HandlerProviderSet contains all handler providers
	HandlerProviderSet = wire.NewSet(
		NewHandlers,
		NewCharacterHandlers,
		NewWebSocketHandlers,
	)
)
