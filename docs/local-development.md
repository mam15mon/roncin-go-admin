# 本地开发

## 推荐运行方式

本地开发直接运行 Go 服务、Ant Design Pro 和 Windows PostgreSQL，不要求 Docker。

```text
Windows
├── Go Kratos 服务：8000（HTTP）、9000（gRPC）
├── Ant Design Pro：8001（开发服务器）
└── PostgreSQL：5432（Windows 本机）
```

Dockerfile 仅用于以后需要单镜像部署时的构建，不是本地开发前置条件。

## Windows PostgreSQL

开发环境使用 Scoop 安装的 Windows PostgreSQL，数据目录为
`C:\Users\admin\scoop\persist\postgresql\data`。可手工查看运行状态：

```powershell
pg_ctl status -D C:\Users\admin\scoop\persist\postgresql\data
pg_isready -h 127.0.0.1 -p 5432
```

`pnpm dev` 会自动检查并启动该实例，无需日常手工执行上述命令。

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

在仓库根目录执行一条命令：

```powershell
pnpm dev
```

该命令会依次启动 Windows PostgreSQL、执行数据库迁移，然后并行启动后端 Air
热重载和前端开发服务。Air 会监听 Go、YAML 和 SQL 文件；每次后端重载前
都会执行幂等迁移。修改 Ent Schema 时仍需要按项目约束生成对应迁移 SQL。

打开 `http://localhost:8001/user/login`。前端开发服务器会把 `/api/*` 请求代理到 Go 的 `http://127.0.0.1:8000`。

健康检查：

```powershell
Invoke-WebRequest http://127.0.0.1:8000/health/live
Invoke-WebRequest http://127.0.0.1:8000/health/ready
```

## Docker 的边界

本地已经使用 Windows PostgreSQL，不要执行 `docker compose up -d postgres`，否则会额外启动一套 PostgreSQL。生产环境是否使用根目录 `Dockerfile`，与本地开发方式互不影响。
