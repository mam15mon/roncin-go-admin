# 执行计划

- [x] 实现登录卡片、顶部切换、图标层级、左侧视觉与独立页脚。
- [x] 同步原有交互测试的控件语义，必要时在实现后补充键盘/选择状态验证；不预写测试。
- [x] 定向运行登录、扫码实例、预载、切换和回调测试；执行修改文件 Biome 与类型检查。
- [x] 主会话真实 Chromium 验收桌面/窄屏/低高度，HTML与动态二维码切换复用及对齐，保存截图与报告。
- [x] 独立 trellis-check 审查并执行一次完整 check:web，发现问题按范围修复。
- [x] 更新扫码规范、记录验收限制，git diff --check 后提交中文 Conventional Commit。
- [x] 仅归档本任务并记录会话。

## 已有批准
用户在上一轮看到五点最终方案后回复“开始吧”，并在任务创建问题中回复“创建任务并继续”。本规划只持久化该既定方案，无新增产品决策，主会话审核后直接启动。

## 验证命令
`pnpm --dir web exec vitest run src/pages/user/login`；`pnpm --dir web tsc`；`pnpm run check:web`；`git diff --check`。
浏览器脚本置于 /tmp，不提交截图、日志、Cookie 或测试凭据。
