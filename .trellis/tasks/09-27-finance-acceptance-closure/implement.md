# 总体执行清单

- [x] 用户审阅最终规划后，激活历史失败子任务，按其 implement.md 实施与检查。
- [x] 激活真实验收子任务并行准备脚本和浏览器；最终实跑包含子任务一修复，写入与清理由各自隔离环境边界约束。
- [x] Trellis 实施与检查代理按各子任务负责范围交付代码、真实运行记录和独立审阅；父会话协调、复核和提交。
- [x] 两项交付完成后，在最终代码上核对报告中的 SHA 与测试范围；影响已验收路径的改动已定向重跑。
- [x] 运行 pnpm run check:fast；真实 PostgreSQL data / migration 两包另有显式运行记录，门禁与集成结果分别记录。
- [x] 生成 research/closure-report.md，分别汇总四项历史失败、HTTP 三场景、浏览器闭环和人工验收状态。
- [x] git diff --check；每组可验证改动独立 Conventional Commit 中文提交；所有必需项完成后归档父子任务。

实施与验收结果以 `research/closure-report.md` 为准。
