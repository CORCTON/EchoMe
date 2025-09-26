package infra

import (
	"fmt"

	"github.com/justin/echome-be/config"
	"github.com/justin/echome-be/internal/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type pgDB struct {
	db *gorm.DB
}

var _ domain.DBAdapter = (*pgDB)(nil)

// InitDB 初始化数据库连接
func NewDB(cfg *config.DatabaseConfig) (*pgDB, error) {
	// 构建数据库连接字符串
	dsn := cfg.GetDSN()

	// 创建数据库连接
	_db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}
	return &pgDB{_db}, nil
}
func (d *pgDB) Get() *gorm.DB {
	return d.db
}

func (d *pgDB) Close() error {
	db := d.Get()
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying database connection: %w", err)
	}
	return sqlDB.Close()
}
