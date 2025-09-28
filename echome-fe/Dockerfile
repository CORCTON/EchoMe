FROM node:22-alpine AS base
# 安装 pnpm
RUN npm install -g pnpm

ARG NODE_ENV
ENV NODE_ENV=${NODE_ENV}

ENV NPM_CONFIG_REGISTRY=https://registry.npmmirror.com/
RUN pnpm config set registry https://mirrors.cloud.tencent.com/npm/

# 依赖安装阶段
FROM base AS deps
# 安装 libc6-compat 依赖
RUN apk add --no-cache libc6-compat
WORKDIR /app

# 拷贝依赖文件
COPY package.json pnpm-lock.yaml* ./
# 安装依赖
RUN pnpm i --frozen-lockfile

# 构建阶段
FROM base AS builder
WORKDIR /app
# 拷贝 node_modules
COPY --from=deps /app/node_modules ./node_modules
# 拷贝项目文件
COPY . .

# 声明构建参数
ARG NEXT_PUBLIC_WS_URL
ENV NEXT_PUBLIC_WS_URL=${NEXT_PUBLIC_WS_URL}
ARG OSS_REGION
ENV OSS_REGION=${OSS_REGION}
ARG OSS_ACCESS_KEY_ID
ENV OSS_ACCESS_KEY_ID=${OSS_ACCESS_KEY_ID}
ARG OSS_ACCESS_KEY_SECRET
ENV OSS_ACCESS_KEY_SECRET=${OSS_ACCESS_KEY_SECRET}
ARG OSS_BUCKET
ENV OSS_BUCKET=${OSS_BUCKET}
ARG API_BASE_URL
ENV API_BASE_URL=${API_BASE_URL}
ARG AUTH_PASSWORD
ENV AUTH_PASSWORD=${AUTH_PASSWORD}
ARG AUTH_SECRET
ENV AUTH_SECRET=${AUTH_SECRET}

# 确保 public 目录存在
RUN mkdir -p public

# 构建时禁用 Next.js 遥测
# ENV NEXT_TELEMETRY_DISABLED=1

# 构建项目
RUN pnpm run build

# 生产镜像
FROM base AS runner
WORKDIR /app

# 运行时禁用 Next.js 遥测
# ENV NEXT_TELEMETRY_DISABLED=1

# 创建用户和用户组
RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

# 拷贝 public 目录
COPY --from=builder /app/public ./public

# 设置 .next 目录权限
RUN mkdir .next
RUN chown nextjs:nodejs .next

# 拷贝构建输出
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

# 切换用户
USER nextjs

# 暴露端口
EXPOSE 3000

ENV PORT=3000

# 设置主机名
ENV HOSTNAME="0.0.0.0"
# 启动命令
CMD ["node", "server.js"]
