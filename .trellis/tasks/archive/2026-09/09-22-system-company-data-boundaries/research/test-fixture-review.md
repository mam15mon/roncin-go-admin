# 测试夹具审查与数据库隔离

- 公司夹具改为独立根节点，部门/团队仍位于公司内；服务端权限测试同时验证跨公司拒绝与进入本公司后的成功路径。
- 检出旧 `TestOrderCreateTransactionPostgres`、`TestOrderIdempotencyPostgres` 直接消费集成连接并设置 `AutoMigrate: true`。本次较宽的测试名称匹配曾命中两例，导致向开发 public 尝试 Ent Schema.Create；两例均在 NewData 初始化阶段以 organizations_base_currency_by_kind CHECK 违反失败，尚未进入业务写入。
- 核对 Ent Atlas 实现：DDL 在事务内执行，错误分支调用 Rollback，成功才 Commit。随后仅以 READ ONLY 查询核对 public：仍为 1 个 headquarters 和 4 个 company，fee_setting_templates 不存在，organizations 无新 CHECK。证据与 DDL 回滚一致。没有执行数据库修复或手工变更。
- 两例已改用 getIntegrationData，正式迁移在随机 Schema 中运行，完成自动删除其隔离 Schema。后续仅以明确测试名执行真实 PostgreSQL 验证。

## 验证结果
- biz、service、server 全包默认测试通过；四层 go vet 通过，git diff --check 通过。
- 真实 PostgreSQL 隔离验证通过：人员候选及越权、汇率套算、订单费用汇率快照、订单提成摘要隐私、订单幂等、订单创建事务/共享主单、公司默认币种。
- 单证读取、公司种子初始化与共享汇率基线测试亦已在隔离 Schema 执行通过。
- 正式迁移暴露旧订单事务夹具的目的港国家码 D1/D2 非法，现用合法 CN 港口码；人员夹具清理按团队→部门并先清理其成员资格，避免根节点约束和外键冲突。
- 尚存 enterprise_image_deletion_integration_test.go 的旧 AutoMigrate 测试不在本次定向执行范围，已通知主会话；全包 PostgreSQL 门禁必须先确保隔离，不能把开发连接直接注入所有旧测试。
