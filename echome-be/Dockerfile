### 全局参数定义
ARG GO_VERSION=1.24.3
ARG ALPINE_VERSION=3.20
ARG APP_NAME=echome-api
ARG APP_PORT=8081

### 构建阶段
FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

### 构建环境配置
WORKDIR /app
ARG APP_NAME
ENV APP_NAME=${APP_NAME}

# 以非root用户身份执行构建操作
RUN addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup

# 安装构建依赖
RUN apk add --no-cache git build-base

# 先复制依赖文件以充分利用Docker缓存机制
COPY --chown=appuser:appgroup go.mod go.sum* ./

# 下载并验证依赖
RUN --mount=type=cache,target=/go/pkg/mod \
    if [ -f go.sum ]; then go mod download; else go mod init ${APP_NAME} && go mod download; fi

# 复制源代码
COPY --chown=appuser:appgroup . .

# 切换到非root用户
USER appuser

# 构建静态链接的二进制文件
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -a -installsuffix cgo \
    -ldflags="-s -w" \
    -o ${APP_NAME} ./cmd/api

### 运行时阶段
FROM gcr.io/distroless/static-debian12:nonroot

### 运行时环境配置
WORKDIR /app
ARG APP_NAME
ARG APP_PORT
ENV APP_NAME=${APP_NAME}
EXPOSE ${APP_PORT}

# 从构建阶段复制二进制文件和配置
COPY --from=builder --chown=nonroot:nonroot /app/${APP_NAME} .
COPY --from=builder --chown=nonroot:nonroot /app/config/etc ./config/etc

# 定义入口点和命令，支持配置覆盖
ENTRYPOINT ["./${APP_NAME}"]
CMD ["-f", "config/etc/config.yaml"]
