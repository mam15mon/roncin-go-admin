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

## ProtoJSON 响应映射与零值存在性

### 1. 适用范围

消费生成客户端响应、增加 DTO → 行模型转换或校验时适用。服务端唯一编码入口为
`server/internal/server/jsoncodec.go`，目前只配置 `UseEnumNumbers: true`，
不会主动输出非 optional 标量的默认值。HTTP 200、集合非空仍可能因单条错误
映射导致 queryFn 整批失败；UI 模板无法替代协议转换。

### 2. 签名与入口

- Proto 字段声明是语义真相源；生成 TS 属性 `?` 只表示 JSON 属性可省略，
  不能据此判断它是不是 Proto `optional`。
- `MasterDataItem.sort_order: int32`、`enabled: bool` 均非 optional，省略分别为
  `0`、`false`；请求中的 `optional bool enabled` 省略表示“不筛选”。
- 主数据公共入口：`pages/master-data/components/masterDataMapper.ts`。
  `mapPersistedMasterDataFields` 统一身份、来源、启停、排序和更新时间，
  `mapPersistedMasterDataItem` 组合通用主数据的代码与名称校验；专用 mapper
  组合公共字段并处理本实体字段。港口/机场中文名可选，不强套通用名称必填。
- `MasterDataTemplate` / `useMasterDataCrud` 消费已转换行模型或显式 `mapItem`，
  模板不感知具体 Proto 类型，不通过字段名猜默认值。

### 3. 契约

1. 仅还原已核对的非 optional 默认值；协议与业务判断保留 optional 字段及消息对象的
   存在性。纯展示的明确投影（如机场城市名输出空文本）可保持既有行为，不得用
   展示投影反推“已设置”或改变请求筛选/业务提交语义。
2. 同一领域的公共字段转换只有一个入口，不能逐页重新定义缺失拒绝或默认值。
   映射先留所属模块，有真实跨模块使用再按依赖边界提取公共 feature。
3. 标识、代码等业务必填条件仍校验；合法零值与缺损业务对象分别处理。
   禁止把转换异常吞成空数组或在全局请求拦截器递归补零。
4. 只有成功取得数据后，才可按已知非 optional bool 语义使用 `!== true`。
   optional bool 的未设置不等于 false，尚未加载也不等于业务否定。
   不机械替换 `success === false`：有些计数接口没有 success 信封，须核对契约。
5. 字符串/数值枚举格式保持生成契约，不增加旧类型转换或历史兼容分支。

### 4. 验证与错误矩阵

| 输入/上下文 | 结果 |
| --- | --- |
| 已加载主数据省略 enabled / sortOrder | 行模型为 false / 0，可正常显示与维护 |
| 显式 false / 0、true / 非零排序 | 保留实际值，不改成新建默认值 100 |
| optional bool 未设置与显式 false | 保留 undefined 与 false 的区别，请求筛选语义不混淆 |
| 缺必填 id / code 等 | 显式抛错，不返回空列表掩盖 |
| 港口/机场缺可选中文名 | 按专用 mapper 显示规则转换，不能整批拒绝 |
| 数据尚未加载或请求失败 | 保留加载/错误状态，不按省略 bool 推导业务结论 |
| 列表字段省略 | 仅按对应 repeated 契约解读为空列表，不掩盖整个响应缺失 |

### 5. 正常、基础与错误场景

- 正常：29 条船代首条无 sortOrder、末条无 enabled，列表、搜索和分页均成功。
- 基础：显式排序与启用值不变，国家和四类专用目录组合公共映射。
- 错误：每个 mapper 在调用公共转换之前仍校验 `enabled === undefined`。
- 错误：为消除缺字段而全局补值，抹掉 optional 及消息对象的存在性。

### 6. 必需测试与门禁

- 服务端用真实 Proto 消息经注册 codec 编码，断言省略非 optional 零值、
  保留 optional 显式零值、未设置 optional 不输出，不修改实际编码策略。
- 前端 fixture 包含真正缺 key 的对象、显式零值、非零及真实缺损对象；
  覆盖公共 mapper、专用 mapper、页面读取和维护路径，不能只用完整对象 mock。
- 主数据复用门禁限定 `map*` 命名且首参为已登记 API DTO 的适配器，验证消费公共入口、
  阻止重复公共字段判断；普通 DTO helper 与请求筛选函数不在范围内。
  门禁自身包含合法 optional 判断不误报的测试。不要全仓禁止 undefined/false 比较。
- `pnpm run check:web` 包含复用检查；具体脚本入口见根 package.json。

### 7. 错误与正确示例

```tsx
// 错误：非 optional false / 0 在真实响应被省略，合法行因此报错。
if (item.enabled === undefined || item.sortOrder === undefined) {
  throw new Error('主数据响应缺少必填字段');
}

// 正确：公共适配器统一还原已核对的字段，专用 mapper 组合它。
const fields = mapPersistedMasterDataFields(item);
// optional 请求筛选保留 undefined，而不是套上述字段规则。
const enabledFilter = query.enabled;

// 错误：未完成请求就把 undefined 解释成业务 false。
const disabled = query.data?.canModify !== true;
// 正确：成功取得对象后才解释 non-optional bool，加载/失败单独处理。
const disabled = query.isSuccess && query.data?.canModify !== true;
```

## 权限能力与数据范围契约

### 1. 适用范围
修改 auth/me、access.ts、角色配置、按钮/路由/页面查询门控时适用。角色范围配置已退役，详见服务端 retire-role-data-scope.md。

### 2. 签名
`CurrentUser.permissionCapabilities: PermissionCapability[]`，每项为
`{ key: string, dataScope: string }`；由 auth.proto 生成客户端类型。
`createPermissionCapabilityChecker(currentUser)` 从 `@/access` 导出，返回按具体权限和
最低范围判断的函数；按钮/路由 access 与角色可授予候选共用该实现。

### 3. 契约
- 服务端从启用角色的具体权限与当前工作台推导有效能力：公司 organization、系统 all；角色本身不再存储或接受可配置范围。
- 前端只能按该权限的 dataScope 判断动作所需范围；roleScopes 只展示角色，
  permissions 只用于展示权限列表/计数，不可作为旧契约回退。
- 新增/编辑角色、角色列表与工作台摘要不再显示或提交 dataScope；Auth.RoleScope 只包含角色标识与名称。旧 self 由正式迁移停用，不因删字段自动成为有效授权。
- 同权限码可能用于不同范围接口：用户列表要求组织级，全部成员关系要求 all，
  调用点应选择正确阈值，不把同码理解为全部接口可用。
- auth/me 只控制入口；真实记录所属组织、状态与审批资格由后端请求时校验。
- 角色表单的可授予候选必须是完整权限字典与当前有效 permissionCapabilities 的交集，
  复用 access 的能力判定；全选、模块批量和统计都消费该候选，不在前端重写系统/公司
  权限分类。完整字典浏览页仍可展示全量目录，不能直接将目录当成可授予集合。
- 编辑角色时若原有 permissionKeys 超出当前可授予候选，保留原权限并明确提示、阻止
  误提交；禁止加载或保存时静默裁剪。切换工作台后候选随当前身份能力重新计算。
- 权限字典首次加载或后台重查失败时禁止提交，不把失败解释为空候选；手动重新加载
  复用 React Query refetch，恢复成功前保留当前角色输入与权限选择。

### 4. 校验矩阵
| 输入 | 结果 |
| --- | --- |
| 缺少 permissionCapabilities，仅有 permissions/roleScopes | 拒绝权限入口，不回退 |
| 权限 A 属 organization，无关权限 B 属 all | A 不具备 all 能力 |
| 权限仅来自旧 self 角色 | 无有效能力，组织资源拒绝 |
| 同权限另有有效 organization 角色 | 按有效角色范围开放 |
| Any 路由通过，但当前 kind 未授权 | 页面精确拒绝且不启动查询 |
| 公司管理员在角色表单全选读写 | 选择该公司的全部有效候选，不混入系统专属权限 |
| 编辑角色含有操作者不能授予的原权限 | 保留原授权事实，提示并阻止提交，不静默撤权 |
| 字典首次加载失败或有缓存时后台重查失败 | 禁止新增/保存，手动重试恢复后才允许提交，原选择不丢失 |

### 5. 正常与错误场景
- 正常：当前公司操作员按海运出口 create 能力看到对应新增按钮。
- 基础：总部与公司切换后刷新当前用户，能力随工作台变化。
- 错误：用所有角色中的最高范围扩大某一个权限。
- 错误：公司角色表单直接全选系统字典，提交后才收到「不能分配超出自身权限范围的角色」。

### 6. 必需验证
覆盖权限来源绑定、同权限多角色、停用角色无能力、缺字段拒绝、工作区切换、
具体业务类型请求门控；保留 lock 补录审批与本人工作台/提成入口。
角色配置还须覆盖公司管理员全选创建、系统工作台候选、普通用户和缺能力、编辑原授权
保留及工作台切换；完整字典页的全量展示不能被候选过滤误伤。

### 7. 错误与正确
错误：`permissions.includes(key) && roleScopes.some(scopeMeetsMinimum)`。
正确：查找 `permissionCapabilities` 中 key 对应的条目，仅比较该条目的范围。

## 海运改配预览与资格

- `SeaOrderReassignmentPreviewData.is_valid` 为非 optional bool；在成功查询返回边界
  归一化后再展示无效 errors/中文兜底，不能直接用原始响应 `=== false` 漏掉省略值。
- `matched` 已有成功边界归一化，其下游 `=== false` 是稳定模型判断，不机械替换。
- 菜单与已经打开的改配弹窗共用明确允许的资格派生值，加载/失效/失败不得继续操作；
  保留现有写成功后的同步阶段保护，不改变服务端权限、并发版本和状态机。
- 回归至少覆盖省略 false、错误列表缺失、加载/失败、缺版本、资格撤回及成功同步。
