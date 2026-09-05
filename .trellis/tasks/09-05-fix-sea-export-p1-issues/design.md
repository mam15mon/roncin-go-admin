# 海运出口四项 P1 修复技术设计

## 设计目标与边界

本任务修复四条已经证实的故障链，不改变已批准的人工业务流程：

1. SE 列表单证入口进入真实海运单证页面；
2. ReleasePod 可以关联真实 Sea MBL/HBL，并在海运单证页面展示；
3. 箱和货物删除发送真实乐观锁版本；
4. 并发拆票失败方稳定返回 409 版本冲突。

新增的关联删除行为只适用于现有 HBL 硬删除命令。MBL 没有硬删除命令；单证作废继续保留放货记录。只要某个 HBL 的关联放货记录中存在 `RETURNED`（已回单），服务端就阻止删除整个 HBL 及全部关联记录。

本任务不新增外部提交、流程节点、单证状态机、审批、共享费用分摊、文件库或独立审计功能。代码继续使用现有操作日志存储；实现中的既有 `writeAudit` 函数名只是当前日志写入机制，不代表扩展产品范围。

## 当前故障链与最小修复面

### SE 列表旧入口

`web/src/pages/orders/list.tsx` 对所有业务类型统一传入 `onOpenDocuments`，公共 `OrderListTemplate` 固定显示“主分单据管理”，因此 SE 也打开 `ShippingDocumentDrawer`。该抽屉调用旧 `OrderShippingDocument` 接口，而 SE 的真实 MBL/HBL 已由 `SeaDocumentSection` 管理。

修复只增加公共列表动作的可配置文案，并由订单列表调用方决定行为：SE 显示“海运单证”并跳转订单详情；其他业务类型仍打开原抽屉。

### ReleasePod 旧关联

`OrderReleasePod.shipping_document_id` 是指向 `OrderShippingDocument` 的可空外键。前端面板只查询旧分单，数据层也只验证旧分单是否属于订单。需要在保留非 SE 行为的同时增加真实 Sea MBL/HBL 引用。

### 箱货删除版本

两个 Remove Proto 已包含必填 `expected_version`，后端也已执行非零和冲突校验；只有 `ContainerDrawer`、`CargoItemDrawer` 的删除调用遗漏该字段。更新调用已经正确发送版本，不改动。

### 并发拆票错误优先级

`SeaOrderChangeUsecase.ExecuteSplit` 在进入事务前调用 `PreviewSplit`。当竞争胜方已经提交，失败方的 Preview 会根据新的 HBL/分配状态先产生 400，导致事务内 `OrderVersion` 检查没有机会返回 409。

仓储 `ExecuteSplit` 已在固定锁序下完整重验 Order、MBL、Link、HBL、货物、箱、分配、费用和守恒。因此执行命令不需要把无锁 Preview 当成提交前门禁；Preview 继续保留为用户主动调用的只读预览接口。

## 数据模型与数据库约束

### OrderReleasePod 新字段

在 `OrderReleasePod` Ent Schema 增加两个可空真实外键：

- `sea_master_bill_id -> sea_master_bills.id`
- `sea_house_bill_id -> sea_house_bills.id`

保留现有 `shipping_document_id` 服务非 SE。新增稳定 CHECK：

```sql
num_nonnulls(shipping_document_id, sea_master_bill_id, sea_house_bill_id) <= 1
```

同时为两个新外键增加索引。Sea MBL/HBL 侧声明对应 `release_pods` 反向边，并将删除策略设为 `NO ACTION`：应用必须先按已批准规则处理放货记录，数据库不得静默把历史关联置空。

正式增量迁移只增加可空列、CHECK、索引和外键，不重写存量数据：

- 非 SE 的旧关联继续保留；
- 未知的历史 SE 旧关联不自动猜测成 MBL/HBL；
- 新写入按订单业务类型严格使用正确引用。

Ent Schema、生成的迁移元数据和正式 SQL 迁移必须使用相同约束语义。

## API 与领域契约

### ReleasePod 引用

`order_release_pod.proto` 引用现有 `SeaDocumentType`，为响应、新增请求和更新请求增加：

```protobuf
SeaDocumentType sea_document_type = <new_number>;
optional string sea_document_id = <new_number>;
```

现有 `shipping_document_id` 不改字段号。API 组合规则：

| 订单类型 | 合法请求 |
| --- | --- |
| SE，无关联 | 三个引用字段均空，`sea_document_type=UNSPECIFIED` |
| SE，关联 MBL | `sea_document_type=MASTER_BILL` 且 `sea_document_id` 非空 |
| SE，关联 HBL | `sea_document_type=HOUSE_BILL` 且 `sea_document_id` 非空 |
| 非 SE，无关联 | 所有引用为空 |
| 非 SE，关联旧分单 | 仅 `shipping_document_id` 非空 |

以下情况返回 400 `ORDER_RELEASE_POD_DOCUMENT_INVALID`：

- 海运类型和 ID 只出现一个；
- 同时提交旧分单和海运单证；
- SE 提交旧分单；
- 非 SE 提交海运单证；
- MBL/HBL 不属于当前组织和订单的当前活动单证关系。

领域对象使用已有 `biz.SeaDocumentType` 与一个可空 `SeaDocumentID` 表达 API 引用，持久化层再映射到两个真实外键，避免把 Ent 类型泄漏到 biz。

### HBL 关联删除

`RemoveSeaHouseBillRequest` 增加：

```protobuf
optional bool remove_related_release_pods = <new_number>;
```

服务端在现有 `Order -> MBL -> Active Link -> HBL` 固定锁序之后，按 UUID 升序锁定关联 `OrderReleasePod`：

1. 任一记录为 `RETURNED`：返回 409 `SEA_HOUSE_BILL_RETURNED_RELEASE_POD_BLOCKED`，零变更；
2. 存在 `PENDING/SIGNED` 且请求未确认关联删除：返回 400 `SEA_HOUSE_BILL_RELEASE_POD_CONFIRMATION_REQUIRED`，零变更；
3. 全部为 `PENDING/SIGNED` 且请求明确确认：先删除这些放货记录，再删除 HBL，并在同一事务写操作日志；
4. 无关联记录：沿用现有 HBL 删除规则。

删除最后一张 HBL 时，`return_to_undetermined` 与 `remove_related_release_pods` 合并在同一次确认请求中，不串联两个互相独立的删除事务。

若请求确认关联删除，service 额外检查调用人具有当前 SE 订单的 `release_pod.delete` 组织范围权限；没有权限返回现有 403。普通无关联 HBL 删除不增加该权限要求。

操作日志沿用 `sea_house_bill.remove`，附加被关联删除的 ReleasePod 数量和 ID，不新增审计页面或审批流。

## ReleasePod 数据事务与固定锁序

重构 `server/internal/data/order_release_pod.go` 中触碰路径，使所有读取通过 `Data.client(ctx)`，写入统一在事务内完成：

```text
Order
→ 当前 SeaMasterBill（仅 SE 引用时）
→ Active Link
→ SeaHouseBill（仅 HBL 引用时）
→ OrderReleasePod（更新时）
→ 操作日志
```

新增/更新先锁定并校验 Order 的组织和可编辑状态，再验证引用：

- MBL 必须是该订单唯一 ACTIVE Link 指向的同组织 MBL；
- HBL 必须同组织、同订单，并且 `master_bill_id` 等于 ACTIVE Link 的 MBL；
- 非 SE 旧分单必须属于同一订单；
- 删除 HBL 与新增/更新 ReleasePod 都先锁 Order，因此关联创建和关联删除不会发生穿透竞态。

查询列表在订单组织校验后通过事务感知客户端读取，不直接使用 `r.data.db`。

## 前端交互设计

### 列表入口

`OrderListTemplate` 增加可选 `documentsActionLabel`，默认仍为“主分单据管理”。`orders/list.tsx`：

- SE：传“海运单证”，点击跳转 `/orders/<order-kind>/<id>`；不挂旧抽屉调用；
- 非 SE：保持默认文案和 `ShippingDocumentDrawer.open`。

测试同时断言 SE 跳转和非 SE 抽屉回归。

### ReleasePod 面板

页面内部提取一份共享的“放货记录单证引用”转换工具，避免面板和海运单证区各自拼接类型：

- SE 调用 `GetSeaOrderDocuments`，构造 `MBL: <master_no>` 和 `HBL: <house_no>` 单选项；
- 非 SE 调用 `ListShippingDocuments`，保持旧分单选项；
- 表单内部使用显式类型和 ID，不把带前缀的显示字符串直接当后端契约；
- 关联候选加载失败时清空旧候选、显示错误并阻止保存错误引用；
- 编辑和列表回显依据 API 的显式字段展示，不猜测 UUID 类型。

### 海运单证页面展示

`SeaDocumentSection` 在具备 `release_pod.read` 权限时同时加载本订单 ReleasePod，并按 `sea_document_type + sea_document_id` 分组：

- MBL 区展示其关联记录；
- 每张 HBL 区展示自己的关联记录；
- 每条显示放货编号、回单编号和状态；
- 无记录显示“暂无关联放货记录”；
- 无读取权限时不请求接口、不泄漏数据；请求失败显示局部错误，不把失败伪装成空列表。

HBL 删除交互：

- 本地发现 `RETURNED`：列出阻断记录，按钮不执行删除；
- 仅有 `PENDING/SIGNED`：合并展示 HBL、关联放货记录以及最后一张 HBL 回到未确定的影响，用户确认后只发一个删除请求；
- 用户取消：不调用删除 API；
- 服务端返回并发阻断/冲突：刷新海运单证与放货记录，不自动重试删除。

前端只展示服务端权限允许的动作。有关联记录但没有 `release_pod.delete` 权限时，提示联系有权限人员，不发起关联删除。

## 箱货删除版本设计

删除调用从记录读取 `version`：

```typescript
expectedVersion: record.version
```

在调用生成客户端之前使用现有版本动作工具或同等的严格守卫确认版本存在且大于零。不得使用 `record.version ?? '1'` 猜测。两个组件测试分别覆盖：

- 真实版本原样传递；
- 缺失/零版本不调用 API并提示刷新；
- 409 交给统一错误处理，不做本地重试。

## 并发拆票设计

`ExecuteSplit` 保留以下事务前逻辑：

- UUID、幂等键、请求指纹和必填 expected versions 校验；
- `validateSplitTargetsAndResults` 的纯输入结构校验；
- 同幂等键重放查询。

移除执行路径中事务前的 `uc.PreviewSplit(...)` 业务状态门禁。执行直接进入 `WithinTransaction`，由 `repo.ExecuteSplit` 先 `FOR UPDATE` 锁定 Order 并比较 `OrderVersion`，再按既有固定锁序重验全部可变事实和守恒。

独立 `PreviewSeaOrderSplit` API 不变，仍为用户提交前展示业务错误的只读功能。Execute 不信任客户端曾经 Preview，也不把 Preview 结果当提交令牌。

错误保持分层：

| 条件 | 结果 |
| --- | --- |
| 请求结构本身非法 | 400 `SEA_ORDER_SPLIT_INVALID_ARGUMENT` |
| 数量守恒失败 | 400 `SEA_ORDER_SPLIT_CONSERVATION_FAILED` |
| 当前业务状态真实阻断且版本未过期 | 400 `SEA_ORDER_SPLIT_BLOCKED` |
| 任一预期版本已过期 | 409 `SEA_ORDER_SPLIT_VERSION_CONFLICT` |
| 同版本并发竞争失败方 | 409 `SEA_ORDER_SPLIT_VERSION_CONFLICT` |

不增加自动重试、事务重试或把所有错误统一改成 409。

## 生成与迁移

契约变更顺序：

1. 修改 `order_release_pod.proto`、`sea_document.proto`；
2. 修改 Ent Schema 及相关反向边；
3. 运行服务端 API/Ent 生成；
4. 添加并审阅正式 SQL 增量迁移；
5. 运行 `pnpm run generate:web-client` 与错误 reason/Proto 常量生成；
6. 实现 service、biz、data 与 Web；
7. 重跑生成并确认幂等。

生成物不得手工修改。迁移在独立临时 PostgreSQL Schema 中从完整迁移链执行，并断言新列、CHECK、FK 的 `NO ACTION`、索引及迁移记录。

## 验证策略

### 后端

- Biz：ReleasePod 引用组合、HBL 关联删除参数、已有拆票静态输入错误不漂移。
- Service：SeaDocumentType/UUID 转换、关联删除额外权限、错误 reason。
- Data 单元/真实 PostgreSQL：同组织关系、跨订单/跨组织拒绝、共享 MBL 成员允许、非当前 HBL 拒绝、RETURNED 阻断、PENDING/SIGNED 原子关联删除、日志失败全回滚。
- 并发：完整 `TestSeaOrderSplitAndReassignment_PostgresIntegration -count=3`，确认每次一成功一 409；保留静态非法和守恒测试。
- Migration：空 Schema 完整迁移及 Ent 生成元数据断言。

### 前端

- 列表：SE 文案/跳转、非 SE 抽屉。
- ReleasePod：SE MBL/HBL 候选、非 SE 旧候选、加载失败、表单映射和回显。
- SeaDocumentSection：按 MBL/HBL 分组展示、无权限、请求失败、RETURNED 阻断、可删除记录确认与取消。
- Container/Cargo：真实版本和缺失版本守卫。
- `pnpm --dir web test`、`pnpm --dir web tsc`，按需要运行 lint/biome/build。

### 全量门槛

- `go -C server test ./...`
- `go -C server vet ./...`
- `pnpm --dir web test`
- `pnpm --dir web tsc`
- 相关真实 PostgreSQL 集成测试
- API、Ent、Web Client 生成幂等检查

## 风险与回滚

- ReleasePod 变更跨 Proto、Schema、迁移、后端和前端，必须作为完整契约批次交付，不能只上线页面或只上线新列。
- HBL 关联删除是破坏性操作，服务端必须忽略前端缓存判断并在锁内重新检查 `RETURNED`。
- 并发拆票修复不能削弱仓储锁内重验；如果测试发现仓储未覆盖某个 Preview 规则，应先把该规则补到锁内，再移除执行前 Preview。
- 执行前再次检查工作区并行改动。若计划文件与来源不明的脏文件重叠，停止对应实现组并报告，不覆盖、不重置。
- 回滚以完整功能提交为单位；正式迁移已应用后不得删除或改写历史迁移，应追加反向迁移并保留存量数据。
