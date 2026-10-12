# 技术设计

前端遵循现有请求客户端可能消费错误并返回 undefined 的契约，动作显式验证成功实体后才能通知、关闭或刷新；保留意见和幂等动作信息。补录所有成功状态流转统一使审核查询失效；将已批准基础与当前可新建补录区分，使撤回重提入口不被待审状态隐藏。

领域快照计算统一 RoundBank(8)。对冲计提预锁订单集合按实际 RECEIVABLE 分配选取，保持 source_changed 检测语义。审批意见按 1000 Unicode 字符贯通领域、Ent 验证与表单；从 Schema 重新生成代码，确认是否影响 SQL 结构。

并发修复将授权父行置于订单锁之前，审查 User → Membership → Role → Order 顺序，并调整月度申请先 User 后 Membership。复核发现直接计提创建/确认亦先锁 Order 再 User，需将既有员工锁提前至来源加载之前，一并消除实际反序。必须检查补录与重提等调用者，保留事务内实时授权及组织范围判定；不得仅局部交换 User/Membership 而留下 Order 反序。

复核补充两条当前业务路径：确认及月申不得先锁员工归属子集后追加来源全集，应先合并完整来源订单按ID一次预锁；方案分配插入会取得 User 外键 KEY SHARE，提成员工快照锁采用 NO KEY UPDATE，避免与 Membership 反序，同时保留与账号修改及审核 SHARE 的互斥。补录决策列历史 SQL 为 varchar(500)，同步批准/退回 Unicode 校验及 Ent Schema，并通过正式 Ent/Atlas Diff 生成扩容迁移。

影响文件位于 web 的费用审核/补录/提成发放组件、server biz/order_fee_review、data/order_fee_review、补录授权、月度申请、对冲计提和审核事件 Schema。必要生成物、定向回归与规范同组交付。
