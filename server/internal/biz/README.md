# 业务领域层

声明纯 Go 领域对象、用例、仓储接口、领域错误与业务规则。通过仓储接口调用持久化，不依赖 service/data，不在领域字段中暴露 Protobuf 消息或 Ent 类型。

完整约束见[服务端协作规范](../../AGENTS.md)与[后端分层规范](../../../.trellis/spec/server/backend/directory-structure.md)；业务入口先查[能力导航](../../../.trellis/spec/server/backend/capability-navigation.md)。
