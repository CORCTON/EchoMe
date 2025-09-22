package infrastructure

import (
	"github.com/google/wire"
	"github.com/justin/echome-be/internal/domain"
)

// Provider sets for Wire
var (
	// RepositoryProviderSet contains all repository providers
	RepositoryProviderSet = wire.NewSet(
		NewMemoryCharacterRepository,
		wire.Bind(new(domain.CharacterRepository), new(*MemoryCharacterRepository)),
		NewMemorySessionRepository,
		wire.Bind(new(domain.SessionRepository), new(*MemorySessionRepository)),
		NewMemoryMessageRepository,
		wire.Bind(new(domain.MessageRepository), new(*MemoryMessageRepository)),
	)
)