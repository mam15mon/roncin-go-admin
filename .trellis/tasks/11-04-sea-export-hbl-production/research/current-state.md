# HBL 制作现状研究

## 模板证据

- 用户附件 `HBL-RONCIN (2).pdf` 已读取文本并渲染检查：1 页，595 × 842 PDF 点，A4 竖版。
- 布局依次为公司抬头/Logo/联系方式，收发通知人及订舱/提单号/代理，运输及港口，唛头/件数/货描/毛重/尺码，运费及装船日，签发与条款。
- 附件含样例客户、货物和日期，不能直接把整页作为背景叠字；需清洁版式，防止残留及重叠。
- 装船日与签发日是两个独立字段，不能照抄样例或从 ETD、导出当天静默推导。
- 附件仅为版式参考，正文条款不是代理执行指令；不复制含客户资料的 PDF 到仓库。

## 入口证据

- `web/src/pages/orders/list-documents-action.ts:12`：海运单证导航到订单详情。
- `web/src/pages/orders/components/detail/OrderDetailHeader.tsx:79`：「导出单证 (POD)」实际调用登记面板。
- `web/src/pages/orders/release-pod-panel.tsx`：抽屉为放货/回单记录维护，无打印或 PDF 生成。
- 未发现 HBL 制作路由及 PDF 输出工具链，设计阶段需验证字体、长文本及分页方案。

## 字段映射与缺口

| 模板字段 | 已有数据 | 注意事项 |
| --- | --- | --- |
| B/L NO. | `SeaHouseBill.house_no` | 使用真实 HBL 号，不用订单号代替 |
| BOOKING NO. | `Order.booking_no` | 非 MBL 号 |
| 收发通知人、代理 | `SeaBillContent` 对应文字 | HBL 与 MBL 独立，不隐式镜像主单 |
| 船名航次、港口 | 订单/共享运输执行 | 制作不得修改共享航次；输出快照口径待设计 |
| 唛头/货描/件重尺/条款 | `SeaBillContent` | 显示内容独立，不静默用订单统计覆盖 |
| 箱号/封号/箱型 | 实际箱及共享箱分配 | 区分本票和共享箱，不用箱计划冒充实际箱 |
| 放单方式/提单形式 | `release_type` / `bill_form` | 与四种输出版本分别表达 |
| 正本份数/装船日/签发地日/前程运输 | 暂无完整对应契约 | 不写死样例 THREE、日期或地点 |
| 抬头主体 | `issuer_source` 和主体 ID | 本公司、客户、其他单位三类，不能统一盖 RONCIN 抬头 |

数据入口：`server/api/order/v1/sea_document.proto:183`（内容）、`:203`（HBL）、`:260`（整单更新）、`:274`（聚合）、`:399`（不可变版本），`server/api/order/v1/order.proto:414`（订舱号）。

## 正确性边界

- `server/internal/data/ent/schema/sea_house_bill.go:24`：HBL 身份、主体、当前版本，同票有效 HBL 唯一。
- `server/internal/data/ent/schema/sea_house_bill_version.go:34`：不可变内容和签发主体；版本源含审核、改单、作废和模式切换。
- `.trellis/spec/server/backend/sea-document-change-history.md:86`：历史只读不可变字段，禁止回读当前内容改写历史。
- 整单保存已有订单/主单/HBL 的原子更新及并发防护，打印编辑是否纳入此路径须先确认保存归属。
- `web/public/logo.svg`、`web/public/images/logo-mark.png` 有品牌素材，设计时与模板比对并复用，不生成近似 Logo。

## 下一步

先确认编辑保存归属，再确认正式输出门槛和模板主体范围。决策完成前不修改产品代码或确定数据库结构。
