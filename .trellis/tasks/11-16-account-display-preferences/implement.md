# 账号展示偏好契约与服务端持久化实施计划

## 启动条件

无实施前置；字段契约以父任务 design.md 为准，完成生成和验证后交付给前端持久化子任务。

须先取得对最新父任务完整方案的审阅批准，再 task.py start 本子任务。

## 步骤

1. 确认最新父任务方案和已有修改；只负责后端契约、biz/service/data/注册/生成及对应迁移，生成客户端由同一所有者执行。
2. 修改 Proto 与 Ent 源，完成用例和仓储实现，再补边界及隔离测试，禁止 TDD。
3. 执行 make -C server api、受影响的 go generate 及 pnpm run generate:web-client，审阅正式 SQL 迁移。
4. 独立 PostgreSQL 应用有序正式迁移，验证并发、隔离、恢复及前置身份；定向 go test/vet、git diff --check 后提交。
5. 交付契约与生成物给下一子任务，父任务最后执行全量 check:fast。

## 完成门

验收结果与未验证限制写入本子任务 research/，所有权和生成物清晰，独立 trellis-check 无阻塞后交付父任务。父最终门禁见[集成计划](../11-16-global-filter-settings/implement.md)。
