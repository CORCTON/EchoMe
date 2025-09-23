# EchoMe 部署说明

## 总览
本项目包含前端 (Next.js) 与后端 (Go Echo) 两个服务，使用 Docker 镜像并通过 GitHub Actions 自动构建与部署到目标服务器 (115.190.101.38)。

## 目录结构新增
- `echome-be/Dockerfile` 后端多阶段构建
- `echome-fe/Dockerfile` 前端多阶段构建（Next.js standalone）
- `deploy/docker-compose.yml` 部署编排示例（生产机上会写入到 /opt/echome/deploy）
- `deploy/install_docker.sh` 服务器初始化脚本
- `.github/workflows/deploy.yml` CI/CD 工作流
- `DEPLOY.md` 当前文档

## 运行时端口
- 后端: 8081 (容器内) -> 服务器 8081
- 前端: 3000 (容器内) -> 服务器 3000

## 服务器准备 (首次)
SSH 登录你的服务器后执行：
```bash
bash -c "$(curl -fsSL https://raw.githubusercontent.com/<your-repo>/main/deploy/install_docker.sh)" || true
# 或者直接用仓库里的脚本 (如果已 clone)
chmod +x deploy/install_docker.sh && ./deploy/install_docker.sh
```
确保已为用户添加 docker 组（可能需要重新登录）。
bash -c "$(curl -fsSL https://raw.githubusercontent.com/<your-repo>/main/deploy/install_docker_cn.sh)"
或本地执行：
```bash
chmod +x deploy/install_docker_cn.sh && ./deploy/install_docker_cn.sh
```
可通过设置变量指定 docker-compose 版本：
```bash
COMPOSE_VERSION=2.27.1 bash deploy/install_docker_cn.sh

## GitHub Secrets 配置
在仓库 Settings -> Secrets and variables -> Actions 中添加：
- `SERVER_SSH_KEY`：部署服务器的私钥（建议使用专用 deploy key）
- `SERVER_USER`：登录用户名（例如 `ubuntu` / `root`）

可选：
- `ALIYUN_API_KEY` 等业务相关变量（如需注入，可改写 docker-compose 和工作流）。

## 工作流触发
- Push 到 `main` 分支并修改 `echome-be/**` 或 `echome-fe/**` 目录即触发对应镜像构建
- 手动：在 GitHub Actions 选择 `CI Build & Deploy` 并 `Run workflow` 可以强制部署 (`force=true`)

## 镜像命名
- 后端：`ghcr.io/<OWNER>/echome-be:latest`
- 前端：`ghcr.io/<OWNER>/echome-fe:latest`

## 手动回滚
若你使用 `:latest`，建议保留历史 tag：
可在工作流中添加 `tags: xxx:latest,xxx:${{ github.sha }}` 以便精确回滚：
```yaml
with:
  tags: ghcr.io/<OWNER>/echome-be:latest,ghcr.io/<OWNER>/echome-be:${{ github.sha }}
```
```bash
ssh -i key.pem $SERVER_USER@115.190.101.38
cd /opt/echome
docker compose -f deploy/docker-compose.yml pull
docker compose -f deploy/docker-compose.yml up -d
docker compose ps
```

## 修改环境变量
在服务器上新增文件 `/opt/echome/.env`，并在 compose 中引用：
```yaml
env_file:
  - /opt/echome/.env
```
然后 `docker compose up -d` 重建。

## 常见问题
1. 无法访问 3000/8081：检查防火墙与安全组是否开放端口。
2. Next.js 资源 404：确认 `Dockerfile` 使用了 standalone 输出并复制了 `.next/static` 与 `public`。
3. Go 应用日志缺失：可通过 `docker logs -f echome-be` 查看。

## 后续可优化
- 增加健康检查 `healthcheck` 字段
- 使用 Traefik / Nginx 反向代理 + HTTPS
- 添加版本 tag + Sentry 发布钩子
- 使用自定义 overlay network 供未来扩展

---
如需调整请在 Issue 中描述。祝部署顺利。
