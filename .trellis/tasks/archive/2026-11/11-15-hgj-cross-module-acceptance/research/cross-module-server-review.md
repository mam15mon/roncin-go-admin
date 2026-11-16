# 四类横切与完整导出：服务端独立复核

## 范围与结论

2026-11-16。已复核本组全部服务端源、Proto/绑定/OpenAPI权限与客户端生成差异、
新增服务/路由/数据库验证文件及现有财务台账号测试扩展，未发现需要修复的确认缺陷。
没有修改产品代码、测试或生成物。本报告不代替真实四类财务/CSV浏览器和最终门禁。

已读取保存的Trellis hook、当前PRD/design/implement与本组相关check上下文、实现验证报告，
并显式核对根/server AGENTS、server/backend和web/frontend的Quality Check。
包发现只列web时仍按实际server变化检查后端；数据库规范、横切导出及工作台/通知合同按实际文件补读。

## 完整导出与跨层检查

- Proto新增独立`ExportOrdersRequest/Response`和静态`GET /api/v1/orders/export`，
  请求无分页字段、业务类型REQUIRED；静态RPC位于动态GetOrder前。生成绑定、
  OpenAPI/client和HTTP权限规则一致。真实路由测试断言operation/handler及合法UUID详情，
  无只按生成文本推断分发的假验收。
- service只做转换和用例调用；以同一`orderListOptionsFromAPI`解析完整筛选、排序、
  optional false和当前公司/实际业务类型read范围，使用原`orderToAPIForPrincipal`
  裁剪受保护子资源。缺类型、非法类型、其他业务/系统工作台均不能借权导出。
- biz `normalizeOrderListFilter`为列表和导出共用内核；普通List保持1..200分页，
  Export仅接受SE/SI/AE/AI且拒绝领域层非零分页，不增加巨型pageSize或循环翻页。
- data一次All完整查询，过滤和排序复用原`orderListQuery/orderListSort`；所有行、
  本票资源、航空事实、海运主分摘要及箱型名称均在`withReadSnapshot`内，辅助仓储
  经`Data.client(snapshotCtx)`取得同一事务客户端。没有连接池旁路或把写事务当读快照。
- 新增箱计划code/name为纯只读投影：按真实spec ID和ContainerSpec Kind批量读取，
  不过滤停用历史项；Get/List/Export统一，service不访问Ent。无数据库新列、
  输入可写字段或新权限。投影没有进入创建意图哈希，箱型改名不改变原计划ID/数量事实。
- `total`来自本次完整行数，`generatedAt`明确新读取时刻，不宣称冻结此前页面行值。
  名称、附件optional零与未授权继续沿原领域/ProtoJSON语义，未恢复业务标签或提成列表摘要。

## 工作台、通知和字符边界

- 工作台service、本人协作ID筛选、近期摘要/分页及两类待办均接通四类；本人、公司、
  订单read与同类型fee/abnormal read仍取交集，没有管理员全公司或跨类权限拼接。
- 正式通知费用路径与模板识别支持四类且绑定精确资源UUID；未知类型和空ID无动作。
  系统通知测试页原资格保持，系统工作台不能读取真实经营订单，纯render/preparer
  测试不调用发送，没有为验收引入外部网络或新通知通道。
- 财务台账真实主单与经营日期源原实现未重写；本组扩展PG至四类并验证现有出口ETD/
  进口ETA及中国自然日，金融金额/汇率算法、账单快照和写事务/锁序保持前置合同。
- MaxRuneLen只同步本期已有用户自由文本的既有字符上限；海运当前值与不可变
  航程版本的船名/航次同链一致，机器标识/币种/ISO/UN未全仓替换。
  `ent/migrate/schema.go`对HEAD无差异，无需空迁移，未修改正式SQL或历史内容。
- 航空危险字段清空的后端行为未改变：service真实ProtoJSON/merge测试分别证明
  optional省略保持与显式空/PG0清除，PG领域保存证明最终持久化；两者未被冒称
  单条HTTP写库，最终HTTP验收由对应脚本完成。

## 本人实际执行

| 检查 | 结果 |
| --- | --- |
| `go -C server test -p 32 ./internal/biz ./internal/service ./internal/server -run 'OrderExport\|Notification\|WorkbenchTodo\|AirDangerousOptional\|OrderList\|SeaMasterBillCandidateRoute' -count=1` | PASS；biz0.039秒、service0.037秒、server0.086秒 |
| `go -C server vet ./...` | PASS |
| `buf lint`（server目录） | PASS |
| 全部本组非生成Go源码和测试 `gofmt -l` | 无输出，PASS |
| `git diff --check` | PASS |
| `ent/migrate/schema.go`对HEAD | 无差异 |

## 沿用证据与未重复项

已读实现报告`cross-module-server-validation.md`及相关实际测试源；其真实PG结果
包括四类完整投影、205行完整排序、箱计划/客户/订单同RR快照、普通200/201边界、
跨公司与写事务拒绝、四类工作台本人权限交集、四类台账号/权威日期、Unicode边界、
AE/AI危险信息持久清空、SE/SI中文船名及不可变版本。最后两项整组5.751秒0SKIP
真实PASS，早期错误夹具/immutable setter/cleanup FK失败和修复均在报告保留。

实现已按正规Ent/API/OpenAPI/web生成入口完成1139文件重复SHA零差异。本复核静态
核对全部源和生成差异，未再次无因生成，也未无因重复已通过的真实PG；这些数据库
与生成结论明确沿用实现证据，不能由普通Go无数据库的跳过项推导。

## 发现与交接

已修复确认问题：无。未修复确认问题：无。全部server源/tests及生成物写权已冻结
并交回主会话；未提交或推送。前端独立复核、真实四类财务/CSV文件、最终check:fast
及生产构建由对应负责人继续统合，不因本报告完成标记父任务已完成。

## 最终全量门禁后的测试合同修正

主会话首轮`check:fast`实际exit1，日志`/tmp/hgj-cross-check-fast.log`中仅确认
服务端`TestFeeReviewMarkdownPreservesTrustedOrderEntry`两通知模板的旧“未注册品类”
负例仍使用`/orders/air-import/{UUID}/fees`。本组已明确接通AI，该路径现在是合法
动作，旧测试预期应同步；首轮失败不算通过，未完成的Web门禁也不能据此算通过。

按主会话限定责任，只修改`server/internal/biz/notification_test.go`该单条夹具，
改用本期未开放的`/orders/land/{UUID}/fees`作为拒绝负例并相应命名。原SE/AE合法
路径、资源UUID不匹配、伪造外链、无入口等断言保留；新增
`notification_four_types_test.go`对SE/SI/AE/AI及两通知结果的合法动作和错资源拒绝
仍保留，没有重复扩写或放宽生产允许路径。产品、生成物、权限、spec和验收脚本均未改。

本人修后执行：`go -C server test -p 32 ./internal/biz -run 'Notification|FeeReviewMarkdown' -count=1`
实际PASS（0.025秒）；该文件`gofmt -l`无输出、`git diff --check`通过。
本次仅测试修正，不要求重建生产包或停止正在运行的隔离浏览器。该文件及本报告
写权现再次冻结交回；后续完整门禁由主会话重新执行，本文不提前声明最终门禁通过。
