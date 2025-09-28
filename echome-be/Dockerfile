# 全局参数
ARG GO_VERSION=1.24.3
ARG APP_NAME=echome-be
ARG APP_PORT=8081

# 构建阶段
FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /app
ARG APP_NAME
ENV APP_NAME=${APP_NAME}

# 复制依赖文件
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

# 复制源代码
COPY . .


# 若 config.yaml 未被预先生成，则用 example 作为兜底（不包含密钥）
RUN if [ ! -f config/etc/config.yaml ] && [ -f config/etc/config.yaml.example ]; then \
            cp config/etc/config.yaml.example config/etc/config.yaml; \
            echo "INFO: copied config/etc/config.yaml.example -> config/etc/config.yaml (no secrets)"; \
        fi

# 编译静态二进制
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -v -a -installsuffix cgo \
    -ldflags="-s -w" \
    -o ${APP_NAME} ./cmd/main

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