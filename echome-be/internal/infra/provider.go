package infra

import (
	"github.com/google/wire"
	"github.com/justin/echome-be/config"
	"github.com/justin/echome-be/internal/domain"
	"gorm.io/gorm"
)

// 从pgDB获取gorm.DB实例
func ProvideGormDB(dbAdapter domain.DBAdapter) *gorm.DB {
	return dbAdapter.Get()
}

var RepositoryProviderSet = wire.NewSet(
	NewCharacterRepository,
	wire.Bind(new(domain.CharacterRepository), new(*CharacterRepository)),
	NewDB,
	wire.Bind(new(domain.DBAdapter), new(*pgDB)),
	ProvideGormDB,
	wire.FieldsOf(new(*config.Config), "Database"),
)
