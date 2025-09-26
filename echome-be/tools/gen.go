//go:build ignore

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/justin/echome-be/config"
	"gorm.io/driver/postgres"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const postgresTcpDSN = "host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai"

func main() {
	// 定义命令行参数
	configPath := flag.String("f", "config/etc/config.yaml", "配置文件路径")
	
	// 解析命令行参数
	flag.Parse()

	// 加载配置
	cfg := config.Load(*configPath)

	// 查找项目根目录
	baseDir, err := findProjectRoot()
	if err != nil {
		log.Fatalf("Failed to find project root: %v", err)
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
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: false, // 使用复数表名
		},
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 配置代码生成器
	genCfg := gen.Config{
		OutPath:      filepath.Join(baseDir, "gen/gen/query"), // 生成的查询代码输出路径
		ModelPkgPath: filepath.Join(baseDir, "gen/gen/model"), // 生成的模型代码输出路径

		Mode: gen.WithDefaultQuery | gen.WithQueryInterface, // 生成默认查询方法和查询接口

		FieldNullable:     true,  // 字段可为空
		FieldCoverable:    false, // 不生成字段覆盖相关代码
		FieldSignable:     true,  // 生成字段符号相关代码
		FieldWithIndexTag: false, // 不生成索引标签
		FieldWithTypeTag:  true,  // 生成类型标签
	}
	genCfg.WithImportPkgPath("github.com/shopspring/decimal")
	genCfg.WithImportPkgPath("cnb.cool/cymirror/ces-services/common/db/optimisticlock")

	// 创建代码生成器实例
	g := gen.NewGenerator(genCfg)
	g.UseDB(db)

	// 配置数据类型映射
	dataMap := map[string]func(columnType gorm.ColumnType) (dataType string){
		"numeric": func(columnType gorm.ColumnType) (dataType string) {
			return "decimal.Decimal" // 将数据库 numeric 类型映射为 decimal.Decimal
		},
	}
	g.WithDataTypeMap(dataMap)

	// 配置特殊字段处理
	autoUpdateTimeField := gen.FieldGORMTag("updated_at", func(tag field.GormTag) field.GormTag {
		return tag.Append("autoUpdateTime") // 自动更新时间字段
	})
	autoCreateTimeField := gen.FieldGORMTag("created_at", func(tag field.GormTag) field.GormTag {
		return tag.Append("autoCreateTime") // 自动创建时间字段
	})
	softDeleteField := gen.FieldType("deleted_at", "gorm.DeletedAt")   // 软删除字段
	versionField := gen.FieldType("version", "optimisticlock.Version") // 乐观锁版本字段

	// 组合所有字段选项
	fieldOpts := []gen.ModelOpt{autoCreateTimeField, autoUpdateTimeField, softDeleteField, versionField}

	// 生成所有表的模型
	allModel := g.GenerateAllTable(fieldOpts...)

	// 应用基本查询方法
	g.ApplyBasic(allModel...)

	// 执行代码生成
	g.Execute()
}

// findProjectRoot 查找项目根目录（包含go.mod的目录）
func findProjectRoot() (string, error) {
	dir, err := filepath.Abs(".")
	if err != nil {
		return "", fmt.Errorf("cannot get absolute path: %w", err)
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// 已到达根目录
			return "", fmt.Errorf("go.mod not found in any parent directory")
		}
		dir = parent
	}
}
