# 生成 wire 依赖注入代码
# 应用名称
APP_NAME = echome-be

# 默认配置文件路径
CONFIG_PATH = config/etc/config.yaml

# 代码编译测试
vet:
	go vet ./...

# 迁移数据库
migrate:
	go run ./tools/migrate.go -f $(CONFIG_PATH)

# 数据库生成代码
gen:
	go run ./tools/gen.go -f $(CONFIG_PATH)

# 构建二进制文件
build:
	go build -o $(APP_NAME) ./cmd/api

# 运行应用程序
run:
	go run ./cmd/api -f $(CONFIG_PATH)

# 构建并运行应用程序
build-run: build
	./$(APP_NAME) -f $(CONFIG_PATH)

# 依赖注入生成
wire:
	wire ./...

# 生成 swag 文档
swag:
	swag init -g cmd/api/main.go

# 清理构建产物
clean:
	rm -f $(APP_NAME)

# 帮助信息
help:
	@echo "可用命令："
	@echo "  make vet          - 运行代码静态分析"
	@echo "  make migrate      - 迁移数据库"
	@echo "  make gen          - 生成数据库代码"
	@echo "  make build        - 构建二进制文件"
	@echo "  make run          - 直接运行应用程序（不构建）"
	@echo "  make build-run    - 构建并运行应用程序"
	@echo "  make wire         - 生成依赖注入代码"
	@echo "  make swag         - 生成Swagger文档"
	@echo "  make clean        - 清理构建产物"
	@echo "  make help         - 显示帮助信息"

# 默认目标
.DEFAULT_GOAL := help