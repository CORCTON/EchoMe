package infra

import (
	"github.com/google/wire"
	"github.com/justin/echome-be/internal/domain"
)

var RepositoryProviderSet = wire.NewSet(
	NewCharacterRepository,
	wire.Bind(new(domain.CharacterRepository), new(*CharacterRepository)),
	NewDB,
)