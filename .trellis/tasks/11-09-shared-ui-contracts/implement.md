# 执行与验证计划

## 启动条件
PRD/design/implement 和研究已形成最终方案。用户已确认底栏条件切换及筛选清空规则；用户已在后续消息明确批准最终方案并要求开始修复、独立 review、合并 main、安全删分支和 push；2026-11-09 已启动。

## 可验证交付包与依赖
1. 财务契约：六列表读取 matchedSummary 的 Proto/biz/data/service 与生成物，保持业务写入边界；先实现再补后端精确筛选/聚合测试。前端正式依赖本包生成类型。
2. 财务展示：公共采纳快照、六类领域投影、单行底栏、跨页选择及筛选清空。依赖包1，保持官方 request 引擎。
3. 导航与改配：可与财务并行，归属 routes/routeUtils/标题测试与改配/资格组件测试，避免同时编辑财务公共模板。
4. 主会话统一更新规范、执行最后独立 trellis-check、提交、归档与日志。沿用本会话已授权的独立 review 后合并 main、安全删除任务分支和普通 push 流程；不强推，不动其他未提交任务目录。

采用一个 Trellis 任务、按上述交付包提交和分工，避免新增无依赖价值的多任务树。多代理提示必须以 Active task 开头，明确文件职责并告知共享工作区；implement/check 子代理不再嵌套同角色。

## 定向验证
- 后端：六列表完全相同筛选命中的 count/amount，跨页不变、不同原币/本币不混加、组织范围、草稿/失效行、零值/负数/小数精度、发票 null与显式零本币、对冲状态筛选。按现有测试基建使用隔离正式迁移库，不重置真实开发库，不并行 Schema.Create。
- 前端：无选择显示 matchedSummary；第一页选2、第二页选3合计5；表头本页全选、跨页取消至零；翻页/页大小/排序保留；已提交筛选/公司切换清空；未提交搜索草稿保留选择；过滤失败后不恢复旧条件勾选。
- 六实体真实字段投影，非费用无利润、未折算提示、双币种/双本币；金额展示统一 formatAmount，金额求和使用现有 decimal 工具或 Decimal.js。
- 用真实框架 deferred A/B 重放竞态，表格/顶部/底栏/导出保持 B；失败/取消不覆盖，首次失败不显示成功零值；不写无意义 sleep。
- 标题回归：两处遗漏路由、SE/AE差异、动态事件迟到、redirect、隐藏路由、无布局与未知 kind；不改变标签身份归组。
- 改配回归：省略/显式 false、errors 缺省、未加载/失败、true缺版本、旧预览迟到、动作资格撤回/加载失败、写成功同步期。
- 单个重型 suite 不超过10例且不超过20秒，必要时按功能拆文件。

## 门禁与交付
每包运行修改文件 Biome、相关定向 Vitest/Go、git diff --check；按类型影响运行 tsc。契约生成按 make -C server api → pnpm run generate:web-client，不手修生成物。全部范围完成且独立 review 无阻塞后执行 pnpm run check:fast；不固定跑 build。
若现有集成环境无法验证，记录具体缺口并在不改真实数据的前提下补安全隔离验证。最终只暂存本任务文件，确认远端 main 进度，普通合并/安全删除分支/推送，不覆盖其他工作。

## 隔离工作区
原checkout有19处并行分页修改及新文件（含FinanceLedgerTemplate），本任务使用 /home/vince/.codex/worktrees/shared-ui-contracts/roncin-go-admin，分支 codex/shared-ui-contracts，基于用户批准规划提交20f14205。原checkout保留main与原未提交分页改动；任务状态改动已仅移到本worktree。合并时必须保护FinanceLedgerTemplate同文件的分页改动，不带入其他并行工作。
