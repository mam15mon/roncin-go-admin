# TagsView 与全站路由现状审计

## 根因

`web/src/components/layout/TagsView.tsx` 当前将 `location.pathname` 同时作为
`TagItem.key` 和 `TagItem.path`。监听到新 pathname 时仅按完整路径查重，因此菜单
列表与其内部的新建、详情、费用、拆票页面都会追加新页签。

现有激活判断使用 `currentPath === tag.path || currentPath.startsWith(tag.path + '/')`，
关闭当前页签也以 `currentPath === closedKey` 为前提。引入稳定菜单键后，这两处都必须
改为按当前路径解析出的页签键判断，不能只修改新增页签逻辑。

## 当前需要归组的路由

| 稳定页签键 | 应归入的当前路由 |
| --- | --- |
| `/orders/sea-export` | 列表、新建、订单详情、费用录入、拆票 |
| `/partners/customers` | 客户列表、新建、详情 |
| `/partners/suppliers` | 供应商列表、新建、详情 |
| `/partners/foreign-agents` | 国外代理列表、新建、详情 |
| `/finance/fees` | 集运费用明细、单票费用详情 |

其余当前菜单入口没有不同 pathname 的内部整页路由：参数设置、主数据、系统管理和企业
资源配置主要使用查询参数或页面内 Tabs。它们仍以自身 pathname 为稳定页签键，但页签
保存的最近地址需要包含 `search` 与 `hash`。

## 动态标题

订单详情与订单费用页在加载真实订单号后发送 `roncin:update-tab-title` 事件，事件携带
当前 pathname。页签改用菜单键后，事件处理必须先解析事件路径所属的稳定页签键，并且
只更新仍停留在该事件页面的页签，避免卸载页面的迟到事件覆盖新页面标题。

## 边界与兼容

- 登录、注册和认证回调继续由现有忽略规则排除。
- 浏览器直接打开内部路由时，以所属菜单键建立页签，同时保留内部路由作为可返回地址。
- 跨菜单跳转按目标 URL 解析目标菜单键：已有则更新，没有则新增，不覆盖来源页签。
- 未匹配明确规则的路径按自身 pathname 建页签，避免宽泛前缀把未知或未来页面误归组。
- 当前实际订单类型配置只有 `sea-export`；归组规则以当前可访问路由为准，后续新增订单
  类型时在集中规则和测试中显式登记，不静默接受无效类型。

## 预计改动边界

- `web/src/components/layout/routeUtils.ts`：集中提供路径到稳定页签键的解析。
- `web/src/components/layout/TagsView.tsx`：按稳定键 upsert 页签，保存最新完整地址与标题，
  并修正激活、关闭、动态标题和图标判断。
- `web/src/components/layout/TagsView.test.tsx`：覆盖纯解析、组件路由切换、直接访问、跨菜单、
  查询参数和现有关闭操作回归。

不需要修改业务页面跳转调用、路由配置、服务端契约或生成代码。
