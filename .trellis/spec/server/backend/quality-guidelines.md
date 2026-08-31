# 质量规范（分层禁令 / 生成物 / 验证命令）

## 分层禁令速查

- `service` 不读 Ent、不写业务规则、不直连数据库。
- `biz` 不依赖 `service`/`data`；领域对象不含 Protobuf/Ent 类型。
- `data` 之外不见 Ent 类型；仓储取客户端必须 `Data.client(ctx)`。
- `server`（internal/server）不做 DTO 转换与业务决策。
- 前端不复制权限真相，只消费 `/api/v1/auth/me` 返回的权限集。

## 生成物清单（只通过生成器更新）

| 生成物 | 生成方式 |
|--------|----------|
| `*.pb.go` / `*_grpc.pb.go` / `*_http.pb.go` / `openapi.yaml` | 修改 `.proto` 后 `make -C server api` |
| 配置绑定代码 | `make -C server config` |
| `web/src/services/roncin/`、`web/types/` | `pnpm run generate:web-client` |
| `web/src/permissions.generated.ts` | 改 `manifest.go` 后 `pnpm run generate:permission-keys` |

生成结果异常时修源文件或生成配置后重新生成，不手工修补；源与生成物同提交。

## 权限

- 权限码唯一登记处：`server/internal/access/manifest.go`。
- 路由级权限在 HTTP 注册处绑定；新增/改名/删除权限码或调整 `Requires` 后必须
  重新生成前端权限键。

## 验证命令（按风险选取最小集）

```bash
go -C server test ./...
go -C server vet ./...
go -C server generate
make -C server api
```

仓库根目录：`pnpm run check:server`、`pnpm run check`（全量）、`pnpm run build`。

## 原则

- 先确认目标、影响范围和验证方式再动手；只实现当前需求，不做猜测性重构、
  无关格式化或一次性抽象。
- 禁止 TDD；功能实现后按风险补针对性验证；修缺陷时至少验证触发场景与相邻路径。
- 提交信息用 Conventional Commits（`feat:` / `fix:` / `refactor:` / `docs:` /
  `test:` / `chore:`），准确描述变更。
