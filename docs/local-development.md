# 本地开发

## 推荐运行方式

本地开发直接在 Linux / WSL 环境下运行 Go 服务和 Ant Design Pro，PostgreSQL 由 Docker Compose 容器提供。

```text
WSL / Linux
├── Go Kratos 服务：8000（HTTP）、9000（gRPC）
├── Ant Design Pro：8001（开发服务器）
└── PostgreSQL：5432（Docker Compose 容器）
```

Go 服务与前端均在 Linux / WSL 环境下直接运行，PostgreSQL 容器由 `pnpm dev` 自动拉起与管理。

## Docker PostgreSQL

安装 Docker 后，可手工查看数据库容器状态和日志：

```bash
docker compose --env-file .env.local ps postgres
docker compose --env-file .env.local logs postgres
```

`pnpm dev` 会自动启动数据库容器，并在容器无响应时重启一次，
无需日常手工执行上述命令。数据库数据保存在 Docker named volume
`roncin-go-admin_postgres-data` 中。

## 环境文件

复制 `.env.example` 为仓库根目录的 `.env.local`，只在本机保存真实密码：

```env
POSTGRES_DB=roncin_go_admin
POSTGRES_USER=roncin
POSTGRES_PASSWORD=本机密码
DATABASE_SOURCE=postgres://roncin:本机密码@127.0.0.1:5432/roncin_go_admin?sslmode=disable
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

```bash
pnpm run migrate:server
```

```bash
pnpm run bootstrap:admin
```

迁移命令使用 `.env.local` 中的 `DATABASE_SOURCE`。管理员初始化只允许迁移完成且
尚无用户的数据库执行；如果数据库中已经存在用户，命令会拒绝继续。

## 启动服务

在仓库根目录执行一条命令：

```bash
pnpm dev
```

该命令会依次启动 Docker PostgreSQL、执行数据库迁移，然后并行启动后端 Air
热重载和前端开发服务。Air 会监听 Go、YAML 和 SQL 文件；每次后端重载前
都会执行幂等迁移。修改 Ent Schema 时仍需要按项目约束生成对应迁移 SQL。

打开 `http://localhost:8001/user/login`。前端开发服务器会把 `/api/*` 请求代理到 Go 的 `http://127.0.0.1:8000`。

健康检查：

```bash
curl http://127.0.0.1:8000/health/live
curl http://127.0.0.1:8000/health/ready
```

## 行业主数据同步

机场使用 OurAirports 每日数据，可在仓库根目录直接联网同步到
`BOOTSTRAP_ORGANIZATION_CODE` 指定的组织：

```bash
pnpm run sync:airports
```

港口使用 UNECE UN/LOCODE 官方发布包。下载 ZIP 后必须显式提供文件和版本：

```bash
pnpm run sync:unlocode -- -source /path/to/loc251csv.zip -release 2025-1
```

两个命令均会先完成全量解析、标准码冲突检查和组织内异来源冲突检查，再开启
写事务；解析不完整或存在致命冲突时不会停用现有数据。机场、港口的网络同步
只更新官方英文信息和来源元数据，不覆盖人工维护的中文名。

## 数据库容器管理

停止容器不会删除数据：

```bash
docker compose --env-file .env.local stop postgres
```

重新启动数据库：

```bash
docker compose --env-file .env.local up -d postgres
```

不要执行 `docker compose --env-file .env.local down -v`，该命令会删除本地数据库
volume。生产环境是否使用根目录 `Dockerfile`，与本地开发方式互不影响。
