# EchoMe Backend

这是EchoMe项目的后端代码，一个基于Websocket的实时语音AI助手系统。后端基于Go语言实现，使用洋葱架构和DDD(领域驱动设计)思想，并使用Wire进行依赖注入。

## 项目结构

项目遵循洋葱架构和DDD思想，主要包含以下目录：

- `cmd/main`: 应用程序入口点
- `config`: 配置相关代码和配置文件
- `internal/app`: 应用程序核心逻辑和依赖注入配置
- `internal/domain`: 领域模型和接口定义
- `internal/handler`: 控制器层，处理HTTP请求和WebSocket连接
- `internal/infra`: 基础设施层，实现数据存储和外部服务集成
- `internal/middleware`: 中间件，处理请求前的通用逻辑
- `internal/validation`: 数据验证相关代码
- `docs`: Swagger API文档
- `tools`: 开发工具和脚本

## 技术栈

- **语言**: Go 1.24.3
- **Web框架**: Echo v4
- **WebSocket**: Gorilla WebSocket
- **依赖注入**: Google Wire
- **配置管理**: Koanf (YAML)
- **数据库**: PostgreSQL + GORM v2
- **日志**: Zap
- **API文档**: Swagger (swaggo)
- **AI服务**: 阿里云语音AI (ASR/TTS/LLM)
- **云存储**: 阿里云OSS
- **搜索增强**: Tavily API

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

### 配置示例

```yaml
server:
  port: "8080"
datebase:
  host: "localhost"
  port: "5432"
  user: "your_db_user"
  password: "your_db_password"
  db_name: "your_db_name"
ai:
  service_type: "alibailian"
  timeout: 30
  max_retries: 3
tavily:
  api_key: "your_tavily_api_key"
aliyun:
  api_key: "your-alibailian-api-key"
  endpoint: "https://dashscope.aliyuncs.com"
  region: "cn-beijing"
  asr:
    model: "paraformer-realtime-v2"
    sample_rate: 16000
    format: "pcm"
    language_hints: ["zh", "en"]
  tts:
    model: "qwen-tts-realtime"
    default_voice: "Cherry"
    sample_rate: 24000
    response_format: "pcm"
  llm:
    model: "qwen-turbo"
    temperature: 0.7
    max_tokens: 2000
```

### 配置说明

- **服务器配置**: 设置后端服务监听的端口
- **数据库配置**: 配置PostgreSQL数据库连接参数（主机、端口、用户名、密码、数据库名）
- **AI服务配置**: 配置AI服务类型、超时时间和重试次数
- **Tavily配置**: 配置Tavily搜索API密钥
- **阿里云配置**: 配置阿里云百炼API相关参数：
  - 基础配置：API密钥、端点、区域
  - ASR配置：语音识别模型、采样率、格式、语言提示
  - TTS配置：语音合成模型、默认音色、采样率、响应格式
  - LLM配置：大语言模型、温度、最大token数

## API端点

### 角色相关
- `GET /api/characters`: 获取所有角色
- `GET /api/characters/{id}`: 获取单个角色
- `POST /api/character`: 创建角色

### WebSocket端点
- `GET /ws/asr`: 语音识别WebSocket连接
- `GET /ws/voice-conversation?characterId={characterId}`: 实时语音对话WebSocket连接

### 系统端点
- `GET /health`: 健康检查端点，返回系统状态和可用服务信息
- `GET /swagger/*`: API文档（Swagger UI）

## 功能说明

### 角色管理功能

角色管理功能支持完整的CRUD操作，主要特点：

1. **角色创建**：支持创建自定义AI角色，包含名称、系统提示词、头像等信息
2. **角色配置**：可配置角色的专业领域、行为模式和个性化特征
3. **角色存储**：角色信息持久化存储在PostgreSQL数据库中
4. **角色检索**：支持按ID和条件查询角色信息

### 实时语音对话功能

项目核心功能，支持与AI角色进行实时语音交互：

1. **语音活动检测(VAD)**：前端使用ONNX模型检测用户语音活动
2. **实时语音识别(ASR)**：通过独立WebSocket通道将音频流传输到阿里云ASR服务
3. **流式文本处理**：接收增量识别结果并实时显示
4. **AI回复生成**：大语言模型基于角色人设和对话上下文生成回复
5. **实时语音合成(TTS)**：将AI回复实时转换为语音流
6. **双向流式通信**：保持WebSocket长连接，支持音频和文本双向流传输

### AI服务集成

项目集成多种阿里云AI服务：

1. **阿里云语音AI**：提供实时语音识别(ASR)和文本转语音(TTS)服务
2. **大语言模型(LLM)**：处理用户输入并生成智能回复
3. **多模态理解**：支持处理文档图像等多模态内容
4. **Tavily搜索**：提供联网搜索增强功能，为对话提供最新信息
5. **阿里云OSS**：用于存储角色头像和文档图像等媒体文件

### 会话与消息管理

完整的会话和消息管理系统：

- 用户可以创建多个与不同角色的会话
- 每个会话包含多条消息记录
- 支持消息的创建、查询操作
- 会话和消息信息持久化存储

### 文档处理功能

支持处理PDF等文档内容：

1. **前端预处理**：PDF.js在客户端将PDF转换为高清图像
2. **直接上传**：图像文件直接上传到阿里云OSS存储
3. **多模态理解**：后端调用多模态大模型进行图像识别分析
4. **内容提取**：自动识别文档结构、文字、图表等关键信息

## 启动验证

应用程序启动时会自动进行以下验证：

1. **配置验证**：检查所有必需的配置项是否正确设置
2. **服务验证**：验证数据库连接和AI服务可用性
3. **路由验证**：确认所有必需的API端点和WebSocket端点已注册

启动成功后，可以通过以下方式验证系统状态：

```bash
# 检查健康状态
curl http://localhost:8080/health

# 访问API文档
open http://localhost:8081/swagger/
```

## 开发注意事项

1. **依赖注入**：修改依赖关系后，需要重新生成依赖注入代码
   ```bash
   make wire
   ```

2. **数据库迁移**：首次运行或修改数据模型后，需要执行数据库迁移
   ```bash
   make migrate
   ```

3. **配置管理**：确保在`config/etc/config.yaml`中正确配置数据库和AI服务参数

4. **Swagger文档更新**：修改API后，需要更新Swagger文档
   ```bash
   make swag
   ```

5. **代码质量**：使用以下命令检查代码质量
   ```bash
   make vet
   go fmt ./...
   ```

6. **环境要求**：确保使用Go 1.24+版本开发，以兼容所有功能

## Swagger文档使用

项目已集成Swagger文档，用于方便地查看和测试API。

### 安装swag CLI工具

```bash
# 安装swag CLI工具（如果尚未安装）
go install github.com/swaggo/swag/cmd/swag@latest
```

### 生成/更新文档

```bash
# 在项目根目录执行命令生成最新文档
make swag
```

这将更新docs目录下的`docs.go`和`swagger.json`文件，确保Swagger文档与实际API保持一致。

### 访问文档

启动服务后，可以通过以下URL访问Swagger UI：
```
http://localhost:8080/swagger/index.html
```

也可以直接访问OpenAPI JSON：
```
http://localhost:8080/swagger/doc.json
```
