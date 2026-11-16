# AI 前端独立复核与自修

日期：2026-11-16。角色 `trellis-check`，活动任务 `.trellis/tasks/11-15-hgj-air-import`。
复核基线为 SI 已交付提交 `74f86b3e`，覆盖全部 tracked 差异与新增非生成前端文件。
不修改后端、生成客户端、任何 e2e、spec 或任务主文档，不提交、推送或归档。

## 合同与范围

读取当前 check.jsonl、PRD/design/implement，父批准模型/字段/验收合同，航空正文与模式、
航空运输/重量、主单方向、进口服务和新 air-import-contract，以及前端 index 与已有状态、
类型、组件/目录及质量约束。逐路径复核而非只检查最后一段补丁：

- AI registry/routes/menu/title，创建 AI+IMPORT、真实 AI 权限和共用航空映射；
  DIRECT 无 HAWB、HOUSE 真分单，普通保存不能切模式或换主单，经营字段清空与 decimal/optional。
- AirMasterFields 的预选/候选、真实公司/类型/ID/版本校验、queryKey、URL 预选仅采纳一次、
  后台刷新不替换已采纳建票基线；共享主单页/成员/历史/继续建票链接保持 AI 身份。
- 共享主单编辑、预览/保存、恢复关联及权重入口；前台15秒/回焦、隐藏/离页、owner/编辑session
  隔离 A→B→A 和离开重进同ID；探测只提醒，不回填编辑版本。
- 详情/费用成功读取严格匹配 Order ID/company/type；AI 服务复用固定五环节，航空服务保存
  只有确切 Order+1 且 MAWB/Link/可选HAWB身份与版本全不变才推进整票基线，不重置草稿。
- Cargo 维度辅助、危险品、ORDER/MASTER 与 ACTUAL/PLANNED 重量入口、费用数量来源、
  进口权威 ETA 展示、状态摘要与 AE/SI 相邻分支；接口仍使用生成客户端。

未发现需要改变已批准公共接口、数据模型或模块边界的事项。本组 AWB PDF/完整 CSV 仍为后续范围。

## 确定发现与已修复

### 1. 共享主单保存迟到关闭编辑稿

文件 `web/src/pages/orders/air-master.tsx`。

原路径在成功写入后先 `setEditOpen(false)`，再等待聚合 `reload()`，最后返回 true；
新输入或新的编辑会话可能被旧 ModalForm 的完成结果关闭。修复后继续使用官方
`GuardedModalForm.onFinish` 生命周期，保留弹窗到读取结束；每段 await 后复验原编辑
serial/revision/实例，只有仍属本次提交且没有后续输入才返回 true。关闭回调也复验 serial，
旧实例不能关闭新编辑。保存期间新正文保持可见和未保存保护。

### 2. 恢复关联借用后台新主单版本、缺少会话保护

同文件原恢复载荷读取当前 query 的 `master.version`，弹窗打开后后台刷新可以为旧恢复稿
借新版本；写入和重读也缺少实例/会话检查。现在打开时复制真实主单快照，固定模式与版本，
每次打开有独立 serial；恢复提交和重读后核对存活、权限、serial 与编辑revision，
旧候选迟到返回空而不写当前选择，关闭同样按 serial。等待期间新原因输入保留，
没有后续输入的正常保存仍由官方生命周期关闭，不增加镜像 loading。

共同资料更新与恢复的成功响应均经既有 `requireAirMasterIdentity` 校验目标 ID、当前公司、
真实 AE/AI 与有效版本，不能仅凭任意 data.id 视为当前目标已成功。没有放宽身份给旧 mock。

### 3. 相邻测试失效缓存键导致场景未发生

`web/src/pages/orders/air-master.edit.test.tsx` 原“主查询变新不覆盖输入”仍失效
`['air-export','master']`，产品已迁移为 `['air-orders','master']`，实际没有执行后台重读。
已改真实 key，并断言 GetMaster 第二次读取确实发生，保留输入和原版本断言。

这些产品修复开始前已通知主会话，由主会话协调浏览器暂停相应最终取证；修复完成后
已明确冻结产品代码，避免把旧整例当最终证据。

## 实际验证

先跑修改触发和正常相邻：

```bash
pnpm --dir web exec vitest run \
  src/pages/orders/air-master.sessions.test.tsx \
  src/pages/orders/air-master.edit.test.tsx \
  src/pages/orders/air-master.contract.test.tsx
```

**3 文件、14 项 PASS，13.30 秒**。随后补齐正常成功关闭与严格写响应身份，并运行最终批次：

```bash
pnpm --dir web exec vitest run \
  src/pages/orders/air-master.sessions.test.tsx \
  src/pages/orders/air-master.edit.test.tsx \
  src/pages/orders/air-master.contract.test.tsx \
  src/pages/orders/air-master.import.test.tsx \
  src/pages/orders/air-master-change-notice.test.tsx
```

**5 文件、26 项 PASS，12.70 秒，无跳过**。覆盖共同保存/恢复写后读取在途的新输入保留、
恢复固定打开版本、两类正常成功关闭、真正后台重查不覆盖稿、预览迟到、刷新取消/失败/成功、
双模式创建/恢复、AI-only/AE借权拒绝、严格主单身份、轮询及 owner 的往返迟到。
新增 sessions 文件 5 项，未堆叠超过10个重型业务渲染，没有 suite 超20秒。

- `pnpm --dir web tsc`：**最终产品冻结后 PASS**（`tsc --noEmit`）。
- `pnpm --dir web exec biome check <全部49个本组非生成TS/TSX文件>`：**PASS**，49文件/77ms，
  无待修问题；清单由基线 diff 与 untracked 合集生成，排除服务生成物和e2e。
- `pnpm run check:architecture`：**PASS**，18条检查通过，扫描560个产品文件/2275条依赖，
  未发现跨领域依赖边界问题。
- `pnpm --dir web exec antd lint src/pages/orders/air-master.tsx --format json`：**0 issues、
  0 skipped、partial=false**。
- `git diff --check -- web`：**PASS**。

前置实现代理已经执行的37文件317项与其他相邻结果见 `air-import-web-validation.md`，
本代理没有无因重复全部套件，也不将其写为本代理新跑结果。Node仍可观察到已有
`TimeoutNaNWarning`，与前置报告定位的happy-dom焦点事件一致；未屏蔽或改依赖，
不宣称stderr为空。

## 未修复项与证据边界

本轮范围内没有遗留的已确认前端缺陷或待产品决定事项。后端正式PG、HTTP权限与事务、
浏览器最终整例、最终全栈门禁仍由主会话/相应代理记录；本报告不把mock或定向通过
冒充真实SQL、浏览器或完整门禁通过。后续航空PDF和四类完整CSV另按依赖任务验收。

**写权已释放：本代理不再修改任何非生成前端产品/tests，仅完成本报告；主会话可接管最终验收。**
