# 订单面包屑统一技术设计

## 设计结论

新增订单专属 `OrderPageHeader`，统一计算并渲染新建、详情、费用和拆票页面的面包屑、
当前标题和主要返回按钮。它复用全站标准 `PageHeaderShell`，订单页面不再各自手写层级。

列表页继续使用路由自动面包屑，因为其当前 `订单管理 > 海运出口` 已与目标一致，且列表
属于 ProTable 页面，不需要额外叠加吸顶详情页头。

## 统一层级模型

```text
订单管理
  └─ 海运出口
       ├─ 新建订单
       └─ SE订单号
            ├─ 费用录入
            └─ 拆票
```

稳定链接：

- 订单管理：`/orders`（按现有路由重定向至默认可用订单列表）；
- 海运出口：`/orders/sea-export`；
- 订单号：`/orders/sea-export/:id`；
- 当前页面：不提供链接。

## OrderPageHeader 契约

组件放在订单模块内，不上升为跨业务通用抽象。建议属性：

```ts
type OrderPageKind = 'create' | 'detail' | 'fees' | 'split';

type OrderPageHeaderProps = {
  page: OrderPageKind;
  orderKind: OrderKind;
  orderId?: string;
  orderNo?: string;
  tags?: ReactNode;
  extra?: ReactNode;
  subTitle?: ReactNode;
};
```

组件通过 `ORDER_KIND_CONFIGS` 取得明确的菜单名称和列表地址，不从显示字符串切割
“订单”二字。为 `OrderKindConfig` 增加专用导航名称，例如 `navigationTitle: '海运出口'`，
使表单标题“海运出口订单”和面包屑菜单名“海运出口”各有清晰语义。

组件内部集中生成：

| page | 上级 breadcrumbs | title | backText / back target |
| --- | --- | --- | --- |
| create | 订单管理、海运出口 | 新建订单 | 返回列表 → 业务列表 |
| detail | 订单管理、海运出口 | 真实订单号或路由 ID | 返回列表 → 业务列表 |
| fees | 订单管理、海运出口、订单号 | 费用录入 | 返回订单详情 → 详情 |
| split | 订单管理、海运出口、订单号 | 拆票 | 返回订单详情 → 详情 |

`PageHeaderShell.breadcrumbs` 只传上级项，当前页仅由 `title` 渲染，从结构上消除重复。

## 标准链接语义

修正 `PageHeaderShell` 已存在但未实现的 `href` 契约：

- 有 `href` 的上级项使用 Umi `Link` 渲染，保持单页应用跳转并支持标准浏览器链接行为；
- 兼容现有 `onClick` 调用，避免影响客商页面；
- 当前页不传 `href/onClick`，保持不可点击；
- 订单新组件统一使用 `href`，不再创建 `<a onClick>`。

## 页面接入

### 新建

- 用 `OrderPageHeader page="create"` 替换当前手写 breadcrumbs。
- 保留现有副标题、表单内容和创建后返回列表行为。
- 无效 kind 继续显示明确 404。

### 详情

- `OrderDetailHeader` 顶部改用 `OrderPageHeader page="detail"`，订单号作为当前标题，锁状态
  放在 `tags`。
- `DocumentDetailLayout` 继续承载下方横向业务操作栏，但不再为订单详情生成第二套面包屑。
- 新增唯一主要返回列表按钮。
- 加载中先用路由 ID 显示页头，加载成功后切换真实订单号；未找到订单时保留同一页头。

### 费用

- 删除 `fees.tsx` 手写导航栏，改用 `OrderPageHeader page="fees"`。
- `extra` 只保留刷新数据；删除右侧“回到订单详情”。
- `OrderFeeHeader` 继续负责锁状态提示和订单摘要；基础信息中的订单号改为普通文本，不再
  作为第四个返回详情入口。
- 加载中与未找到状态保留公共页头，订单号未知时使用路由 ID。

### 拆票

- 改用 `OrderPageHeader page="split"`，当前 title 只显示“拆票”。
- 保留业务说明副标题、HOUSE 标签和刷新按钮。
- 面包屑订单号返回详情；顶部主要返回按钮同样返回详情。
- 底部“取消返回”继续作为放弃当前拆票操作的业务按钮。

### 列表

- 保持 `PageContainer` 自动面包屑和现有页面标题，不引入第二个页头。
- 增加回归测试确保 route name 仍产生 `订单管理 > 海运出口` 所需配置；不修改列表业务。

## 无效业务类型

- 移除详情与费用中的 `|| ORDER_KIND_CONFIGS['sea-export']` 静默兜底。
- 在发起数据请求和构建业务页头前校验 `parseOrderKind` 结果。
- 无效 kind 显示明确错误状态和返回有效订单入口的按钮，不展示海运出口面包屑假象。
- 不增加旧路径兼容或自动纠错。

## 测试策略

1. `OrderPageHeader` 组件矩阵测试：四种页面的文字、链接、当前项和返回目标。
2. `PageHeaderShell` 测试：`href` 生成真实链接，`onClick` 兼容，当前项不可点击，标题不重复。
3. 新建/详情/费用/拆票接入测试：页面不再保留旧手写面包屑和重复返回按钮。
4. 异常状态测试：加载中、订单不存在、无效 kind 均有正确导航且无海运出口误标。
5. 既有拆票与订单锁测试回归，确保页头重构不影响业务按钮和请求。

## 兼容、风险与回滚

- 不修改 URL、路由权限或接口，书签和现有跳转继续有效。
- 详情动作较多，继续保留独立横向工具栏，避免把所有按钮塞入页头造成窄屏溢出。
- `PageHeaderShell` 是公共组件，新增真实链接行为可能影响现有样式；必须对客商页现有
  `onClick` 路径做回归测试。
- TagsView 复用规则已由独立任务提交；本任务不得继续修改 TagsView 文件或把页签行为夹带
  进面包屑改造。
- 回滚只涉及前端组件与订单页面，不涉及数据迁移。
