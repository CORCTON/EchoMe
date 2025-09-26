//go:build ignore

package main

import (
	"flag"
	"fmt"

	"github.com/google/uuid"
	"github.com/justin/echome-be/config"
	"github.com/justin/echome-be/internal/domain"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const postgresTcpDSN = "host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai"

func main() {
	// 初始化zap日志
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	zap.ReplaceGlobals(logger)

	configPath := flag.String("f", "config/etc/config.yaml", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		zap.L().Fatal("Failed to load config", zap.Error(err))
	}

	// 构建数据库连接字符串
	dsn := fmt.Sprintf(postgresTcpDSN,
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.Port,
	)

	// 连接数据库
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		zap.L().Fatal("Failed to connect to database", zap.Error(err))
	}
	if err != nil {
		zap.L().Fatal("Failed to connect to database", zap.Error(err))
	}

	// 执行迁移
	zap.L().Info("Running database migrations...")

	// 创建角色表
	err = db.AutoMigrate(&domain.Character{})
	if err != nil {
		zap.L().Fatal("Failed to migrate characters table", zap.Error(err))
	}

	// 创建索引
	err = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_characters_name ON characters (name)").Error
	if err != nil {
		zap.L().Fatal("Failed to create index on characters.name", zap.Error(err))
	}

	// 检查是否需要插入默认数据
	var count int64
	db.Model(&domain.Character{}).Count(&count)
	if count == 0 {
		zap.L().Info("No existing characters found, inserting default characters...")
		insertDefaultCharacters(db)
	}

	zap.L().Info("Database migration completed successfully")
}

// insertDefaultCharacters 插入默认角色数据
func insertDefaultCharacters(db *gorm.DB) {
	defaultCharacters := []*domain.Character{
		{
			ID:          uuid.New(),
			Name:        "小助手",
			Prompt:      "你是一个友善、耐心的AI助手，总是乐于帮助用户解决问题。你说话温和，回答详细且有用。",
			Avatar:      nil,
			Voice:       lo.ToPtr("xiaoyun"), // 阿里云小云语音
		},
		{
			ID:          uuid.New(),
			Name:        "专业顾问",
			Prompt:   "你是一个专业的技术顾问，具有丰富的技术知识和经验。你的回答准确、专业，善于用简单的语言解释复杂的技术概念。",
			Avatar:   nil,
			Voice: lo.ToPtr("zhiwei"), // 阿里云志伟语音（男声）
		},
	}

	for _, character := range defaultCharacters {
		if err := db.Create(character).Error; err != nil {
			zap.L().Warn("Failed to insert default character", zap.String("name", character.Name), zap.Error(err))
		}
	}
}
