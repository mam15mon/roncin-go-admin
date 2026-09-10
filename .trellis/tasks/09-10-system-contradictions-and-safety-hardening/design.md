# 系统核心业务矛盾与安全加固技术方案

## 1. 总体策略

本方案沿现有 `service → biz → data` 边界修复，不引入新平台层。核心方法是把当前混在一起的门禁和投影拆成可验证的领域概念：

1. 订单“内容写入资格”与“生命周期流转资格”分开。
2. 财务“有效结清金额”成为列表过滤、列表展示和详情展示的共同口径。
3. 发票释放账单成为一个带固定锁序和版本推进的原子事务。
4. 权限判断使用权限 + 数据范围，不使用角色名称作为安全语义。
5. 前端只消费服务端已有安全契约，不通过降低接口权限解决调用方式错误。

## 2. 订单设计

### 2.1 内容门禁

在 `server/internal/data/order_lock.go` 保留一个统一内容门禁，建议命名为 `ensureOrderBusinessContentEditable`，明确其用途。校验顺序如下：

1. 订单类型合法；
2. `termination_status`：
   - `TERMINATING` → 新增稳定冲突错误 `ORDER_TERMINATION_IN_PROGRESS`；
   - `TERMINATED` → 新增稳定冲突错误 `ORDER_TERMINATED`；
3. `closure_status != OPEN` → 新增稳定冲突错误 `ORDER_CLOSED`；
4. `locked_at != nil` → 保留带 metadata 的 `ORDER_BUSINESS_LOCKED`。

该顺序使用户首先看到不可逆程度更高的生命周期原因，也避免已终止且仍带历史锁信息时误提示“先解锁即可编辑”。

`lockOrderAndEnsureBusinessEditable` 同步改名表达内容语义。`order_release_pod.go` 删除重复函数，改为复用统一门禁。实施时用 `rg` 枚举全部调用点并分类：

- 业务内容写入口：继续调用新门禁；
- 生命周期命令：移除调用，改用专属状态校验；
- 纯读取：不得新增门禁。

### 2.2 生命周期事务

`TransitionStatus`：

- Order `FOR UPDATE`；
- 校验版本与 `event.FromStatus`；
- 显式要求 `termination_status = ACTIVE`、`closure_status = OPEN`；
- 业务锁规则保持现状：主流程推进属于业务写入，锁定后不得推进。

`TransitionTermination`：

- Order `FOR UPDATE` 后校验版本、来源状态、closure 仍为 OPEN；
- 不调用内容生命周期门禁；
- `ACTIVE → TERMINATING` 仍要求订单未业务锁定，保持“锁定资料不可发起退关”的现行约束；
- `TERMINATING → ACTIVE/TERMINATED` 与 `TERMINATED → ACTIVE` 不因内容门禁而失败；
- 目标 `TERMINATED` 时执行 2.3 的 Link 结束逻辑；
- Order、Link、生命周期事件、审计同事务提交。

`TransitionClosure`：

- Order `FOR UPDATE` 后校验版本、来源状态；
- 不校验业务锁；
- 关闭时在事务内重验：主流程已 `DOCUMENT_RELEASED` 或终止已完成、无活动异常、无未开账费用；
- 反结案沿现有状态图执行，不受历史业务锁阻断；
- 保持版本递增、事件与审计原子性。

### 2.3 Link 结束与恢复策略

仅当 SE 订单最终流转到 `TERMINATED`：

1. 已先锁 Order；
2. 查询该订单活动 Link，按 Link ID 排序并 `FOR UPDATE`；
3. 若超过一条活动 Link，返回结构冲突，不做静默修复；
4. 更新为 `ENDED`，设置 UTC `ended_at`、稳定中文 `ended_reason = "订单退关"`，`version + 1`；
5. 不修改 MBL/TransportExecution 内容版本。

`TERMINATED → ACTIVE` 只恢复订单状态，不修改历史 Link。后续显式建立新 Link 时仍由数据库活动 Link 唯一索引兜底。

### 2.4 自拼汇总

修改 `server/internal/data/order_query.go:ListConsolidationSummaries`：

- Link 仍以 `status = ACTIVE` 定位当前 MBL 成员；
- 成员 Order 额外要求 `shipment_type = LCL`、`termination_status = ACTIVE`；
- 使用 `WithSeaHouseBills`，仅加载当前/活动身份所需记录，按 house no 稳定排序并去重；
- 删除本查询对 `WithShippingDocuments` 的依赖；
- 件数、毛重、体积、成员数、house numbers 必须基于同一成员集合计算。

## 3. 财务台账与核销候选设计

### 3.1 有效结清金额

在 `settlement.go` 内定义单一语义：

```text
effective_settled_amount(bill) =
  SUM(active verification allocations whose verification is ACTIVE)
  + SUM(active netting allocations whose netting is CONFIRMED)
```

要求：

- `feeLedgerFinancialProgressPredicate` 的 SQL 子查询使用该定义；
- Ent 预加载分别加载有效核销和有效对冲分摊，禁止把两个一对多表直接 join 后聚合，避免笛卡尔乘积；
- `ResolveFeeLedgerFinancialProgress` 的参数/局部命名从 `verifiedAmount` 收敛为 `settledAmount`，状态枚举保持现有 API 契约；
- 反核销或反对冲后的 inactive allocation 不计入；状态非 ACTIVE/CONFIRMED 的父单据也不计入。

### 3.2 列表与详情共用投影

抽出 data 层小型投影辅助函数，将一个费用及其活动账单关系转换为 `FeeLedgerItem` 所需的：

- bill no；
- invoice amount/是否开票；
- effective settled amount；
- financial progress；
- finance locked。

`ListFeeLedger` 与 `GetFeeLedgerOrderDetail` 使用同一加载条件和投影函数。详情仍按 Order 读取费用，但必须加载投影所需的账单、发票关系、两类有效分摊；不从列表接口回调或二次分页。

### 3.3 候选谓词下推

账单：

- `finance_verification.go:ListCreationCandidates` 调用 bill repo 时设置 `OnlyUnsettled: true`；
- 复用现有 `billUnsettledPredicate`，不复制 SQL。

资金流水：

- 在 `biz.FinanceCashflowFilter` 增加仅供仓储调用的 `OnlyUnverified bool`；
- data List 在 Count/Query 共用谓词中加入余额条件：流水金额大于该流水的有效核销 allocation 合计；
- 仅计算 allocation `active = true` 且父 verification `status = ACTIVE`；
- 候选调用设置 `OnlyUnverified: true`；
- Go 层保留金额转换和防御性非正数跳过，但不能依赖它保证候选完整性。

此变更为 Biz/Data 内部过滤器，不需要扩展公共 Proto。

## 4. 发票设计

### 4.1 状态矩阵

| 命令 | DRAFT | ISSUED | CANCELLED | RED_FLUSHED |
| --- | --- | --- | --- | --- |
| Cancel | 允许，语义“取消” | 允许，语义“作废” | 拒绝 | 拒绝 |
| RedFlush | 拒绝 | 允许 | 拒绝 | 拒绝 |

Data 层在 Invoice `FOR UPDATE` 后执行权威状态判断，Biz 层保留输入校验。禁止只靠前端隐藏按钮。

### 4.2 原子释放与锁序

Cancel/RedFlush 使用相同内部事务步骤，避免两套实现漂移：

1. `FinanceInvoice FOR UPDATE`；
2. 版本和状态校验；
3. 查询活动 `FinanceInvoiceBill`，按 ID `FOR UPDATE`；
4. 提取 Bill IDs，去重并按 UUID 升序；
5. `FinanceBill ... ORDER BY id FOR UPDATE`，验证数量、组织与当前状态；
6. 将活动关联置为 inactive；
7. 每张账单 `version + 1`，受影响行数必须与 Bill IDs 数量相等；
8. 更新发票终态与 `version + 1`；
9. 写审计。

锁序在两个发票终态命令中完全一致。账单取消当前先锁 Bill，再只读检查活动 invoice link；并发集成测试用于验证该路径不会出现部分提交或永久互锁。若实施验证发现反向锁等待，则同阶段统一调整相关 FinanceBill 入口的锁序，不留临时重试或超时回退。

### 4.3 前端提示

`web/src/pages/finance/invoices/index.tsx`：

- DRAFT：按钮“取消”，确认说明“取消草稿并释放关联账单”；
- ISSUED：按钮“作废”，危险确认明确说明只有在线下税控/开票系统满足并完成作废条件时才能执行，系统不会代替税务判断；
- ISSUED 的“红冲”保持独立操作；
- 成功提示与审计原因使用对应术语。

## 5. 权限与前端设计

### 5.1 订单锁资格投影

将当前单用户判断和候选列表查询收敛到同一“有效 lock grant”规则：

- 用户 enabled；
- membership enabled；
- role enabled；
- role 持有 `business.order.<type>.lock`；
- role data scope 覆盖目标订单组织：
  - ORGANIZATION：membership organization 等于目标组织；
  - ORGANIZATION_TREE：目标组织等于 membership organization 或为其后代；
  - ALL：覆盖任意组织；
- 不按 role code 排除 `administrator`。

候选结果仍保存实际 `membership_id`、`role_id` 和 DingTalk UserID 快照。一个用户有多个合格 grant 时，按用户去重并用稳定顺序选择可追溯的 grant。回调先校验“审批人存在于请求快照”，再以同一规则复核当前资格。

bootstrap 分支显式处理：

- `GetOrderLockState`/`LockOrder`：视为可锁；
- 解锁：继续优先 `ADMIN_EMERGENCY`；
- 候选查询：不添加 bootstrap；
- 回调：bootstrap 不绕过候选快照。

同时更新 `.trellis/spec/server/backend/order-lock-and-document-version.md`，移除“有效非 administrator 业务角色、非 bootstrap 才可锁”的旧契约，写入上述新边界。

### 5.2 对冲目标组织解析

`server/internal/server/auth.go` 的财务权限表加入：

- `FinanceNettingRead: writable=false`；
- `FinanceNettingCreate: writable=true`；
- `FinanceNettingConfirm: writable=true`；
- `FinanceNettingReverse: writable=true`。

不新增 Cancel 权限，Cancel/Reverse 继续共享 Reverse。测试必须验证权限名、目标组织解析及 read/write 附加访问差异。

### 5.3 用户页调用分流

`UsersPanel`：

- `ListRoles` 仅在 `canReadRoles` 时调用；
- `ListOrganizations` 仅在 `canReadOrganizations` 时调用；
- 当前组织信息从 `initialState.currentUser.currentOrganization` 取得，作为普通组织管理员的唯一组织选项/默认值；
- 只有 `canReadAllUserMemberships`/外部授权权限成立时才渲染或触发跨组织流程。

`UserFormModal`：

- 普通创建/编辑使用父组件提供的当前组织 roles；
- pending external provider 仅在相应 `canAuthorizeWeComUsers`/`canAuthorizeDingTalkUsers`（现为 ALL）时开放，并按目标组织调用 `ListOrganizationRoles`；
- membership modal 保持全局权限要求。

后端 `ListOrganizationRoles` scope 不改，因而本阶段预计无 Proto/生成物变化。

## 6. 数据迁移与兼容性

- 不修改 Schema，无迁移。
- 不回填历史退关订单的活动 Link；上线前若验收环境存在脏数据，只报告明确数量和示例 ID，不自动清理。是否一次性修复数据需用户另行授权。
- 不改变 API 字段和枚举；预计无需生成 OpenAPI 客户端。
- 若实施中发现必须修改 Proto/Manifest，必须暂停该阶段并先更新任务设计与生成清单，不能手改生成物。

## 7. 测试设计

### 7.1 单元/SQL 级

- 订单状态图及新错误映射；内容门禁四类阻断。
- 台账进度算法：核销、对冲、混合、撤销、超额防御。
- bill/cashflow unsettled 谓词进入 Count 和 Query。
- 发票状态矩阵。
- access 六类型与用户页请求条件。
- auth 网关 netting read/write 矩阵。

### 7.2 PostgreSQL 集成

- 每类代表性订单子资源在 TERMINATING/TERMINATED/CLOSED 下写入失败；正常态成功。
- 锁定 DOCUMENT_RELEASED 订单结案；终止完成/取消/恢复；Link 原子结束和不自动恢复。
- 自拼汇总 HBL 与退关过滤。
- 台账列表筛选和详情在有效对冲下完全一致。
- 账单与 cashflow 各超过 200 条已结清数据后的未结清候选仍可见。
- 发票 Cancel/RedFlush 的状态、活动 link、账单 version、审计、回滚和竞争。
- 订单锁组织/树/全局/administrator/bootstrap/撤权回调矩阵。

### 7.3 前端

- 发票两种确认文案与允许动作。
- 组织管理员不请求全组织/跨组织角色；全局管理员仍可执行跨组织流程。

## 8. 风险与控制

- **状态机误封**：生命周期命令禁止调用内容状态门禁，以状态图测试锁定。
- **金额重复累计**：两类 allocation 分开聚合/预加载，不做多表直接 join 聚合。
- **锁序死锁**：发票终态命令共享固定锁序，多 ID 排序，使用真实 PostgreSQL 并发测试。
- **权限放大**：不降低 `ListOrganizationRoles`；管理员资格以权限和组织范围为准，不以名称直接放行。
- **历史关系误恢复**：退关恢复不复活 Link，历史仍可审计。

## 9. 需求追踪矩阵

| 需求 | 设计落点 | 主要验证 |
| --- | --- | --- |
| ORD-01 | 2.1 内容门禁 | 四类状态阻断 + 代表性子资源集成测试 |
| ORD-02 | 2.2 生命周期事务 | 锁后结案、终止完成/取消/恢复 |
| ORD-03 | 2.3 Link 策略 | Link 结束事实、回滚、恢复不复活 |
| ORD-04 | 2.4 自拼汇总 | 活动成员与当前 HBL |
| FIN-01 | 3.1 有效结清金额 | 核销/对冲/混合/撤销与筛选一致 |
| FIN-02 | 3.2 共用投影 | 列表和详情字段逐项一致 |
| FIN-03 | 3.3 谓词下推 | 两类 `> 200` PostgreSQL 数据集 |
| INV-01 | 4.1 状态矩阵、4.3 前端提示 | 所有允许边/非法边和 UI 文案 |
| INV-02 | 4.2 原子释放 | Bill version、回滚和并发 |
| ACC-01 | 5.1 锁资格投影 | 组织/树/全局/admin/bootstrap/撤权 |
| ACC-03 | 5.4 用户页分流 | 组织管理员无越权请求、全局流程回归 |
| ACC-04 | 5.2 对冲目标组织 | read/write 附加组织矩阵 |
