# 持久化与仓储层

实现 biz 仓储接口，集中处理 Ent 查询、持久化转换与驱动错误映射。客户端经 Data.client(ctx) 取得；事务和锁复用统一封装，跨仓储操作加入 WithinTransaction 提供的上下文。

完整约束见[服务端协作规范](../../AGENTS.md)与[后端分层规范](../../../.trellis/spec/server/backend/directory-structure.md)；业务入口先查[能力导航](../../../.trellis/spec/server/backend/capability-navigation.md)。
