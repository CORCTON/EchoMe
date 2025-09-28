package handler

import (
	"github.com/google/wire"
)

// Provider sets for Wire
var HandlerProviderSet = wire.NewSet(
		NewHandlers,
		NewCharacterHandlers,
		NewWebSocketHandlers,
	)