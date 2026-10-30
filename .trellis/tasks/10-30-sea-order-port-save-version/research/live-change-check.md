# 海运详情资料变动提醒：最小接入研究

## 当前确认范围

当前已批准行为为：海运详情前台停留期间每 15 秒轻量检查，切回页面立即检查；实际发生变化时常驻提示「这个订单已经发生变动，请重新刷新。」。检查只提示，不覆盖用户输入或编辑版本。刷新有未保存内容时先确认，保存仍由服务端校验兜底。全站多人编辑已有单据采用同一规则，但本任务只落地海运详情，不迁移所有表单。

## 1. 脏状态与刷新入口

- `web/src/components/ui/order-template/types.ts:15` 的 `OrderFormTemplateActions` 仅有 `resetTo`；内部 `dirtyRef` / `internalDirty` 由模板独占，不能从 actions 读取。
- 无需扩展模板接口：`web/src/components/layout/tabCloseGuard.ts:97` 的 `isTabDirty(tabKey, draftScope)` 同时读取当前表单守卫与持久草稿 `hasTabDraft`。它覆盖恢复草稿、保存失败、存储不可用时仍存在的内存修改，比 `Form.isFieldsTouched()` 更贴合现有未保存语义。
- `OrderFormTemplate` 用户编辑时注册脏守卫并将完整 store 持久化；`resetTo` / 保存成功清理草稿与脏状态。候选回填等程序变更不单独标脏。
- 详情已有 `refreshOrderDataAndResetForm`，通过身份与 `explicitRefreshTokenRef` 保护完整刷新、重置与迟到结果；刷新确认应包在调用它之前，取消不调用、不清草稿，失败沿用原输入。
- 详情顶部「更多 → 刷新数据」和新增提示中的「刷新」应复用同一确认入口，避免一处保护一处直接覆盖。确认框须在确认执行时重验订单/工作区身份；已有 `useModalOwnerSignal` 只覆盖组件卸载，A→B 同组件切换还要显式核对订单身份。

## 2. 内部页签和浏览器前台

- 仓库不存在 KeepAlive。`TagsView.tsx:216` 点击页签只 `history.push(tag.path)`；`AppLayout.tsx:237` 仅渲染单个 `Outlet`。切到其他页面后原页面卸载；同一路由订单参数变化时，详情表单通过 `key={orderFormIdentity}` 重建。
- 因此不需要给全站布局新增「页签激活」事件或维护隐藏实例清单。返回订单页会复用现有详情查询加载/草稿恢复；监测 Hook 随当前订单身份启动即可。
- 浏览器返回需要同时考虑 `focus`、`blur`、`visibilitychange`。已有 `components/ui/order-list-template/useOrderAutoRefresh.ts` 展示前台监听模式，但它是用户勾选的 60 秒 ProTable 轮询，不能直接拿来充当详情检查。
- `document.visibilityState === 'visible'` 配合窗口焦点可停止后台检查。相同一次前台恢复通常连续触发 focus/visibilitychange，应按「后台→前台」状态跃迁触发一次，不能每个事件都请求。

## 3. Query 接入与去重

- 保持 `utils/queryClient.ts` 的全局 `refetchOnWindowFocus:false`、`refetchOnReconnect:false` 与禁用自动重试策略。
- 建议海运详情专属监测 Hook 使用独立 `useQuery`；key 包含工作区/组织、品类、订单 ID，查询只调用 `orderServiceGetOrder({ id }, { signal, skipErrorHandler:true })`。读取失败抛错，`meta:{silent:true}`，不能把 undefined/失败信封当成「没有变化」。
- 周期调度交给 Query `refetchInterval:15000`，后台/无合法加载身份时停用；`refetchIntervalInBackground:false`。可见性监听只控制生命周期与触发 `refetch`，不另写 data/loading/request 竞态系统。
- 局部浏览器恢复触发 `refetch({cancelRefetch:false})`，同 key 在途读取由 Query 去重；同时按前台状态跃迁去重事件。不要同时打开另一套全局 focus 自动查询。
- TanStack Query 5.103.1 默认 focusManager 只监听 visibilitychange（本地 `node_modules/.../query-core/src/focusManager.ts:16`），单独使用 `refetchOnWindowFocus:true` 无法覆盖返回原本仍可见的浏览器窗口场景，故仍需局部 focus 处理。
- 页面首次已完成完整详情加载时，可将当前订单响应作为监测初值，避免同一时刻重复初始 GET；之后轮询/真实恢复前台才主动重读。内部页签返回已有完整详情查询，不要同时无条件再发另一套初始 GET。
- `OrganizationWorkspace.tsx` 按用户/组织 key 重建并清理 QueryClient；生成请求支持 AbortSignal。monitor key 与组件身份必须一致，A→B→A 不消费旧实例结果。

## 4. 轻量版本指纹

单次 `GetOrder` 足够，无需周期调用当前 `useOrderDetailData.loadData()` 的主数据/人员/完整单证聚合。

| 受保护内容 | GetOrder 可比较字段 | 证据 |
| --- | --- | --- |
| 订单资料 | `id`、`version` | `order.proto` 的 `Order` |
| MBL 身份与正文 | `seaMasterBill.masterBillId`、`version` | `order.proto:875`；`order_convert.go:240` |
| 实际航程 | `seaMasterBill.transportExecutionId`、`transportExecutionVersion` | 同上，TE 有独立版本 |
| 单证模式/活动关联 | `seaDocumentLinkVersion`、`seaDocumentStructure` | `order.proto:410`；也可由 `seaDocumentSummary.linkVersion/documentStructure` 取得 |
| HBL 内容/号码/状态 | 活动 Link 版本 | GetOrder 不返回独立 HBL 版本，但 `updateLockedSeaHouseBill` 更新 HBL 后推进 Link（`sea_document.go:368`）；正式 HBL amendment 与 mode 也推进 Link（`sea_document_change_amendment.go:200`、`sea_document_change_mode_change.go:210`） |

版本可直接按规范化字符串做相等判断，不能转 `Number`；身份变化、关联消失和模式切换同样视为变动。包含 TE 版本，避免只检查订单/MBL而漏掉共享航程更新。

## 5. 基线与本次保存的竞态

- 检测比较基线必须来自当前编辑快照，而非监测刚读到的新版本。草稿恢复使用草稿自身版本；已有完整 Form store 中保留订单、MBL、TE摘要、Link等版本/身份。
- 监测结果只进入常驻提醒，绝不 `setFieldsValue`、推进版本或重置表单。多次发现同一变化不反复弹 notification；检测失败不得清除已有提醒。
- 保存进行中、结果未知或已提交待重读期间暂停外部变动判断，避免把自己的在途写入误报成外部变动。成功完整重置后更新监测参考点；不能再用保存前缓存的监测结果与新基线比较并立刻误报。
- 显式刷新、普通保存和正式单证动作各有既有版本推进规则，保持它们的差异：正式动作重读只允许采纳可证明的一次增量；更晚并发写入仍保留旧编辑版本并提示刷新。
- 原订单快照检查在途期间用户保存/刷新，旧检查返回后不得对新基线重新设置提醒；建议监测 query 身份或本地消费范围包含基线代次，交给 Query 隔离结果，复用已有显式刷新令牌处理 UI 副作用，不再增加全局状态。
- 用户确认刷新后、完整刷新结束前继续编辑也是需要覆盖的路径：沿用提交/刷新冻结或在完成前检查用户编辑状态，不能清除请求发起后新增的输入。

## 6. 建议新增回归

1. 当前详情前台停留，15 秒前不请求，15 秒检查返回新订单/MBL/TE/Link版本后出现常驻提示，输入与原版本不变。
2. 无变化不提示；相同变化连续多次不重复弹窗；失败不伪装为一致，不清既有提醒。
3. 隐藏/失焦停止轮询；恢复前台立即检查，focus+visibilitychange 同次只请求一次；未激活订单不在后台请求。
4. 干净页面刷新直接重置；有普通输入/恢复草稿时先确认；取消、失败都保留输入；成功刷新清提醒与草稿。
5. A→B、A→B→A、同用户换组织，旧检测、旧刷新和旧确认不得更新当前页面。
6. 自己保存成功不误报；检查在途后自己保存/显式刷新，旧检查迟到不重新告警；未知保存结果期间不自动覆盖。
7. 仅HBL变化（Link推进）与仅共享TE变化都可探测；uint64大版本按字符串比较不丢精度。
8. 保留真实 Form、独立 QueryClient、真实焦点事件；时间驱动只使用假定时器，避免增加重型单文件长尾。

## 7. 保存冲突文案增量复核

已只读复核 `use-sea-order-save.ts`、其测试与详情缺基线文案：并发提示只在 HTTP/业务 code 409 且 reason 属于明确并发集合时收敛，号码占用、主单更正限制与航程业务冲突保留原原因。`getRequestErrorStatus` 已支持 response 与 data 两种现有信封；4xx 清未知提交快照后回到 idle，未自动刷新或重试，不改变输入保留规则。未发现此增量中的可操作问题；实施代理已报告相关 49 例、tsc、Biome通过，此处不重复运行。

## 8. 首轮门禁噪声

`/tmp/roncin-sea-port-check-web.log` 的 3 个 ECONNREFUSED AggregateError 均为 localhost:3000，日志无测试文件/请求 URL，不能据此可靠归因到某个未 mock API。相同数量与模式也存在本任务之前的 `/tmp/order-detail-footer-check-web-final.log`，属于既有噪声；本次新增定向港口回归无 stderr。未修改无关测试，也未把网络失败当作本任务测试已验证的后端错误路径。
