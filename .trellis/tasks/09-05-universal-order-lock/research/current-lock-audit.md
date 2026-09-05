# 当前订单锁实现审计

## 结论

现有数据库已经在统一 `orders` 表保存锁状态，HTTP API 也按订单 ID 动态解析业务类型，因此
无需为六种业务类型复制接口或锁表。扩展工作的关键是把“业务类型 → 权限”的解析统一化、
把 SE 单证快照做成锁定事务中的条件分支，并让所有订单业务写入口共同执行同一个事务内门禁。

## 已确认事实

- `server/api/order/v1/order.proto:88` 定义 SE、SI、AE、AI、LAND、RAIL 六种业务类型。
- `server/internal/data/ent/schema/order.go:41` 的 Ent 枚举和 `orders` 表已接受六种类型；同一实体
  已有 `locked_at`、`locked_by`、`lock_generation` 和 `version`。
- `server/internal/access/manifest.go:117-123` 只声明四种权限业务类型；
  `server/internal/access/manifest.go:287` 只为 SE 生成 lock 权限；
  `server/internal/server/auth.go:120-190` 的任一类型检查和 API/Biz 映射也只覆盖四种。
- `server/internal/data/order_lock.go:46-48` 的统一业务写门禁仅在 SE 时生效。
- `server/internal/data/order_lock.go:120-190` 的角色资格和候选查询固定使用
  `business.order.se.lock`。
- `server/internal/data/order_lock.go:457-481` 的允许动作、活动海运提单要求和普通编辑权限均为
  SE 专用逻辑；非 SE 会被明确阻断。
- `server/internal/data/order_lock.go:535` 的锁定事务拒绝非 SE，后续逻辑无条件读取 MBL、Link、
  HBL 并创建不可变版本。
- `server/internal/data/ent/schema/order_lock_record.go:27-29` 强制每条锁记录都有 MBL 与 MBL
  版本，当前模型无法表达非 SE 锁记录。
- `server/api/order/v1/order_lock.proto:126-217` 的锁状态和历史 DTO 没有业务类型，且 MBL 引用
  是必填字符串，不能准确表达非 SE 记录。
- `server/internal/data/dingtalk_approval_gateway.go:116-128` 当前 OA 表单只发送操作票号和可选
  解锁原因；`server/internal/data/dingtalk_approval_inbox.go:252-257` 在批准生效时仍按 SE 权限
  复验审批人。

## 写门禁覆盖审计

已经调用 `ensureOrderBusinessEditable` 的共享写路径包括：

- 订单草稿更新、状态/终止/结案流转；
- 订单标签和费用标签；
- 里程碑、附件、人员、货物、集装箱、放货 POD、异常；
- 订单费用新增、修改、确认/撤回、作废；
- SE 单证、箱货分配、拆票与改配。

`server/internal/data/order_shipping_document.go:45-183` 的非 SE 通用装运单证新增、修改、状态
流转和删除只在事务外读取订单并排除 SE，没有在事务内先 `FOR UPDATE` 锁订单，也没有执行
业务锁门禁。这是当前审计发现的明确绕过入口，实施时必须补齐并增加并发回归。

## 前端现状

- `web/src/pages/orders/common.ts:205-220` 当前只正式声明 `sea-export` 页面配置，本任务不扩展
  其他五类完整页面。
- `web/src/pages/orders/detail.tsx:121-187` 只为 SE/sea 加载锁状态，并仅对 SE 做失败关闭。
- `web/src/pages/orders/components/detail/OrderDetailHeader.tsx:116-127` 和锁定/解锁弹窗文案、按钮
  显示均硬编码 SE。
- `web/src/pages/orders/fees.tsx` 未读取业务锁状态，仍只按财务锁控制费用编辑；服务端虽然会
  阻断 SE 锁单费用写，但页面不能及时呈现只读状态。
- `web/src/pages/orders/order-fee-panel.tsx` 同样只按财务锁和权限控制写按钮；详情页入口被禁用
  不能替代组件自身的失败关闭。

## 设计约束

- 权限仍以 `server/internal/access/manifest.go` 为唯一真相，前端键必须生成。
- LAND/RAIL 若不加入权限业务类型字典，HTTP 中间件无法对其订单 ID 解析 read/update/lock，
  因此至少必须登记现有通用订单权限及新的 lock 权限；这不等于开放页面或支持创建命令。
- 非 SE 没有可证明的专属不可变单证模型，只允许写空的 SE 快照引用，不能创建虚拟快照。
- 订单业务锁不得代替费用、账单、发票、核销或提成门禁。锁单后费用编辑禁止，但使用既有
  已确认费用生成账单仍由财务规则决定并保持可用。
- 当前 OA 创建接口使用同一个 process code。共用模板时必须先把模板调整为通用标题，并支持
  业务类型、操作票号、申请人、锁定代次和可选解锁原因字段，再部署发送新字段的代码。
