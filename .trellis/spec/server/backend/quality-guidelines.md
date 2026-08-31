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

## 场景：同前缀静态 HTTP 路由与参数路由

### 1. 适用范围 / 触发条件

- 同一 Proto service 中存在相同 HTTP 方法、相同路径前缀的静态路径与参数路径时
  适用，例如 `/commissions/employees` 与 `/commissions/{id}`。
- Kratos v3 的 HTTP Router 按注册顺序匹配；动态参数路由先注册会把静态片段当作
  参数，导致请求进入错误处理器。

### 2. 签名

```proto
rpc ListCommissionEmployees(ListCommissionEmployeesRequest)
    returns (ListCommissionEmployeesResponse) {
  option (google.api.http) = {get: "/api/v1/finance/commissions/employees"};
}
rpc GetCommission(GetCommissionRequest) returns (GetCommissionResponse) {
  option (google.api.http) = {get: "/api/v1/finance/commissions/{id}"};
}
```

### 3. 契约

- 同 HTTP 方法、同前缀下，所有静态路径 RPC 必须在参数路径 RPC 之前声明。
- 调整顺序必须修改 `.proto` 真相源，再运行 `make -C server api`；禁止手改
  `*_http.pb.go`。
- 路径、请求/响应类型、operation 和权限注解保持原契约，除非任务另有要求。

### 4. 校验与错误矩阵

| 条件 | 必须得到的结果 |
|------|----------------|
| 请求静态路径 | 命中对应静态 RPC 的 operation 与处理器 |
| 请求合法参数路径 | 命中参数 RPC，路径参数正确解码 |
| 参数路由先于静态路由 | 视为路由契约错误，Router 回归测试必须失败 |
| 生成物与 Proto 顺序不一致 | 生成幂等检查失败，禁止提交 |

### 5. Good / Base / Bad

- Good：`/export`、`/employees`、`/candidates` 均声明在 `/{id}` 之前。
- Base：合法 UUID 详情请求仍命中 `GetCommission`。
- Bad：`/{id}` 先声明，使 `/employees` 被解码为 `id = "employees"`。

### 6. 必需测试

- 使用真实 `http.NewServer` 和生成的 `Register*HTTPServer` 注册路由，通过
  `httptest` 请求验证实际分发。
- 同时断言 transport operation、实际调用的 Service 方法和路径参数；禁止仅检查
  生成文件文本或路由列表顺序。
- 测试至少覆盖全部相邻静态路径和一个合法参数路径，并验证把动态路由移到前面时
  静态用例会失败。

### 7. 错误与正确示例

```proto
// 错误：动态路由会先吃掉 employees。
rpc GetCommission(...) returns (...) { option (google.api.http) = {get: "/commissions/{id}"}; }
rpc ListCommissionEmployees(...) returns (...) { option (google.api.http) = {get: "/commissions/employees"}; }

// 正确：静态路由先注册。
rpc ListCommissionEmployees(...) returns (...) { option (google.api.http) = {get: "/commissions/employees"}; }
rpc GetCommission(...) returns (...) { option (google.api.http) = {get: "/commissions/{id}"}; }
```

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
