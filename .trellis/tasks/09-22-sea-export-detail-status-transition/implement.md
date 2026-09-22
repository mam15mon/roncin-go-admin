# 海运出口详情页直接流转状态实施计划

## 1. 实施步骤

- [x] 扩展 `OrderStatusSection` 的输入契约，用服务端 `allowedTargetFlowStatuses` 标记合法目标，
      以 Ant Design `Button` 提供鼠标和键盘可操作节点，并保留只读/禁用提示。
- [x] 将退关徽标扩展为 `未退关 / 退关中 / 已退关` 三态动作入口，将结案徽标扩展为
      `未完结 / 已完结` 两态动作入口；单动作直接确认，退关中用下拉菜单选择完成或取消。
- [x] 在 `order-detail-transitions.tsx` 增加固定目标的主流程确认函数：展示当前/目标状态、
      可选原因，调用生成客户端并让异步 `onOk` 接管提交 loading；三个确认函数均防止空响应
      被误报为成功。
- [x] 在 `detail.tsx` 编排动作权限、当前工作台与锁状态门禁，防止重复确认框；流转成功后并行
      刷新详情和锁状态，不触碰详情表单草稿。
- [x] 将生命周期提交门禁按服务端契约拆分：主流程/发起退关检查业务锁；完成/取消退关、恢复、
      完结和反结案不检查业务锁，只复核工作台和允许动作，修复现有页头的前端误拦截。
- [x] 将完结与反结案原因统一为必填，对齐现有 Proto/Biz 契约，并补充空原因不发请求测试。
- [x] 从 `OrderDetailHeader` 移除退关与结案按钮及对应 props，费用、锁单、保存、异常和业务类型
      专属动作保持不变。
- [x] 从 `OrderListTemplate` 类型和菜单中移除 `onTransitionStatus`，从 `list.tsx` 移除旧弹窗
      挂载，并删除仅供列表使用的 `TransitionModal.tsx`。
- [x] 新增小型定向测试文件，覆盖单一目标、已配舱双目标、退关三态、结案两态、只读/禁用、
      固定目标提交、版本参数、单次请求、成功刷新与列表/页头旧入口清理；按组件和页面拆分，
      单文件不超过 10 个重型页面用例。
- [x] 调整服务端 `orderAllowedActions`：所有 `ACTIVE + OPEN + 未锁定` 订单均投影 `EDIT`，再由
      现有 Principal 权限与组织范围裁剪；`CLOSED`、`TERMINATING`、`TERMINATED` 和已锁定订单
      保持不可编辑。
- [x] 删除 `UpdateDraft` 仓储中仅允许 `DRAFT` 的重复状态门禁，继续复用统一内容写门禁并保留
      `ForUpdate`、`expectedVersion`、业务类型和引用校验。
- [x] 补充服务端定向测试，覆盖 `BOOKED`/后续流程允许编辑，以及退关、结案、锁定、旧版本仍
      被拒绝；补充前端详情回归，确认服务端返回 `EDIT` 时非草稿订单保持可编辑。
- [ ] 将 SE 表单分节收敛为单一五卡片构建器；新建与详情使用相同 key、标题、顺序、字段归属
      和栅格，删除 `isDetail` 下返回另一套组合的分支。
- [ ] 按“业务与客户、订舱与运输、货物与商业、提单信息、内部人员”重组现有字段；第二卡内部
      分订舱与主单、航线与船期、箱量与截关三组，详情动作置于标题或页签工具栏。保留既有
      form name、条件校验、候选搜索、payload 映射和只读规则。
- [ ] 备注实现空内容初始收起、有内容完整展开及主动收起后的“有备注”提示；支持异步回填、
      草稿恢复、同单保持用户选择和切单重置。订舱/配舱备注就近放置，操作备注在第二卡底部
      平级区域，不增加摘要和二次编辑操作。
- [ ] 新建与详情复用同一张提单卡的 MBL/HBL 页签；DIRECT 仅显示 MBL，HOUSE 可切换 HBL，
      切换时保留草稿；详情继续使用现有 `SeaDocumentSectionComponent` 的单一数据所有者和已落库
      版本动作，新建不发订单详情请求。
- [ ] 修改新建提交成功路径：只在创建响应存在 `data.id` 时提示成功、轮换幂等键并进入详情；
      空响应、缺 ID 或请求失败不跳转、不清表单、不轮换幂等键。
- [ ] 把 SE 详情的三张后置卡合并成“关联与记录”，页签为“操作记录｜同批订单｜拆票与改配记录”，
      默认操作记录，保留查看全部抽屉；仅消费已有准确总数，未知时省略数字，预览数量不作总数。
- [ ] 接入模板现有提交失败链路：适用隐藏字段必须校验；先切提单页签/展开备注，再滚动聚焦
      首个错误。DIRECT 跳过 HBL 校验及提交，但切回 HOUSE 时本地已填值仍保留。
- [ ] 补充定向测试：新建/详情五卡片合同一致、备注位置与折叠值保留、HOUSE/DIRECT 页签、
      新建成功按返回 ID 跳转、空响应保留页面、无底部操作栏、详情专属区块顺序，以及单证
      页签调整后的详情回填与动作回归；增加从未打开 HBL 的错误定位、备注初始化及手动收起、
      未知/失败/预览记录总数、其他订单类型后置区域不受影响的回归。

## 2. 验证命令

开发期先运行：

```bash
pnpm --dir web exec vitest run src/pages/orders/components/detail/OrderStatusSection.test.tsx src/pages/orders/detail-status-transition.test.tsx src/pages/orders/order-detail-transitions.test.tsx src/components/ui/order-list-template/OrderListTemplate.test.tsx
pnpm --dir web exec vitest run src/pages/orders/new.test.tsx src/pages/orders/new-create-idempotency.test.tsx src/pages/orders/order-kinds/sea-export/form-adapter.test.ts src/pages/orders/templates/components/sea/SeaDocumentSection.test.tsx
pnpm --dir web exec biome check src/pages/orders/components/detail/OrderStatusSection.tsx src/pages/orders/components/detail/OrderStatusSection.test.tsx src/pages/orders/components/detail/OrderDetailHeader.tsx src/pages/orders/order-detail-transitions.tsx src/pages/orders/order-detail-transitions.test.tsx src/pages/orders/detail.tsx src/pages/orders/detail-status-transition.test.tsx src/pages/orders/list.tsx src/components/ui/order-list-template/OrderListTemplate.tsx src/components/ui/order-list-template/types.ts
antd lint web/src/pages/orders/components/detail/OrderStatusSection.tsx --format json
antd lint web/src/pages/orders/order-detail-transitions.tsx --format json
```

提交前运行：

```bash
pnpm --dir web tsc
git diff --check
```

本轮 UI 最终验收运行前端门禁；只有新增服务端改动或此前服务端验证被新改动影响时再扩展后端检查：

```bash
pnpm run check:web
```

## 3. 风险与复核点

- 详情页现有表单只读策略依赖编辑动作，主流程流转必须独立检查 `TRANSITION_FLOW`，不能把
  “无编辑权限”等同于“无流转权限”。
- `allowedTargetFlowStatuses` 是合法迁移边唯一真相，禁止根据节点序号推导可点击状态。
- 锁状态未知或刷新中必须失败关闭；弹窗打开后提交前再次检查门禁。
- “锁状态失败关闭”只适用于主流程和发起退关；完成/取消退关、恢复、完结与反结案是生命周期
  命令，不得复用内容写门禁。
- 退关中存在“完成”和“取消”两个合法动作，必须让用户显式选择；不得把点击徽标默认解释为
  其中任意一个动作。
- 当前草稿订单无 `CLOSE` 动作，“未完结”只能禁用提示，不得为了视觉可点击而发送必然失败请求。
- 编辑资格以统一内容门禁为真相源，禁止只放开前端或只增加 `EDIT` 动作而遗漏仓储中的草稿
  状态检查；否则会出现“表单可填但保存必然 409”。
- 放开主流程状态不能绕过 `termination_status`、`closure_status`、业务锁与版本校验；测试必须
  同时覆盖允许态和四类阻断相邻路径。
- 状态成功刷新不得误清用户尚未保存的详情表单草稿；主流程刷新使用普通 `loadData`，不调用
  显式表单 `resetTo` 路径。
- 删除公共 `OrderListTemplate` 属性后必须运行 TypeScript 检查，确认没有残留调用方。
- 五卡片统一不能通过复制 JSX 达成；字段组件和分节合同必须单源，防止新建/详情再次漂移。
- 保持详情单证查询的单一状态所有者，禁止为了新建页签而新增另一套详情请求或重复回填 Form；
  切单不能串数据，页签切换不能丢失草稿。
- 创建成功跳转必须取服务端返回 ID，不得从订单号、路由或本地值猜测；空响应不得伪造成功。
- 用户明确不要底部按钮，不得顺带引入 `StickyFooterBar` 或模板默认 submitter。
- 实施前后均复核工作区；如出现其他任务的未提交文件，暂存和提交时只包含本任务文件及当前
  Trellis 任务目录，不覆盖或提交其他改动。

## 4. 提交计划

- 一组完成并验证后提交：`feat(order): 统一海运订单流转与表单工作流`。
- 当前回合仅交付计划；实施后按可验证分组提交代码。计划文档可单独以
  `docs(order): 完善海运订单五卡片改版计划` 提交。
- 提交前精确检查暂存文件，只包含本组明确归属的变更；本任务必要的服务端集成测试应随对应
  实现提交，不能按文件类别一律排除。其他任务和已有未提交工作保持原样。
