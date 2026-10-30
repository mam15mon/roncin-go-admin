# Research: 海运共同段与本票最终目的港的前端体验

- Query: 现有海运出口新建、详情、共享候选、航程调整、拆票改配及单证页，是否正确区分共同运输段和分单最终目的地；如何落实「默认同港、仅例外修改」的最小体验。
- Scope: internal
- Date: 2026-10-30

## Findings

### 1. 用户范围与结论

用户已在本任务规划中选择：本期仅维护现有共同段和本票目的地，不做后续分段航次管理；默认各票最终港口与共同段卸货港相同，常规不重复填，确有后续去别港时才修改本票最终港；手动修改包括主动清空不能被默认值覆盖。

**现有字段边界基本具备，但缺默认值与清楚的信息架构。** 新建和详情共用「航线与船期」组，把起运港、目的港、卸货港、中转港四个字段排在同一行，船期只写 ETD/ETA。用户很容易把独立目的港与共享段卸货港当成两次填同一件事。

### 2. 找到的文件

| 文件 | 职责 |
| --- | --- |
| `web/src/pages/orders/templates/sea-template.tsx` | 新建与详情共享的五卡片结构、组标题和字段布局 |
| `web/src/pages/orders/templates/components/sea/SeaTransportSection.tsx` | 主单候选提示、共享航程摘要、船期、港口与目的地字段 |
| `web/src/pages/orders/templates/components/sea/useSeaMasterBillCandidate.ts` | 候选请求、航次选择、差异采纳、共享只读来源 |
| `web/src/pages/orders/templates/components/sea/sea-shared-voyage.ts` | 共享字段集合、Form 值映射及差异比较 |
| `web/src/pages/orders/order-kinds/sea-export/form-adapter.ts` | 创建默认值、创建/更新 payload 与详情初始化 |
| `web/src/pages/orders/new.tsx` | 来源订单加载、创建身份隔离与初值组装 |
| `web/src/pages/orders/sea-house-order-context.ts` | 从已有主单新增分单的最小来源上下文 |
| `web/src/pages/orders/components/drawers/SeaTransportExecutionUpdateModal.tsx` | 共享航程预览、受影响订单展示与正式执行 |
| `web/src/pages/orders/order-kinds/sea-export/SeaExportDetailFeatures.tsx` | 共享航程调整与改配入口及权限 |
| `web/src/pages/orders/components/drawers/SeaOrderReassignmentModal.tsx` | 当前票切换目标 MBL/TE，非追加后续运输段 |
| `web/src/pages/orders/components/split/SplitResultsSection.tsx`、`splitUtils.ts` | 拆票目标共享身份与航程选择；当前未提供各结果票最终目的地输入 |
| `web/src/pages/orders/templates/components/sea/SeaDocumentSection.tsx`、`SeaBillContentFormFields.tsx` | MBL/HBL 独立正文表单 |
| `web/src/components/ui/order-template/OrderFormTemplate.tsx`、`components/layout/formDraft.ts` | 草稿恢复、显式清空、程序回填持久化 |

### 3. 已具备的共享与独立边界

- `sea-shared-voyage.ts:6-14`、`:66-73` 的 `SeaVoyageValues` / `seaVoyageFields` 仅包含起运、卸货、中转、船名航次、ETD、ETA，**没有 `destinationLocationId`**。
- `sea-shared-voyage.ts:50-63` 的 `seaVoyageFormValues` 不复制本票目的地；`:96-145` 的差异比较也不比较目的地。
- `useSeaMasterBillCandidate.ts:254-263` 自动继承仅填空白共享字段；`:277-280` 的显式「采用主单航程」也只更新共享字段。
- `SeaTransportSection.tsx:639-707` 的港口区中，共享起运/卸货/中转字段使用 `readonly`，`:670-677` 的 `destinationLocationId` 不使用共享只读条件，仍可独立修改。
- `SeaMasterBillFields.inheritance.test.tsx:15-65` 已验证目的地、客户、货物、费用不随继承改变，目的港 Select 保持可编辑；`SeaMasterBillFields.identity.test.tsx:58-76` 验证身份变化不清独立目的地。
- `form-adapter.ts:226-236`、`:455-465` 分别传递 destination 和 discharge，且 ETD/ETA 属于共享航程字段，不存在独立「最终目的港 ETA」字段；`:333-339` 详情直接读取服务端两种地点与共同船期。

### 4. 真实缺口：当前没有「最终目的港默认卸货港」

- `form-adapter.ts:35-61` 创建默认值仅含业务枚举、单证默认正文和人员等，没有目的港或卸货港默认。
- `new.tsx:219-232` 来源创建只预填船公司、主单号和 HOUSE 模式。
- `sea-house-order-context.ts:29-67` 的来源上下文只提取 MBL/TE 身份，不复制来源订单的目的地。这一点应保留：不能让来源 A 的独立雅加达目的地默认流入本票 B。
- `SeaTransportSection.tsx:670-677` 目前目的港是一直展开的空下拉，缺少默认提示及「本票例外」的表达。
- 未在本次检查的前端文件中找到 `destinationLocationId ← dischargeLocationId` 自动默认逻辑；所以不能对用户声称目前已经默认。

### 5. 原始“资料尚未填全”提示的实际作用

- `SeaTransportSection.tsx:100-107` 检查选中的 TE 是否缺船名、航次、起运港、卸货港、ETD、ETA；中转港可为空且不属于缺项检查。
- `:317-323` 提示是 `type="info"`，自身不是校验规则，未直接禁止提交。
- `useSeaMasterBillCandidate.ts:212-222` 的关联准备条件不要求所有航程字段完整，只要求候选/TE 身份版本有效、模式允许、无差异、非详情。
- 但是缺失共享字段与本票已填值不同会进入差异列表（`sea-shared-voyage.ts:130-145`），校验在 `useSeaMasterBillCandidate.ts:303-306` 阻断；「采用」会把本票字段覆盖为共享空值。
- `SeaScheduleDateFields` 的 ETD/ETA 始终必填且共用时只读（`SeaTransportSection.tsx:567-601`）。因此已有共同段缺 ETA 时，新票采用后 ETA 为空且只读，不能建票，必须先补已有共同段。现有 `SeaMasterBillFields.inheritance.test.tsx:112-158` 明确验证了这条路径。
- 缺船名/航次或港口但 ETD/ETA 齐时，并非所有缺项都会独立阻断；当前是笼统 info，无法让用户判断哪些只是资料未齐、哪些导致本次不能保存。
- 已有订单链接为新标签打开（`:325-338`），详情菜单维护入口由 `SeaExportDetailFeatures.tsx:175-183` 的当前写入能力和 `reassign` 权限控制。提示目前无视是否有此权限，可能把无权限用户引向不可见入口。

### 6. 共同段编辑、改配、拆票的能力边界

- `SeaTransportExecutionUpdateModal.tsx:37-70` 输入只包含共同段 origin/discharge/transit、船名航次、ETD/ETA，没有本票目的地。
- `:181-237` 先预览再执行，携带 TE 版本、原因、外部确认和幂等键；`:399-405` 展示精确关联票数。该入口可直接复用，不应另建一个未预览的「补全共享数据」副作用接口。
- `SeaOrderReassignmentModal.tsx:340-344`、`:519` 明确是把当前票切换到目标 MBL/TE，并更新其权威共同航程；这不是给当前票追加「新加坡→雅加达」后续运输段。其当前表单没有 destination 输入（港口输入为 origin/discharge/transit）。
- `splitUtils.ts:6-32` 的 `ResultConfig` 无 destination；`:43-76` 的目标组包只对共同段建 NEW/CANDIDATE/CURRENT。`SplitResultsSection.tsx:379-419` 采用候选时也不处理 destination。本期每票目的地可在结果订单详情独立修改；若要求拆票时直接逐结果指定目的港，需单独计入范围，不能假定当前已有。
- 当前一主单多个 `transportExecutions` 在 UI 是「本票共用航程」单选（`SeaTransportSection.tsx:235-249`），候选不能默认第一条；不能把这个单选列表误说成按先后顺序排列的全程多段路线。

### 7. HBL 正文港口/交付地：当前不存在对应表单字段

- `server/api/order/v1/sea_document.proto:182-200` 的 `SeaBillContent` 仅有收发通知人、唛头、品名、件重尺、付款/运输条款、提单形式、放单方式、条款、国外代理。
- 该消息 **没有 `portOfLoading`、`portOfDischarge`、`placeOfReceipt`、`placeOfDelivery` 或独立船期字段**，所以前端没有可追踪的 HBL 港口/交付地自动填入逻辑。
- `SeaDocumentSection.tsx:197-227` 分别读取 MBL/HBL 正文入 Form；`SeaBillContentFormFields.tsx:422-438` 的“从订单带入”只搬品名与件重尺，不搬地点。
- 本期若仅操作票展示和保存独立最终港，无需假造 HBL 正文新字段；若未来需要实际提单打印交付地，要在服务端 Proto 和单证不可变快照里正式设计，不能在前端以隐藏副本冒充已存在单证字段。

### 8. 本期最小 UI 建议（结合用户“默认一样、别再重复填”）

1. **共同段分组**：复用 `TemplateFieldGroup`（`components/FieldGroup.tsx:7-27`），将现有「航线与船期」明确为「共同运输段」，起运港→共同段卸货港紧邻，保留船名航次与共同段 ETD/ETA。ETA 辅助说明是到共同段卸货港，不是到本票最终港。
2. **目的地默认与例外**：本票最终目的港默认等于共同段卸货港。常规显示“最终目的港：与卸货港一致”，给一个「更改最终目的港」的轻入口，例外时展开现有 `destinationLocationId` Select。无需新增必填“是否中转/是否不同”流程，也不要把共同段和最终港两个空输入都交给用户重复录入。
3. **默认只属于创建意图**：普通新建选择卸货港、匹配已有主单继承卸货港、从来源新建确定 TE、主动采用共同段，都可在本票仍处默认状态时更新最终港。**不得从来源订单复制它的最终目的港**。已有订单详情保持当前已保存的目的地，不因挂载或共同段后台查询自动改写。
4. **手动覆盖保护**：只要手动选择或清空最终港，就退出自动跟随；后续候选重读、主单身份变化或「采用航程」不改这一手动值。恢复草稿保留它自己的模式与值，显式清空同样属于用户意图；用户可通过明确的「恢复与卸货港一致」操作重新采用默认。
5. **共同资料不全**：列明实际缺项，如“共同段尚缺：ETA”；区分可保存提醒与必填阻断。用已有订单链接/同一共享调整入口补共同段，不能在新票输入时悄悄修改所有旧票。若优化为本页按钮打开维护，也必须明确“将修改共同段并影响 N 票”，复用预览/外部确认/权限检查；这属于是否纳入本期的额外交互决策。
6. **保留当前安全边界**：版本号仍由系统内部传递；共享段调整仍预览影响范围；目的地编辑仍由本票保存；近期加入的变动提示与草稿保护不能被默认回填绕过。

### 9. 默认值实现需要防的陷阱

- `OrderFormTemplate.tsx:159-170` 在 layout effect 恢复草稿，`formDraft.ts:81-86` 把 `undefined` 序列化为 `null` 保留主动清空。新增默认 Effect 必须读恢复后的完整 store，不能只看 `destination` 是否 falsy；否则清空的草稿会被新默认反复填回。
- `new.tsx:47-56` 按来源路径给创建内容 key，来源 A/B 与普通创建有不同草稿身份；`:146-154` 来源只初始化一次。默认状态必须放在这份 Form/草稿身份内，不能依赖跨订单共享 ref 或候选 query 缓存。
- `OrderFormTemplate.tsx:40-53` 观察程序回填并在已脏时保存完整快照，因此默认模式/用户覆盖意图需要随同 Form store 持久化；单靠 `isFieldTouched` 不能证明恢复草稿里一份空值是未触及还是主动清空。
- 不应把“当前 destination 与 discharge 恰好相等”自动推断成长期跟随授权。保存后的订单目的地是明确事实，共同段后来改港不能仅凭相等就改写它。
- `form-adapter.ts:228/:457` 当前用 `value || undefined` 传目的地，主动清空的服务端语义需与后端研究一起核对；本报告仅确认前端草稿会记录清空，不断言当前请求能清掉已保存字段。

### 10. 定向验证建议

优先扩展真实 Form / Query 测试：

- 现有 `SeaMasterBillFields.inheritance.test.tsx`：继承、采用共同段不改手动目的地；默认值跟随目标卸货港。
- `SeaMasterBillFields.identity.test.tsx` / `source.test.tsx`：换主单、指定来源航次、A→B→A 与迟到结果不覆盖手动目的地或草稿。
- 新增小型目的地默认行为 Suite：普通新建选卸货港、共享候选与来源创建默认；手动选择、主动清空、恢复默认、恢复草稿及卸货港后续改变。
- `sea-create-layout.test.tsx` / `sea-template.test.tsx` / `sea-form-labels.test.tsx`：同一共同段分组与“本票最终目的港”的清楚文案、默认态仅一处业务输入。
- `SeaTransportExecutionUpdateModal.test.tsx`：调整 payload 不含本票 destination，仍展示精确受影响订单，独立目的地不被统一调整。
- `form-adapter.test.ts`：共同段 discharge 和本票 destination 不同也原样组包，默认同港时两字段按真实值传递，显式清空语义与服务端一致。
- `sea-house-order-context.test.ts` / 新建来源测试：不复制来源分单的独立目的地，只由选定共同段给本票默认。

## Related Specs

- `.trellis/spec/web/frontend/sea-master-voyage-inheritance.md`
- `.trellis/spec/web/frontend/state-management.md`（初始化、草稿显式清空与程序回填）
- `.trellis/spec/web/frontend/component-guidelines.md`
- `.trellis/spec/server/backend/sea-export-document-contract.md`
- `.trellis/spec/domain/glossary.md`、`architecture-map.md`

## External References

本报告为内部源码研究，没有新增外部业务断言或引用第三方资料；此前会话的真实业务资料由主会话维护，不作为本报告的代码证据。

## Caveats / Not Found

- 未运行浏览器或测试；以上属于源码与既有测试断言的研究，不冒充新行为已经验证或实现。
- 后续具体转船航次/承运人/时间不在用户已选本期范围，当前 UI 没有该能力；不可把现有改配或多 TE 单选当成逐段运输管理。
- 只写本任务 research 文件，未修改产品、规范、任务状态或 Git。
