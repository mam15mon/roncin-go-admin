# 类型安全

## 生成物（禁止手改）

| 文件 | 来源 |
|------|------|
| `web/src/services/roncin/` | OpenAPI（服务端契约变更后 `pnpm run generate:web-client`） |
| `web/src/permissions.generated.ts` | 后端 `manifest.go`（`pnpm run generate:permission-keys`） |
| `web/src/enums.generated.ts`、`errorReasons.generated.ts` | 生成产物，同样禁手改 |

修改服务端契约后运行生成命令，源文件与生成物放同一提交审阅。

## 权限键编译期对齐

`web/src/access.ts` 引用的权限键名来自 `permissions.generated.ts`；后端
`internal/access/manifest.go` 改动后必须重新生成，否则编译期即暴露漂移。

## 场景：消费 Proto/OpenAPI 枚举

### 1. 适用范围

- 触发条件：前端根据生成客户端响应中的枚举字段执行过滤、分支、排序或状态展示。
- 目的：防止前端把 HTTP 数字枚举与字符串枚举名比较，导致真实数据被全部过滤。

### 2. 签名

- 响应示例：`API.MasterDataItem.kind?: number`。
- 生成常量示例：
  `MasterDataKind.MASTER_DATA_KIND_SERVICE_TYPE: 8`。

### 3. 契约

- 业务代码必须从 `@/enums.generated` 导入并使用生成常量。
- 禁止复制枚举名字符串或裸数字作为第二套真相。
- 不为旧字符串格式增加兼容分支；契约变化应修改 Proto 并重新生成。

### 4. 校验与错误矩阵

| 输入 | 处理 |
| --- | --- |
| 等于生成数字常量 | 进入对应业务分支 |
| 其他合法枚举值 | 不进入该分支 |
| `undefined` | 按字段可选语义处理，不猜测默认值 |
| 字符串枚举名或数字字符串 | 视为契约不匹配，不兼容、不静默纠正 |

### 5. Good / Base / Bad

- Good：API 返回数字 `8`，代码与生成的服务类型常量比较。
- Base：字段为 `undefined`，由调用方按可选字段语义处理。
- Bad：代码使用 `"MASTER_DATA_KIND_SERVICE_TYPE"` 或裸数字 `8` 比较。

### 6. 必需测试

- 测试数据使用生成数字常量构造，形态满足生成的 API 类型。
- 断言正确枚举进入目标分支，其他枚举不进入。
- 若业务还依赖代码等字段，同时断言字段在转换后未丢失。
- 测试期望也应引用生成常量，避免生产代码与测试共同保护旧字面量。

### 7. 错误与正确示例

```typescript
// 错误：HTTP 返回数字时永远无法匹配。
item.kind === 'MASTER_DATA_KIND_SERVICE_TYPE';

// 错误：当前能工作，但枚举值变化时生产代码和测试容易一起滞后。
item.kind === 8;

// 正确：生成枚举是唯一真相源。
item.kind === MasterDataKind.MASTER_DATA_KIND_SERVICE_TYPE;
```

## 校验

- TypeScript 严格模式，禁止 `any` 兜底绕过生成类型。
- 提交前按风险运行 `pnpm --dir web tsc` 与 `pnpm --dir web biome:lint`。

## Convention: 禁止借用「数值恰好相同」的兄弟枚举

**What**：每个 Proto 消息的枚举字段必须用它自己的生成常量消费
（如 `WorkbenchCommissionStatus.WORKBENCH_COMMISSION_STATUS_DRAFT`），
禁止因为另一个枚举（如 `FinanceCommissionStatus`）成员名和数值当前恰好一致
就借用它。已实际踩中：工作台抽屉用财务提成枚举消费工作台状态，tsc 不报错、
测试全绿，构成第二套真相，任一侧插入枚举值即静默错位。

**Why**：两个 Proto 枚举是独立真相源，「当前数值相同」是巧合不是契约；
TypeScript 结构类型对 `Record<number, …>` 键与 `status?: number` 不做枚举
来源区分，编译期无法拦截。

**Example**：

```tsx
// 错误：数值巧合可运行，但枚举重排即静默破坏筛选与文案
import { FinanceCommissionStatus } from '@/enums.generated';
status ?? FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT;

// 正确
import { WorkbenchCommissionStatus } from '@/enums.generated';
status ?? WorkbenchCommissionStatus.WORKBENCH_COMMISSION_STATUS_DRAFT;
```

**Prevention**：code review 与 trellis-check 时对每个生成枚举导入核对
「导入名与消费字段所属消息一致」；测试数据同样使用字段所属枚举构造。

## 服务端布尔响应的零值省略陷阱

### Convention: 判断后端 false 一律用 `!== true`，禁止 `=== false`

**What**：服务端 `internal/server/jsoncodec.go` 的 protojson 未开启
`EmitUnpopulated`，**非 optional 布尔字段为 false 时会在 JSON 响应中被整体省略**，
前端拿到的是 `undefined` 而不是 `false`。因此对后端布尔的严格 `=== false`
比较永远不成立，构成死分支。

**Why**：proto3 标量字段默认不序列化零值。已实际踩中：`can_modify`（订单
「已锁单」标签死分支）、`configuration_complete`（建账工作台三处配置守卫死分支）
与 `tax_inclusive`（订单费用「不含税单价」列初版 `=== false` 死分支，后按本规范
改为 `!== true` 才能识别 wire 上字段缺失的历史不含税行）。

**Example**：

```tsx
// 错误：false 被省略成 undefined，条件永不成立
{order.canModify === false && <Tag>已锁单</Tag>}

// 正确
{order.canModify !== true && <Tag>已锁单</Tag>}
```

**判定真值时用 truthy 即可**（`success`、`financeLocked` 等 true 才序列化的场景
不受影响）；只有需要区分「明确 false」的分支必须用 `!== true`。新增后端响应
布尔消费点时同步检查此规则。

## 权限能力与数据范围契约

### 1. 适用范围
修改 auth/me、access.ts、角色范围配置、按钮/路由/页面查询门控时适用。

### 2. 签名
`CurrentUser.permissionCapabilities: PermissionCapability[]`，每项为
`{ key: string, dataScope: string }`；由 auth.proto 生成客户端类型。

### 3. 契约
- 服务端从持有具体权限的有效角色中取最高范围，并应用当前工作台限制。
- 前端只能按该权限的 dataScope 判断动作所需范围；roleScopes 只展示角色，
  permissions 只用于展示权限列表/计数，不可作为旧契约回退。
- 有效范围为 organization、organization_tree、all。self 已停用，旧值清晰展示
  “已停用/需调整”，新建和保存拒绝；禁止自动改为 organization。
- 同权限码可能用于不同范围接口：用户列表要求组织级，全部成员关系要求 all，
  调用点应选择正确阈值，不把同码理解为全部接口可用。
- auth/me 只控制入口；真实记录所属组织、状态与审批资格由后端请求时校验。

### 4. 校验矩阵
| 输入 | 结果 |
| --- | --- |
| 缺少 permissionCapabilities，仅有 permissions/roleScopes | 拒绝权限入口，不回退 |
| 权限 A 属 organization，无关权限 B 属 all | A 不具备 all 能力 |
| 权限仅来自旧 self 角色 | 无有效能力，组织资源拒绝 |
| 同权限另有有效 organization 角色 | 按有效角色范围开放 |
| Any 路由通过，但当前 kind 未授权 | 页面精确拒绝且不启动查询 |

### 5. 正常与错误场景
- 正常：当前公司操作员按海运出口 create 能力看到对应新增按钮。
- 基础：总部与公司切换后刷新当前用户，能力随工作台变化。
- 错误：用所有角色中的最高范围扩大某一个权限。

### 6. 必需验证
覆盖跨角色范围拼接、同权限多角色、self 停用、缺字段拒绝、工作区切换、
具体业务类型请求门控；保留 lock 补录审批与本人工作台/提成入口。

### 7. 错误与正确
错误：`permissions.includes(key) && roleScopes.some(scopeMeetsMinimum)`。
正确：查找 `permissionCapabilities` 中 key 对应的条目，仅比较该条目的范围。
