# 订单类型注册薄底座技术设计

## 1. 设计原则

1. `OrderFormTemplate` 管表单生命周期；
2. 订单详情页面管通用数据、锁单、保存与显式刷新竞态；
3. `OrderKindDefinition` 管稳定类型元数据和表单适配入口；
4. 有状态的品类专属交互由正常 React 扩展组件管理；
5. 后端权限 Manifest 管操作能力，前端注册表不复制权限真相；
6. 未注册类型 fail-closed，不存在默认 Sea/Air 分支；
7. 首期只有 SE，所有抽象必须能由当前真实迁移行为证明必要性。
8. 本任务不修改后端；等 SI 有真实领域规则时再设计服务端业务类型分发。

## 2. 目标结构

```text
web/src/pages/orders/
├── order-kinds/
│   ├── types.ts
│   ├── registry.ts
│   ├── registry.test.ts
│   └── sea-export/
│       ├── definition.ts
│       ├── form-adapter.ts
│       ├── form-adapter.test.ts
│       ├── SeaExportDetailFeatures.tsx
│       └── SeaExportDetailFeatures.test.tsx
├── common.ts                    # 只保留通用选项、搜索与格式化
├── new.tsx
├── detail.tsx
├── list.tsx
├── fees.tsx
└── ...
```

不把所有代码塞进一个 `sea-export.tsx`。定义文件只组合稳定元数据、纯表单适配器与详情扩展
组件；字段转换和有状态 UI 分开放置。

## 3. 注册契约

### 3.1 元数据与表单适配器

```ts
export type OrderKind = 'sea-export';
export type OrderTransportMode = 'sea' | 'air' | 'land' | 'rail';

export interface OrderCreateDefaultsContext {
  creator?: TemplateProps['creator'];
  serviceTypeOptions: SelectOption[];
  cargoCategoryOptions: SelectOption[];
}

export interface OrderKindFormAdapter {
  buildSections(props: TemplateProps): TemplateSection[];
  buildCreateDefaults(
    context: OrderCreateDefaultsContext,
  ): Partial<CreateOrderFormValues>;
  buildCreatePayload(values: CreateOrderFormValues): API.CreateOrderRequest;
  buildDetailInitialValues(
    order: API.Order,
    shippingDocs: API.OrderShippingDocument[],
    personnel: API.OrderPersonnel[],
  ): OrderDetailFormValues;
  buildUpdatePayload(
    orderId: string,
    expectedVersion: string,
    values: OrderDetailFormValues,
  ): API.UpdateOrderRequest;
}

export interface OrderKindDefinition {
  readonly kind: OrderKind;
  readonly businessType: OrderBusinessType;
  readonly tradeDirection: TradeDirection;
  readonly transportMode: OrderTransportMode;
  readonly title: string;
  readonly navigationTitle: string;
  readonly form: OrderKindFormAdapter;
  readonly DetailFeatures?: React.ComponentType<OrderDetailFeaturesProps>;
}
```

首期表单值仍使用现有 SE 类型，避免为了未实现品类设计虚假的泛型联合。SI 首次接入时，再以
真实字段决定是引入判别联合还是按运输方式拆分值类型；本任务不把所有未来字段预先设为可选。

### 3.2 注册与解析

```ts
export const ORDER_KIND_REGISTRY = {
  'sea-export': seaExportDefinition,
} as const satisfies Record<OrderKind, OrderKindDefinition>;

export function getOrderKindDefinition(
  kindOrPath?: string,
): OrderKindDefinition | undefined;
```

解析规则：

1. 空字符串或 `undefined` 返回 `undefined`；
2. 直接 key 只接受注册表自有属性；
3. 路径只抽取 `/orders/<kind>` 的 `<kind>` 后按同一注册表查询；
4. 未注册类型返回 `undefined`；
5. 不接受数值业务枚举反查路由，不做近似匹配或默认值。

运行时代码不得通过 `as OrderKind` 绕过解析；只有解析成功返回的定义可进入数据 Hook。

## 4. 所有调用方迁移

注册表替换的是完整配置真相，不只是新建/详情模板选择。

| 调用方 | 迁移后消费内容 |
|---|---|
| `list.tsx` | 标题、kind、businessType、transportMode |
| `new.tsx` | 元数据、Sections、默认值、创建转换 |
| `detail.tsx` | 元数据、Sections、详情映射、更新转换、详情扩展 |
| `fees.tsx` | kind、businessType、标题 |
| `OrderPageHeader.tsx` | 已解析定义或页面直接传入的标题，不再查旧字典 |
| `use-order-create-options.ts` | transportMode、businessType |
| `use-order-detail-data.ts` | transportMode、businessType |
| `list-resources.ts` | transportMode、businessType |
| `list-query.ts` | kind、businessType、标题 |

`common.ts` 删除类型配置和解析器，只保留与所有订单类型共享的选项、合作方搜索和主数据工具。

公共 `OrderListTemplate` 不应反向依赖页面注册表。其 `orderKind` 改为不带业务知识的稳定字符串，
业务类型列显示由列表行已经映射好的 `businessType` 文案提供；删除公共组件中的 `kindMap` 和
“未知即海运出口”兜底。这样注册表仍是页面业务真相，公共模板保持纯展示。

## 5. SE 表单适配器

### 5.1 Sections

`seaExportDefinition.form.buildSections` 直接调用现有 `getSeaTemplateSections`。本任务不搬动各
Section Builder，避免无收益的大规模文件移动；适配器是唯一选择入口，不复制字段 JSX。

### 5.2 创建默认值

适配器从显式上下文计算现有默认值：

- `orderDate = dayjs()`；
- 传统货代默认运输模式；
- FCL 默认装运类型；
- CIF 默认贸易条款；
- 按现有 `recommendedServiceIDs` 计算推荐服务类型；
- 优先按 `code === GENERAL`、再按 `label === 普货` 选择默认货类；
- 回填当前创建人及组织。

默认值函数只读传入上下文，不读取全局 initialState 或表单 ref。

### 5.3 创建和更新转换

将现有转换重命名为明确的 SE 边界，例如：

- `buildSeaExportCreatePayload(values)`；
- `buildSeaExportDetailInitialValues(order, docs, personnel)`；
- `buildSeaExportUpdatePayload(orderId, expectedVersion, values)`。

这些函数不再接收 `OrderKindConfig`，也不计算 `isSea`。业务枚举和贸易方向来自
`seaExportDefinition` 的稳定常量，或由定义在调用转换时显式注入；不得继续使用裸数值 `1`。

为控制重构风险，优先在原文件内进行语义明确的重命名和删除分支，再按职责移动到
`order-kinds/sea-export/form-adapter.ts`。整个过程中只有一个实现，不保留旧导出包装器。

## 6. 有状态详情扩展

### 6.1 为什么不用纯 `renderModals`

现有 SE 功能拥有 change-actions 请求序号、目标身份 ref、多个弹窗开关、共享运输执行 ID 和
订单切换清理 effect。若用注册表中的纯函数渲染，会迫使 `detail.tsx` 继续持有全部状态，或把
十余个 setter/ref 放进 context，无法实现真正解耦。

因此定义提供一个正常 React 组件 `DetailFeatures`。该组件由 React 渲染并合法使用 Hook，
通过 render-prop 向通用详情布局贡献 UI 描述。

```ts
export interface OrderDetailFeatureContribution {
  headerActions?: React.ReactNode;
  moreMenuItems?: MenuProps['items'];
  appendSections?: OrderFormTemplateSection[];
  overlays?: React.ReactNode;
  refreshTypeState?: () => Promise<void>;
}

export interface OrderDetailFeaturesProps {
  context: OrderDetailFeaturesContext;
  children: (
    contribution: OrderDetailFeatureContribution,
  ) => React.ReactNode;
}
```

页面结构概念如下：

```tsx
const DetailFeatures = definition.DetailFeatures ?? EmptyDetailFeatures;

return (
  <DetailFeatures key={orderFormIdentity} context={detailFeaturesContext}>
    {(features) => renderCommonDetailLayout(features)}
  </DetailFeatures>
);
```

动态变化的是组件类型，不是 Hook 调用；订单身份变化通过 key 重挂载扩展，旧状态不会复用。

### 6.2 Context 边界

Context 只提供跨边界所需的稳定业务输入和命令：

- 当前 `orderId`、`order`、`orderFormIdentity`；
- `businessWritesDisabled` 与阻断原因；
- 绑定当前业务类型的权限判断函数；
- 海运扩展实际需要的搜索函数和箱型候选项；
- `refreshOrderAndLock()`：业务写成功后的普通快照刷新，不清草稿；
- `ensureBusinessWriteAllowed()`：复用通用锁单提示。

Context 不暴露草稿键、dirty setter、显式刷新 token、通用页面的局部 state 或模板 actions ref。

### 6.3 Contribution 合并规则

- `headerActions` 插入现有“异常情况”和“更多操作”之间，保持拆票、改配顺序；
- `moreMenuItems` 放在通用更多菜单项之前，保持共享航次、共享箱位于顶部；
- `appendSections` 放在通用审计时间线之前，保持同批订单和拆票/改配历史顺序；
- `overlays` 与通用费用、异常、放货面板并列挂载；
- `refreshTypeState` 只供锁状态同步或类型相关成功回调使用，不参与显式刷新令牌消费。

### 6.4 Header 接口

`OrderDetailHeader` 删除 SE 命名的 `canSplit/onOpenSplit/canReassign/onOpenReassign` 等 props，
改为一个 `businessActions?: ReactNode` 插槽。按钮本身留在 SE 扩展中，从而保持样式和禁用提示，
同时让通用 Header 不认识拆票与改配。

## 7. 权限边界

不设计 `OrderCapability`：

- 后端 Manifest 已决定 operation 是否对某业务类型存在；
- `access.canOrder` 决定当前用户是否具备该 operation；
- `order.allowedActions` 决定当前订单状态是否允许动作；
- `businessWritesDisabled` 决定锁单等运行时写入口是否关闭。

SE 扩展组合这三类现有事实控制按钮。注册表只决定“由哪个模块实现 UI”，不复制“用户是否
可操作”的判断。

## 8. 生命周期与竞态边界

- `OrderFormTemplate` 不修改；
- 显式刷新 token 继续由 `detail.tsx` 持有；
- SE change-actions 的 request id 和身份 ref 移入 `SeaExportDetailFeatures`；
- 订单身份变化时，通用详情和 SE 扩展都以同一个 `orderFormIdentity` 为 key；
- 扩展迟到请求必须同时校验请求序号和订单身份；
- 扩展触发普通数据刷新不得调用 `resetTo`，只有通用“刷新数据”成功且 token 有效时可重置
  表单与草稿。

## 9. 兼容与迁移

项目未上线且明确不保留历史兼容：

- 一次性迁移全部调用方；
- 删除旧导出，不提供 deprecated alias；
- 不迁移浏览器数据，草稿 key 仍使用相同 kind/path，因此现有 SE 草稿自然保持原语义；
- 不注册未来类型占位项；
- 不修改 API 或持久化数据。
- 不改写服务端 SE 白名单，也不提前创建只有一个实现的 validator registry。

## 10. 测试设计

### 注册表契约

- `sea-export` 与合法路径解析为同一引用；
- 空值、未知值、SI/AE/AI/LAND/RAIL 返回 `undefined`；
- 注册项元数据与生成枚举一致；
- 代码检索门禁证明旧配置符号和本地 kindMap 消失。

### SE 转换等价

- 用固定表单夹具断言创建请求完整结构；
- 用固定订单/单证/人员夹具断言详情初始值；
- 用固定详情表单夹具断言更新请求；
- 覆盖默认服务、普货 fallback、创建人、日期、海运单证、人员、箱量和空值处理；
- 测试最终请求值，不能只 mock 并断言适配器被调用。

### 页面与扩展行为

- 列表、新建、详情、费用未知类型 404 且不发请求；
- 新建、详情使用同一注册定义；
- 拆票/改配按钮顺序和阻断原因；
- change-actions 同订单乱序、A/B 与卸载迟到；
- 弹窗成功后的普通刷新；
- 共享箱在订单切换时关闭并清除旧运输执行身份；
- 通用详情没有 Sea 专属 import。

### 生命周期回归

- 模板草稿恢复、resetTo、保存成功/失败；
- 详情显式刷新同订单乱序、A/B、ABA；
- readonly、锁单、组织切换和 tab 关闭守卫。

## 11. 回滚

本任务无数据迁移。若 SE 行为回归，整体回滚前端注册层提交即可；不得通过恢复旧 API 兼容
桥、增加 fallback 或清空 sessionStorage 规避问题。
