# Roncin Go Admin 服务端

基于 [go-kratos/kratos](https://github.com/go-kratos/kratos) v3 的货代后台服务：
Protobuf 契约优先（HTTP + gRPC 双传输）、Ent + PostgreSQL 持久化、Wire 依赖注入。
目录约定、分层边界与权限清单规则见根目录 `AGENTS.md` 与 `server/AGENTS.md`。

## 目录结构

```text
api/                  Protobuf 契约与生成的绑定代码（唯一契约真相源）
cmd/                  入口指令
  server/               主服务入口
  bootstrap-admin/      冷启动初始化根组织与超级管理员；--sync-permissions 手工同步权限
  migrate/              版本化迁移执行器（迁移后自动同步权限清单）
  sync-airports/        OurAirports 机场同步 CLI
  sync-regions/         全国行政区划同步 CLI
  sync-unlocode/        UN/LOCODE 海港同步 CLI（需人工下载发布包）
  export-permission-manifest/   导出权限码清单供前端权限键生成
  generate-access-rules/        由 Proto 契约生成接口权限规则
configs/              运行配置（config.yaml、config.production.yaml 等，不含凭据）
internal/
  biz/                  领域对象、用例、仓储接口与业务规则
  data/                 Ent 仓储实现与持久化转换
  service/              传输层 DTO 转换与用例调用
  server/               HTTP/gRPC 注册、中间件与静态资源服务
  access/               权限 Manifest 与权限码
  conf/                 配置 Protobuf（make config 生成）
  platform/             日志、遥测等平台能力
  webassets/            生产前端静态资源内嵌
migrations/           版本化 SQL 迁移脚本（用法见 migrations/README.md）
openapi.yaml          生成的 OpenAPI 文档
```

## 代码生成

```bash
make init      # 安装 wire、buf 等生成器
make api       # 生成 API 绑定代码（含 go:generate 权限规则）
make config    # 生成配置 Protobuf
make all       # 全量生成、Wire 与模块整理
```

修改 `.proto`、权限 Manifest 或 Ent Schema 后必须重新生成对应代码，不手工修改
生成物。前端 OpenAPI 客户端在仓库根目录执行 `pnpm run generate:web-client` 更新。

## 本地运行

```bash
go run ./cmd/server -conf ./configs
```

默认端口见 `configs/config.yaml`：HTTP `0.0.0.0:8000`、gRPC `0.0.0.0:9000`。
开发期推荐在仓库根目录使用 `pnpm dev` / `pnpm run dev:server`（Air 热重载），
数据库迁移与初始化使用 `pnpm run migrate:server` 和 `pnpm run bootstrap:admin`。

## 测试与检查

```bash
go test ./...
go vet ./...
```

仓库根目录的 `pnpm run check:server` 会额外执行 Proto lint 和 govulncheck。

## 构建与部署

```bash
make build     # 输出 bin/ 下的可执行文件
```

生产构建使用仓库根目录 `pnpm run build`：前端产物内嵌进单一二进制
`server/bin/roncin-server`，同域提供 `/api/*`、健康检查与 React 静态资源，
容器化部署使用仓库根目录 `Dockerfile`。
