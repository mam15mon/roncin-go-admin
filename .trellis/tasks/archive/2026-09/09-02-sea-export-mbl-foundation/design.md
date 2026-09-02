# 海运出口共享主单基础设计

## 范围

阶段 1 建立可被后续阶段扩展的运输执行、MBL 和当前/历史成员关系，并完成订单创建、候选确认、航程校验、身份更正门禁和页面基础。HBL 完整内容、箱货分配、拆票、改单版本和共享费用明确留给后续任务。

## 数据模型

### `SeaTransportExecution`

- `organization_id`
- `carrier_id`：实际承运主体，可与 MBL 签发主体不同
- `origin_location_id`、`discharge_location_id`、可选 `transit_location_id`
- 分离的 `vessel_name`、`voyage_no`
- 可选 `etd`、`eta`
- `version`

阶段 1 每张 MBL 指向一个当前运输执行，一个运输执行允许多张 MBL。后续多航段扩展属于同一运输执行边界，不把航程字段复制回 Order/MBL。

### `SeaMasterBill`

- `organization_id`
- `issuer_partner_id`
- `transport_execution_id`
- `master_no`、`normalized_master_no`
- `status=DRAFT`
- `version`

唯一约束：`organization_id + issuer_partner_id + normalized_master_no`。阶段 1 的规范化结果为实时大写后的合法原值；不删除任何字符。

### `SeaMasterBillOrderLink`

- `organization_id`
- `master_bill_id`
- `order_id`
- `status=ACTIVE|ENDED`
- `started_at`、可选 `ended_at`、`ended_reason`
- `version`

数据库部分唯一约束保证每个 `order_id` 最多一条 `ACTIVE`。活动 SE 订单创建事务结束时必须存在一条 `ACTIVE`；历史关系只追加终止信息。

### 现有模型处理

- 移除 `OrderConsolidation` 作为号码归组真相；不保留双写。
- 当前 `OrderShippingDocument` 中 HBL 相关能力不得继续拥有 MBL 归组职责。阶段 1 只保留后续阶段需要的真实 HBL 身份入口或在同次迁移中替换为明确 HBL 实体；不得为 DIRECT 创建记录。
- `Order` 的最终目的地继续归本票；共享主航程从运输执行读取。旧的重复航程字段不作为共享真相，是否物理删除以生成和现有消费者迁移的最小完整闭环为准，但禁止继续双向写入。

## API 契约

- 新增共享主单候选查询，输入签发主体、MBL 和本票航程，返回是否命中、主单 ID、版本、摘要、成员数和逐字段冲突。
- `CreateOrderRequest`/`UpdateOrderRequest` 使用明确的海运 MBL 选择对象：新建身份，或携带候选 ID、预期版本的确认关联；禁止用一个布尔值同时表示两种意图。
- 订单响应返回当前 MBL 摘要和运输执行摘要，旧消费者不再从第一张 HBL 推断 MBL。
- 静态候选路由声明在任何 `/{id}` 动态路由之前。

## 命令流程

### 新建 MBL

1. service 解析 UUID 和 DTO，不写业务规则。
2. biz 校验 SE 门禁、MBL 格式和明确意图。
3. `WithinTransaction` 中创建 Order，锁定/校验主体与位置引用，创建运输执行、MBL 和 ACTIVE 成员。
4. 唯一冲突映射为“发现已有主单，请确认关联”，整个事务回滚。
5. 提交后普通上下文重读完整响应。

### 确认关联已有 MBL

1. 请求携带 MBL ID 和 `expected_version`。
2. 事务中以固定顺序锁定 Order（新建时为待创建上下文）、MBL 和运输执行；首次写读直接 `FOR UPDATE`。
3. 重验组织、签发主体、规范化号码、版本和航程。
4. 只创建 ACTIVE 成员，不更新 MBL/运输执行。
5. 任一冲突原样返回并回滚。

### 单票身份更正

只有 DRAFT、单成员、无受影响下游事实时允许；锁定 MBL 和成员，校验预期版本、原因和新唯一键。若命中已有 MBL，返回需要走确认关联的领域错误，不合并。

## 页面

- 主单号在输入/粘贴时仅执行 ASCII 小写转大写。
- 其他非法字符保留在输入框并显示校验错误，不静默删除。
- 主单签发方通过服务端合作伙伴关键字搜索选择。
- 候选摘要列出签发方、MBL、起运港、卸货港、船名、航次、ETD/ETA 和成员数；冲突逐项显示，存在冲突时不能确认。
- 最终目的地保持本票字段，候选摘要不得把它伪装成共享卸货港。

## 错误与并发

- 格式/必填错误：400 中文领域错误。
- 候选待确认：409，携带稳定 reason 和候选摘要所需 ID；不把完整敏感内容塞入错误文本。
- 版本或候选变化：409“已被更新，请刷新后重试”。
- 航程冲突：409，结构化冲突列表由正常候选响应表达，最终命令仍以领域错误拒绝。
- 唯一索引和 ACTIVE 部分唯一索引作为最后防线；不得以先查后写代替约束。

## 迁移

正式迁移在变更前检查旧主分单相关业务表是否为空。由于用户确认无历史数据，迁移不回填、不双写、不读取旧格式；非空时迁移或实施前检查必须停止并报告。

## 回滚

Ent Schema、SQL 迁移、Proto、生成物、后端实现和前端表单作为阶段 1 原子提交回滚。数据库回滚只适用于尚未承载业务数据的开发环境。
