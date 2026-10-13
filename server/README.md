# Roncin Go Admin 服务端

基于 [go-kratos/kratos](https://github.com/go-kratos/kratos) v3 的货代后台服务：
Protobuf 契约优先（HTTP + gRPC 双传输）、Ent + PostgreSQL 持久化、Wire 依赖注入。
目录约定、分层边界与权限清单规则见根目录 `AGENTS.md` 与 `server/AGENTS.md`。

## 目录结构

```text
api/                  Protobuf 契约与生成的绑定代码（唯一契约真相源）
cmd/                  入口指令
  server/               主服务入口
  bootstrap-admin/      按正式迁移创建唯一初始管理员或补齐其工作台授权；--sync-permissions 手工同步权限
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

首次初始化按 `.env.example` 设置 `BOOTSTRAP_ADMIN_USERNAME`、`BOOTSTRAP_ADMIN_DISPLAY_NAME`
和至少 12 位的 `BOOTSTRAP_ADMIN_PASSWORD`，执行 `pnpm run bootstrap:admin`。命令先建立
唯一初始管理员身份，再创建默认北京（BJ）/成都（CD）公司，并为系统管理及所有实际启用公司
补齐成员关系与各公司独立的 `administrator` 绑定。公司角色只包含公司允许的权限，
部门/团队复用公司工作台，不重复加入；后来手工授予系统管理员角色的普通账号不自动加公司。
后台手工创建公司也在同一事务内把此初始管理员加入，要求其系统成员、管理员角色及绑定完整有效；
缺失时明确要求先执行 bootstrap 补齐，再创建公司，不在创建公司时顺带恢复系统资格。

重复执行同一命令时，只补缺失项，保留原账号姓名、密码、已分配的其他角色及公司配置。
此时用户名、姓名、密码可留空；若提供用户名必须匹配原初始管理员，
`BOOTSTRAP_ORGANIZATION_CODE` 必须匹配原系统工作台。账号表非空却没有初始管理员、
存在多个初始管理员或身份配置冲突时明确失败，不会接管普通同名账号。
系统工作台必须唯一，配置编码不得指向另一套工作台；角色 `self` 数据范围已失效，保持配置并要求人工处理。
停用的初始账号或系统管理/系统管理员资格使整次授权失败；停用公司、公司管理员角色
或原公司成员关系保持停用并逐公司报告跳过原因，需人工恢复后重跑。数据库/审计错误
回滚整次可处理项，成功摘要区分新增成员、绑定、已满足和跳过，含跳过项时不表示全部成功。

已有环境补建默认公司可执行 `pnpm run bootstrap:admin --seed-branches`（同一管理员补齐及停用规则）。
开发业务种子必须先执行 `pnpm run bootstrap:admin`，再执行 `pnpm run sync:dev`；
开发人员及经营数据仍明确归属北京公司，不恢复已移除的上海/重庆默认种子，也不隐式启用
任何初始管理员资格。权限目录常规更新仍由 `migrate:server` 同步；bootstrap 的身份和
系统资格检查先于授权写入，不调用会修改全部管理员角色的全局权限同步。

## 测试与检查

```bash
go test ./...
go vet ./...
```

仓库根目录的 `pnpm run check:server` 会额外执行 Proto lint 和
govulncheck。扫描由 `scripts/check-server-vuln.mjs` 统一执行；脚本仅对精确
匹配 `github.com/xuri/excelize/v2@v2.11.1-0.20260728235842-f98df08a8f6a`
的 `GO-2026-6452` 做审计豁免，因为该伪版本已包含上游完整修复提交
`f98df08a8f6aed8bc3b193115d1abb5b1ea2e433`，且有内存与临时文件路径的
恶意工作簿回归测试。
任何其他可达漏洞、Excelize 版本变化或扫描异常仍会使检查失败。

## 构建与部署

```bash
make build     # 输出 bin/ 下的可执行文件
```

生产构建使用仓库根目录 `pnpm run build`：前端产物内嵌进单一二进制
`server/bin/roncin-server`，同域提供 `/api/*`、健康检查与 React 静态资源，
容器化部署使用仓库根目录 `Dockerfile`。
