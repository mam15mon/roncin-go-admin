# 实施验收记录

- 实施与独立检查由 Trellis 子代理完成；检查修复停用原单位维护及补录规则读取共享锁。
- Go biz/service 定向测试、数量规则 helper 与管理页交互等 Vitest 定向测试、tsc、修改文件 Biome、git diff --check 通过。
- 独立 PostgreSQL 临时库执行正式迁移，数量规则新增/更新、停用原单位维护、迁移新库与旧 DAY=true 升级回放全部通过。
- check:fast 两次受全量高并发下无关 sea-document-visibility 用例时序抖动阻断；该文件单跑 5/5 通过。
- 最终独立 check:server 完整通过，包括 Go 全量测试、vet、分层、Proto lint、漏洞扫描；按仓库既有审计豁免处理 1 项漏洞。
- 前端完整门禁中的 lint/类型/架构/生成校验通过；全量 Vitest 降至 maxWorkers=8 后 179 文件通过、1 跳过，1040 用例通过、12 跳过。
- pnpm run migrate:dev 已执行成功，按开发流程重录同日迁移校验和，并应用增量 DAY 修正；实查 BL/CONT=true、CBM/DAY=false。
- 未重置开发数据；其他会话配置改动不纳入提交。临时测试库使用后删除。
