package domain

import (
	"github.com/google/wire"
	"github.com/justin/echome-be/internal/domain/character"
	"github.com/justin/echome-be/internal/domain/conversation"
)

var ServiceProviderSet = wire.NewSet(
	character.NewCharacterService,
	conversation.NewConversationService,
)
