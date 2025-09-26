package service

import (
	"github.com/google/wire"
	"github.com/justin/echome-be/internal/domain"
	"github.com/justin/echome-be/internal/service/character"
	"github.com/justin/echome-be/internal/service/conversation"
	"github.com/justin/echome-be/internal/service/webrtc"
)

var ServiceProviderSet = wire.NewSet(
	webrtc.NewWebRTCService,
	conversation.NewConversationService,
	wire.Bind(new(domain.ConversationService), new(*conversation.ConversationService)),
	character.NewCharacterService,
	wire.Bind(new(domain.CharacterService), new(*character.CharacterService)),
)
