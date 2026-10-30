# 共享箱工作台独立 review

## 审查范围

对照本任务 R2/R3、AC2/AC3，审查 SeaSharedContainerDrawer.tsx 及错误状态、原工作台两组测试；本轮 reviewer 未编写该前端实现。追踪查询成功/错误/空态、候选重试草稿、关闭和订单/航程切换迟到、写操作权限和成功反馈。

## 发现及修复

P1：两个新 query 使用 `Array.isArray(response.data)` 作为必要成功条件，但服务端 `marshalProtoJSON` 未开启 EmitUnpopulated，合法空 repeated 集合被省略。真实成功报文可以是 `{success:true,message:"OK"}`，新实现会把无共享箱/无候选订单误显示为加载失败。

已局部修复：仅在 `success === true && data === undefined` 时按照正式 ProtoJSON 空集合语义规范化为 `[]`。仍拒绝 undefined 响应、无成功标记且无 data、`success:false`、非法 data 类型。现有带数组的响应继续正常消费。成功空集合回归新增缺省 data 变体，两组测试通过。

## 通过核对

- 403（HTTP 与业务封套）、5xx、网络失败、请求层 undefined 与真正空集合分别反馈；共享箱失败不渲染 0、暂无或正常创建入口。
- 列表与候选各自 query/refetch，候选失败重试不重查列表、不清 drafts；本地草稿仅按共享箱身份/版本初始化，人工编辑后不被候选分页或刷新覆盖。
- 外层按订单、航程、open keyed 重挂；每次打开独立 query 实例；关闭、切订单/航程的旧响应不写新实例。
- 写操作仍携带 orderId 锚点及 expectedVersion；按钮消费 canCreate/canUpdate/canDelete，父页结合业务锁与公司写范围传入。服务端继续独立鉴权；此处不新增或放宽权限。
- 无读成功态时隐藏共享箱卡片和写入口；候选错误保留已经加载的箱详情与本地草稿。

## 验证与边界

- 两测试文件共 18 项通过，12.49 秒整轮；新增省略 data 变体通过。
- 修改文件 Biome 通过；git diff --check 通过。
- `pnpm --dir web tsc` 通过；全前端门禁由主会话归集。
- 本轮未修改领域守恒、事务/版本、共享箱新建弹窗体系；没有剩余已确认的 R2/R3 阻断缺陷。实际视觉/键盘以主会话浏览器验收为准。

## 最终草稿竞态独立复核（6246344c）

已独立检查实现者追加的 `useLayoutEffect` 初始化和 `draftsRef` 快照修复，以及 `SeaSharedContainerDrawer.initialization.test.tsx`，本评审者未编写该修复。

- 初始化 effect 在提交阶段同步完成当前共享箱初始草稿，首个可交互界面不再等待被动 effect 回填。原有 ID/版本初始化判断、人工编辑后刷新不覆盖及切箱初始化规则保持。
- 全部四条草稿赋值路径（空选择清空、箱分配初始化、手动改数量、全部填入）均经过唯一 `setDrafts` 包装；包装先同步更新 ref，再更新渲染 state，未发现绕过入口直接修改 state 的路径。
- 保存请求与确认请求都用 `buildAllocationInputs(draftsRef.current)`；确认守恒检查也用当前 ref，避免只修载荷但校验仍读取旧渲染摘要。UI 的列表与摘要仍由 state 驱动，不在渲染期间写 ref。
- 新测试覆盖首次候选渲染被动阶段填入，以及同一事件批次将12改7后立即保存；第二个场景没有等待重渲染，能捕捉旧闭包。实现者受控复现保留 layout 只回退载荷读取后 expected7/received12，证据与源码因果一致。
- 本轮只读复核，没有修改产品或临时回退版本。主会话记录的最终完整测试340文件1830项、lint/tsc均通过；不重复运行全门禁。

结论：上述两处修复共同覆盖初始化覆盖和提交旧闭包两个窗口，没有发现新的阻断问题，可进入合并收尾。
