# Roncin 货代后台

Roncin 货代后台采用 Kratos v3、Ent、PostgreSQL 与 Ant Design Pro。前后端位于同一仓库，开发期分别启动，生产可由 Go 服务同域提供 API 与 React 静态资源。

## 目录

- `server/`：Kratos API、领域逻辑、权限、数据访问与可观测性。
- `web/`：Ant Design Pro 管理后台。
- `docs/`：架构、权限、接口、开发与运维说明。

## 基础命令

```bash
pnpm install
pnpm dev
pnpm run dev:web
pnpm run dev:server
pnpm run check
pnpm run build
```

其中 `pnpm dev` 会检查 WSL 中运行的本机 PostgreSQL、执行数据库迁移，并并行启动前后端；`dev:web` 与 `dev:server` 用于只启动其中一端。

运行所需环境变量见 `.env.example`。按[本地开发说明](docs/local-development.md)在 WSL 中安装、启动并配置 PostgreSQL 后，配置 `.env.local` 即可。

## 行政区划数据

业务中的中国省、市、区县三级行政区划不随代码内置，需要单独同步。在完成
[本地开发说明](docs/local-development.md)中的首次初始化后执行：

```bash
pnpm run sync:regions
```

该命令实时调用民政部全国行政区划信息查询平台接口，抓取全量省市区数据并按
编码幂等写入 `administrative_regions` 表，可重复执行以更新数据。运行时从
`.env.local` 读取 `DATABASE_SOURCE`。
