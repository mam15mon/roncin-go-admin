# 进口业务横切影响与空运单证复用

以下是当前源码调研结论与设计建议，不是已批准设计；未运行测试。

| 影响 | 当前证据 | 进口实施必须验证 |
| --- | --- | --- |
| 类型和方向 | `server/internal/biz/order_usecase.go:321` 只接受 SE/AE+EXPORT | 四类合法配对，拒绝类型方向错配 |
| 权限 | `server/internal/access/manifest.go:133`、`:303` 已生成 SI/AI 通用权限，`:315` 高级变更仅 SE | 使用真实业务类型授权，前端路由判定同步，禁止借出口权限 |
| 汇率 | `server/internal/biz/order_fee.go:521` 出口 ETD/进口 ETA、`:538` 中国自然日 | 进口 ETA 变化刷新适用费用，ETD 不误触发 |
| 权威运输 | `server/internal/data/air_order_query.go:41`、`:48` 仅认 AE | AI 详情/列表/筛选/报表读取同一权威来源 |
| 费用主单号 | `server/internal/data/fee_ledger_master_no.go:22`、`:24` 仅 SE/AE | 进口号码展示与搜索一致 |
| 重量权限 | `server/internal/data/air_weight.go:33`、`:77`、`:98`、`:204` 硬编码 AE | AI 如复用重量确认，按目标类型检查权限 |
| 审核快照 | `server/internal/data/order_lock_apply.go:50`、`:97` 仅出口单证快照 | 进口审核锁包含真实单证内容及版本 |
| 终止恢复 | `server/internal/data/order_write_lifecycle.go:76`、`:105` 仅 SE/AE 专项 | 进口恢复关联、历史与状态原子一致 |
| 放货凭证 | `server/internal/data/order_release_pod.go:36` 非 SE 依赖旧通用引用 | 不将进口接到过时通用单证假装完成 |
| 工作台 | `server/internal/service/workbench.go:31`、`data/workbench.go:452`、`:521`、`:829`、`workbench_todo.go:20` 仅出口 | 进口订单、异常、未建账费用不漏计 |
| 通知 | `server/internal/biz/notification.go:232`、`:262` 仅出口订阅/跳转 | 类型范围、深链和受控预览，不向真实员工发调试通知 |
| 编号 | `server/internal/data/order_write.go:29`、`:33` 及 `biz/orderconfig.go:200` 通用流水 | 复用编号，无需另建进口序列平台 |
| 列表导出 | `web/src/components/ui/order-list-template/OrderListToolbar.tsx:134` 有 props，但 `OrderListTemplate.tsx:714` 未接表格导出 | 区分入口存在/接通，明确当前页或全量筛选 |
| 单证跳转 | `web/src/pages/orders/list-documents-action.ts:4`、`:25` 仅 SE/AE 且 AE 进详情 | 四类真实单证入口，不能四类都进海运 HBL |

## 可复用的 AWB 工程机制

- PDF 按需加载、变更保护、单证及关联版本：`web/src/pages/orders/hbl/HblEditor.tsx:67`、`:122`、`:148`、`:217`。
- 前台 15 秒/回焦检查：`hbl/use-hbl-data.ts:29`、`:46`、`:60`。
- 内容、公司和模板指纹：`hbl/model.ts:92`。
- 同一 Blob 预览/下载/打印：`hbl/pdf/HblPdfPreview.tsx:34`、`:46`、`:54`、`:71`。
- 同域中文字体和长文本续页机制：`hbl/pdf/font.ts:7` 及现有 PDF 组件。
- 服务端版本化打印稿模式可参考 `server/internal/biz/sea_house_bill_print.go`，空运另建领域内容类型，不能依赖 `SeaBillContent`。
- 共享实现只在真实复用后提取；空运页面不得深引 `pages/orders/hbl/...`。

## 尚需用户决定的单证语义

主分运单结构化及制作/预览/PDF打印、无HAWB直单均已获用户后续明确批准。仍需收敛进口外部单证与本公司签发语义、直单共享/转换约束、具体输出版式；多航段和复杂改配/拆拼仍等待范围决定。新决定需同步现有契约、Schema与测试，不能靠兼容分支绕过。

> 状态同步：后续用户已确认直单一主一票不混HOUSE、高级空运与批量导入后续、完整筛选CSV本期；最新具体方案以父prd.md/design.md最终复核稿为准，本研究中的候选/历史未决项不再单独决定实施范围。
