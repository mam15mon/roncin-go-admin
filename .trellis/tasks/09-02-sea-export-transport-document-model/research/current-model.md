# 当前海运主分单模型调查

## 仓库证据

- `server/internal/data/ent/schema/order.go`：`Order` 是现有操作票聚合，包含客户、承运人、起运港、卸货港、最终目的地、船名航次、ETD/ETA、货物、箱、费用、账单和提成关系，并已有 `version`。
- `server/internal/data/ent/schema/order_consolidation.go`：当前共享对象仅按“组织 + 业务类型 + 规范化主单号”唯一，没有签发主体、航程、版本或独立成员关系。
- `server/internal/data/ent/schema/order_shipping_document.go`：当前记录同时强制关联订单、共享主单和非空 HBL，无法表达只有 MBL 的 DIRECT 业务。
- `server/internal/data/order_sync.go`：`resolveOrderConsolidation` 按号码自动复用共享记录，并可在普通订单保存中更新共享主单属性，存在静默归组和跨票覆盖风险。
- `web/src/pages/orders/order-plan-fields.tsx` 与 `components/docs/OrderMasterDocGroupCard.tsx`：当前表单允许一票录入多组 MBL，并把 MBL/HBL 组合为重复表单组。
- `web/src/pages/orders/templates/components/sea/SeaBasicInfoSection.tsx`：已有承运人选择，但承运人不等同于实际签发 MBL 的船公司或上游 NVOCC。
- `web/src/pages/orders/common.ts`：合作伙伴选择已支持按角色服务端检索，可用于签发主体候选，不应退回自由文本或一次性全量加载。

## 阶段 1 受影响链路

```text
海运订单页面
  → CreateOrder/UpdateOrder 契约
  → service DTO 转换
  → biz 主单匹配、关联和门禁
  → data 统一共享事务
  → Order + 运输执行 + MBL + 当前成员关系
  → 查询返回共享主单摘要
  → 页面展示/确认已有 MBL
```

## 风险

- 仅在前端限制一票一 MBL，其他 API 入口仍可破坏不变量。
- 先查询后创建而无数据库唯一约束，并发时会产生重复 MBL。
- 事务内先共享锁后升级写锁，可能形成锁升级死锁；首次需要修改的行必须直接 `FOR UPDATE`。
- 关联动作若顺带用订单输入更新共享 MBL，会重现当前跨票覆盖问题。
- `carrier_id` 不能直接重命名为主单签发方；承运人、订舱代理和 MBL 签发主体可能不同。
- 当前为开发阶段且用户确认没有历史业务数据；实施前仍须检查相关表行数。若事实变化，停止破坏性迁移并返回规划，不增加静默兼容分支。
