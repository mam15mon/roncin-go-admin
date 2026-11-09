# 传输适配层

把 Protobuf 请求转换为 biz 输入，执行边界参数校验、调用用例并转换响应。业务规则留在 biz，数据库与 Ent 查询留在 data，不在本层跨越边界直连持久化。

完整约束见[服务端协作规范](../../AGENTS.md)与[后端分层规范](../../../.trellis/spec/server/backend/directory-structure.md)；业务入口先查[能力导航](../../../.trellis/spec/server/backend/capability-navigation.md)。
