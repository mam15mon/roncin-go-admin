# 本地开发

## 推荐运行方式

本地开发直接在 Linux / WSL 环境下运行 Go 服务、Ant Design Pro 和 PostgreSQL，不依赖 Docker。

```text
WSL / Linux
├── Go Kratos 服务：8000（HTTP）、9000（gRPC）
├── Ant Design Pro：8001（开发服务器）
└── PostgreSQL：5432（WSL 本机服务）
```

所有开发进程均在 Linux / WSL 环境下直接运行。`pnpm dev` 会检查 PostgreSQL 是否就绪，但不会启动或管理数据库服务。

## PostgreSQL 准备

在 WSL 中安装 PostgreSQL 服务端和客户端，并启动服务：

```bash
sudo apt update
sudo apt install postgresql postgresql-client
sudo service postgresql start
```

确认服务状态：

```bash
sudo service postgresql status
pg_isready -h 127.0.0.1 -p 5432
```

首次使用时，根据 `.env.local` 中的 `POSTGRES_USER` 和 `POSTGRES_DB` 创建角色与数据库：

```bash
sudo -u postgres createuser --pwprompt roncin
sudo -u postgres createdb --owner=roncin roncin_go_admin
```

命令中的 `roncin` 和 `roncin_go_admin` 应替换为实际配置值。数据库服务的启停和数据目录由 WSL 的 PostgreSQL 服务管理。

## 环境文件

复制 `.env.example` 为仓库根目录的 `.env.local`，只在本机保存真实密码：

```env
POSTGRES_DB=roncin_go_admin
POSTGRES_USER=roncin
POSTGRES_PASSWORD=本机密码
DATABASE_SOURCE=postgres://roncin:本机密码@127.0.0.1:5432/roncin_go_admin?sslmode=disable
RONCIN_INTEGRATION_DATABASE_SOURCE=postgres://roncin:本机密码@127.0.0.1:5432/roncin_go_admin_integration?sslmode=disable
BOOTSTRAP_ADMIN_USERNAME=admin
BOOTSTRAP_ADMIN_DISPLAY_NAME=系统管理员
BOOTSTRAP_ADMIN_PASSWORD=请替换为至少12位密码
BOOTSTRAP_ORGANIZATION_CODE=HQ
BOOTSTRAP_ORGANIZATION_NAME=总部
OTEL_ENABLED=false
```

`.env.local` 已被 Git 忽略，不要提交真实配置。

### PostgreSQL 集成测试库

需要验证真实事务锁、并发幂等和失败回滚时，单独创建集成测试数据库，禁止复用
`DATABASE_SOURCE` 指向的日常开发库：

```bash
sudo -u postgres createdb --owner=roncin roncin_go_admin_integration
```

把该数据库的连接地址写入 `.env.local` 的
`RONCIN_INTEGRATION_DATABASE_SOURCE`。测试会自动建表，并使用带唯一后缀的组织数据；
用例结束后会清理本次业务数据，但仍应保持数据库隔离。

在仓库根目录显式加载 `.env.local` 并运行账单、核销共享事务与订单事务测试：

```bash
node --env-file-if-exists=.env.local scripts/run-with-env.mjs \
  go -C server test ./internal/data \
  -run 'Test((FinanceBill|Verification)CreateSharedTransaction|OrderCreateTransaction)Postgres$' \
  -count=1 -v
```

未配置该变量时，集成测试会明确跳过，不会回退到日常开发数据库。

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

该命令会先检查本机 PostgreSQL 是否可连接，执行数据库迁移，然后并行启动后端
Air 热重载和前端开发服务。Air 会监听 Go、YAML 和 SQL 文件；每次后端重载前
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

## 数据库服务管理

停止服务不会删除数据：

```bash
sudo service postgresql stop
```

重新启动 PostgreSQL：

```bash
sudo service postgresql start
```

生产环境是否使用根目录 `Dockerfile`，与本地开发方式互不影响。
