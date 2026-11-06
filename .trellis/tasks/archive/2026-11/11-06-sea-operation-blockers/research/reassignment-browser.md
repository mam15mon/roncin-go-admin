# 改配与同票操作链路真实浏览器验收

## 环境与方法

- 日期：2026-11-06；浏览器使用 Asia/Shanghai 时区。
- 隔离 PostgreSQL 数据库：`roncin_sea_ops_20261106_a1afc6`；服务端 `127.0.0.1:8010`，前端 `127.0.0.1:8013`。
- 仅使用隔离认证状态与正式 API 创建的测试票，未操作原开发库、未自动修复既有开发订单。
- Playwright 操作真实页面与确认框，不 mock 业务请求；记录实际预览、执行载荷和响应，再读取正式 API 与数据库核对事实。
- 外部确认时间由脚本明确填写；改配执行监听真实路由 `/sea-order-change/reassignment/execute`。

## NEW 与 CANDIDATE

最终一轮使用隔离订单 `SE2026110600002`（`01a1afdc-b415-7850-93a3-1d555aa4d98f`）。该票经过前序脚本调试，最终一轮的起始版本为 5；每次正式动作仍独立验证预期版本和版本递增。

| 路径 | 订单版本 | HBL 不可变版本数 | 结果 |
| --- | --- | --- | --- |
| NEW，同母单新航次 | 5 → 6 | 5 → 6 | 原 MBL ID 保留，TE 与活动 Link 切换，船名/航次/ETD/ETA 按输入保存 |
| CANDIDATE，已有母单和航次 | 6 → 7 | 6 → 7 | 加入隔离目标票 `SE2026110600007` 的 MBL/TE，目标成员数 1 → 2，目标票资料和原 HBL 不变 |

- 两次真实执行均为 HTTP 200 且 `success=true`，返回真实改配事件 ID：NEW 为 `01a1aff4-dfbd-769d-94f0-62ebd2b968d4`，CANDIDATE 为 `01a1aff5-1059-79b8-bd47-4c663668b55f`。
- 分别断言执行目标与最后成功预览目标完全相同，订单/Link 预期版本与对应预览一致；CANDIDATE 同时携带 MBL 和 TE 的预期版本。
- 原订单号、流程状态、HBL 身份与号码保持；旧快照逐项读取比较不变，新快照指向目标 MBL。
- 表单中的航程日期实际进入 RFC3339 请求，不只断言控件显示。
- 原候选票的共享成员数属于派生统计，允许随加入增加；原订单正文、版本、航程及 HBL 实体与内容均保持。

## 同一票：拆票 → 改配 → HOUSE 转 DIRECT

起点为已通过真实拆票验收的原票 `SE2026110600003`（`01a1afdc-b43e-77ce-b2e0-f6fbc735e484`），不是另外创建一张无关票替代串联验收。

| 阶段 | 订单版本 | 验证事实 |
| --- | --- | --- |
| 已拆票原票 | 2 | 正式改配历史接口包含 SPLIT 事件，有实际货物明细 |
| NEW，不同船公司新母单 | 3 | 退出原共享批次，加入新建且仅有本票的 MBL；HBL 身份保持并追加当前快照 |
| HOUSE → DIRECT | 4 | 真实预览返回 executable，正式确认后结构为 DIRECT；原 HBL 形成 VOIDED 快照，历史继续可读 |

- HBL ID 为 `01a1afdc-b442-7326-a6c9-6a3f87cfeb93`，最终保留 3 份不可变版本；改配前和改配后的旧版本均逐项核对未被改写。
- 模式切换执行使用刚完成改配后的订单版本、HBL 实体版本及当前快照 ID，并携带真实外部确认时间。
- 串联结束后 SPLIT、REASSIGNMENT 历史仍在；实际货物明细与拆票完成时逐项一致，未发生隐式货量覆盖或删除。
- 重新打开最终 DIRECT 详情成功，无“页面暂时无法打开”，无浏览器 pageerror。

## 数据库持久化复核

只读 SQL 直接关联 `orders`、活动 `sea_master_bill_order_links`、`sea_master_bills`、`sea_transport_executions` 与 `shipping_lines`，排除仅在响应转换时覆盖字段而掩盖持久化遗漏的情况。

| 订单 | 真实持久化船公司 | Order/TE/MBL 身份一致 | 起运/卸货/中转一致 | ETD/ETA 一致 |
| --- | --- | --- | --- | --- |
| SE2026110600002 | 地中海航运 | 是 | 是 | 是 |
| SE2026110600003 | 中远海运集运 | 是 | 是 | 是 |

两票数据库中的船公司 ID/名称同时与正式 API 一致；订单航程分别为 `CANDIDATE SHIP / CV003`、`CHAIN SHIP / CHAIN001`。船期均保存为 `2026-11-12T08:00:00Z` 和 `2026-11-22T08:00:00Z`，对应页面输入的上海时间 16:00。

本次浏览器验收实际发现并推动修复了两个后端问题：改配未同步订单持久化航程投影；Ent 同字段先 Clear 再 Set 导致 SQL 重复赋值。最终复核使用两项均修复后重启的后端，早期失败执行已事务回滚。

## 证据与观察边界

- 驱动脚本：`/tmp/roncin-sea-operation-blockers/reassignment-browser.mjs`、`split-reassignment-mode-browser.mjs`。
- 真实请求、响应、前后状态：同目录 `reassignment-results.json`、`split-chain-results.json`。
- 最终独立只读复核：`reassignment-final-verify.mjs`；输出 `reassignment-final-verification.json`，执行成功，包含数据库一致性结果和最终页面检查。
- 截图：`reassignment-new-success.png`、`reassignment-candidate-success.png`、`split-chain-success.png`、`split-chain-final-confirmed.png`。
- 链路脚本的两个业务步骤均通过。其末尾曾把 React 开发警告当作运行异常判失败；独立只读复核按实际类别保留警告，并重新核对 API、数据库、快照、货物和最终详情，未重复执行已成功的业务动作。
- 仍观察到三类 React 开发警告：未连接的 useForm、showCount 属性落到 DOM、controlled input 转 uncontrolled；已如实保存在最终证据中。它们未产生 pageerror、错误页或数据丢失，本轮不据此宣称全站开发警告已清零。
- CANDIDATE 匹配前，自动 NEW 预览可能因目标已存在返回 409；正式匹配并选择候选后可正常执行。409 的中文业务原因已通过现有 `getRequestErrorMessage` 展示，不再被泛化为“缺少版本”。

## 定向测试的稳定性修正

- 改配最终 6 文件、12 用例通过；保留真实字段注册、执行确认、乱序、关闭重开、冲突、写失败及幂等断言。
- 仅对包含 toast 可见性断言的两个文件使用 jsdom。最小 DOM 对照证明：没有 `ant-message-stack` 祖先的消息，HappyDOM 仍误匹配其 `:where(...).ant-message.ant-message-stack:not(...) .ant-message-notice:not(...)` 规则并套用 opacity:0；同一选择器在 jsdom 返回不匹配。
- 未删除可见性断言、未覆盖产品 CSS、未添加固定 sleep；消息保留默认生命周期。重型冲突与写迟到场景已拆文件，以满足单文件耗时约束。
