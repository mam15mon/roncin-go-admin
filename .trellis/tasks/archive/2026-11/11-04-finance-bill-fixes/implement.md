# 实施计划

- [x] 阅读 PRD、设计与 jsonl 指向规范。
- [x] 实现本组需求；不得修改其他组所有文件。
- [x] 实现后补触发场景、相邻正常路径、失败零写入/并发回归。
- [x] 定向 go test（数据库场景使用隔离 PostgreSQL 与正式迁移，不接受 SKIP）；前端涉及时 Vitest/Biome/tsc。
- [x] 独立 trellis-check 复审与修正；git diff --check 后分组中文提交。

父任务负责全量集成门禁与规范同步。本组不得独立运行漏洞扫描。
