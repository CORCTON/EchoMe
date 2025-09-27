package service

import (
	"github.com/google/wire"
	"github.com/justin/echome-be/config"
	"github.com/justin/echome-be/internal/domain"
	"github.com/justin/echome-be/internal/service/character"
	"github.com/justin/echome-be/internal/service/conversation"
	"github.com/justin/echome-be/internal/service/webrtc"
)

var ServiceProviderSet = wire.NewSet(
	webrtc.NewWebRTCService,
	ProvideConversationService,
	wire.Bind(new(domain.ConversationService), new(*conversation.ConversationService)),
	character.NewCharacterService,
	wire.Bind(new(domain.CharacterService), new(*character.CharacterService)),
)

func ProvideConversationService(
	aiService domain.AIService,
	characterService domain.CharacterService,
	tavilyConfig *config.TavilyConfig,
) *conversation.ConversationService {
	return conversation.NewConversationService(aiService, characterService, tavilyConfig.APIKey)
}
