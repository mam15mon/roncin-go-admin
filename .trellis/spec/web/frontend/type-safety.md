# 类型安全

## 生成物（禁止手改）

| 文件 | 来源 |
|------|------|
| `web/src/services/roncin/` | OpenAPI（服务端契约变更后 `pnpm run generate:web-client`） |
| `web/types/` | Umi 生成 |
| `web/src/permissions.generated.ts` | 后端 `manifest.go`（`pnpm run generate:permission-keys`） |
| `web/src/enums.generated.ts`、`errorReasons.generated.ts` | 生成产物，同样禁手改 |

修改服务端契约后运行生成命令，源文件与生成物放同一提交审阅。

## 权限键编译期对齐

`web/src/access.ts` 引用的权限键名来自 `permissions.generated.ts`；后端
`internal/access/manifest.go` 改动后必须重新生成，否则编译期即暴露漂移。

## 校验

- TypeScript 严格模式，禁止 `any` 兜底绕过生成类型。
- 提交前按风险运行 `pnpm --dir web tsc` 与 `pnpm --dir web biome:lint`。
