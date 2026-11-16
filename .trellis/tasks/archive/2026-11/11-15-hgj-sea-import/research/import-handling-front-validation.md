# 固定进口服务面板验证

本记录由主会话填写，范围为 `web/src/pages/orders/import-handling/`，不替代 SI 后端、
海运共用页面与归档 PDF 的其他分工验收。当前产品使用生成的三个 OrderService 接口，
字段合同见 `.trellis/spec/server/backend/order-import-handling.md`。

## 实现范围

- `model.ts`：五项完整 wire 校验、省略 false 与 optional 完成时间、Dayjs↔RFC3339、
  服务完整快照请求、取消项目、既有完成事实/换单代理改变原因判定。
- `ImportHandlingPanel.tsx`：本票只读摘要及展开历史、官方 ModalForm 生命周期、
  供应商/责任人/本票附件候选、取消确认与原因、完成事实和未承接分离。
- 当前公司、真实进口类型、read/update 和父可写策略门控；CLOSED 服务变更须先重开。
  查询每 15 秒仅前台当前实体运行，隐藏/离页停止，回焦立即检查。
- 编辑固定打开时服务/Order.version；新探测只提醒，检查失败不清提醒。
  重新读取先确认，失败/取消/刷新在途编辑不覆盖输入；A→B→A 的消费代际隔离。
  保存等待期间的新输入保留并继续要求重新读取，保存失败不关闭。
- 独立保存回调仅传明确成功响应和前版本，由主详情的精确基线判定保护整票草稿，
  不把整票经营字段混入服务请求，也不直接写 flowStatus。

## 实际检查

```bash
pnpm --dir web exec vitest run \
  src/pages/orders/import-handling/model.test.ts \
  src/pages/orders/import-handling/ImportHandlingPanel.test.tsx \
  src/pages/orders/import-handling/ImportHandlingPanel.change.test.tsx \
  src/pages/orders/import-handling/ImportHandlingPanel.polling.test.tsx
```

4 文件、11 例全部 PASS，完整执行 17.94 秒，单文件均低于 20 秒。
覆盖完整五步/非法缺失或重复/false省略/optional未设、任意子集及清空时间保留其他字段、
原因、精确大整数版本、完成变动、失败保存/在途新输入、取消确认、旧基线持续提醒、
读取失败及刷新取消、CLOSED/公司不符、15秒/隐藏/回焦/非当前实体。
使用真实 React Query、官方表单与 Modal，历史表在此交互测试以轻量替身隔离，
后端与浏览器测试负责实际历史请求和前后事实内容。

首轮失败包括测试导出 vi.hoisted 的方式、未等待初次读取可写、Antd Modal 重复标题/旧隐藏
DOM 引用及成功回调的稳定归一化字段断言，已修正并实际重跑；未降低业务断言。

```bash
pnpm --dir web exec biome check src/pages/orders/import-handling
pnpm --dir web tsc
antd lint web/src/pages/orders/import-handling/ImportHandlingPanel.tsx --format json
git diff --check
```

当前 7 文件 Biome、完整 tsc、组件 antd lint（0 问题）和 diff 检查通过。
离线查询 Form/Checkbox/Modal/Descriptions/Alert API 后实施，未添加依赖。
运行保留依赖侧 TimeoutNaNWarning，不以该噪音掩盖失败。

## 未完成的本组证据

- 本记录不宣称 mock 场景为真实 SQL 或浏览器通过。SI 后端扩展 PG、独立检查、
  真实浏览器、归档 PDF 实图、生产 build 及风险匹配最终门禁仍由当前任务统合。
- 未向开发库应用迁移或清数据，未向外站/真实员工发通知。
