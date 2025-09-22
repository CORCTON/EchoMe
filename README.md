<img width="9050" height="2849" alt="EchoMe" src="https://github.com/user-attachments/assets/b92c1c49-339b-43b5-8161-08d7b740ff54" />

# EchoMe
EchoMe - Voice Agent

## 项目结构 / Project Structure

本项目采用前后端分离的架构设计：

### 📁 目录说明 / Directory Structure

```
EchoMe/
├── echome-fe/   # 前端项目 (Next.js Frontend)
├── echome-be/   # 后端项目 (Go Backend)
└── README.md    # 项目说明文档
```

### 🛠️ 技术栈 / Tech Stack

#### 前端 (Frontend) - echome-fe/
- **框架**: Next.js
- **描述**: 用户界面和交互逻辑

#### 后端 (Backend) - echome-be/
- **语言**: Go
- **描述**: API服务和业务逻辑

## 🚀 快速开始 / Quick Start

### 前端开发 / Frontend Development
```bash
cd echome-fe/
pnpm install && pnpm dev
```

### 后端开发 / Backend Development
```bash
cd echome-be/
# 待添加 Go 项目后的编译和运行命令
```

## � 部署 / Deployment

本仓库提供 GitHub Actions 自动化部署，分别针对前端 (FE) 与后端 (BE)。支持：

- 监听 `main` 分支对应目录变更自动部署
- 手动触发 (workflow_dispatch)
- 回滚基础能力（通过保留历史 release 目录与备份）

### 🗂️ 目录与脚本

`deploy/deploy_fe.sh` 前端部署脚本 (服务器执行)
`deploy/deploy_be.sh` 后端部署脚本 (服务器执行)
`.github/workflows/deploy-fe.yml` 前端 CI/CD 工作流
`.github/workflows/deploy-be.yml` 后端 CI/CD 工作流

### 🔐 必需 Secrets (在 GitHub 仓库 Settings -> Secrets -> Actions 配置)

| Secret Key | 说明 |
|------------|------|
| `SERVER_HOST` | 服务器地址，例如: `115.190.101.38` |
| `SERVER_USER` | SSH 用户，例如: `root` 或 `deploy` |
| `SERVER_PASSWORD` | SSH 密码（或改为使用 SSH Key：可新增 `SERVER_SSH_KEY`）|
| `SERVER_PORT` | (可选) SSH 端口，默认 22 |

> 推荐改为免密钥方式：使用 `SERVER_SSH_KEY`（`appleboy/*` actions 支持）。若使用密钥，将 workflow 中的 `password:` 字段替换为 `key:`。

### 🧩 服务器前置准备

1. Node.js (建议 20+) & pnpm (若无会尝试 `corepack enable`)
2. 可选：pm2 (提供进程守护与重启) `npm i -g pm2`
3. Go 环境仅在本地/CI 构建时需要，服务器不必须（我们上传已编译二进制）
4. 目标目录（默认）：
	- 前端：`/opt/echome-fe`
	- 后端：`/opt/echome-be`
	手动创建并确保拥有写权限：`mkdir -p /opt/echome-fe /opt/echome-be`

### 🔄 部署流程概述

前端：
1. CI 安装依赖并执行 `pnpm build`
2. 打包 `.next`、`public`、`package.json` 等为压缩包上传服务器 `/tmp`
3. 服务器执行 `deploy_fe.sh`：
	- 解压到 `releases/<timestamp>-<sha>`
	- 切换 `current` 软链
	- 安装生产依赖并通过 pm2 或 nohup 启动

后端：
1. CI 使用 `go build` 交叉编译 Linux amd64 二进制
2. 打包二进制与 `config`、`docs` 上传服务器 `/tmp`
3. 服务器执行 `deploy_be.sh`：
	- 解压到 `releases/<timestamp>-<sha>`
	- 切换 `current` 软链
	- 尝试 systemd (若存在 `echome-be.service`) -> pm2 -> nohup 级别启动

### 💪 回滚策略（手动）

1. 登录服务器，查看 releases：`ls -ltr /opt/echome-fe/releases` 或 `/opt/echome-be/releases`
2. 更新软链指向：`ln -sfn /opt/echome-fe/releases/<old> /opt/echome-fe/current`
3. 重启对应进程 (pm2/systemd/nohup)。

### ▶️ 手动触发部署

GitHub -> Actions -> 选择 `Deploy Frontend` 或 `Deploy Backend` -> Run workflow。

### 📝 可扩展方向

- 增加镜像构建与 Container Registry 推送
- 增加健康检查 + 自动回滚逻辑
- 增加数据库迁移步骤 (如后续引入)
- 增加 Slack / 飞书 / 钉钉 通知

### ⚠️ 注意事项

- 默认端口：FE `3000`（可在手动触发时传入），BE 在代码配置文件中指定。
- 若使用 systemd，创建 `/etc/systemd/system/echome-be.service` 示例：
  ```ini
  [Unit]
  Description=EchoMe Backend
  After=network.target

  [Service]
  Type=simple
  WorkingDirectory=/opt/echome-be/current
  ExecStart=/opt/echome-be/current/echome-be -f config/etc/config.yaml
  Restart=on-failure
  User=root

  [Install]
  WantedBy=multi-user.target
  ```
  然后：`systemctl daemon-reload && systemctl enable echome-be && systemctl start echome-be`

---

如需添加灰度、镜像或回滚自动化，请提出进一步需求。

## �📝 开发说明 / Development Notes

- `echome-fe/` 目录将包含 Next.js 前端应用
- `echome-be/` 目录将包含 Go 后端服务
- 项目采用前后端分离架构，便于独立开发和部署
