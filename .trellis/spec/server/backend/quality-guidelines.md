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

## 场景：按权限解析跨组织读写范围

### 1. 适用范围 / 触发条件

- 业务接口允许当前工作区用户读取或写入其他组织的数据时适用。
- 新增跨组织能力时必须保留“角色、权限、数据范围、追加组织访问”的来源关系，
  禁止先把用户全部权限与全部组织访问分别拍平后再组合。
- 当前采用显式组织谓词的薄底座；不得为了省略参数而增加全局 Ent interceptor、
  mutation hook 或 `SystemScope` 隐式豁免。

### 2. 签名

```go
type RoleGrant struct {
    RoleID               uuid.UUID
    Permissions          map[string]struct{}
    DataScope            DataScope
    OrganizationAccesses []OrganizationAccess
}

type PermissionOrganizationScope struct {
    ReadableOrganizationIDs []uuid.UUID
    WritableOrganizationIDs []uuid.UUID
}

func (p *Principal) ResolvePermissionOrganizationScope(
    permission string,
) (PermissionOrganizationScope, error)

// 普通聚合：ID 与允许组织必须在同一数据库查询中求交。
FindAuthorized(ctx context.Context, id uuid.UUID, organizationIDs []uuid.UUID)

// 订单还必须保持业务类型与组织集合成对，不能生成笛卡尔积。
type OrderOrganizationScope struct {
    BusinessType    OrderBusinessType
    OrganizationIDs []uuid.UUID
}
```

### 3. 契约

- Resolver 只合并持有目标 `permission` 的启用角色；多个匹配角色的结果可取并集，
  不持有该权限的角色不得贡献 `data_scope` 或追加组织。
- 基础范围按该角色的 `all`、`organization_tree`、`organization`、`self` 解析；
  `self` 在组织维度只包含当前组织，业务实体的本人条件仍由对应领域追加。
- 基础范围同时进入 readable 与 writable；显式组织访问总是进入 readable，只有
  `writable=true` 才进入 writable；停用组织不进入任何结果。
- Service 选择当前接口的具体权限码和读/写集合；Biz 接收明确的组织 ID；Data 使用
  `Data.client(ctx)`，并把实体 ID 与 `OrganizationIDIn(...)` 放在同一 Ent 查询中。
- 创建、更新、确认、取消、状态流转分别按自己的写权限解析 writable，禁止用 read
  权限或当前组织猜测写范围；范围外目标不得静默改写为当前组织。
- 角色新增、编辑和成员角色分配的提权检查必须复用同样的来源绑定语义：对目标角色的
  每个权限，只能由操作者同样持有该权限的角色共同覆盖数据范围和组织访问。
- 当前不兼容旧的订单专用角色组织字段，不双读、不双写；契约变化必须同步 Proto、
  Ent、迁移、OpenAPI、前端客户端和仓库内调用方。

### 4. 校验与错误矩阵

| 条件 | 必须得到的结果 |
|------|----------------|
| 没有任何角色持有目标权限 | 返回 `ErrPermissionDenied`，不得回退到当前组织 |
| 北京访问项为只读，执行列表或详情读取 | readable 包含北京，可正常读取 |
| 北京访问项为只读，执行更新或状态流转 | writable 不含北京，返回权限错误（HTTP 403） |
| 角色 A 有权限、角色 B 有北京访问但无该权限 | 北京不得进入该权限的范围 |
| 目标实体 ID 存在但组织不在允许集合 | 仓储查询不可见，按领域约定映射为无权或不存在 |
| 追加组织已停用 | readable / writable 都不包含该组织 |
| 角色管理试图配置操作者无法覆盖的权限或范围 | 返回 `ErrAdminPrivilegeEscalation` |
| 订单权限覆盖不同业务类型 | 每个业务类型独立绑定组织集合，不得交叉组合 |

### 5. Good / Base / Bad

- Good：天津角色同时拥有 `system.order.sea_export.read` 和北京只读访问，读取范围为
  天津 + 北京，写范围仍只有天津。
- Base：没有追加组织时，继续按该角色原有 `data_scope` 访问当前组织、组织树或全部
  启用组织。
- Bad：订单角色只有查看权限，财务角色只有北京访问；把两者拍平后允许查看北京订单。
- Bad：先按 ID 全局读取实体，再在 Service 内存中判断组织；这会扩大数据暴露面，且
  子资源、预加载或审计很容易在判断前越界。

### 6. 必需测试

- Resolver 单元测试：四种 `data_scope`、停用组织、read/write 差异、无权限拒绝、
  同权限多角色并集，以及不同权限角色不得借用范围。
- 角色管理测试：创建、更新和成员角色分配均覆盖跨角色借权反例与只读提升为可写反例。
- 每个接入聚合至少验证列表、详情、一个写操作的组织集合传递；Data 层断言最终 SQL/Ent
  谓词同时包含主键（适用时）和显式组织集合。
- 订单额外验证 `BusinessType + OrganizationIDs` 成对谓词；账单额外回归事务、锁、版本
  和状态流转没有因组织范围接入而绕过。
- 跨层验收至少包含“天津工作区读取北京只读资源”和“同一资源写入被拒绝”两条路径。

### 7. 错误与正确示例

```go
// 错误：权限和组织访问分别拍平，会把不同角色拼成不存在的能力。
permissions := unionAllRolePermissions(roles)
organizationIDs := unionAllRoleOrganizations(roles)

// 正确：先按具体权限选择角色，再解析并显式传递组织集合。
scope, err := principal.ResolvePermissionOrganizationScope(permission)
if err != nil {
    return err
}
bill, err := repo.Get(ctx, id, scope.ReadableOrganizationIDs)
```

## 验证命令（按风险选取最小集）

```bash
go -C server test ./...
go -C server vet ./...
go -C server generate
make -C server api
```

仓库根目录：`pnpm run check:server`、`pnpm run check`（全量）、`pnpm run build`。

### 专用 PostgreSQL 测试不得以 Skip 充当通过

依赖 `RONCIN_INTEGRATION_DATABASE_SOURCE` 的真实 PostgreSQL 测试在变量为空时会
明确 `Skip`。普通 `go test ./...` 即使退出码为 0，也只能证明默认测试通过，不能证明
事务、锁、并发和 PostgreSQL 约束用例已经执行。

```bash
# 错误：只看退出码，把 SKIP 误报为真实数据库测试通过。
go -C server test ./...

# 正确：为质量门显式注入一次性 PostgreSQL，并在 -v 日志中确认目标用例为 PASS。
RONCIN_INTEGRATION_DATABASE_SOURCE="postgres://..." \
  go -C server test -v ./internal/data -run 'Postgres$' -count=1
```

- CI 中承担合并门禁的 PostgreSQL Job 必须显式提供专用连接串，禁止回退到开发库。
- 验收记录必须区分 `PASS`、`SKIP` 和“测试文件存在”；不得只记录命令最终退出码。
- 测试使用随机业务键或一次性数据库隔离，结束后检查夹具、临时数据库和角色无残留。
- 新增关键事务/锁测试时，同步确认它进入真实 PostgreSQL Job，而不只是进入普通
  `go test ./...` 的包扫描范围。

## 原则

- 先确认目标、影响范围和验证方式再动手；只实现当前需求，不做猜测性重构、
  无关格式化或一次性抽象。
- 禁止 TDD；功能实现后按风险补针对性验证；修缺陷时至少验证触发场景与相邻路径。
- 提交信息用 Conventional Commits（`feat:` / `fix:` / `refactor:` / `docs:` /
  `test:` / `chore:`），准确描述变更。
