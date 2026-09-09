# 订单类型注册薄底座 PRD

## 1. 目标与用户价值

在不改动现有海运出口（SE）业务行为的前提下，为订单列表、新建、详情、费用页及其数据
Hook 建立唯一的订单类型注册入口，使路由类型、业务枚举、运输方式、表单适配和详情专属
能力不再散落在页面条件分支中。

首期只注册并迁移 SE，不实现 SI、AE、AI、LAND 或 RAIL。完成后，后续订单类型应通过新增
独立定义和业务模块接入，而不是继续扩大 `new.tsx`、`detail.tsx`、`common.ts` 或通用
Payload 构建器中的 `if/else`。

本任务保留已经验收通过的订单草稿、dirty、只读恢复和显式刷新令牌语义；注册层只负责
业务类型分发，不接管表单生命周期或请求竞态控制。

## 2. 已确认事实

- 当前产品只开放 `sea-export`；`OrderKind`、`ORDER_KIND_CONFIGS` 和 `parseOrderKind` 定义在
  `web/src/pages/orders/common.ts:189-223`。
- `parseOrderKind` 不只服务新建与详情，还被列表、费用页、列表资源 Hook、创建资源 Hook、
  详情数据 Hook、列表查询、页面头部及相关测试消费；注册层迁移不能只改两个页面。
- `OrderListTemplate` 在公共组件内另有一套包含 `sea-import`、`air-export`、`truck`、
  `customs` 等值的 `OrderKind` 联合和本地 `kindMap`，与当前产品注册事实已经漂移。
- 新建页使用“非 sea 即 air”，详情页使用“非 air 即 sea”的不同 fallback；若直接增加
  LAND，新建与详情会选择不同且都错误的模板。
- 当前创建默认值不仅依赖创建人，还依赖服务类型候选项和“普货”候选项；定义接口不能把
  默认值上下文缩减为 `creator`。
- `order-create-payload.ts` 通过 `category/businessType` 判断 `isSea`；
  `orderDetailHelpers.ts` 通过海运字段是否存在反推 `isSea`，两者都不是可扩展的类型分发。
- `detail.tsx` 直接持有海运改配、共享航次、共享箱和拆票历史的请求、状态、按钮与覆盖层；
  单纯提供 `renderModals(context)` 无法让扩展模块独立持有这些 Hook 与本地状态。
- 订单操作权限的唯一真相源已经是后端权限 Manifest；`split/reassign/amend/void/switch` 等
  SE 专属操作已由 Manifest 的 `businessTypes` 限制。前端再维护同义 capability 集合会形成
  第二套能力真相。
- `OrderFormTemplate` 独占草稿键、dirty、恢复与清理；统一显式刷新令牌实际位于订单详情
  页面编排层，不在 `OrderFormTemplate` 内。
- 后端枚举和权限目录已认识 SE、SI、AE、AI、LAND、RAIL，但
  `normalizeOrder` 目前明确只允许 SE + Export；其他类型尚未具备领域模型和创建契约。
- 项目处于快速上线阶段，不需要旧前端配置 API、旧类型别名或历史路由兼容层。
- 本任务确定为纯前端重构；后端继续保持仅支持 SE + Export，直到 SI 接入任务依据真实领域
  模型设计业务类型分发。

## 3. 需求范围

### R1：订单类型唯一注册入口

- 新增唯一的 `OrderKindDefinition` 注册表，首期只挂载 `sea-export`。
- 定义至少包含稳定路由类型、业务枚举、贸易方向、运输方式、标题信息、表单适配器和可选
  详情扩展组件。
- 注册项必须通过 TypeScript 穷尽约束声明；禁止依赖 `as any`、数值字面量或静默默认类型。
- 类型解析支持直接 kind 和订单业务路径输入；未知、空值或未注册类型返回 `undefined`。
- 页面收到未注册类型后明确展示 404，不得落入 Sea/Air/Land 任一默认实现。

### R2：删除旧配置真相与兼容桥

- 将列表、新建、详情、费用页、页面头部、列表查询、列表资源 Hook、创建资源 Hook、详情数据
  Hook 及相应测试全部迁移到新注册入口。
- 迁移完成后删除 `common.ts` 中的 `OrderKind`、`OrderKindConfig`、`ORDER_KIND_CONFIGS` 和
  `parseOrderKind`；不得保留桥接函数、旧别名或双写配置。
- 清理 `OrderListTemplate` 内与产品注册表冲突的本地 `OrderKind` 联合、`kindMap` 和调用方
  `as any`。公共列表模板只消费页面提供的稳定显示结果，不维护订单类型业务知识。

### R3：SE 表单适配器

- SE 定义负责选择现有海运 Sections、生成新建默认值、构造创建请求、将订单聚合映射为详情
  表单值、构造更新请求。
- 新建默认值上下文必须显式提供现有默认逻辑需要的创建人、服务类型候选项和货类候选项。
- SE 创建与更新转换不得再接收通用 `config` 后判断 `isSea`，也不得通过字段存在性猜测业务
  类型；调用 SE 适配器本身就是类型证据。
- 可以复用或移动既有纯函数，但不得复制同一套字段转换；每个转换方向只有一个实现来源。
- 本任务不得改字段名、校验规则、日期转换、空值处理、人员装配、海运主分单或箱量请求语义。

### R4：页面容器按定义编排

- `new.tsx` 只负责路由、权限、主数据状态、模板生命周期与提交结果；Sections、默认值和创建
  Payload 由当前定义的表单适配器提供。
- `detail.tsx` 保留订单身份、数据加载、锁单、保存、显式刷新令牌、通用费用/异常/放货、
  状态流转、草稿动作和错误呈现；Sections、初始值和更新 Payload 由定义提供。
- 列表页、费用页和通用 Hook 读取同一个注册定义中的稳定元数据，不再读取旧配置对象。
- `OrderFormTemplate` 的 props、草稿身份算法、dirty、layout-effect 恢复、`resetTo` 和提交清理
  不得因本任务改变。

### R5：详情专属扩展边界

- 将海运改配动作、共享航次、跨订单共享箱、同批订单、拆票/改配历史及其本地状态和请求
  生命周期移入 SE 详情扩展模块。
- 详情扩展必须以正常 React 组件形式持有 Hook 和本地状态；禁止从注册字典动态调用 Hook，
  也禁止把所有海运 `useState`、ref 和 setter 反向塞入一个巨型 context。
- 通用详情头提供稳定的“类型专属操作”插槽；SE 的拆票与改配按钮通过该插槽保持现有位置、
  顺序、颜色、禁用原因和权限语义。
- 扩展可向通用详情布局贡献类型专属头部操作、更多菜单项、后置 Sections 和覆盖层；通用页面
  不再直接 import `SeaOrderReassignmentModal`、`SeaTransportExecutionUpdateModal`、
  `SeaOrderChangeHistoryDrawer`、`SeaSharedContainerDrawer` 或海运 change service。
- 扩展只能通过页面提供的明确刷新命令刷新订单快照与锁状态；不得读写模板草稿、dirty 或
  显式刷新令牌。

### R6：权限与类型能力不重复建模

- 不在前端注册表维护与权限 Manifest 同义的 `OrderCapability` 集合。
- 操作是否对某业务类型存在、当前用户是否可用，继续由 `access.canOrder(businessType,
  operation)` 和后端 `allowedActions` 决定。
- 类型扩展组件的存在只表达 UI/流程实现位置，不得用来绕过权限或锁单判定。

### R7：规范与测试同步

- 更新前端组件或状态管理规范，记录“注册表是订单类型元数据与适配入口的唯一真相；权限
  Manifest 是操作能力唯一真相；OrderFormTemplate 是表单生命周期所有者”。
- 测试必须覆盖注册解析、未知类型 fail-closed、所有运行时调用方迁移、SE 新建/详情转换、
  详情扩展行为和已验收的草稿/显式刷新竞态矩阵。

## 4. 验收标准

- [ ] **AC-1**：唯一注册表只注册 `sea-export`，直接 kind 与合法订单路径均解析到同一对象；
  空值、未知类型、SI/AE/AI/LAND/RAIL 均返回 `undefined`。
- [ ] **AC-2**：列表、新建、详情和费用页遇到未注册类型均呈现明确 404，不发起目标订单或
  类型主数据请求，不选择任一默认模板。
- [ ] **AC-3**：产品代码中不存在旧 `ORDER_KIND_CONFIGS`、`OrderKindConfig`、
  `parseOrderKind`、兼容桥、重复 `OrderKind` 联合、本地订单类型 `kindMap` 或相关 `as any`。
- [ ] **AC-4**：新建页只通过 SE 表单适配器取得 Sections、默认值和创建 Payload；创建请求与
  重构前在字段、默认值、人员、海运单证和箱量上行为等价。
- [ ] **AC-5**：详情页只通过 SE 表单适配器取得 Sections、初始值和更新 Payload；更新请求与
  重构前行为等价，且不再通过字段存在性推断运输方式。
- [ ] **AC-6**：通用详情页不直接依赖列出的 Sea 覆盖层或 sea change service；SE 扩展独立
  持有其请求与本地状态，并保持拆票、改配、共享航次、共享箱、同批订单和历史入口行为。
- [ ] **AC-7**：拆票/改配按钮的显示、顺序、颜色、禁用原因、权限、锁单保护与路由不变；
  类型扩展的迟到请求不能污染切换后的订单身份。
- [ ] **AC-8**：订单列表标题、业务类型显示、筛选、主数据加载、费用页权限与详情导航继续
  使用 SE 元数据，且公共列表模板不再维护业务类型名称表。
- [ ] **AC-9**：前端未新增与权限 Manifest 重复的 capability 表；SE 专属操作仍以
  `access.canOrder`、`allowedActions` 和锁单策略共同控制。
- [ ] **AC-10**：`OrderFormTemplate` 生命周期契约和详情统一刷新令牌实现无改动；现有同订单
  乱序、A/B 和 ABA 测试保持通过。
- [ ] **AC-11**：相关定向 Vitest、修改文件 Biome、`pnpm --dir web tsc`、
  `git diff --check` 通过；任务最终执行一次 `pnpm run check:web` 并通过。
- [ ] **AC-12**：实现、测试与项目规范描述同一所有权边界，不存在未记录的兼容分支或提前
  注册未实现订单类型。

## 5. 不包含范围

- 不实现或开放 SI、AE、AI、LAND、RAIL 的菜单、路由、表单、接口或数据模型；
- 不修改 Proto、OpenAPI、数据库 Schema、生成客户端或权限 Manifest；
- 不修改服务端 `normalizeOrder` 或提前建立只有一个 SE 元素的 validator map；
- 不把空运现有占位模板改造成可用业务；
- 不设计未来品类的具体字段、必填规则、单证流程或状态机；
- 不修改 `OrderFormTemplate` 的生命周期内部实现；
- 不保留旧订单配置 API、历史类型别名或静默 fallback；
- 不引入 IoC 容器、反射、动态模块加载、微前端或运行时插件发现；
- 不因文件移动顺手重写现有 SE 业务规则。

## 6. 风险与约束

- `detail.tsx` 当前同时承载通用编排与 SE 专属交互，抽离时最容易丢失按钮顺序、弹窗状态、
  change-actions 竞态门禁和刷新链路，必须用行为测试而非仅快照验证。
- 注册定义不得演变为包含所有页面状态的“上帝对象”；纯表单转换、类型元数据和有状态详情
  扩展必须保持独立文件与契约。
- 公共 `OrderListTemplate` 的现有类型表包含未开放类型，迁移时只能删除这份重复真相，不能
  借机开放菜单或改变列表列结构。
- SE 转换函数重命名或移动时，必须用重构前后的固定输入/输出夹具证明请求等价，不能只断言
  新函数被调用。
