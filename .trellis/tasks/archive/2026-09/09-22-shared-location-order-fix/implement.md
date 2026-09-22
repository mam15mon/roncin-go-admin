# 实施
1. 修改统一地点谓词、海运摘要读取与订单保存地点校验。
2. 实现后补真实PG共享港口列表、匹配、同码覆盖及港口机场资格回归。
3. 定向测试后check:fast；只读核查现有数据，逐项审查并提交归档。
用户已批准上述修复范围并要求完成。

## 实施结果
- 摘要使用共享或本公司港口引用范围；已保存引用不受候选同码遮蔽或停用影响。
- 港口和机场写入复用现有候选范围并校验启用。旧共享 ID 若已被本地同码行遮蔽，保存须重选，不自动改写。
- 开发库只读核查：17 条运输执行均使用共享起运港；34 个地点引用中，缺失或跨公司引用为 0。

## 验证记录
- 真实 PostgreSQL 隔离 Schema：`TestSharedLocationOrderReadPostgres`、`TestSharedLocationOrderValidationPostgres` 通过。
- 相邻真实 PostgreSQL：`TestSeaMasterBillBatchRulesPostgres`、`TestIndustryReferenceBaselineSharingPostgres`、`TestOrderCreateTransactionPostgres` 通过。
- 回归测试显式注入 `RONCIN_INTEGRATION_DATABASE_SOURCE`，并非因环境缺失而跳过。
- 手工审查确认修复仅限 data 层查询条件，无接口、Schema 或前端变更；未修改业务数据。
- `pnpm run check:fast` 全部门禁通过（120.8 秒）；前端 166 个测试文件通过、1 个跳过，937 用例通过、12 个跳过；后端测试、vet、分层和漏洞检查通过。
- `git diff --check` 通过。
