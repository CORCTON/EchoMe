package infra

import (
	"github.com/google/wire"
	"github.com/justin/echome-be/internal/domain/ai"
	dc "github.com/justin/echome-be/internal/domain/character"
	"github.com/justin/echome-be/internal/infra/aliyun"
	"github.com/justin/echome-be/internal/infra/character"
	"github.com/justin/echome-be/internal/infra/db"
)

// RepositoryProviderSet 包含所有仓库提供者
var RepositoryProviderSet = wire.NewSet(
	db.NewDB,
	db.NewQuery,
	character.NewCharacterRepository,
	wire.Bind(new(dc.Repo), new(*character.CharacterRepository)),
	aliyun.ProvideAliClient,
	wire.Bind(new(ai.Repo), new(*aliyun.AliClient)),
)