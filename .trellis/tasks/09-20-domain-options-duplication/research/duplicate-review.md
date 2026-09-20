# 第二阶段实仓重复候选审阅

扫描日期：2026-09-20。默认阈值 60，按同语言函数结构比较；此报告是审查快照，不是自动合并清单。已逐一打开所有候选函数，并检查 DTO 导入和关键调用关系。

## 扫描结果

- 命令：`pnpm run report:duplicates --format json --output /tmp/roncin-duplicates.json`。
- 覆盖 461 个前端文件、410 个 Go 文件，7631 个函数（前端 4232、Go 3399）；3320 个达到默认阈值，1174 个仅参与 exact。
- 8 组候选，0 解析/执行错误；已审阅全部 8 组。
- 两次 JSON 与两次文本输出分别逐字节一致；一次实仓执行约 5.14 秒（包含 pnpm 与 Go 启动，仅供本机参考）。
- 排除：测试文件 378、非目标源码 230、生成后缀/声明文件 90、生成标记 782，跳过目录 4。
- [报告快照](./duplicate-report.json) 保存全部候选与统计，省略较长的 exactOnly 函数明细；重新运行命令可获得完整实时结果。

## 逐组结论

| 候选 | 实际证据 | 审查结论与后续建议 |
| --- | --- | --- |
| 汇率导入模板下载 | `web/src/pages/finance/exchange-rates/components/ExchangeRateImportModal.tsx:51` 与 `ExchangeRatesPanel.tsx:99` 的 handleDownloadTemplate，含同一接口、base64 解码、下载和消息流程 | 确实同义重复。后续可在汇率页面模块内提取下载动作或 Hook；loading/message 由消费组件传入或持有，无需上提全局领域平台。 |
| 财务组织 ID 集合校验 | `server/internal/biz/finance_cashflow.go:121`、`finance_invoice.go:218`、`finance_verification.go:149`、`settlement.go:146` 均拒绝空集合、Nil UUID 和重复 UUID | 同 biz 包中相同纯校验，可后续提取共享验证。不能顺带修改不同调用方的权限/数据范围判定；保留各用例测试。 |
| 业务标签 DTO 转换 | `server/internal/service/order_fee.go:400` 与 `settlement_fee_tag.go:14` 外观相同，但 v1 分别指向 api/order/v1 和 api/finance/v1 | 合理重复，返回值属于不同生成协议类型。扫描不解析跨文件 import 别名，此组不能直接合并，禁止为了减组数引入反射/泛型 DTO 框架。 |
| 三处业务标签列渲染 | `web/src/pages/finance/bills/components/billColumns.tsx:38`、`finance/fees/components/feeLedgerColumns.tsx:120`、`orders/order-fee-panel-columns.tsx:35` 相同 Tag 样式和空值展示 | 值得后续按业务标签领域共享小型展示组件；列宽、搜索和页面操作仍留调用方。先检查已有 business-tag 组件目录，避免另建同类组件。 |
| 订单费用候选增量合并 | `web/src/pages/orders/use-order-fee-options.ts:127` 与 `:152` 对结算单位和费用设置分别合并 base/extras 并绑定订单身份 | 同文件相似算法但候选实体、状态和闭包来源不同。可在需要维护时提取局部纯合并函数；不在本期创建通用缓存/状态框架，尤其保留订单身份隔离。 |
| access → biz 订单类型转换 | `server/internal/server/auth.go:390` 与 `server/internal/service/order_query.go:418` 相同枚举映射 | 重复事实成立，但跨 server/service 层；不能从一个传输层直接引用另一个来消除重复。若未来枚举维护成本增加，再评估中立适配位置与依赖方向，本期保留。 |
| biz → access 订单类型转换 | `server/internal/server/auth.go:428` 与 `server/internal/service/order_query.go:437` 相同反向映射 | 与上一项一起审议适配归属，不建立双向层间依赖；当前保留。 |
| 对冲与核销的汇兑损益展示 | `web/src/pages/finance/nettings/index.tsx:456` 与 `verifications/index.tsx:397` 相同正负金额与币种展示 | 可后续提取财务金额展示小组件；两种业务的计算、说明标题及损益口径保持在各自模块。 |

## 能力限制

- 不解析跨文件类型/别名语义，局部同名自由变量也可能绑定不同外层值；每组必须回到调用方审阅。
- 不对跨语言、不同算法实现的同义逻辑、仅部分片段相似作完备判断。
- exact-only 数量是覆盖限制，不是扫描错误；详细原因随 JSON 返回。
- 本次只交付工具与建议，不修改上述业务实现。未发现候选不能证明不存在重复。
