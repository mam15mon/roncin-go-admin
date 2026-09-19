# ADR 0010: 订单类型注册表与三类真相边界

- 状态：已采纳
- 日期：2026-09-09（建立订单类型唯一注册入口）
- 主题：前端

## 背景

系统规划六类订单（SE/SI/AE/AI/LAND/RAIL），当前只交付海运出口。此前
订单类型字典散落多处：通用 layout 维护 kind→标题映射、列表模板维护
kind→文案映射、各页维护业务枚举→路由映射。任何新增类型都要改 N 处，
且「谁能对某类订单做什么」的能力判断容易和「这类订单的 UI 在哪」耦合成
第二套权限真相。

## 决策

- 订单类型元数据的唯一所有者是
  [../../../web/src/pages/orders/order-kinds/](../../../web/src/pages/orders/order-kinds/)
  注册表（`ORDER_KIND_REGISTRY` + `getOrderKindDefinition`），以
  `satisfies Record<OrderKind, OrderKindDefinition>` 穷尽约束；只注册已
  真实交付的类型，不提前注册占位类型。
- 三类真相严格分界，不得越权：

  | 真相 | 唯一所有者 |
  |------|-----------|
  | 类型元数据与表单/详情适配入口 | `order-kinds/` 注册表 |
  | 操作能力（谁能做什么） | 后端权限 Manifest + `access.canOrder` + `order.allowedActions`（ADR 0002） |
  | 表单生命周期（草稿、dirty、resetTo） | `OrderFormTemplate` |

- 空值、未知、未注册 kind 一律返回 `undefined`，页面据此展示 404；禁止
  默认兜底为海运出口，未知类型不得进入数据 Hook 或发起请求。
- API 只返回业务枚举时用 `getOrderKindDefinitionByBusinessType` 反查，
  不得在各页维护数字枚举到路由的第二套映射。
- 通用组件（layout、`OrderListTemplate`）不反向依赖注册表：页签用中性
  占位标题，真实标题由订单页加载后经 `roncin:update-tab-title` 回填。
- 类型详情扩展（`DetailFeatures`）是普通 React 组件，经 render-prop
  贡献 `headerActions`/`moreMenuItems`/`appendSections` 等插槽；通用
  Header 不认识具体业务动作。

## 理由

- 注册表 + `satisfies` 穷尽让「新增类型漏改一处」从运行期遗漏变成编译期
  错误。
- 通用/类型专属代码双向隔离（通用不认识类型、类型扩展不碰模板内部），
  是模板复用体系（ADR 0008）能容纳多品类的前提。
- 404 显式失败优于静默兜底：把「未开放」误显示成海运出口会造成错单。

## 后果

- 新增订单类型 = 注册一个 definition + 实现对应扩展组件，不再触碰
  layout 与列表模板。
- 运输方式相关的地点/站点逻辑必须穷尽分发，land/rail 显式抛「尚未开放」。
- 契约全文见
  [../web/frontend/component-guidelines.md](../web/frontend/component-guidelines.md)
  「订单类型三类真相边界」。
