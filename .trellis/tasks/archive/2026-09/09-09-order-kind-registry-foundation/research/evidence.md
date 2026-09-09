# 订单类型注册薄底座调研证据

## 1. 生命周期边界

- `web/src/components/ui/order-template/OrderFormTemplate.tsx:63-109`：模板拥有草稿键、dirty、
  layout-effect 恢复和 `resetTo`。
- `web/src/pages/orders/detail.tsx:82-129,272-317`：统一显式刷新令牌属于详情页面编排，不在
  `OrderFormTemplate` 内。新注册层不得搬运或复制该令牌。

## 2. 当前类型配置与 fallback

- `web/src/pages/orders/common.ts:189-223`：`OrderKind` 只有 `sea-export`，配置只支持
  `category: 'sea' | 'air'`。
- `web/src/pages/orders/new.tsx:155-160`：新建页使用“sea，否则 air”。
- `web/src/pages/orders/detail.tsx:381-386`：详情页使用“air，否则 sea”。
- 两处分支对第三种运输方式的默认结果相反，不能通过继续扩展 `category` 修复。

## 3. 配置真实调用面

`rg "parseOrderKind|ORDER_KIND_CONFIGS|OrderKindConfig" web/src` 证明调用方包括：

- `list.tsx`、`new.tsx`、`detail.tsx`、`fees.tsx`；
- `OrderPageHeader.tsx`；
- `list-query.ts`、`list-resources.ts`；
- `use-order-create-options.ts`、`use-order-detail-data.ts`；
- 对应测试。

因此只迁移新建和详情会保留双重配置真相。

## 4. 公共列表模板漂移

- `web/src/components/ui/order-list-template/types.ts:3-10`：另有包含 `rail/truck/customs` 的
  `OrderKind` 联合，与产品当前路由和计划中的 LAND 命名不一致。
- `web/src/components/ui/order-list-template/OrderListTemplate.tsx:129-143`：组件内部维护本地
  `kindMap`，未知值默认显示“海运出口”。
- `web/src/pages/orders/list.tsx:121` 与 `list-query.ts` 使用 `as any` 跨越两套类型。

## 5. SE 表单转换

- `web/src/pages/orders/new.tsx:222-278`：创建默认值依赖创建人、服务类型候选和货类候选，
  不能只传 creator。
- `web/src/pages/orders/order-create-payload.ts:86-267`：创建转换用 category/裸数字 1 计算
  `isSea`，然后决定海运单证和普通 shipping documents。
- `web/src/pages/orders/components/detail/orderDetailHelpers.ts`：更新转换通过
  `seaDocumentStructure/seaMasterBillMasterNo/seaDocument` 是否存在反推业务类型。
- `web/src/pages/orders/templates/air-template.tsx:114-119`：未接通的空运模板把航班号写入
  `vesselVoyage`，说明未来类型不能仅复用当前扁平字段完成真实接入。

## 6. SE 详情状态

- `web/src/pages/orders/detail.tsx:1-230`：通用详情直接 import sea change service 与四个 Sea
  覆盖层，并持有 change-actions 请求序号、目标身份、弹窗开关、共享运输执行 ID。
- `web/src/pages/orders/components/detail/OrderDetailHeader.tsx`：拆票和改配以专用命名 props
  写入通用 Header。
- 这些状态需要由正常 React 组件接管；纯注册字典函数不能安全动态调用 Hook。

## 7. 权限与后端支持度

- `server/internal/access/manifest.go:280-304`：绝大多数通用操作覆盖全部业务类型；拆票、改配、
  改单、作废、Switch 通过 `businessTypes: [SE]` 限制，Manifest 已是能力真相。
- `server/internal/biz/order_usecase.go:290-296`：领域创建仍只允许 SE + Export。
- `server/internal/biz/order.go:17`：错误文案明确为“当前仅支持海运出口订单”。
- 当前没有第二个可用类型，只有一个元素的后端 validator map 不减少复杂度；推荐延后到 SI
  真实领域规则设计时处理。

## 8. 历史决策

- 已归档任务 `09-09-consolidate-order-draft-lifecycle` 明确：模板独占草稿生命周期，页面持有
  订单身份与显式刷新命令；不保留旧 API 或历史 sessionStorage 兼容。
- 2026-09-09 讨论确认：先建立只迁移 SE 的薄注册层，再按 SI → AE → AI → LAND 独立交付，
  不在底座任务提前实现其他品类。
