# 验证记录

- 真实表单与既有方案列表：2文件7项通过，9.17秒。
- TestCommissionRuleToNettingConfirmationPostgres 与 TestCommissionNettingSourcePostgres：专用PostgreSQL隔离Schema，PASS，14.768秒；新用例覆盖无编码客商、业务CreateRule与分配、候选、金额、草稿确认、重复与旧版本拒绝。
- 实际开发环境验收见acceptance.md，金额6.66666667，已确认未发放。
- 首轮check:fast发现新增测试ref的null/undefined类型不符，已修正；重新执行完整check:web与check:server中。
- trellis-check独立review因外部服务额度403未运行，主会话已接手核对，另尝试原生review回退；结果待补录。
