# 实施与验证

- [x] 用户确认最终方案，任务切换 in_progress。
- [x] 派发 trellis-implement；读取 before-dev、后端专项规范，前端变更前读取 roncin-web-stack；实施后补测试（禁止 TDD）。
- [x] 修正终态单证读取与动作集；检查详情操作后状态刷新，仅修改必要路径。
- [x] 添加真实退关路径及相邻异常回归，先运行 Go 定向测试和前端定向 Vitest、修改文件 Biome；类型影响时运行 tsc。
- [x] 派发 trellis-check，复核组织隔离、错误不吞、终态只读、基线保护及改配历史选择。
- [ ] 因涉及后端读取和前端详情生命周期，最终执行 pnpm run check:fast；不因普通修复运行构建。
- [ ] 补充海运单证读取规范，git diff --check，仅提交本任务改动；保留已有未跟踪 10-30-sea-order-page-layout 任务。
- [ ] 记录验证、提交号与限制，归档任务并记录会话。
