# 调研：订单详情移除持久草稿与有损离开确认

- 查询：移除已有订单详情的本地草稿存储和自动恢复；所有会丢输入的离开先确认，确认放弃；确定 dirty、版本与单证基线的最薄接入方式。
- 范围：仓库内部，只读产品代码；写入仅限本任务 research。
- 日期：2026-11-09。

## 结论

用户最新明确选择：切换、关闭或其他会丢输入的离开均先确认；确认后放弃修改。取消之前“切换页签保留输入”的要求。因此本期不做内存会话，也不做 KeepAlive。

最薄实现：已有海运、空运订单详情取消 draftPathname 与直接 storage 读取；复用现有模板事件 dirty + tabCloseGuard + useRouteLeaveGuard，再补浏览器 beforeunload。初始化保持干净；真实修改后任何导致 Outlet 卸载的路由变化先确认，取消保留当前活表单，确认后由 React 正常卸载。关闭一次确认复用 isTabDepartureConfirmed，避免页签和路由两次弹框。新建订单持久草稿和待确认创建幂等记录保持现有契约。

以下“内存会话备选”仅保留为为何未选择该方案的技术证据，不属于本期设计或实现范围。

## 文件定位与当前真实控制流

| 文件 | 职责与证据 |
| --- | --- |
| `web/src/pages/orders/detail.tsx:95` | 包装模板用 useState 固定首次 initialValues，后续后台重读不自动覆写编辑值。 |
| `web/src/pages/orders/detail.tsx:307` | 详情额外直接读取持久草稿；恢复时 version、MBL 正文版本、关联版本和 HBL expectedVersion 均取草稿原值，不允许拿新响应补齐。 |
| `web/src/pages/orders/detail.tsx:171` | `documentBaseline/currentDocuments` 独立于 Form store，保留本次编辑的单证基线。 |
| `web/src/pages/orders/detail.tsx:1345` | 模板身份为 kind+orderId+用户公司 scope；海运、空运详情均提供 draftPathname/draftScope，均触发模板持久化。 |
| `web/src/components/ui/order-template/OrderFormTemplate.tsx:168` | 只有 tabKey+draftPathname+draftScope 齐全才生成持久草稿键。 |
| `web/src/components/ui/order-template/OrderFormTemplate.tsx:189` | 首次可编辑且加载完成时 getFormDraft→setFieldsValue，并标 dirty；仅一次。 |
| `web/src/components/ui/order-template/OrderFormTemplate.tsx:351` | 真实 Form onValuesChange 标 dirty 并保存完整 store。 |
| `web/src/components/ui/order-template/OrderFormTemplate.tsx:41` | 子观察器监听 JSON 字符串快照，用户开始修改后，后续 setFieldValue/setFieldsValue 程序写值也落完整草稿；观察器不负责首次标 dirty。 |
| `web/src/components/ui/order-template/OrderFormTemplate.tsx:204` | applyValues 是显式用户带入入口：合并字段→dirty→完整快照。 |
| `web/src/components/ui/order-template/OrderFormTemplate.tsx:212` | resetTo 先 resetFields、再回填新值，清 dirty 和草稿。 |
| `web/src/components/ui/order-template/OrderFormTemplate.tsx:229` | onFinish 返回非 false 后清 dirty/草稿；失败返回 false 保留。 |
| `web/src/components/layout/formDraft.ts:79` | sessionStorage 序列化把 undefined 变成 null，用于保留显式清空；正是前次箱量恢复 null 的来源。 |
| `web/src/components/layout/tabCloseGuard.ts:110` | isTabDirty 同时读取活跃 guard 和 hasTabDraft；详情移除持久化后不能继续让残留 storage 草稿影响 dirty 判定。 |
| `web/src/components/layout/OrganizationWorkspace.tsx:27` | 用户/公司身份改变重建整个工作区 Provider，适合挂内存编辑会话。 |
| `web/src/app/AppLayout.tsx:237` | 直接 Outlet，普通导航卸载页面，不存在 KeepAlive。 |
| `web/src/pages/workbench/useWorkbenchOverview.ts:26` | 注释明确路由页签按 Outlet 卸载/重挂。 |
| `web/src/components/ui/list-query-state/index.tsx:16` | 已有工作区内存状态范本：Provider 内 Map、页签代际、同步事件写入、卸载不回写；可复用模式而不能混入列表查询 store。 |

## 最终推荐：有损离开确认的最薄改动

### 1. 移除详情持久草稿

- 删除 detail.tsx:307 的 restoredDraft storage 读取和其合并分支；Form 初始值、version、MBL/HBL 版本全部来自本次详情与单证响应。
- detail.tsx:1345 的模板继续提供稳定 tabKey 与 actionsRef，停止提供 draftPathname，从而不触发共享模板的存储/恢复；用户公司身份仍可沿用 draftScope 作实例 key 和外部变动查询 scope，命名暂不做无关全仓重构。
- 详情的 isTabDirty 不再附带 draftScope 扫描 storage；只看当前活跃 dirty guard。TagsView 关闭检测也不能让旧详情草稿继续触发关闭提示，应按当前消费者契约处理此边界。
- 不读取、不迁移遗留详情草稿；不误删新建/待确认创建记录。

### 2. 复用关闭和路由保护

- 详情通过现有 useRouteLeaveGuard 注册数据路由 blocker，包含页签切换、菜单跳转、页面返回、代码 history 导航和浏览器站内 back/forward。只有真实路由变化且当前活跃表单 dirty 时确认。
- 当前 useRouteLeaveGuard.ts:17 已复用 isTabDepartureConfirmed；TagsView 已先确认的关闭不再触发第二个离开确认。
- 在 useTabCloseGuard 或工作区统一层补 beforeunload，存在未保存活跃 guard 时 preventDefault + returnValue；使用浏览器原生文案，不承诺可自定义。
- 取消离开保留当前 Form 与所有版本；确认离开后正常卸载，不补保存、不存草稿、不恢复输入。
- isDirty 使用 dirtyRef 同步读，避免真实用户修改后立即关闭的状态提交窗口。

### 3. 保存、重置与结果未知

- 保存失败/校验失败保留 dirty；resetTo/原生 reset/明确保存成功按模板现有生命周期清 dirty。没有任何新旧草稿存储时，页面重新打开总是当前服务端值。
- 海运在途、unknown、refresh 状态保持 useSeaOrderSave 当前同键重试与冻结行为，不重写保存平台。活表单仍 dirty，因此离开前会提示；确认意味着放弃本地待处理状态，重新进入以服务端实际数据为准。
- 推荐关闭文案明确“未保存的修改将丢失”；保存结果待确认时不要声称“订单未保存”，因为请求可能已写入。若需追加“保存结果尚未确认，离开后请核对订单”提示，仅在已有状态上呈现，不引入存储或新重试层。
- 已知旧机制风险：useSeaOrderSave pending/status 存页面 ref，路由离开会丢失同键确认状态。该行为并非本期新增；有损离开必须先经确认，本期不承诺重新进入继续重试原载荷。创建接口的独立持久幂等记录不受影响。

### 4. 依然保留的基线保护

- 保留 OrderDetailFormTemplate 固定首次初值、documentBaseline、explicitRefreshTokenRef、外部变化 15 秒/回焦检查以及确认刷新后 resetTo；这些属于当前活表单正确性，不是持久草稿兼容层。
- 删除 storage 恢复版本分支后，当前编辑的 version/MBL/HBL expectedVersion 仍来自实际 Form store，不能保存时偷换最新 order.version。
- readonly 状态变化后已有 dirty 不得失去关闭保护；当前模板 enabled=!readonly 条件需要调整或在详情有独立 guard，避免被他人锁单后可无提示丢输入。

### 5. 最终验收重点

1. 初始化和候选完成后切页签/关闭不提示。
2. 修改客户业务编号后，切工作台提示；取消保留输入；确认进入工作台，再开详情只显示服务端值，不恢复旧 storage 草稿，也不触发箱量 null 崩溃。
3. 单关、关其他、关全部只确认一次；取消不关、不丢输入；当前页签不在批量范围时不能被误清。
4. 刷新/关闭浏览器页面只使用原生 beforeunload。
5. 保存/重置成功后离开不提示；保存失败或外部锁单后仍提示；确认刷新失败不清 dirty。
6. 海运与空运详情均停止 storage；新建草稿恢复和创建结果未知同键确认仍通过。

## 备选方案证据：切换保留需要哪些数据（本期不实施）

### 1. 表单值与初始化

- 按稳定 tabKey 保存唯一详情编辑会话；身份 key 仍保留 kind/orderId，用户公司由 Provider 生命周期隔离。
- 保存 `getFieldsValue(true)` 的完整当前值，包括未注册的 version、主单候选版本、MBL/HBL 正文版本、日期对象和显式 undefined。
- 内存快照不走 JSON 序列化，不产生“undefined→null→日期复活”链路；不用 sessionStorage、localStorage 或模块级全局 Map。
- 没有用户修改时不建脏会话；初始化、后台刷新、候选元数据变化不能首次标 dirty。
- 已有脏会话返回时，以原完整快照初始化 Form，并恢复 dirty；不要先以新服务端版本合并旧输入。
- 首次可写时的草稿恢复逻辑不能照搬到内存会话：若已编辑后外部锁单变成只读，回来仍须展示已保留输入并继续关闭保护。
- 事件/观察器同步保存快照，禁止卸载 cleanup 才写；页签已关闭或公司已切换后旧租约不能重建记录。

### 2. 海运单证与版本基线必须成组保留

`detail.tsx:793` 保存读取完整 store，用其中 version 构造 UpdateOrder；`sea-document-form.ts:72` 比较编辑值和 currentDocuments，并使用 expectedMblVersion、expectedLinkVersion、HBL ID/expectedVersion。

因此内存会话必须同时保留：

1. 当前完整 Form values（包含原版本）。
2. 编辑单证 baseline（`API.SeaOrderDocuments`），与上述 values 同时采纳。
3. dirty 标志。

`detail.tsx:663` 的独立主单/分单/模式命令会选择性推进 Form 版本并通过 `mergeSeaDocumentBaseline` 推进对应 baseline；其他未保存输入及无关版本保持原值。内存会话应同步这两部分变化，不能只监听一个字段快照，否则切换返回后可能用过期单证 baseline 误判未改内容为修改。

### 3. 待确认保存状态不能在切换中丢失

`use-sea-order-save.ts:30` 的 pending 是页面内 ref，包含同一 payload 和 committed；status 为 idle/saving/unknown/refresh。`use-sea-order-save.ts:39` 身份切换清 pending。现在路由卸载会天然丢失它。

工作区会话应保存该订单的最薄提交状态：原同键 payload、committed 和 status（或等价状态），使切换回来：

- unknown 重试原键原载荷，不形成新写入意图。
- refresh 已知写入成功，只重试读取，不能再写一次。
- 在途写入完成后，只有仍有效且未关闭的会话可采纳结果。
- 确认关闭/切换公司后，不允许迟到回调复活会话或污染新实例。

仅保留文本值而丢弃此状态会让“切换不丢输入”的承诺在真实保存异常时不完整。若不纳入本期，则须明确阻断会丢失待确认状态的离开；用户已选择有损离开先确认，本期不引入内存会话。

## dirty 判定与模板接入的共用证据

- 现有 template 的 dirtyRef 已适合事件驱动，不必为了此任务引入全字段深比较；用户首次真实修改、applyValues 显式带入才标脏。
- 继续保留程序回填观察：用户已修改后，相关派生字段与版本变化进入内存快照；纯初始化不能标脏。
- 用 `isDirty: () => dirtyRef.current` 注册 guard，可避开 state commit 前的立即关闭窗口；当前传 internalDirty 布尔值存在短暂时序差。
- 详情不再传 draftPathname，模板 storage 路径自然关闭；新建路径仍维持现有存储配置，不复制整套 OrderFormTemplate。
- shared template 的 `resetTo`、成功 onFinish、原生 reset 必须清当前 dirty；失败不清。独立单证业务命令不能误清其他普通字段的 dirty。
- `readonly` 是当前写权限/锁态，不应删除先前输入的关闭保护；当前 `enabled: !readonly && enableCloseGuard` 会在锁态变化后注销 guard，需要验证并收敛。
- 遗留详情 storage 草稿不读取、不迁移到内存；旧草稿不能继续作为页签 dirty 来源。无需全库扫 storage，也不应误删新建订单/待确认创建记录。

## 刷新、重置、外部更新的保留规则

| 场景 | 应保留行为 |
| --- | --- |
| 普通完整详情后台重读、锁状态同步 | 不覆写当前活表单输入和提交基线（保留 `OrderDetailFormTemplate` 首次初值语义）。 |
| 外部 15 秒/回焦检查 | `use-order-change-notice.ts:151` 比较实际编辑 Form 版本；即使服务端最新版本变化，旧 Form 不被换基线，继续提示。 |
| 用户确认刷新 | 最新数据成功后才通过 resetTo 替换 Form/baseline 并清 dirty；失败继续保留原输入、dirty、版本与提醒（detail.tsx:449）。 |
| 重置修改 | resetTo 使用最新明确选定快照，清当前未保存状态（detail.tsx:358）。 |
| 海运保存成功且重读完成 | seaSave reload 回调同步替换 Form/baseline（detail.tsx:188），模板完成清 dirty。 |
| 已写入但重读失败 | 保留当前输入和“重试读取”状态，不能自动重发写入。 |
| 订单 A→B | 不让 A 的迟到显式刷新 reset B；现有身份+token 保护必须保留（detail.tsx:252、411）。 |
| A→B→A，用户已确认离开 A | 重开 A 读取当前服务端值，不恢复旧输入；仍保留各次挂载间迟到请求的隔离。 |
| 确认关闭后重开 | 新表单以当前服务端值初始化，先前未保存内容不复活。 |

## 与新建草稿及创建幂等记录的边界

- `new.tsx:482` 继续提供规范 tabKey/draftPathname/draftScope；本期不撤销新建长表单持久草稿。
- `use-sea-order-create.ts:36` 使用独立 `roncin:pending-order-create:` 存储，保存原 JSON 规范化 payload+幂等键+成功 ID；用于超时刷新后确认是否已建单，绝不能按普通草稿一起移除。
- `new.tsx:242` 成功会先 flushSync resetTo 清 guard/草稿，再导航并 clear 待确认创建意图。共享模板变化需验证不重新弹放弃编辑确认。
- 用户当前已有未提交 `new.tsx/new.test.tsx/new-source.test.tsx/component-guidelines.md` 修改，本任务不要覆盖；只在必要共享影响下沟通合并。

## 受影响测试与改写原则

1. `OrderFormTemplate.test.tsx:210` 等：新建持久化、缺身份禁用、readonly 首次恢复、成功失败清理等保持；新增详情不传 draftPathname 时初始化干净/事件标脏/保存失败/重置成功的验证。
2. `OrderFormTemplate.draft-snapshot.test.tsx:35`：仍是新建草稿行为，不能因详情移除草稿而删掉；程序回填首次修改与显式清空语义继续保留。
3. `detail-draft-lifecycle.test.tsx:381` 起：测试 mock 自行模拟 sessionStorage 恢复，必须改为取消恢复后的实际契约；保留 A/B 隔离、迟到刷新、失败保留、重置与保存幂等有价值断言。
4. `detail-port-save.test.tsx:87`、`:120`：旧草稿注入应改为真实用户编辑或实际 Form 值，继续验证旧 version 冲突、缺版本不借新基线、确认刷新采纳新版本；另加遗留草稿被忽略。
5. `detail-port-save-master.test.tsx`：独立单证动作的 MBL/HBL/共享航程版本不得顺带升级，保留。
6. `detail-change-notice.test.tsx:67`：把 storage 断言改为实际 Form/提交版本和 isTabDirty；保留取消刷新、刷新失败、外部提醒、确认框换单失效。
7. `detail-termination-refresh.test.tsx:58`：去掉 storage 预埋，使用真实编辑保留输入与原提交版本；状态从可编辑变只读后依然关闭保护。
8. `order-container-draft.test.tsx`：前次 null 修复覆盖真实序列化，改路径到新建仍有价值；另新增详情确认放弃后重开不恢复旧 null 的回归。
9. `use-sea-order-save.test.tsx`：同键重试、committed 只重读及旧身份迟到不污染新实例继续通过，不为此任务扩成内存恢复平台。
10. `new-create-idempotency.test.tsx`、`use-sea-order-create.test.tsx`：作为相邻路径验证，保护原创建快照与待确认语义。
11. 真实 data router 集成验证：改客户业务编号→工作台→取消/确认→再进详情；关闭单签/其他/全部复用一次确认、刷新 beforeunload，以及保存或重置成功不再提示。

## 相关规范

- `.trellis/spec/web/frontend/state-management.md`：草稿身份、显式清空、创建待确认、详情外部变化和显式刷新生命周期。
- `.trellis/spec/web/frontend/organization-switch.md:18`：用户公司切换重建工作区；离开失败保留输入。
- `.trellis/spec/web/frontend/component-guidelines.md:328`：表单生命周期由 OrderFormTemplate 所有，详情扩展不能暴露 dirty setter/草稿键。
- `.trellis/spec/web/frontend/component-guidelines.md:518`：已有订单后台重查保持表单挂载，不用骨架卸载输入。
- 根 `AGENTS.md`：保存必须校验服务端 expectedVersion，前台外部变化检查不可自动覆盖未保存输入或替换基线。

## 外部资料

无；本调研以当前仓库代码与测试为证据，未涉及需要查询外部文档的 API 新用法。

## 限制与未发现项

- 未发现现有页面 KeepAlive，也未发现订单详情专用内存编辑会话；列表内存查询 Provider 是可参考模式。
- 未运行测试或浏览器复现；本轮为规划研究，不修改产品代码。
- 弹窗独立 dirty、页签批量关闭、beforeunload 及离开导航确认由另一研究主题负责；这里仅记录与订单会话的接点。
