# 航空单证布局实施前复核

日期：2026-11-16（北京时间）。角色：独立复核；本轮暂不修改产品代码，等待实现代理完成后再复核和补局部修正。

## 范围与已有合同

- 已读本任务prd/design/implement、航空工作区规范、航空正文与模式合同。
- 已查 `AirDocumentSection.tsx`、`AirWaybillFields.tsx`、`form-adapter.ts`、`air-save-baseline.ts`、`AwbEditor.tsx`、`OrderFormTemplate.tsx`、`sea-form-error-reveal.ts`；海运 `SeaBillContentFormFields.tsx` 仅作为布局参考，不修改。
- 订单详情的HAWB由整票保存：原Master身份/版本不变，Order恰增一，House/Link按正文实际变化恰增一；`acceptsAirSaveBaseline` 不得因布局或带入动作被旁路。
- 专用AWB制作页是另一保存生命周期：原主单或分单目标、独立revision、preview指纹、同步错误与打印保存快照不可由订单布局组件改写。
- `serializeAirWaybillContent` 的空子消息清理只作用于声明及汇总：保留 `{}`、显式空字符串、false和十进制字符串`'0'`。

## 当前海运对齐缺口

1. 订单variant目前将五个抬头字段顺次排在两列网格中，通知人与第二通知人分别占格，导致代理和后续唛头/品名的行关系与海运不同。
2. 航空order文本域当前固定2行，海运抬头4行、货描5行。当前MAWB摘要是Descriptions文本，与编辑HAWB和海运文本框的边界/高度不同。
3. 航空低频区直接暴露七个折叠组，需要再包一层“更多运单资料”，同时保留内部所有字段及错误揭示链。
4. 缺少明确带入入口。不能直接照搬海运现有异步setFieldValue实现，因为本任务明确要求迟到响应与标脏保护。

## 实现建议与复核点

### 通知人切换

- 两个字段保持独立路径：`notifyPartyText`与`secondNotifyPartyText`，禁止将一个字段当另一字段的临时缓冲。
- 推荐两份Form.Item常驻，只切display/hidden；若条件挂载，必须保留独立key与preserve，并证明validateFields仍带出两个现有值及显式清空。
- 通知人错误reveal须选中正确页签；父HAWB/MAWB页签先按正文前缀切换；低频字段错误需同时打开外层“更多”和内部原分区。
- 第二通知人有值标记取该字段当前值，不从显示页签反推。切换本身不应标脏。

### 资料带入与并发

- 带入只写本次目标正文的单一字段。客户→shipperText，国外代理→agentText，订单goodsDescription→goodsDescription；不连带覆盖唛头、费率、运价行或件重尺。
- partner GET经生成客户端及React Query管理；查询身份包含当前公司、订单/新建实例、来源ID。请求开始与完成均校验来源和目标字段，不只检查组件是否仍挂载。
- 捕获目标字段原值和编辑代际；请求期间用户继续输入、选择别的客户/代理、切换公司或正文目标时，不得带入旧结果。A→B→A也要有实例/代际保护，不能仅比相同的最终ID。
- 覆盖非空目标前确认，确认期间继续输入必须再次核验；取消、请求失败或身份变化保留原稿。空源资料提示，禁止静默清空目标。
- `form.setFieldValue`不会触发Form.onValuesChange；必须通过现有模板`applyValues`/明确onApply回调标脏。仅setFieldsTouched不足以触发页签离开保护和新建草稿生命周期。
- 解析客商响应校验success、ID及当前公司/允许共享归属；不能用缺响应当空档案、也不能从当前工作区猜单证主体。

### 只读与两个variant

- MAWB只读展示复用排列但不能注册name字段或改写本票Form的主单版本。以textarea readOnly/value呈现时不参与整票保存；保留完整换行和长文。
- 父Form disabled或只读时，通知人页签、折叠、预览/历史导航仍可用；明确禁用所有带入与编辑写动作，不能用全区ConfigProvider componentDisabled=false误放开写入。
- `variant=full`保持专用AWB字段注册、数值校验、Form.List路径、纯日期和独立保存协议。新布局仅应用order variant及只读常用区。
- order常用区不重复注册低频组中已经迁出的同名字段，避免多个Form.Item绑定同一路径。

## 实现后验证计划

- 通知人A→B→A、切MAWB/HAWB、显式清空、validateFields与序列化的完整值保持。
- 已有收发文本覆盖确认取消/接受，请求期间继续输入、来源切换、公司/目标切换、迟到成功不落地。
- 真实OrderFormTemplate带入后dirty与离开保护，失败不标脏，单纯切页签不标脏。
- 父Form disabled下可切通知人与折叠，不能带入或编辑；只读MAWB没有表单注册，原expectedMasterVersion不变。
- full模式航空现有字段测试相邻回归；真实浏览器1440/390对照由主会话执行，截图不能替代保存正确性测试。

## 实施后复核与实际验证

- 实现代理已按建议将通知人两份字段常驻隐藏，增加主单ID/候选ID的带入代际保护、Form.Item长度校验，并移除不标脏的静默fallback；带入按钮仅在父onImport存在时展示。
- 主会话的`airDocumentImportPatch`只接受airHouseContent或airMaster.waybillContent两个合法目标，new/detail以模板`applyValues`调用，因此字段深合并与dirty由已有模板管理。
- 已核实客商公司等值判断符合当前合同：`organization-shared-masterdata.md`明确系统不存在集团、客户/供应商/国外代理仅当前公司经营；`data/partner.go:Get`按organization_id精确读取。无需添加跨公司共享兼容分支。
- 新增独立`air-document-import.integration.test.tsx`，使用真实OrderFormTemplate、actionsRef、带入按钮与patch，验证通知人切换不标脏、带入立即标脏且保留其他正文/低频字段。
- 新增独立`air-document-readonly.review.test.tsx`，父Form disabled下DIRECT主单仍可切第二通知人，只读文本框保持4行且不向父Form注册airMaster/airHouseContent；原主单版本基线保留。
- 更新`air-document-workspace.test.tsx`的旧MAWB断言：由Descriptions改为只读文本框，仍验证主分切换和本票输入/基线。
- 实际命令：独立integration+readonly两文件2项通过（6.17秒）；air-document-workspace+full AirWaybillFields两文件7项通过（13.60秒）。只检查上述修改测试的Biome通过。

## 状态与限制

当前产品代码继续由实现代理负责，本复核未改产品代码或海运页面。独立测试覆盖了程序带入dirty及父表单只读这一实施风险；其他带入race/覆盖测试由实现代理执行。未运行全量门禁、浏览器或数据库验证，最终门禁和1440/390真实浏览器对照由主会话负责。

## 保存回归长时间阻塞诊断

- 主会话集成保存回归仅显示RUN且单worker CPU100%。对该既有测试worker的Node Inspector CPU采样显示，1334次样本停在happy-dom的`CSSStyleDeclarationComputedStyle.getComputedStyle`；栈来自`openAirHouseRemarks`找不到可见“签发与备注”后Testing Library生成可访问角色错误明细的`prettyRoles/getRoles/isInaccessible`。这不是产品React渲染循环。
- 原fixture没有先打开新增外层“更多运单资料”。修正为先限定HAWB tabpanel，再按标题文案定位真实`role=button`元素、等待其可见、顺序展开外层与“签发与备注”；没有使用hidden:true、跳过可见断言或放宽超时。
- 修复后原保存/race/stale三文件8项通过（22.51秒整轮）。由于原保存suite首次单跑22.35秒超过20秒约束，将首次签发及同页改单1项移至`detail-air-house-first-release.test.tsx`，其余3项留原文件，断言不变。
- 拆分后两文件4项通过（15.47秒整轮；原3项12.91秒，新单项11.12秒），相关3文件Biome和git diff检查通过。产品代码无需改动。
