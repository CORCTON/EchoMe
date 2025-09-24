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

### 中国大陆环境（推荐使用加速镜像脚本）
如果服务器在中国大陆，官方脚本可能较慢，可使用镜像优化版本：
```bash
bash -c "$(curl -fsSL https://raw.githubusercontent.com/<your-repo>/main/deploy/install_docker_cn.sh)"
```
或本地执行：
```bash
chmod +x deploy/install_docker_cn.sh && ./deploy/install_docker_cn.sh
```
可通过设置变量指定 docker-compose 版本：
```bash
COMPOSE_VERSION=2.27.1 bash deploy/install_docker_cn.sh
```

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

## 手动 SSH 部署调试
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

## 使用 Traefik 自动申请 HTTPS（最简单方法）

若想让项目通过域名自动获得 Let's Encrypt 证书，可以在远程服务器上使用 Traefik 作为反向代理。仓库已经在 CI 部署脚本中加入了一个最小的 Traefik 配置：

- 需要在 GitHub 仓库的 Secrets 中添加：
  - `SERVER_USER`、`SERVER_PASSWORD`（当前默认使用 password 登录）；
  - `SERVER_HOST`：用于 SSH 连接的主机地址，可以是服务器公网 IP（例如 115.190.101.38）或域名；
  - `SERVER_DOMAIN`：用于 TLS/Traefik 的域名（必须是一个可被 DNS 指向服务器 IP 的域名，例如 example.com），注意 `SERVER_HOST` 与 `SERVER_DOMAIN` 可能不同；
  - `LETSENCRYPT_EMAIL`：用于注册 Let's Encrypt 的联系邮箱（可选，若为空会使用 admin@SERVER_HOST）。

- 工作流会在服务器 `/opt/echome/deploy` 生成 `docker-compose.yml`，并创建 `deploy/letsencrypt/acme.json`（权限 600）。Traefik 会监听 80/443，并为 `echome-fe`（主域名或 www）与 `echome-be`（api. 子域）申请证书。

- 简单测试与调试步骤：
  1. 确认域名 DNS 指向服务器公网 IP（A 记录）；
  2. 在 GitHub Actions 中运行工作流（可用 `workflow_dispatch` 手动触发并传入 `force=true`）；
  3. 登录服务器：
     ```bash
     ssh $SERVER_USER@your.server
     cd /opt/echome
     sudo docker compose -f deploy/docker-compose.yml logs -f traefik
     ```
     观察 Traefik 日志中 ACME 申请流程；若挑战失败，日志会显示错误原因（DNS、端口被防火墙占用、或 80/443 未开放）。
  4. 若证书成功，浏览器访问 https://your.domain 应显示前端服务，并且证书由 Let's Encrypt 签发。

注意：在中国大陆环境，访问 Let's Encrypt 可能会更慢或失败，可考虑使用自签证书或商业 CA。若你需要使用 Cloudflare、Aliyun 之类的 DNS 验证来申请证书（DNS-01），可以扩展 Traefik 的 certificatesResolvers 配置并在远程环境中注入相应 API Key。

---
如需调整请在 Issue 中描述。祝部署顺利。
