# AI前端实施与定向验证

日期：2026-11-16。责任：本代理独占非生成web产品与对应单测；不修改server、生成物、`web/tests/e2e/`、规范或任务主文档，不提交Git。浏览器及最终全栈门禁由主会话单独记录。

## 实施范围

- `config/routes.ts`、`order-kinds/types.ts/registry.ts`和新增`air-import/definition.ts`注册AI列表、新建、详情、费用及共享主单路由。集中路由继续持有标题；已有access真实AI能力直接复用。
- `air-export/form-adapter.ts`导出公共`buildAirCreatePayload`，AI只组合共享映射和AI/IMPORT身份；保留DIRECT无HAWB、HOUSE真实分单、普通保存不可换模式或主单、金额/日期/显式清空及结构化正文。
- `air-master.tsx`、`AirMasterFields.tsx`、`air-master-change-notice.ts`和新增`air-export/identity.ts`贯穿真实类型、公司、主单ID与版本；候选显式AE/AI，成员/继续建票/主单链接保持当前类型。queryKey携带类型；候选迟到门禁包含类型。主单读取缺失类型/公司也明确失败，不为旧mock放行。URL预选只采纳一次，后台重读不借新版本覆盖建票基线。
- 共享主单页按公司/类型/主单key重建；编辑卸载使旧操作失效。主单前台15秒/回焦/隐藏停止沿用React Query；探测增加每次身份独立owner，A→B→A不能复用旧探测缓存或迟到结果。
- `detail.tsx`复用已有`ImportHandlingPanel`服务面板。`sea-import/service-save-baseline.ts`入口复用公共Order校验，并为AI核对Order恰加一及MAWB/Link/可选HAWB身份和版本未变；成功只推进Order.version和本人探测参考，不reset主表。普通航空保存同时严格检查成功响应的类型与主单身份。
- `AirExportDetailFeatures`、`AirWeightPanel`、货物drawer、费用页/行内数量辅助/摘要及单证动作覆盖AI。重量queryKey带类型，ORDER/MASTER和ACTUAL/PLANNED仍独立。进口费用摘要使用权威主单ETA经营时刻；进口状态不提供出口flow按钮。

## 定向验证

最终合并批次命令（全部相对web目录，不是全量test入口）：

```bash
pnpm --dir web exec vitest run \
  src/components/layout/TagsView.test.tsx \
  src/pages/orders/air-master-change-notice.test.tsx \
  src/pages/orders/air-master.contract.test.tsx \
  src/pages/orders/air-master.edit.test.tsx \
  src/pages/orders/air-master.import.test.tsx \
  src/pages/orders/air-master.permissions.test.tsx \
  src/pages/orders/components/detail/OrderStatusSection.import.test.tsx \
  src/pages/orders/components/fees/OrderFeeHeader.test.tsx \
  src/pages/orders/components/fees/OrderFeeTableTabs.quantityAssist.test.tsx \
  src/pages/orders/detail-air-house-save-direct.test.tsx \
  src/pages/orders/detail-air-house-save-race.test.tsx \
  src/pages/orders/detail-air-house-save-stale.test.tsx \
  src/pages/orders/detail-air-house-save.test.tsx \
  src/pages/orders/detail-air-import-service-save.test.tsx \
  src/pages/orders/detail-import-service-save.test.tsx \
  src/pages/orders/fees.test.tsx \
  src/pages/orders/import-handling/ImportHandlingPanel.change.test.tsx \
  src/pages/orders/import-handling/ImportHandlingPanel.polling.test.tsx \
  src/pages/orders/import-handling/ImportHandlingPanel.test.tsx \
  src/pages/orders/list-documents-action.test.ts \
  src/pages/orders/list-query.test.ts \
  src/pages/orders/list.test.tsx \
  src/pages/orders/new-create-idempotency.test.tsx \
  src/pages/orders/new.test.tsx \
  src/pages/orders/order-kinds/air-export/AirWeightPanel.test.tsx \
  src/pages/orders/order-kinds/air-export/air-save-baseline.test.ts \
  src/pages/orders/order-kinds/air-export/form-adapter.test.ts \
  src/pages/orders/order-kinds/air-import/form-adapter.test.ts \
  src/pages/orders/order-kinds/air-import/service-save-baseline.test.ts \
  src/pages/orders/order-kinds/registry.test.ts \
  src/pages/orders/order-kinds/sea-import/service-save-baseline.test.ts \
  src/pages/orders/templates/air-create-layout.test.tsx \
  src/pages/orders/templates/air-document-mode.test.tsx \
  src/pages/orders/templates/air-import-master-selection.test.tsx \
  src/pages/orders/templates/air-preselected-master.test.tsx \
  src/pages/orders/use-order-detail-data.test.ts \
  src/pages/orders/use-order-fee-options.test.ts
```

最终合并批次结果：37个测试文件、317项测试全部通过，退出0，总耗时29.54秒；没有单个suite超过20秒。

其他检查：

- `pnpm --dir web tsc`：最终生成客户端与owner修改后通过（`tsc --noEmit`，退出0）。
- 对本代理47个新增/修改TS、TSX文件运行`pnpm --dir web exec biome check <精确文件列表>`：47文件通过，无修改、无warning。
- `git diff --check -- web`：通过。
- `antd info Alert/Descriptions/Form --format json`及`antd demo Alert basic --format json`：已查API。
- 分别执行`antd lint web/src/pages/orders/air-master.tsx --format json`、`antd lint web/src/pages/orders/templates/components/air/AirMasterFields.tsx --format json`、`antd lint web/src/pages/orders/components/fees/OrderFeeHeader.tsx --format json`：三份均issues=[]、partial=false。

新增及补充断言覆盖：AI DIRECT/HOUSE新建payload、已有主单只提交ID/版本、optional省略/显式字符串0/清空、AI候选和严格身份、AI-only主单访问/AE-only拒绝、主单往返迟到、航空服务确切版本及真实表单草稿保留、AI重量范围/来源及只读、AI服务任意子集及15秒/隐藏/回焦、列表AI类型/ETA、进口ETA摘要、进口无出口流转、集中标题及AE/SI相邻路径。

开发期间首个16suite批次有1个旧夹具失败：`air-create-layout.test.tsx`缺少真实businessType，严格候选门禁下不发请求。已为夹具补齐AE身份并通过后续6suite33tests。未放宽生产门禁。其他早期批次15suite85tests和AI专项7suite46tests通过。

焦点事件测试可观察到Node `TimeoutNaNWarning`。已用`NODE_OPTIONS=--trace-warnings`单独复现定位至happy-dom `EventTarget`事件分派及`air-master.edit.test.tsx:115`原有焦点事件；不修改依赖、不屏蔽warning。该警告与测试失败分开记录，不宣称stderr全零。

## 验收边界与交接

本报告只证明上述定向前端测试与静态检查，覆盖父V1/V3–V8/V11/V16/V18的前端部分，不替代真实PostgreSQL、API、权限中间件或事务验收。未自行运行浏览器、PDF、生产build、`check:fast`；这些由主会话及浏览器代理负责。本组不宣称AWB制作/PDF或完整CSV已交付。

产品代码已向主会话明确冻结并释放写权；后续独立检查代理可审阅所有非生成web改动。现有用户/其他代理改动均保留，无reset、无清库、无commit或push。
