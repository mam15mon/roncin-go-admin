# 实施与验证计划

## 启动门禁
用户已明确批准“开始实施，独立review 完了合并回主 push”。2026-11-09 完成上下文校验并执行task.py start；交付时沿用该合并与推送授权，无需重复询问。

## 工作区与协作
分支codex/finance-current-workspace基于497d3209，复用已附着的隔离worktree；原checkout /home/vince/roncin-go-admin 的未跟踪海运布局任务保留。修改前git status，不覆盖并行工作。通过原生trellis-implement/trellis-check代理分工，所有代理显式workdir、不再派生代理、不提交；主会话协调生成、文档、检查和提交。

## 有序交付
1. 后端契约包：一个owner负责settlement.proto、公司候选纵向删除、全部后端stub、源生成及web生成物。先改源再生成，其他前端owner不写services/roncin与enums.generated。保留purpose4/9，reserved其余值；无数据库操作。
2. 前端列表包：一个owner负责fees/cashflows/invoices/nettings/verifications五列表及cashflows内登记弹窗、费用台账建账入口/纯过滤与相关测试；不修改InvoiceCreateModal和VerificationWorkbench（下一包owner）。按需要与后端生成协调。
3. 前端工作台包：一个owner负责features/finance/bill-creation、InvoiceCreateModal、VerificationWorkbench、订单费用页建账入口与相关测试；同步父props边界要与列表owner沟通，避免共同重写父页面。
4. 提成包：当前三owner收尾释放slot后由一个owner负责整个commissions模块列表/导出/方案/生成/名单/月度申请与调整及测试，避免同目录跨人修改。
5. 主会话同步AGENTS/spec，组合定向结果后派独立trellis-check审查全基线到最终工作树；发现修复后复验，完整门禁通过提交产品/生成物/规范同一组变更，单独归档与日志。

## 功能后定向验证
- 六列表/方案与提成子视图：当前公司请求、旧恢复外公司不覆盖、重置/翻页/排序仍同公司；缺身份/非company不请求；无公司候选调用，展示列和导出公司名称保留。
- 所有创建弹窗：无需选择公司就能查询正常候选，预览/提交正确；注入外公司Form值不能覆盖；无身份/无权限不提交，失败保留输入。开票/核销只验证契约允许的参数及实体归属，不写不存在的organizationId。
- 固定费用/选中账单/资金/方案来源缺失或外公司拒绝，不自动改挂；跨页选择与两种建账模式保留。
- 提成候选真实可操作，防已删除Form依赖导致永久disabled；方案员工/名单/复制当前公司一致，月度申请/调整不跨公司。
- 全局公司切换取消/失败保留输入，成功销毁旧工作区和弹窗；关闭重开/来源变化/预览迟到不污染新会话。复用OrganizationWorkspace真实测试，不新建第二套状态机。
- 后端：真实HTTP/gRPC注册与反射验证旧公司候选缺席、结算单位候选保留；用途4/9不变、reserved有效，0/退役用途拒绝；scope与资金/核销/建账/开票现有相邻用例保留。

## 命令与最终门禁
先实现后测试（禁止TDD）。定向pnpm --dir web exec vitest run <相对web路径>，改动文件Biome、tsc；重型suite超过10深层用例或20秒按功能拆分。Ant Design组件改动前按已加载ant-design技能查询本地CLI API，改后antd lint --format json；不顺手处理无关弃用。

后端受影响包go test/vet、buf lint、生成重跑幂等、git diff --check；全任务一次稳定完整pnpm run check:fast，跨层契约删除是完整门禁依据。无构建入口/依赖变更不固定build。无Schema变更不新建迁移或操作数据库；若需要实际数据库验证另用隔离库，SKIP不能当实际通过。

## 交付记录
记录实际执行/未执行的验证，不能把研究推断当PASS。独立review无阻塞后按中文Conventional Commits提交。任务归档、journal及后续合并/推送按用户已有授权的适用范围与最终指示执行，安全删除仅已合并分支，不强推或强删未合并工作。
