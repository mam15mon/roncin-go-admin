# 独立评审：权限路由与实际箱入口

评审范围：本评审者未编写的 R1 后端 HTTP 注册顺序修复，以及 R4/R5 实际箱入口、摘要与独占箱写后通知。R2/R3 共享箱查询由本评审者实现，不在本份独立评审结论范围内。

## 结论

已检查的 R1/R4/R5 代码没有遗留阻断问题。后端修复只改变静态共享箱集合和动态订单详情的注册顺序，没有放宽权限、组织范围或请求锚点要求。实际箱通过已有 ContainerDrawer 维护；写入后只失效 actual-containers 查询，未调用整票 resetTo、loadData 或 refreshOrderAndLock，因此不会主动覆盖未保存的订单字段或计划箱量。

## 发现并修复

1. 实际箱摘要最初要求 data 必须为数组；后端 ProtoJSON 对合法空 repeated 字段会省略 data。这会把真实空箱列表显示为错误。改为仅当 success=true 且 data 缺省时按空数组处理；undefined 整体响应、success=false、无成功标识的缺省数据及非法 data 仍视为失败。补充成功省略 data 的回归。
2. detail-shared-container-workbench 的调用断言仍只有一个参数；同步新增 skipErrorHandler:true，保持真实请求签名验证。
3. 实际箱贡献类型注释位置错位，已就地修正。
4. 普通实际箱查看入口和只读查询重试的 disabled=false 经真实父 Form disabled 测试确认有必要：审核锁限制写入，不应封锁查看。尝试删除后锁定查看测试明确失败；按主会话决定保留并增加中文注释，未增加新上下文抽象。Drawer canCreate/canUpdate/canRemove 仍全部受 businessWritesDisabled 和具体权限约束。

## 验证

- 前端四个定向文件（detail-shared-container-workbench、SeaActualContainers、ContainerCargoDelete、SeaExportDetailFeatures）初轮 21 项通过。
- 后端真实 HTTP 鉴权/绑定、正常订单详情路由及共享箱鉴权中间件定向测试通过：`go -C server test -p 32 ./internal/server -run 'TestSharedContainerHTTPAuthorizationAndBinding|TestSharedContainerRegistrationPreservesOrderDetail|TestAuthorizationMiddlewareSharedContainerRequests' -count=1`。
- 补空 repeated 语义与保留只读查看后再次运行实际箱/共享箱详情两个文件，7 项通过；类型检查、修改文件 Biome lint、git diff --check 通过。

## 边界

HTTP 测试以真实路由、鉴权中间件、Service 和 Usecase 验证权限及查询绑定，只替换仓储；它不能证明真实数据库中锚点与航程关系。该关系沿用原 data.ensureSharedAnchorExecution，需专用数据库集成测试，SKIP 不计通过。浏览器双分辨率、实际服务版本及截图对应运行实例由主会话验收，不在本报告中虚构结论。列表旧实际箱入口保持原组件和权限传递；新增 onChanged 为可选回调，不影响未提供该回调的旧调用方。

## 全量验收追加：共享箱草稿竞态修复证据（实现者记录）

全量测试发现首次“全部填入→保存”偶发缺分配。草稿初始化由被动 useEffect 改为 useLayoutEffect，确保界面交互前先初始化，并保留切箱初始化及已编辑刷新保护。另用唯一 setDrafts 同步推进 draftsRef，提交及确认守恒检查读取当前事件快照，避免连续操作读取上次渲染闭包。

新增 initialization 回归：候选首次可交互时填入，以及同一个点击事件中先将件数从12改7、立即点击保存且中途不等待渲染。受控复现只把提交读取改回闭包 drafts（保留 layout 修复），第二项稳定失败：expected 7, received 12；日志 /tmp/shared-draft-stale-closure-repro.log（临时产物不入库）。恢复 draftsRef 后两项通过；原9项同时通过，tsc通过。这证明提交快照修复具有实际必要性，不以延迟测试掩盖问题。
