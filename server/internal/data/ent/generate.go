package ent

// 单据发号通过行锁串行分配同一规则的序列，避免并发创建订单时产生重复编号。
//go:generate go run -mod=mod entgo.io/ent/cmd/ent generate --feature sql/lock ./schema
