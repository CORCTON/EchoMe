package domain

import (
	"gorm.io/gorm"
)

// DBAdapter 数据库适应器
type DBAdapter interface {
	Get() *gorm.DB
	Close() error
}
