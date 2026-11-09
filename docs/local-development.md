# 本地开发

## 推荐运行方式

本地开发直接在 Ubuntu / Linux 环境下运行 Go 服务、Ant Design Pro 和 PostgreSQL，不依赖 Docker。

```text
Ubuntu / Linux
├── Go Kratos 服务：8000（HTTP）、9000（gRPC）
├── Ant Design Pro：8001（开发服务器）
└── PostgreSQL：5432（本机服务）
```

所有开发进程均在 Ubuntu / Linux 环境下直接运行。`pnpm dev` 会检查 PostgreSQL 是否就绪，但不会启动或管理数据库服务。

## PostgreSQL 准备

在 Ubuntu / Linux 中安装 PostgreSQL 服务端和客户端，并启动服务：

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

命令中的 `roncin` 和 `roncin_go_admin` 应替换为实际配置值。数据库服务的启停和数据目录由系统的 PostgreSQL 服务管理。

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
BOOTSTRAP_ORGANIZATION_NAME=系统管理
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
`RONCIN_INTEGRATION_DATABASE_SOURCE`。每个用例在专用集成库中创建随机隔离 Schema，
按顺序执行 `server/migrations/*.sql` 正式迁移，结束时删除该 Schema；不使用 Ent
运行期自动建表，仍须与日常开发库隔离。

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

迁移命令使用 `.env.local` 中的 `DATABASE_SOURCE`。首次初始化要求账号表为空，
创建唯一初始管理员；重复执行定位同一管理员，只补齐系统工作台及实际启用公司的成员和
管理员绑定，不修改密码、不覆盖其他账号、不重新启用已停用资格。账号表非空但无唯一
初始管理员、用户名冲突或系统资格失效时拒绝。首次创建密码至少 12 位，重复补齐无需
再次提供密码。具体规则见[管理员初始化](../.trellis/spec/server/backend/company-admin-provisioning.md)。

随后可执行 `pnpm run sync:all` 导入公共资料。开发演示数据仅在开发库执行
`pnpm run sync:dev`，且须先完成管理员初始化。

## 启动服务

在仓库根目录执行一条命令：

```bash
pnpm dev
```

该命令会先检查本机 PostgreSQL 是否可连接，执行 `migrate:dev`，然后并行启动后端
Air 热重载和前端开发服务。Air 虽监听 Go、YAML 和 SQL 文件，后续重载只重编译和启动 Go，
不会再次执行迁移。新增或修改迁移后须显式运行 `pnpm run migrate:dev`，或重启
`pnpm dev`；生产仍执行严格校验的 `pnpm run migrate:server`。Ent Schema 变更须生成
正式迁移 SQL，不以热重载代替迁移。

打开 `http://localhost:8001/user/login`。前端开发服务器会把 `/api/*` 请求代理到 Go 的 `http://127.0.0.1:8000`。

健康检查：

```bash
curl http://127.0.0.1:8000/health/live
curl http://127.0.0.1:8000/health/ready
```

## 行业主数据同步

机场、港口、航司、船公司等同步到全局公共目录，不按公司复制，也不读取
`BOOTSTRAP_ORGANIZATION_CODE` 作为同步目标。系统工作台负责公共维护，各公司按权限消费。

```bash
# 一次同步全部公共资料；这些根目录脚本已带 -apply，会写库
pnpm run sync:all

# 单独同步机场（默认下载 OurAirports CSV）
pnpm run sync:airports

# 单独同步海港（默认读取仓库归档 ZIP 与配套中文对照）
pnpm run sync:unlocode
```

若先预览，直接使用不带 `-apply` 的 Go CLI，不能将根目录同步脚本当只读预览：

```bash
node --env-file-if-exists=.env.local scripts/run-with-env.mjs \
  go -C server run ./cmd/sync-unlocode
```

机场以 IATA、港口以 UN/LOCODE 全局匹配。解析、冲突预检通过后才在事务内写入，
写事务中再次校验；同来源本版缺失记录停用，异来源同码冲突拒绝，不抢占人工记录。
机场同步保留人工中文名；港口同步采用配套中文对照，更新时会写入对照值，不适用
“所有同步永不覆盖中文”的说法。换用其他港口版本时须同时提供匹配的 ZIP、中文 CSV
及必要版本参数，见[机场港口同步](airport-port-sync.md)和[归档数据说明](../server/seeds/README.md)。

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
