# 保存成功判定与路由保护：技术核查

日期：2026-10-12。只读源码核查，尚未实施。

## 当前正式响应契约

| 目标接口 | 契约位置 | 响应 |
| --- | --- | --- |
| 客商创建 | `server/api/partner/v1/partner.proto:908` | success/code/message/data(Partner)/trace_id |
| 客商更新 | `server/api/partner/v1/partner.proto:916` | success/code/message/data(Partner)/trace_id |
| 订单更新 | `server/api/order/v1/order.proto:720` | success/code/message/data(Order)/trace_id |
| 账单更新 | `server/api/finance/v1/settlement.proto:779` | success/code/message/data(FinanceBill)/trace_id |
| 流水创建 | `server/api/finance/v1/settlement.proto:832` | success/code/message/data(FinanceCashflow)/trace_id |

这些目标接口都有实体响应，不是204空响应契约。正式实现仍需核对service返回是否填写必要ID，成功与空报文/undefined必须区分，不能仅凭await未throw判成功。

## 生成客户端与错误选项

`web/src/services/roncin/partnerService.ts:52`、`:81`，`orderService.ts:86` 已接受并透传options；不需要手改生成物来使用skipErrorHandler。保持统一request既有错误消费约定，通过目标调用方显式判定成功/处置错误，避免全局契约变化或通知重复。
账单/流水现有catch分支已能保留弹窗，但全局消费undefined不会进入catch；修复必须覆盖该形态，不能只加catch。

## 路由保护可用边界

`web/src/router/index.ts:7` 使用createBrowserRouter。客商可沿既有tabCloseGuard注册脏状态，并在真实路由阻断处复用确认语义；须避免页签关闭/组织切换与路由阻断重复确认。相邻同菜单记录身份切换也纳入保护，不能仅判断tab key不同。
此处确认的是技术入口，未声称当前客商已经实现保护。详细验收与行为仍由本子任务prd/design规定。
