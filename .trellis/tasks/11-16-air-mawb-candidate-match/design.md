# 设计：空运 MAWB 同号主单自动判断

## 边界

- 纯前端，集中在 `web/src/pages/orders/templates/components/air/`；
  后端、Proto、生成客户端不变。
- 新增 `useAirMasterCandidate.ts`（参照 `useSeaMasterBillCandidate`），
  `AirMasterFields.tsx` 只负责呈现；`AirMasterCandidateHint` 与「选择共享主单」下拉删除。
- `airExportFormAdapter` 不改：继续以内部字段 `airMasterMode`（NEW/EXISTING）、
  `airMasterId`、`airMasterExpectedVersion` 决定载荷；这些字段改由 Hook 自动维护，
  不再由用户选择。

## 数据流

1. `Form.useWatch(['airMaster','masterNo'])` → 规范化（去空格与连字符）→
   通过模 7 校验后 300ms 防抖得到 `debouncedMasterNo`。
2. React Query `['air-orders','master-candidate',{organizationId,businessType,masterNo}]`
   调 `airExportServiceListAirMasterWaybills({businessType, keyword, page:1, pageSize:20})`，
   经 `requireAirMasterIdentity` 校验身份后按 `content.masterNo === masterNo` 精确筛选；
   `skipErrorHandler` + `meta.silent`，错误由卡片与校验呈现。
3. 派生状态：`checking | none | blocked | modeMismatch | conflict | ready | error`。
   - 差异比较字段：航空公司、航班号、起运/目的机场、计划起飞/到达（与主单摘要一致的
     共同运输字段）；空值不算差异，日期按分钟比较。
4. Effect（只写变化值，避免循环）：
   - `ready`：自动带入本票空字段，写 `airMasterMode='EXISTING'`、`airMasterId`、
     `airMasterExpectedVersion`、`airMasterSelection`；
   - 其他状态：写 `airMasterMode='NEW'` 并清空 id/version/selection（保留用户已填值）。
5. 「采用主单资料」：`setFieldsValue({ airMaster: candidate.content })`，随后自然进入 `ready`。
   「切换为主单模式」：`setFieldValue('airDocumentMode', candidate.documentMode)`。
6. 校验：Hook 暴露 `validate()`，作为主运单号字段的 validator，读取当前表单值
   防止旧结论穿透（同海运写法）；状态变化时 `validateFields(['airMaster','masterNo'],{dirty:true})`。
7. `ready` 时主单字段 `disabled`（以主单为准）；航线/申报小节与主单备注照常展示但只读。

## URL 预选（R10）

保留 `preselected` 查询，改为只向表单写入 `airMaster: content`（含 masterNo）与
`airDocumentMode`；随后由同一 Hook 命中并进入 `ready`，不另起一条关联路径。

## 取舍

- 不新增后端核对接口：列表接口包含匹配 + 前端完全相等对 11 位 MAWB 已是精确核对，
  可关联性也已由后端计算；新增接口只会复制同一逻辑。若后续需要成员分单号预检再评估。
- 不做「多航程」选择：空运主单只有一份共同资料。

## 风险

- 自动写入 `airMasterMode` 等内部字段与 `AirDocumentSection`、`AirOrderWaybillFields`
  对 `airMasterMode` 的读取联动：需回归单证卡在关联主单时的 MAWB 只读逻辑。
- 新建草稿恢复：草稿含 `airMasterMode=EXISTING` 时恢复后由 Hook 重新核对，以最新结论为准。

## 回滚

纯前端，按提交 revert。
