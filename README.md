# Roncin 货代后台

Roncin 货代后台采用 Kratos v3、Ent、PostgreSQL 与 Ant Design Pro。前后端位于同一仓库，开发期分别启动，生产由 Go 服务同域提供 API 与 React 静态资源。

## 目录

- `server/`：Kratos API、领域逻辑、权限、数据访问与可观测性。
- `web/`：Ant Design Pro 管理后台。
- `docs/`：架构、权限、接口与运维说明。

## 基础命令

```powershell
pnpm install
pnpm run dev:web
pnpm run dev:server
pnpm run check
pnpm run build
```

运行所需环境变量见 `.env.example`。本地数据库可使用 `docker compose up -d postgres` 启动。

