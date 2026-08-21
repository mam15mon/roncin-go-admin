# 本地开发

## 推荐运行方式

本地开发直接运行 Go 服务、Ant Design Pro 和 WSL PostgreSQL，不要求 Docker。

```text
Windows/WSL
├── Go Kratos 服务：8000（HTTP）、9000（gRPC）
├── Ant Design Pro：8001（开发服务器）
└── PostgreSQL：5432（WSL）
```

Dockerfile 仅用于以后需要单镜像部署时的构建，不是本地开发前置条件。

## WSL PostgreSQL

先在 WSL 中确认数据库已经启动，并确认数据库、用户和密码与项目配置一致：

```bash
sudo service postgresql status
psql -h 127.0.0.1 -U roncin -d roncin
```

如果 Go 服务也在 WSL 中运行，优先使用 `127.0.0.1`。如果 Go 服务在 Windows 中运行，现代 WSL2 通常也会转发到 `127.0.0.1`；连接失败时，再使用 `hostname -I` 查看 WSL 地址，并确保 PostgreSQL 的 `listen_addresses` 和 `pg_hba.conf` 允许该连接。

## 环境文件

复制 `.env.example` 为仓库根目录的 `.env.local`，只在本机保存真实密码：

```env
DATABASE_SOURCE=postgres://roncin:本机密码@127.0.0.1:5432/roncin?sslmode=disable
BOOTSTRAP_ADMIN_USERNAME=admin
BOOTSTRAP_ADMIN_DISPLAY_NAME=系统管理员
BOOTSTRAP_ADMIN_PASSWORD=请替换为至少12位密码
BOOTSTRAP_ORGANIZATION_CODE=HQ
BOOTSTRAP_ORGANIZATION_NAME=总部
OTEL_ENABLED=false
```

`.env.local` 已被 Git 忽略，不要提交真实配置。

## 首次初始化

先显式执行版本化数据库迁移，再初始化管理员：

```powershell
pnpm run migrate:server
```

```powershell
pnpm run bootstrap:admin
```

迁移命令使用 `.env.local` 中的 `DATABASE_SOURCE`。管理员初始化只允许迁移完成且
尚无用户的数据库执行；如果数据库中已经存在用户，命令会拒绝继续。

## 启动服务

开两个终端：

```powershell
pnpm run dev:server
```

```powershell
pnpm run dev:web
```

打开 `http://localhost:8001/user/login`。前端开发服务器会把 `/api/*` 请求代理到 Go 的 `http://127.0.0.1:8000`。

健康检查：

```powershell
Invoke-WebRequest http://127.0.0.1:8000/health/live
Invoke-WebRequest http://127.0.0.1:8000/health/ready
```

## Docker 的边界

如果本地已经有 WSL PostgreSQL，不要执行 `docker compose up -d postgres`，否则会额外启动一套 PostgreSQL。生产环境是否使用根目录 `Dockerfile`，与本地开发方式互不影响。
