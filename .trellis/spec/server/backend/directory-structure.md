# 分层结构与依赖方向

## 目录职责

| 目录 | 职责 | 禁止 |
|------|------|------|
| `server/api/` | Protobuf 契约（HTTP/gRPC 唯一真相源） | 手改生成物 |
| `server/internal/service/` | DTO ↔ 领域对象转换、边界校验、调用用例 | 读 Ent、写业务规则、直连数据库 |
| `server/internal/biz/` | 领域对象、用例、仓储接口、领域错误、业务规则 | 依赖 `service`/`data`；泄漏 Protobuf/Ent 类型 |
| `server/internal/data/` | Ent 仓储实现、持久化转换、驱动错误映射 | Ent 类型出本层 |
| `server/internal/server/` | 传输注册、中间件、健康检查、静态资源 | DTO 转换、业务决策 |
| `server/internal/access/` | 权限 Manifest 与权限码（单一真相源） | 前端复制权限真相 |
| `server/internal/platform/` | 日志、请求元数据等平台能力 | 承担业务逻辑 |

## 依赖方向

```
service → biz ← data（data 实现 biz 定义的仓储接口）
```

- `biz` 通过接口反转依赖，不 import `data`。
- Ent 实体与 Protobuf 消息都不得出现在 `biz` 的领域对象字段中；
  转换发生在 `service`（proto ↔ 领域对象）与 `data`（ent ↔ 持久化对象）。

## 契约变更流程

1. 修改 `server/api/**/*.proto`。
2. `make -C server api` 生成绑定代码，检查差异。
3. 仓库根目录 `pnpm run generate:web-client` 更新 OpenAPI 与前端客户端。
4. 源文件与生成物放同一提交。
