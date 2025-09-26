//go:build ignore

package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/justin/echome-be/config"
	"github.com/justin/echome-be/internal/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const postgresTcpDSN = "host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai"

func main() {
	configPath := flag.String("f", "config/etc/config.yaml", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg := config.Load(*configPath)

	// 构建数据库连接字符串
	dsn := fmt.Sprintf(postgresTcpDSN,
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.Port,
	)

	// 连接数据库
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 执行迁移
	log.Println("Running database migrations...")

	// 创建角色表
	err = db.AutoMigrate(&domain.Character{})
	if err != nil {
		log.Fatalf("Failed to migrate characters table: %v", err)
	}
	log.Println("✓ Characters table migrated successfully")

	// 创建索引
	log.Println("Creating indexes...")
	err = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_characters_name ON characters (name)").Error
	if err != nil {
		log.Fatalf("Failed to create index on characters.name: %v", err)
	}
	log.Println("✓ Indexes created successfully")

	// 检查是否需要插入默认数据
	var count int64
	db.Model(&domain.Character{}).Count(&count)
	if count == 0 {
		log.Println("No existing characters found, inserting default characters...")
		insertDefaultCharacters(db)
	}

	log.Println("✅ Database migration completed successfully")
}

// insertDefaultCharacters 插入默认角色数据
func insertDefaultCharacters(db *gorm.DB) {
	defaultCharacters := []*domain.Character{
		{
			ID:          uuid.New(),
			Name:        "小助手",
			Description: "友善的AI助手",
			Persona:     "你是一个友善、耐心的AI助手，总是乐于帮助用户解决问题。你说话温和，回答详细且有用。",
			AvatarURL:   "",
			VoiceConfig: &domain.VoiceProfile{
				Voice:      "xiaoyun",      // 阿里云小云语音
				Model:      "cosyvoice-v3", // 阿里云TTS模型
				SpeechRate: 1.0,            // 正常语速
				Pitch:      0,              // 正常音调
				Volume:     0.8,            // 80%音量
				Language:   "zh-CN",        // 中文
			},
		},
		{
			ID:          uuid.New(),
			Name:        "专业顾问",
			Description: "专业的技术顾问",
			Persona:     "你是一个专业的技术顾问，具有丰富的技术知识和经验。你的回答准确、专业，善于用简单的语言解释复杂的技术概念。",
			AvatarURL:   "",
			VoiceConfig: &domain.VoiceProfile{
				Voice:      "zhiwei",       // 阿里云志伟语音（男声）
				Model:      "cosyvoice-v3", // 阿里云TTS模型
				SpeechRate: 0.9,            // 稍慢语速
				Pitch:      -50,            // 稍低音调
				Volume:     0.9,            // 90%音量
				Language:   "zh-CN",        // 中文
			},
		},
	}

	for _, character := range defaultCharacters {
		if err := db.Create(character).Error; err != nil {
			log.Printf("Warning: Failed to insert default character %s: %v", character.Name, err)
		} else {
			log.Printf("✓ Inserted default character: %s", character.Name)
		}
	}
}

// mustParseUUID 解析UUID，如果失败则panic
func mustParseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		panic(err)
	}
	return id
}
