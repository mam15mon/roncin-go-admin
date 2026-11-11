# 六项修复交付后的续审

2026-11-11，B1–B6 已完成修复、独立 review 与 Git 交付，main 与 origin/main 核对为
`2fb5594c`（含交付补记）。继续只读审计，暂停前端测试，不写业务数据库；父任务未归档。

## 首轮续审范围

- 核对 `partner_shipping_preset.go` 的默认预设清理、Create/Update 启停及默认状态组合，
  并追踪 `biz/partner_shipping_preset.go` 的归一化与独立 Service/RPC 入口。
- 对比 `enterprise_resource.go` 停用资源时清默认逻辑及正式迁移中的唯一默认索引。
- 检查资源行、父档案行与关联行的锁序疑点；不能只根据看到先锁资源再锁父档案就断言死锁，
  必须证明真实交错及需要冲突的锁模式。

## 暂定状态

当前未新增已确认问题编号。独立复核排除了所述默认切换死锁：清默认只更新关联行非外键字段，
不会因外键值未变而额外锁另一资源行，缺少形成循环等待的边。

禁用预设与默认状态组合存在 Create/Update 不一致的静态 API 边界问题：Create 的
`enabled=false,isDefault=true` 会清旧默认并将禁用资源设为默认；Update 也先清旧默认，
但最终以 `isDefault && enabled` 保存本项。主档详情确有 PresetsSection→ShippingPresetSection
的 RPC 消费入口，但该 UI 新建默认 enabled=true 且无启停开关，不能写成正常 UI 直接可触发。
尚未动态复现或证明订单自动带入影响，暂记待证，不混入已经修复的 B1–B6。

## 动态确认结果

用户要求实际确认后，禁用默认组合已在 SHIPPER 路径经真实 Service→Biz→Data→正式迁移 PostgreSQL 复现，创建和更新均清除原有效默认，对照正常。现升格为 B7/P2，详见 [动态证据及复现源码](preset-default-dynamic-confirmation.md)。上文暂定状态保留为此前时点；本轮未修复产品。
