package infra

import (
	"github.com/google/wire"
	"github.com/justin/echome-be/gen/gen/query"
	"github.com/justin/echome-be/internal/domain"
	"gorm.io/gorm"
)

var RepositoryProviderSet = wire.NewSet(
	NewDB,
	ProvideQuery,
	NewCharacterRepository,
	wire.Bind(new(domain.CharacterRepository), new(*CharacterRepository)),
)

// ProvideQuery 提供query.Query实例给CharacterRepository
func ProvideQuery(db *DB[*gorm.DB]) *query.Query {
	return GetQuery()
}
