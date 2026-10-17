# 执行计划

- [x] 用户授权创建任务、修复体检 bug、回归 main 和安全删分支。
- [x] 干净 main 基线核查，创建 codex/system-audit-bugfix。
- [x] PRD、设计、执行计划收敛，准备子代理上下文清单。
- [ ] 呈现最终方案，取得后续实施确认，再 task.py start。
- [ ] Trellis 实现代理分别负责写入反馈、缓存隔离、财务导出；主会话负责 CI/真实库。
- [ ] 实现后补回归，写调用审计和数据库 PASS/FAIL/SKIP 记录。
- [ ] 各批次定向测试、Biome、类型检查及 diff 检查后提交。
- [ ] Trellis 检查代理独立复核，修复发现，更新规范。
- [ ] 浏览器失败反馈、身份切换和 CSV 样本复核。
- [ ] 最终 check:fast 与完整 PostgreSQL data/migration 验证。
- [ ] 逐项完成验收、提交、任务归档和 journal。
- [ ] 干净状态切 main 快进合并，核对祖先关系，git branch -d 安全删除本次分支。
- [ ] 复核 main 最新提交、分支消失、工作树干净，完成目标。

## 验证命令与风险

开发中只跑受影响 vitest 文件（路径相对 web，确认匹配数量）、修改文件 Biome、pnpm --dir web tsc、git diff --check。真实库完整 data/migration 包分别执行，禁止使用开发连接。浏览器请求拦截，禁止将模拟重置密码/财务写入开发库。

最终 pnpm run check:fast；Provider 涉生产入口时增加 pnpm run build。一次性资源清理后查残留，凭据和原始 trace 不入库。公共模板六消费者必须覆盖，工作区隔离不得破坏钉钉预热/callback；财务 index.tsx 写入代理和导出代理按分支/props 协调。
