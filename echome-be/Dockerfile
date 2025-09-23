# 全局参数
ARG GO_VERSION=1.24.3
ARG APP_NAME=echome-be
ARG APP_PORT=8081

# 构建阶段
FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /app
ARG APP_NAME
ENV APP_NAME=${APP_NAME}

# 以非root用户身份执行构建操作
RUN addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup

# 变更工作目录所有者，避免以非 root 用户构建时写入 /app 权限被拒绝
RUN chown appuser:appgroup /app

# 确保 Go module cache 目录权限正确
RUN mkdir -p /go/pkg/mod && chown -R appuser:appgroup /go

# 安装构建依赖
RUN apk add --no-cache git build-base

# 先复制依赖文件以充分利用Docker缓存机制
COPY --chown=appuser:appgroup go.mod go.sum* ./

# 下载并验证依赖
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

# 复制源代码
COPY . .


# 编译静态二进制
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -v -a -installsuffix cgo \
    -ldflags="-s -w" \
    -o ${APP_NAME} ./cmd/api

# 运行时阶段
FROM scratch

WORKDIR /app
ARG APP_NAME
ARG APP_PORT
ENV APP_NAME=${APP_NAME}
EXPOSE ${APP_PORT}

# 复制二进制和配置文件
COPY --from=builder /app/${APP_NAME} .
COPY --from=builder /app/config/etc ./config/etc

# 入口点和默认命令
ENTRYPOINT ["./echome-be"]
CMD ["-f", "config/etc/config.yaml"]