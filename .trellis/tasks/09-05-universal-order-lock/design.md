# 全业务类型订单锁定与解锁技术设计

## 设计目标与边界

把现有 SE 专属订单业务锁改造成六种订单共用的一套能力：统一状态、统一锁定/解锁 API、统一
写门禁、按业务类型隔离的权限和同一套钉钉 OA 解锁审批。锁定的本质是冻结订单业务事实及
订单费用，而不是阻断读取或已经具备自身状态机的下游财务处理。

本任务只扩展锁能力，不实现 SI、AE、AI、LAND、RAIL 的完整页面、创建流程或专属单证模型。
SE 继续形成 MBL/HBL 不可变版本；其他类型不创建无法由现有领域模型证明的快照。

## 总体架构

```text
Order.business_type
  │
  ├─ HTTP 鉴权：business.order.<type>.<operation>
  ├─ 允许动作：read / update / lock + 数据范围 + 当前锁状态
  ├─ 锁定事务
  │    ├─ 通用：Order 状态、代次、版本、OrderLockRecord、审计
  │    └─ 仅 SE：MBL/HBL 不可变版本与快照引用
  ├─ 解锁事务
  │    ├─ bootstrap admin → ADMIN_EMERGENCY
  │    ├─ 对应类型 lock 业务角色 → ROLE_DIRECT
  │    └─ 对应类型 update 普通编辑人 → DINGTALK_APPROVAL
  └─ 所有业务写入口：Order FOR UPDATE → ensureOrderBusinessEditable
```

API 路径保持不变，业务类型始终从目标订单读取，不允许客户端提交另一个类型来选择权限或
快照分支。这样可避免“请求说 SE、数据库是 AI”之类的越权或数据错配。

## 业务类型与权限解析

### 权限目录

在 `internal/access` 增加 LAND、RAIL，并用一份穷举的业务类型集合驱动 `Valid`、代码、中文名、
Manifest 和“任一订单权限”判断。`OrderLock` 的适用类型由仅 SE 改为六种，最终生成：

- `business.order.se.lock`
- `business.order.si.lock`
- `business.order.ae.lock`
- `business.order.ai.lock`
- `business.order.land.lock`
- `business.order.rail.lock`

每个 lock 权限只依赖同类型的 read 和 update，不产生跨类型依赖。现有 SE 权限键和角色关联
原样保留；新增权限通过迁移命令的 Manifest 同步进入权限表，只有 administrator 按既有规则
补挂缺失权限，普通业务角色必须由管理员显式分配对应类型的 lock 权限。

LAND/RAIL 同时进入现有订单权限业务类型字典，使 HTTP 路由可以解析其 read、update 和 lock。
这只登记既有通用操作权限，不改变 `OrderUsecase.Create` 等尚未支持这些业务的领域规则，也不
新增路由或页面。

### 后端资格解析

data 层不再保存 SE 权限字符串。目标订单加锁读取后，把其 Ent 业务类型转换为
`access.OrderBusinessType`，验证属于六种合法值，再通过 `access.OrderPermission(type, operation)`
取得 lock/update 权限键。角色资格与候选查询都接收解析后的 lock 权限键：

1. User、Membership、Role、RoleAssignment 启用；
2. Membership/Role 属于目标订单组织，数据范围覆盖目标订单；
3. Role 不是 `administrator` 且显式持有目标类型 lock 权限；
4. User 不是 bootstrap admin。

HTTP 中间件补齐 LAND/RAIL 的 API/Biz 映射和任一类型遍历。中间件先按目标订单类型检查权限，
仓储事务再复验角色事实，形成传输层与数据层双重防护。只持有 SE lock 的用户不能锁定、直接
解锁或批准 AI 订单。

## 持久化模型与迁移

### Order

继续复用现有字段：

- `locked_at`、`locked_by`：当前是否锁定及实际锁定人；
- `lock_generation`：每次成功锁定加一，解锁不改变；
- `version`：成功锁定和成功解锁都加一。

不增加按类型拆分的锁字段，不自动锁定历史订单。

### OrderLockRecord

增加不可变、必填 `business_type` 快照；把 `master_bill_id` 和
`master_bill_version_id` 改为可空。增加与 Ent Schema 同名同表达式的数据库 CHECK：

```text
business_type = 'SE'
  → master_bill_id 与 master_bill_version_id 均非空
business_type IN ('SI', 'AE', 'AI', 'LAND', 'RAIL')
  → 两个 SE 专属引用均为空
```

该 CHECK 同时排除非法业务类型和一半为空的歧义记录。SE 的 HBL 快照仍由现有子表表达；非 SE
必须为零条 HBL 快照，应用层创建路径和 PostgreSQL 集成测试共同保证这一点。

### OrderUnlockRequest

增加不可变、必填 `business_type` 快照。它用于历史 API、OA 表单和批准生效时的资格复验，
避免审批过程中依赖页面传值。创建请求时必须与 Order 和当代 OrderLockRecord 的业务类型相同。

### 正式迁移顺序

1. 为锁记录和解锁请求增加可空 `business_type`；
2. 锁记录从关联 Order 回填，解锁请求从关联锁记录回填；现有历史应全部得到 `SE`；
3. 对回填结果做显式断言，发现空值或非法类型立即让迁移失败；
4. 把两列设为 `NOT NULL`，不保留会掩盖调用方遗漏的数据库默认值；
5. 解除两个 SE 引用的 `NOT NULL`，增加稳定命名的 CHECK；
6. Ent Schema、生成的 migrate metadata 和正式 SQL 保持同源。

现有外键、`(order_id, generation)`、组织级幂等键和活动解锁请求部分唯一索引不变。

## API 与领域契约

在 `order_lock.proto` 引用订单业务类型枚举，并做向后兼容的字段追加：

- `OrderLockStateData.business_type`；
- `OrderLockRecordData.business_type`；
- `OrderUnlockRequestData.business_type`；
- `OrderLockRecordData.master_bill_id/master_bill_version_id` 改为 optional。

字段编号只追加或保留原编号，不复用旧编号。对 SE 响应，两个 MBL 字段继续存在且值不变；
非 SE 响应不返回这些可选字段，HBL 快照为空。service 只做 Biz ↔ Proto 映射，不推导权限或
快照规则。

Biz 的 `OrderLockState`、`OrderLockRecord`、`OrderUnlockRequest` 增加 `OrderBusinessType`；锁记录
的两个 MBL ID 改为指针。现有错误 reason 保持稳定，只把 SE 专属中文错误消息改成通用订单
业务锁角色表述。

## 锁定事务

`LockOrder` 保持统一 `Data.WithTx`，执行顺序如下：

1. `FOR UPDATE` 锁定目标 Order，验证组织归属并取得权威业务类型；
2. 解析同类型 lock 权限并复验调用人是有效业务角色成员；
3. 在持有 Order 锁后查询组织级幂等记录，同键同指纹返回原事实，异指纹返回稳定 409；
4. 验证业务类型合法、Termination 为 ACTIVE、Closure 为 OPEN、当前未锁且版本匹配；
5. 若为 SE，调用现有单证快照分支，保持 Order → MBL → Link → HBL 的固定锁序，创建或复用
   MBL/HBL 不可变版本；若为其他类型，跳过全部海运实体查询；
6. 设置锁定人/时间，`lock_generation + 1`，`version + 1`；
7. 创建携带业务类型的 OrderLockRecord；SE 写 MBL/HBL 引用，非 SE 写空引用和零 HBL 快照；
8. 写审计，details 至少包含业务类型、订单、代次和版本；SE 继续附带单证版本信息；
9. 事务提交后用普通上下文重读状态和记录。

为避免在一个超长方法中复制两套流程，现有 SE MBL/HBL 快照逻辑下沉为只在 SE 分支调用的
私有原语，返回可空 MBL 引用和 HBL 快照集合；通用 Order 状态、记录、幂等和审计只实现一次。

`GetOrderLockState` 仅对 SE 查询活动海运提单并要求其存在；其他五类不触碰海运表。所有类型
都要求 ACTIVE、OPEN 和对应 lock 角色才可锁。返回允许动作时：

- 未锁定：只有对应类型 lock 业务角色可锁；
- 已锁定：bootstrap admin 可紧急直解；对应类型 lock 业务角色可直接解锁；否则只有具备对应
  类型 update 权限且覆盖订单范围的用户可申请审批；
- 活动审批存在时普通编辑人不能重复创建，但直解路径仍可取代旧申请。

## 解锁与钉钉审批

### 三路分流

`RequestOrderUnlock` 在锁住 Order 和当代 OrderLockRecord 后校验二者业务类型一致，再按固定
顺序执行：

1. bootstrap admin → `ADMIN_EMERGENCY`；
2. 持有目标类型 lock 权限的有效非管理员业务角色 → `ROLE_DIRECT`；
3. 持有目标类型 update 权限的普通编辑人 → `DINGTALK_APPROVAL`；
4. 其他用户拒绝。

前两条在同一事务内创建已批准请求、清空当前锁、递增 Order version、关闭 LockRecord、取代
旧活动审批并写审计。第三条按目标类型 lock 权限查询全部候选，保存候选快照、请求和 outbox。
没有候选或相关人员缺少钉钉绑定时仍保存 `CONFIGURATION_FAILED`，不缩小候选集、不跨类型
借用审批人，也不加入 bootstrap admin。

### 共用 OA 模板

六种类型继续读取同一个 `Security.DingTalk.approval_process_code`。钉钉模板标题调整为通用的
“订单解锁审批”，OA 创建命令增加以下展示上下文：

- 业务类型中文名及代码；
- 操作票号；
- 申请人显示名；
- 锁定代次；
- 可选解锁原因。

候选人仍作为同一个 OR 节点，任意一个合格成员批准即可。business type、代次、票号和原因从
不可变 OrderUnlockRequest 读取；申请人显示名在准备派发时从实际 User 读取，不接受浏览器
传值。外部网络调用继续发生在事务外，process code 和候选 DingTalk UserID 使用 outbox 快照。

批准 Inbox 生效时除现有实例、状态、代次、版本、候选快照校验外，还要验证 Order、LockRecord、
UnlockRequest 的业务类型一致，并按该类型 lock 权限实时复验审批人。现有 SE 活动审批经迁移
回填后仍解析到 `business.order.se.lock`，无需重建实例或改写候选。

部署前必须先在现有 OA 模板加入上述通用字段；模板仍使用同一 process code。已创建的审批
实例保存自己的历史表单，不受模板字段更新影响。

## 统一业务写门禁

`ensureOrderBusinessEditable` 删除 SE 条件：任何合法业务类型只要 `locked_at != nil` 就返回
`ORDER_BUSINESS_LOCKED`，metadata 保持订单 ID/编号、代次、时间和锁定人。所有调用必须位于
事务内，并在真实业务写入前持有目标 Order 的 `FOR UPDATE` 锁。

现有调用点因此自动覆盖六种类型。额外修复通用装运单证仓储：新增、修改、状态流转和删除
均在同一 `WithTx` 内先锁 Order、校验组织/非 SE 适用范围、执行门禁，再锁或创建子实体；删除
事务外“先查订单再写”的竞态窗口。

订单费用 Add、Update、Transition、Remove 继续通过 `lockOrderForFeeMutation` 先锁 Order 并执行
业务锁，再执行提成、账单等已有财务门禁。费用标签也继续锁定全部涉及订单并按 UUID 排序。
锁单只阻止费用业务事实变化；读取费用和用已确认费用创建账单不走业务写门禁，保持可用。

实现结束后用搜索审计所有直接写 `Order` 及其业务子实体的仓储入口，确认没有新增或遗漏的
绕过路径；财务账单、发票、收付、核销、提成实体不被误接入订单业务锁。

## 前端复用与失败关闭

在订单领域内抽取可复用的锁状态 hook 和锁控件，不建立全局第二套权限表：

- hook 只调用生成的 `OrderLockService` 客户端，暴露 state/loading/error/refresh；
- 控件从服务端 `business_type` 和 `can_*` 字段渲染锁摘要、锁定、三路解锁和历史抽屉；
- 通用文案按生成枚举映射业务类型，SE 锁定确认额外提示会固定 MBL/HBL 版本，其他类型只提示
  冻结业务资料和费用；
- 不根据前端 `access.canOrder` 重算角色资格或审批候选。

现有订单详情对任何已配置的订单类型都加载锁状态；未成功加载或已锁定时，表单、状态流转、
POD、异常、拆票/改配和费用写入口失败关闭。锁定/解锁成功后同时刷新订单数据与锁状态，确保
新的 Order version 用于下一次命令。

费用详情页复用同一 hook：锁状态未加载或已锁定时仍展示费用列表、汇总和账单创建，但禁用
新增、编辑、确认、撤回、作废和标签维护，并展示业务锁原因。快速费用抽屉接收相同的只读
状态，不能仅依赖“详情页入口按钮被禁用”。服务端仍是最终门禁，前端禁用只改善交互。

本任务不扩展 `ORDER_KIND_CONFIGS`，不新增五类路由、表单或导航入口。

## 兼容、风险与回滚

- 现有 SE 锁记录和请求通过权威关联回填业务类型；不修改锁代次、时间、操作人、候选或外部
  实例 ID。
- Proto 只追加业务类型，并把原字符串字段改为 wire-compatible optional；SE JSON 仍返回原
  MBL 字段，非 SE 才省略。
- 新权限不会自动赋给普通角色。发版后必须由管理员按职责为各业务类型角色显式配置 lock；
  未配置时允许动作会清晰返回“未分配对应业务类型锁定角色”。
- OA 模板字段必须先更新再部署；若模板未更新，钉钉会明确派发失败，系统不得改走直接解锁。
- 迁移回滚会重新要求 MBL 字段 NOT NULL，因此数据库一旦产生非 SE 锁记录就不能直接降级；
  回滚应用前必须先阻止新锁定，并确认不存在非 SE 锁记录。不得删除历史记录来强行降级。
- 业务锁与写入竞争依赖同一 Order 行锁序；不得增加自动重试来掩盖死锁或版本冲突。

## 任务拆分决策

不创建独立子任务。Schema、Proto、权限、锁事务、OA 请求和当前页面共享同一个业务类型契约，
拆成可独立归档的子任务会产生不可运行的中间状态。实施仍按三个可验证提交分组，每组完成
局部生成和测试后再提交。
