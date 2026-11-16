# 独立复核记录

日期：2026-11-16（北京时间）。复核范围：`5cb906bd..e4b79007` 的产品差异及随后只读动作修复。复核代理原负责 B 航空工作区，本次重点复核未负责的 A 模板、主会话组合和 C 批量摘要。未提交变更，未重复运行全量门禁。

## 已查范围

- A：`air-template.tsx`、`AirMasterFields.tsx`、`AirDocumentSection.tsx`、`AirWaybillFields.tsx` 的六列布局、DIRECT/HOUSE 分支、主单只读正文、分单可编辑正文、隐藏页签和低频字段错误展开。
- 主会话：`detail.tsx` 的三个航空插槽、独立 `airTransportSnapshot`、进口服务前移及服务保存回调；页头独立单证菜单与海运制单分组。
- C：`list.tsx`、`list-query.ts`、`air-list-columns.tsx`、公共列表列设置；Proto→biz→data→service→TS 的本票重量确认摘要读取链。
- B相邻：父表单 disabled 对已存在读入口的影响，未保存货量与确认按钮关系。

## 发现与修复

### P2 审核只读继承禁用阻止查看和核对

当订单父 ProForm 因审核禁用时，航空区域的实际货物查看、MAWB/HAWB 制作导航、历史及重量参数/重新核对会继承禁用。确认写入口虽然已有 `canUpdate`，读入口却不能使用，不符合锁定仍可查看的既有合同。主会话同时在浏览器中观察到了该问题。

修复：

- `AirExportDetailFeatures` 的制单导航和历史按钮显式 `disabled={false}`。
- `AirCargoWorkspace` 的实际明细打开和失败重试显式解除继承禁用，抽屉写权限继续由 canCreate/canUpdate/canRemove 控制。
- `AirWeightPanel` 局部 `ConfigProvider componentDisabled={false}` 允许读参数和重新核对；重量确认仍只在 canUpdate 时出现，且未保存货量、依据变化及提交入口继续阻止写入。
- 主会话修复 `AirMasterFields` 的查看主单与成员按钮；本次补父 Form disabled 回归验证。

回归验证：审核只读父表单下四个单证读按钮、实际货物、主单成员、来源选择、重新核对仍可用，确认重量入口不出现。

## 复核结论

- 独立运输摘要消费最新已读取 `order.airExport.master` 的只读 prop；本票 `airExport`、`airMasterExpectedVersion` 和未保存字段仍留在原 Form 基线。摘要更新未被误当作允许旧稿提交的新版本。
- 进口服务只是从 append 移至 prepend；成功保存的 `acceptsImportServiceSaveBaseline`、目标身份、刷新代际和确切 Order+1 判断保留，未新增 reset 主表行为。
- MAWB正文只读渲染不注册正文输入；HAWB与新建MAWB保持各自路径，DIRECT不挂载虚构分单。隐藏主分单页签 forceRender 保留输入，错误事件先切页签再展开对应低频分区。
- 列表只消费后端摘要，无逐行 HTTP 请求。后端按已授权订单公司分组，批量读取每个来源最新 ORDER 确认、委托货量和实际明细；过期检查复用与重量预览相同的事实/指纹构造，未混用 MASTER 或两种来源。
- `confirmationStale` 非 optional 布尔省略按 false；未确认来源缺席，明确重量字符串 `0` 不被当空。参数错误不被吞成零摘要。
- 未发现额外阻塞实施的确定问题。

## 实际验证

- `pnpm --dir web exec vitest run src/pages/orders/order-kinds/air-export/AirCargoWorkspace.test.tsx src/pages/orders/order-kinds/air-export/AirExportDetailFeatures.readonly.test.tsx src/pages/orders/templates/air-transport-summary.test.tsx`：3文件8项通过，8.26秒。
- `pnpm --dir web exec vitest run src/pages/orders/templates/air-document-workspace.test.tsx src/pages/orders/templates/air-document-mode.test.tsx src/pages/orders/list-query.air-weight.test.ts src/pages/orders/air-list-columns.test.tsx src/pages/orders/components/detail/OrderDetailHeader.test.tsx`：5文件21项通过，28.54秒，零运行警告。
- 只读修复涉及的6个TSX文件 Biome 检查通过；`pnpm --dir web tsc`通过；`git diff --check`通过。

## 限制

本复核未重跑全量门禁、真实 PostgreSQL 或浏览器 e2e，由主会话和 C 代理的独立验收负责。源码核对可证明调用边界与参数传递，不替代真实数据库并发、完整权限组合和浏览器视觉验收。
