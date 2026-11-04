# 执行计划

- [x] 四项审查与修复建议获得用户实施及合并授权；创建隔离 worktree 和 codex/sea-business-fixes 分支。
- [x] 编写并复核 PRD/设计/上下文清单。
- [ ] 实现 R1：服务端候选关联正文门禁、前端载荷、相邻回归测试。
- [ ] 实现 R2/R3/R4：共享箱事务门禁、改配阻断及同母单新航次、回归测试。
- [ ] 运行定向 Go / Vitest，修改文件 Biome 与必要 tsc；测试数据库使用隔离实例和正式迁移。
- [ ] 分组本地提交（Conventional Commits 中文说明）；独立 trellis-check review 并修复遗漏。
- [ ] 更新现有海运与审核锁规范；运行 pnpm run check:fast。
- [ ] 归档任务、记录 journal，确认 main 未漂移后合并；保留用户已有任务目录。

## 所有权
实施 A 负责 R1 的 order_write_sea_document.go / 创建相关 biz 守卫与前端 form-adapter 及对应测试。
实施 B 负责 sea_order_change*.go 与 sea_cargo_allocation.go 及其测试（R2-R4）。
协调者负责任务、规范、测试环境、整体验证、提交与合并。两个实施者不得改对方文件，新增冲突先协调。

## 命令
- go -C server test ./internal/biz ./internal/data ./internal/service -run '<受影响用例>' -count=1
- pnpm --dir web exec vitest run <受影响测试>
- pnpm --dir web exec biome check <修改前端文件>
- pnpm --dir web tsc
- git diff --check
- pnpm run check:fast

## 风险
共享箱固定锁序和成员重验最重要；前端只省略字段不能替代服务端保护。未跑的检查不得写为通过。主仓库已有 main 比 origin/main 领先 14 个提交，不进行未经要求的远端推送。
