package ent

// 单据发号通过行锁串行分配同一规则的序列，避免并发创建订单时产生重复编号；
// 注册审批通知按确定性任务幂等键 On Conflict Do Nothing 入队，重复确认注册
// 不产生重复通知任务。
//go:generate go run -mod=mod entgo.io/ent/cmd/ent generate --feature sql/lock,sql/upsert ./schema
