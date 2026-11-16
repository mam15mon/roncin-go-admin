# 四类订单横切与完整导出：服务端验证

## 实施边界

本组按已批准父设计接通SE/SI/AE/AI已有业务事实的读取与输出，不增加新业务类型、独立费用算法、流程平台、业务标签、异步导出平台或新权限。所有写入/锁/版本保持已有合同；代码和生成物未提交，最后整体验收由主会话统合。

- `OrderService.ExportOrders`为`GET /api/v1/orders/export`，独立无page/pageSize请求，`businessType`必填且只接受本期四类；余列表筛选与排序全部沿现行语义。标准响应携`Order[] data`、`total`、`generatedAt`及trace字段。
- service复用`orderListOptionsFromAPI`作唯一筛选转换和真实类型权限解析；沿`orderToAPIForPrincipal`裁剪箱号/附件optional数量及动作，不能借另一类型或系统治理权限。显式非法列表businessType也直接拒绝，不把它误当未筛选全部类型。
- biz共用筛选规范化，普通List仍验证1..200，Export拒绝任何非零分页；仓储抽出`orderListQuery`唯一筛选内核，Export直接完整All一次，不分页拼接或巨型pageSize。全部行、资源摘要、海空主分单投影在`Data.withReadSnapshot`中经`Data.client(snapshotCtx)`读取；total取同次完整结果行数，生成时刻明确。
- 静态Export RPC在动态GetOrder之前声明；HTTP中间件识别Export真实businessType，不能把它当详情ID或只因有任一类型read放行。
- `OrderContainerRequest`追加只读`containerSpecCode/containerSpecName`。biz纯展示值、service转换，data批量按真实`KindContainerSpec`取名称代码，Get/List/Export共用；不按Enabled过滤历史项，不新增列。Export内字典名称也属于相同RR快照，前端无需另查会变化字典。
- 工作台范围解析、本人协作ID、费用/异常交集及近期订单摘要/分页均覆盖四类，仍限定当前公司、本人真实OrderPersonnel及各类read+分类read。没有扩大为管理员全公司查询。
- 通知费用动作和受控内容准备覆盖四类，URL必须匹配当前资源UUID。系统通知测试页仍无经营读取资格、不能选真实Order；预览验证只用纯render和已有preparer，不调用网络发送。
- 财务台账主单搜索/展示已有SE/SI和AE/AI真实关联实现，本组扩展其真实PG矩阵至四类，确认活动关系与本次终止保留关系、旧关系排除、公司范围及完整汇总。经营日期报表已有出口ETD/进口ETA及航空权威主单路径，本组实际重跑已有跨中国午夜专项，不把创建接通代替日期证据。

## 本期中文边界修复

主会话最终F表审计发现领域按Unicode字符计数而Ent按UTF-8字节限制。本组仅将本期已有用户自由文本按原字符上限同步MaxRuneLen：Order客户/内部/订舱号100、合约100、工厂及收发简称200、船名航次100、货描/特殊要求/四备注1000；AirMasterWaybill.notes1000；SeaTransportExecution及不可变Version的vessel_name128/voyage_no64。机器标识、幂等键、币种、ISO时刻、UN等未机械替换。

所有Schema表列类型、Size、索引、CHECK和迁移生成声明未变化；正规Ent生成后除Schema源validator之外无生成差异，因此不创建空SQL迁移。航空仍拒绝夹带海运VesselVoyage，未为测试或中文长度放宽业务适用性。

危险品取消清空沿既有optional存在性合同：service真实ProtoJSON解析“未传”保留原UN/CLASS/申报截止/PG，“显式空+PG UNSPECIFIED=0”清除；真实PG用例验证合并后完整领域值保存和重读，不在data测试反向依赖service。最终真实HTTP验收由主会话脚本衔接，本报告不把分层组合冒称直接HTTP写库。

## 已执行定向结果

- `go -C server test -p 32 ./internal/biz ./internal/service ./internal/server -run 'OrderExport|Notification|WorkbenchTodo|AirDangerousOptional' -count=1`：三包PASS。包含四类专用导出真实HTTP生成路由/Authorization/RequiredFields/Service/Usecase（只stub仓储）、静态Export与合法UUID详情实际operation/handler、同类公司成功/他类与系统拒绝，service筛选转换/false存在性/子资源裁剪/非法参数，以及纯通知无发送和工作台交集。
- 首轮真实PG：`TestOrderExportCompleteSnapshotPostgres`3.16秒PASS、`TestOrderBusinessTextUnicodeBoundariesPostgres`2.53秒PASS，均0 SKIP。完整四类真实建票与205条真实DIRECT关联、中文多行/零/6.7183、普通200上限/201拒绝、公司隔离、升降序、重复读快照与写事务拒绝。
- 扩展真实PG：`TestFeeLedgerMasterNoPostgres`四子场景SE/SI/AE/AI全部PASS10.68秒；`TestFinanceReportAuthoritativeAirAndImportDatePostgres`PASS2.39秒；`TestOrderExportCompleteSnapshotPostgres`PASS3.16秒；`TestWorkbenchFourTypeIntersectionPostgres`PASS2.58秒。均0 SKIP。
- 扩展中文表格时一次失败为夹具将海运专属VesselVoyage塞进航空请求，明确AIR_WAYBILL_INVALID；修正夹具不放宽产品后，`TestOrderBusinessTextUnicodeBoundariesPostgres`PASS2.68秒，AE/AI真实创建/重读、合法中文与Ent+1、危险字段保存后重读均通过。后续海运船名独立真实PG首次确认旧不可变Version字节validator阻断，已按同字段链同步修复。

数据库连接仅通过`.env.local`在Node子进程环境读取`DATABASE_SOURCE`并注入`RONCIN_INTEGRATION_DATABASE_SOURCE`，不输出凭据。全部数据用已有`getIntegrationData`随机隔离Schema和正式有序SQL迁移，结束只清理本次Schema；未清开发库或使用Schema.Create。普通Go测试的SKIP不计为真实PG通过。

## 最终收口结果与冻结

- 最后真实PG命令为`go -C server test -p 32 ./internal/data -run '^TestOrderExportCompleteSnapshotPostgres$|^TestSeaOrderVesselVoyageUnicodePostgres$' -count=1 -v`，通过安全Node子进程环境注入连接。最终整组 **PASS 5.751秒、0 SKIP**：完整导出与箱计划RR快照3.13秒，SE/SI中文船名航次及不可变快照2.55秒。日志`/tmp/hgj-cross-final-snapshot-unicode-pg.log`。
- 箱计划测试创建真实3箱计划及`EXPORT-OLD/原箱型名称`，先建立RR快照，再由外连接修改订单备注、客户名称和箱型名称；同一Export的订单与名称投影保持旧值，下一请求读取新值。箱型code本身是既有immutable字段，测试不伪造修改code或用原生SQL绕过；名称变化足以证明同一快照边界。
- 海运测试通过真实SE/SI Create→Get验证显式`船名 / 航次`，包含100字符组合及64字符中文航次；通过真实事务创建不可变版本，读回完整船名和航次。当前/版本validator的128/64原限不变；另在正式Schema下验证合法上限和+1拒绝。
- 最终Go定向三包通过：`OrderExport|Notification|WorkbenchTodo|AirDangerousOptional|OrderList|SeaMasterBillCandidateRoute`，biz0.036秒、service0.035秒、server0.086秒。包含原列表筛选/排序及候选静态路由相邻。
- `go -C server vet ./...`最终通过；`buf lint`最终通过；所有本组非生成Go经gofmt，`git diff --check`通过。
- 正规`make -C server api`、`pnpm run generate:web-client`、`go -C server generate ./internal/data/ent`完成。在最终源定型后，对1139个Proto/绑定/OpenAPI/client/枚举/权限规则/Ent相关文件作SHA256，再原样重复三生成入口，**变化文件数0**。`ent/migrate/schema.go`相对HEAD无差异，不添加空迁移；Manifest未改，不产生新权限。
- 最后箱计划增量初次验证编译报试图调用immutable MasterDataItem.code的不存在setter，已将并发变更收敛到真实可改name；随后业务断言通过但单票fixture的逐表清理漏新计划/版本FK而整例FAIL。按项目既有复杂场景`deferCleanupToSchema=true`交随机Schema辅助入口统一清理后重跑取得真正PASS，未关闭业务校验或忽略cleanup失败。首轮失败不计最终通过。

全部server产品源、tests及所有生成物写权现已冻结并交回主会话，无未修复确认问题，未提交或推送。研究报告不再编辑。前端、真实财务/CSV文件及父全量门禁由主会话与相应负责人继续验证，本报告不提前宣称它们通过。
