package service

import (
	"github.com/google/wire"
	"github.com/justin/echome-be/internal/domain"
	"github.com/justin/echome-be/internal/service/character"
	"github.com/justin/echome-be/internal/service/conversation"
	"github.com/justin/echome-be/internal/service/webrtc"
)

// Provider sets for Wire
var (
	// ServiceProviderSet contains all service providers
	ServiceProviderSet = wire.NewSet(
		character.NewCharacterService,
		webrtc.NewWebRTCService,
		conversation.NewConversationService,
		wire.Bind(new(domain.ConversationService), new(*conversation.ConversationService)),
	)
)
