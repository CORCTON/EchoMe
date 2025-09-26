# EchoMe Backend

这是EchoMe项目的后端代码，一个利用AI进行角色扮演的应用程序。后端基于Go语言实现，使用DDD(领域驱动设计)架构，并使用Wire进行依赖注入。

## 项目结构

项目遵循DDD架构，主要包含以下目录：

- `cmd/api`: 应用程序入口点
- `config`: 配置相关代码
- `internal/app`: 应用程序核心逻辑和依赖注入
- `internal/domain`: 领域模型和接口定义
- `internal/usecase`: 用例层，实现业务逻辑
- `internal/infrastructure`: 基础设施层，实现数据存储等
- `internal/interfaces`: 接口层，处理HTTP请求
- `client`: 第三方服务客户端

## 技术栈

- Go 1.21+
- Echo: HTTP路由框架
- Gorilla WebSocket: WebSocket支持
- Wire: 依赖注入
- koanf: 配置管理
- 阿里云百炼API: AI对话生成

## 安装依赖

```bash
# 安装项目依赖
go mod tidy
```

## 生成依赖注入代码

项目使用Wire进行依赖注入，需要生成依赖注入代码：

```bash
# 在项目根目录执行
make wire
```

## 运行项目

```bash
make run
```

## 配置

项目使用koanf进行配置管理，配置文件格式为YAML。默认配置文件路径为`config/etc/config.yaml`。

### 主要配置项

#### 服务器配置
- `server.port`: 服务器端口

#### AI服务配置

##### 阿里云百炼配置
- `aliyun.api_key`: 阿里云百炼API密钥
- `aliyun.endpoint`: 阿里云百炼API端点

##### AI服务选择
- `ai.service_type`: AI服务类型，可选值：`alibailian`

#### WebRTC配置
- `webrtc.stun_server`: STUN服务器地址

## API端点

### 角色相关
- `GET /api/characters`: 获取所有角色
- `GET /api/characters/{id}`: 获取单个角色
- `POST /api/character`: 创建角色（语音克隆并创建角色）

### 会话相关
- `POST /api/sessions`: 创建会话
- `GET /api/sessions?userId={userId}`: 获取用户的所有会话
- `GET /api/sessions/{id}`: 获取单个会话
- `GET /api/sessions/{id}/messages`: 获取会话中的所有消息
- `POST /api/sessions/{id}/messages`: 发送消息

### WebSocket端点
- `GET /ws/asr`: 语音识别WebSocket连接
- `GET /ws/tts`: 文本转语音WebSocket连接
- `GET /ws/webrtc/{sessionId}/{userId}`: WebRTC信令WebSocket连接
- `GET /ws/voice-conversation/{sessionId}/{characterId}`

### 系统端点
- `GET /health`: 健康检查端点，返回系统状态和可用服务信息
- `GET /swagger/*`: API文档（Swagger UI）

## 功能说明

### 角色创建功能

主要特点：

1. 通过语音克隆创建角色，自动设置角色ID为语音ID
2. 角色信息与数据库保持一致，包含ID、Name、Prompt、Avatar、CreatedAt和UpdatedAt字段
3. 创建角色的API需要传入：
   - `audio`（可选，当需要自定义音色时必须）
   - `name`（必须，角色名称）
   - `prompt`（必须，角色提示词）
   - `avatar`（可选，角色头像）
   - `flag`（必须，布尔值，标识是否需要自定义音色）

### 聊天功能

AI对话生成功能进行了优化，现在具有以下特点：

1. 当角色上下文为空时，会使用默认提示词："你是一个友好、专业的AI助手，会用自然的方式回答用户的问题。"
2. 无论是同步还是流式响应模式，都会应用相同的角色上下文处理逻辑

### AI服务集成

项目支持两种AI服务提供商：

1. **阿里云百炼API**：通过`client/aliyun_bl.go`实现
实现了`domain.AIService`接口，通过工厂模式（`client.NewAIServiceFromConfig`）根据配置动态选择使用哪种服务。

### 会话管理

- 用户可以创建多个与不同角色的会话
- 每个会话包含多条消息
- 发送消息后，系统会自动生成AI回复

### WebRTC支持

项目提供WebRTC信令服务，支持实时音视频通信功能。

## 启动验证

应用程序启动时会自动进行以下验证：

1. **配置验证**：检查所有必需的配置项是否正确设置
2. **服务验证**：验证所有依赖服务是否正确初始化
3. **路由验证**：确认所有必需的API端点和WebSocket端点已注册

启动成功后，可以通过以下方式验证系统状态：

```bash
# 检查健康状态
curl http://localhost:8081/health

# 访问API文档
open http://localhost:8081/swagger/
```

## 开发注意事项

1. **依赖注入**：修改依赖关系后，需要重新生成依赖注入代码

2. **内存存储**：当前项目使用内存存储数据，重启服务后数据会丢失

3. **AI服务配置**：使用前需要配置对应AI服务的API密钥和相关参数

4. **服务验证**：应用程序启动时会自动验证所有服务的配置和可用性

5. **安全的角色仓库查询**：角色仓库查询功能已实现安全访问，包括：
   - 正确处理数据库模型与领域模型的转换
   - 完善的空指针检查和错误处理
   - 使用上下文参数进行安全的数据库操作
   - 正确解析JSON字段并处理异常情况

## Swagger文档使用

项目已集成Swagger文档，用于方便地查看和测试API。

```bash
# 安装swag CLI工具（如果尚未安装）
go install github.com/swaggo/swag/cmd/swag@latest

# 在项目根目录执行命令生成最新文档
make swag
```

这将更新docs目录下的`docs.go`、`swagger.json`和`swagger.yaml`文件，确保Swagger文档与实际API保持一致。