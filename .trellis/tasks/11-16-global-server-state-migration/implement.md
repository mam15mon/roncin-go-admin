# 实施与验证计划

## 阶段门禁

- [x] 用户同意创建任务，已核查代码、测试和规范。
- [x] PRD 完成最终结构收敛，设计、清单和真实代理上下文齐备。
- [x] 已展示最终摘要，用户明确批准实施，并授权最终 review、合并及安全删除任务分支；已执行 task.py start。

## 协作边界

使用 Trellis implement/check 子代理，派发提示以 `Active task: .trellis/tasks/11-16-global-server-state-migration` 开头。原生注入优先，缺失时子代理自行加载任务与规范；state-management.md 超过单文件注入上限，按 research/context-loading.md 补读，不能仅依赖被截断的注入文本。最多三个实现代理并行，主会话负责接口协调、整合与提交。

明确文件所有权；代理不得覆盖用户或其他代理工作，不独立提交。公共接口稳定后再并行迁移消费方。费用重叠文件由主会话最后整合。

## 第一批：公共协议与缓存（R1、R5）

- [x] 复核实时 git status 和初始清单，保存实施前差异基线。
- [x] 确认 RemoteSearchSelect 资源键和订单/货币查询前缀最小接口。
- [x] 迁移公共下拉和必要 SearchFilterTemplate 参数、两套缓存、公开导出及 clear/forceRefresh 调用方。
- [x] 修复 useMasterDataCrud 及涉及辅助函数的失败响应守卫。
- [x] 验证防抖乱序、父依赖变化、已选名称、卸载、工作区隔离、失效后新数据、undefined/合法空 repeated。
- [x] 定向回归、修改文件 Biome、tsc、架构和 diff 检查后提交。

## 第二批：查询消费方（R2、R4）

共享协议稳定后按三组不重叠文件并行：

- [x] 公共/基础组：PartnerQuickAddSelect、订单列表资源、组织/编号规则、货币/行政区划、企业资源及客商辅助区块。
- [x] 财务组：账单编辑/变动检查、建账分组货币、申请/来源/员工候选、账单/提成/发票详情与策略设置。同步移除父级 refreshToken 驱动。
- [x] 订单组：同批订单、审批、变更历史、动作能力、单证/版本/附件候选、放货关联提单，保留官方表格 request。
- [x] 各组记录清单每项的 key、enabled/轮询、meta、写后失效及测试结果。
- [x] 回归账单刷新确认与本人保存两个窗口、组织选择不重取整树、历史分页、正文/选择不被重查覆盖。
- [x] 主会话审查并分组提交。

## 第三批：汇率及命令（R3、R5）

- [x] 主会话在最新费用改动上整合表单/行内汇率 Query，不恢复旧费用弹窗。
- [x] 订单组：航程预览、海运抬头带入、空运影响预览、单证模式/修订/作废预览。
- [x] 财务组：提成计算、牌价抓取、导入预检 mutation，保留 token/载荷/指纹/幂等语义。
- [x] 覆盖手工与系统汇率切换、缺 ETD/ETA、失败/过期预览、关闭重开、未知结果与新输入保护。
- [x] 实现后更新针对性测试，真实 Query Provider、不 mock Query 本体、不采用 TDD；验证后分组提交。

## 第四批：全局检查与收尾（R6）

- [x] Trellis check 子代理逐条复核 PRD/清单、调用方、失效前缀及独立缓存残留。
- [x] 更新 state-management、hook-guidelines、capability-navigation 中旧缓存/手写查询指引。
- [x] 原 31 + 2 及命令补充清单都有已完成或具源码依据的处置记录。
- [x] 执行完整 `pnpm run check:web`；因跨领域公共组件与全站状态迁移，需完整前端验收，无需固定跑后端全量或 build。
- [x] 可运行浏览器时复核代表性入口，未执行的真实环境验收记录原因。
- [ ] finish-work、提交复核、任务归档与会话记录（代码与门禁已完成，按收尾流程执行）。

## 验证命令

```bash
pnpm --dir web exec vitest run <相对web目录的测试路径>
pnpm --dir web exec biome check <修改文件>
pnpm --dir web tsc
pnpm run check:architecture
git diff --check
```

优先运行已有 RemoteSearchSelect、list-resources、use-fee-exchange-preview、OrderFeeTableTabs、BillEditModal/变动检查、CommissionCreateModal/ApplicationsPanel、客商辅助区块、单证、企业资源、组织与货币回归。实施时列出真实文件，确保匹配用例数大于零。不用 `pnpm test -- <file>` 冒充定向测试。

不得关闭规则、静音警告、删除断言或放宽超时。超过 20 秒的测试按功能拆分，单文件不堆超过 10 个重型渲染用例。

## 提交保护

提交前检查 index，使用明确路径/差异，仅提交本任务内容。原本暂存的 FeeFormModal 删除及其他会话变化不得带入。重叠文件无法安全分离时先协调并保留源差异，不整体 git add。

每批记录中文 Conventional Commit 提交号、验证结果和剩余项，不把仅完成规划报告成完成迁移。
