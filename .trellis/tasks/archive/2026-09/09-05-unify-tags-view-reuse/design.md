# 全站 TagsView 页面复用技术设计

## 设计目标

页签的“身份”和“当前地址”必须分离：稳定菜单入口决定页签唯一性，用户最后访问的
内部页面决定再次点击页签时去哪里。标题是当前页面的呈现属性，可以变化但不能参与
页签去重。

```text
当前 location（pathname + search + hash）
                  │
                  ▼
      集中的页签归属解析规则
                  │
       ┌──────────┴──────────┐
       ▼                     ▼
稳定 key（菜单入口）    最新 path（完整地址）
       │                     │
       └────── upsert ───────┘
                  │
                  ▼
          同菜单只保留一个 TagItem
```

## 核心模型

保留现有 `TagItem` 数据结构，但重新定义字段语义：

- `key`：稳定页签键，对应菜单入口 pathname；用于去重、激活和关闭判断。
- `path`：该页签最后访问的完整地址，包含 pathname、search 和 hash；用于点击、重新加载
  及关闭其他页签后的跳转。
- `title`：当前 `path` 对应标题。路由变化时先使用静态解析标题，业务数据加载后允许由
  现有动态标题事件更新。
- `closable`：保持现有语义，工作台固定不可关闭。

## 归属解析

在 `routeUtils.ts` 中新增单一公共解析函数和集中规则表，输入 pathname，返回稳定页签键。
规则按具体程度排列：

1. 海运出口列表、新建、详情、费用、拆票 → `/orders/sea-export`；
2. 客户新建/详情 → `/partners/customers`；
3. 供应商新建/详情 → `/partners/suppliers`；
4. 国外代理新建/详情 → `/partners/foreign-agents`；
5. 集运费用详情 → `/finance/fees`；
6. 其他路径 → 规范化后的自身 pathname。

规则只接受当前真实存在的业务入口，不用宽泛的任意参数匹配吞掉未知路由。新增菜单子路由
时在同一规则表和测试矩阵中登记。

## 状态更新

### 初始化和路由变化

从 `useLocation()` 取得 pathname、search、hash，构造：

```text
key   = resolveTabKey(pathname)
path  = pathname + search + hash
title = resolveRouteTitle(pathname)
```

- 初始化直接访问内部页面时，创建 `[工作台, 所属菜单页签]`，页签 `path` 保留内部地址。
- 后续 location 变化时按 `key` upsert：不存在则追加；已存在则原位更新 `path` 和静态标题。
- 原位更新保持页签顺序，不会因内部跳转把页签移动到末尾。

### 激活与关闭

- 当前激活项通过 `resolveTabKey(currentPathname) === tag.key` 判断。
- `computeNextActivePath` 接收当前稳定页签键，而不是拿 pathname 与关闭键直接比较。
- 关闭非当前页签不跳转；关闭当前页签仍按“前一个、后一个、工作台”规则跳转到对应
  TagItem 的最新 `path`。
- 关闭其他、左侧、右侧与重新加载继续使用各 TagItem 最新 `path`。

### 动态标题

保留 `roncin:update-tab-title` 事件，避免修改订单详情和费用页面。事件处理流程：

1. 从事件 `path` 解析 pathname 和稳定页签键；
2. 找到同键页签；
3. 仅当该页签当前 `path` 的 pathname 仍等于事件 pathname 时更新标题；
4. 若用户已跳到同菜单的另一个内部页面，则忽略迟到事件。

这既保留真实订单号标题，也避免同一稳定键导致旧页面事件误改新页面标题。

## 查询参数与直接访问

- 页签键只由 pathname 决定，查询参数不会产生新页签。
- 页签目标地址保存 search 和 hash，因此系统管理、参数设置等页面的内部 Tab 状态可随
  页签切换恢复。
- 标题仍按 pathname 解析；现有查询参数不改变标题。
- 未知路径退回自身 pathname 作为键，不与已知菜单错误合并。

## 变更边界与兼容

- 不修改业务页面中的 `history.push`，所有行为由 TagsView 中央状态统一处理。
- 不修改 Umi 路由、权限、菜单渲染或页面缓存策略。
- 不修改 TagsView 样式与 DOM 视觉结构。
- 不持久化 TagsView 到浏览器存储；刷新后仍按当前 URL 重建工作台和当前所属菜单页签，
  与现有生命周期一致。
- 不触碰正在进行的海运配舱只读分单展示任务文件。

## 风险与回滚

- 最大风险是稳定键与最新地址混用，造成页签不高亮、关闭当前项不跳转或动态标题错位；
  通过纯函数测试和组件路由切换测试覆盖。
- 查询参数进入 `path` 后，测试与比较必须区分 `key` 和 `path`，不能恢复为同值假设。
- 回滚只需回退 TagsView、routeUtils 与对应测试提交，不涉及数据迁移或服务端状态。
