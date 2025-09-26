package infra

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/justin/echome-be/config"
	"github.com/justin/echome-be/gen/gen/query"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"moul.io/zapgorm2"
)

const postgresTcpDSN = "host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai"

var key keyType = struct{}{}

type keyType struct{}

type Transactional[T any] interface {
	Transaction(fn func(tx T) error, options ...*sql.TxOptions) error
}

type DB[T Transactional[T]] struct {
	query T
}

var instance any

func newDB[T Transactional[T]](query T) *DB[T] {
	database := &DB[T]{query: query}
	instance = database
	return database
}

func (d *DB[T]) Get(ctx context.Context) T {
	db, ok := ctx.Value(key).(T)
	if !ok {
		return d.query
	}
	return db
}

func Transaction[T Transactional[T]](ctx context.Context, fn func(ctx context.Context) error, options ...*sql.TxOptions) error {
	db := instance.(*DB[T])
	return db.query.Transaction(func(tx T) error {
		ctx = context.WithValue(ctx, key, tx)
		return fn(ctx)
	}, options...)
}

// NewDB 初始化数据库连接并返回DB实例
func NewDB(cfg *config.DatabaseConfig) (*DB[*gorm.DB], error) {
	logger := zapgorm2.New(zap.L())
	logger.IgnoreRecordNotFoundError = true

	// 构建数据库连接字符串
	dsn := cfg.GetDSN()

	// 创建数据库连接
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		SkipDefaultTransaction:   true,
		PrepareStmt:              true,
		Logger:                   logger,
		TranslateError:           true,
		DisableNestedTransaction: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// 初始化query包，设置全局Q变量
	query.SetDefault(database)

	return newDB(database), nil
}

// GetQuery 返回全局的query.Query实例
// 这个函数可以在整个应用中使用，获取已初始化的查询对象
func GetQuery() *query.Query {
	return query.Q
}

// Close 关闭数据库连接
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying database connection: %w", err)
	}
	return sqlDB.Close()
}
