# 集成与最终验收记录

## 范围与保护

用户已批准规划并明确授权最终 review、合并及安全删除分支。任务工作区为 `/home/vince/.codex/worktrees/server-state-migration/roncin-go-admin`，分支 `codex/global-server-state-migration`；最终交付目标为源工作区 `/home/vince/roncin-go-admin` 的 main。

源主工作区原有 AGENTS.md、OrderFeeHeader.tsx、OrderFeeHeader.test.tsx、feeSummary.ts 四个未提交文件持续保留。任务分支相对 main 未修改这些路径。主分支其他会话费用提交 47b9dd45 已通过 e0b714b7 同步入任务分支，不恢复删除的 FeeFormModal。

## 实现与分组提交

- dee1312a：公共 Query 协议、响应守卫及两套旧结果缓存退出，30 文件；公共 10 文件 74 例通过。
- 28ad3458：客商、组织和企业资源，11 文件 37 例通过；地区乱序/失败、组织选择不重取均覆盖。
- 4e5ab166：订单资源和客商快建，7 文件 35 例通过。
- b541b59a：测试 wrapper 在 rerender 时保留同一 QueryClient 与外层 Context。
- f9f87462：表单和行内汇率 Query，32 文件 87 例通过；同币种去重及保存后清旧基线另有回归。
- 91ec95be：财务查询、预检与账单编辑主动变动检查；实施组原定向 48 文件 225 例，审阅新增 7 文件 29 例通过。
- ee491086：公共候选 signal 贯通至真实客户端、货币直接及聚合依赖失效、主数据首载失败及 wire enabled 修复；专项和全部调用方回归通过。
- ad0ade3e：四个旧重渲染回归移除重复 Provider，恢复同一查询和列表上下文；费用 19 例、财务恢复 10 例通过。
- a899e7a0：订单/单证查询和预览 mutation；实施定向 29 文件 104 例、审阅最终 8 文件 37 例全部通过，无单 suite 超 20 秒。

原始 31 + 2 + 7 清单全部处置，见 completion-matrix.md；各组精确修改路径、发现和定向命令见 foundation/finance/order-implementation.md 与 review-core/finance/orders.md。

## 交叉审阅结论

三组 check 审阅均已完成，无遗留待决设计或实现问题。主要补修：币种启停失败不关闭确认、首次失败不伪造空主数据、ProtoJSON 非 optional enabled 省略正确映射、取消信号贯通、写入在途冻结关闭/输入、历史抽屉关闭取消及重开隔离、单证预览绑定实体与版本、避免就地错误与全局重复通知。

`enabled: false` 不移除 Query observer 的契约与真实取消回归已写入状态管理规范。state-management、hook-guidelines 和 capability-navigation 已同步当前 Query/缓存/辅助函数契约。

## 最终完整门禁

在任务 worktree 执行 `pnpm run check:web`，退出码 0。2026-11-16 最终日志 `/tmp/roncin-state-check-web-final.log`：

- 权限键和 Proto 常量一致性、架构、主数据 mapper、重复函数扫描通过。
- Biome lint、TypeScript 通过；Biome 原有 11 warnings + 2 info，不关闭规则。
- Vitest：577 文件通过、1 文件跳过；3004 测试通过、11 测试跳过；总计 578 文件/3015 测试，耗时 252.43 秒。
- 最后订单活动键/silent 调整另经 8 文件 37 例与 TypeScript/Biome 验证通过。
- `git diff --check` 通过。

首轮全量曾有 9 例失败，均已修复后重跑最终门禁；不把首轮结果算成通过。主因是测试 wrapper 更新后旧用例 rerender 重复套 Provider，以及新增 signal/options 的旧精确断言和模拟队列清理，原业务断言保留。

测试输出仍有 SWR focus TimeoutNaNWarning（源 main 基线已复现）、happy-dom 伪元素提示及一次 localhost:3000 ECONNREFUSED 输出；未静音、未放宽超时，最终无失败测试或未处理错误汇总，不声称零噪音。

## 浏览器和验收边界

真实 Chromium 的代表性交互复核通过：组织选择不重取；账单回焦发现变化、保留输入；取消刷新不变、确认刷新接受新快照并清提醒，pageerror=0。API 全部本地合成拦截，未使用真实凭据或写数据库，不表述为真实后端/生产验收。详见 browser-verification.md。

本次仅改前端状态和相关文档，无后端契约、依赖或构建配置变更，按批准计划使用完整前端门禁；未固定运行后端全量或生产 build。

## 收尾

代码与文档均已按组提交，finish-work 已归档任务（5ee3d253）并记录第 215 次会话（bbe74331）；main 已 fast-forward 合并全部提交。合并前后的四个用户未提交文件经 SHA256 和完整 binary diff 比对一致，暂存区保持为空。

`codex/global-server-state-migration` 已确认被 main 完整包含，并使用 `git branch -d` 安全删除。应用 archive_worktree 返回 `The task's primary checkout cannot be removed.`，尝试关联原主目录也被拒绝（仅接受 linked worktree）；遵守应用保护，保留干净的临时工作树并切至 main 对应 detached HEAD，不强制删除当前聊天主检出。独立 Vite 服务已停止。未创建 PR 或推送。
