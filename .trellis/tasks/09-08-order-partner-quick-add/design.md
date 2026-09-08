# 订单表单伙伴字段快捷新增技术设计

## 设计目标与边界

本任务在现有海运订单模板中增加订单专属的伙伴快捷创建字段，复用既有伙伴创建接口和公共
`QuickCreateModal`。实现只修改 Web 前端，不新增接口、权限码、全局状态或数据兼容逻辑。

最小职责划分如下：

```text
订单新建页 / 详情页
  └─ 计算并传递模板上下文（含有效 readonly）
      └─ SeaBasicInfoSection
          └─ PartnerQuickCreateField（字段、下拉入口、本地选项、快捷弹窗）
              ├─ partnerServiceCreatePartner
              └─ QuickCreateModal（通用弹窗框架与附加动作）

添加公司详情
  └─ /partners/<role>/create?legalName=...
      └─ partner-detail 创建模式读取并预填
```

伙伴角色、订单字段和目标路由由一个静态配置对象传给字段组件，不在组件内部通过字段名猜测
角色或路由。该组件属于订单页面域，放在海运模板组件目录附近；不提升到 `@/components/ui`，
避免公共 UI 层持有订单字段和伙伴角色等业务知识。

## 有效只读状态

订单详情页定义并复用同一个有效只读值：

```ts
const formReadonly =
  !hasAction(OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT) ||
  businessWritesDisabled;
```

该值同时传给 `OrderFormTemplate.readonly` 与 `TemplateProps.readonly`。新建页不传时按 `false`
处理。伙伴字段只有在 `!readonly && access.canCreatePartners` 时提供 `popupRender` 新增入口。

这项统一也会让模板内部已经依赖 `TemplateProps.readonly` 的操作组件与整个表单的权限状态保持
一致。实施时必须运行现有海运模板定向测试，确认没有把纯查看内容隐藏或改变字段值。

## `QuickCreateModal` 公共契约

在现有 Props 上增加两个可选属性，默认值保持当前行为：

```ts
interface QuickCreateExtraAction<TFormValues> {
  label: ReactNode;
  onClick: (form: FormInstance<TFormValues>) => void;
}

interface QuickCreateModalProps<TFormValues, TResult> {
  // existing props...
  extraAction?: QuickCreateExtraAction<TFormValues>;
  centered?: boolean;
}
```

- `centered` 默认 `false`，原样传给 Ant Design Modal；订单快捷新增显式传 `true`。
- 仅在 `extraAction` 存在时使用 Modal 的 footer render 扩展默认 footer，顺序为附加动作、取消、
  确认；取消和确认继续使用 Modal 提供的 `CancelBtn`、`OkBtn`，保留现有 loading、文案与关闭
  语义。
- 附加动作使用 `htmlType="button"`，直接调用 `onClick(form)`，不调用 `validateFields`、
  `handleSave` 或 `onSubmit`。
- 保存中禁用附加动作，避免创建请求与页面跳转并发。
- 不改变当前 `onSubmit`、成功回调、错误提示和重置规则。订单伙伴组件遇到响应缺少 ID 时主动
  抛错，从而进入既有失败路径并保留输入。

## 订单伙伴字段组件

新增 `PartnerQuickCreateField`，建议契约如下：

```ts
interface PartnerQuickCreateFieldProps {
  name: 'customerId' | 'bookingAgentId' | 'foreignAgentId';
  label: string;
  roleType: PartnerRoleType;
  createPath: string;
  required?: boolean;
  readonly?: boolean;
  request: (keyword?: string) => Promise<SelectOption[]>;
  onPartnerChange?: (option?: SelectOption) => void;
}
```

组件内部持有：

- 下拉显隐状态；
- 快捷弹窗显隐状态；
- 最近快捷创建成功的本地伙伴选项列表。

`ProFormSearchableSelect` 保留 `allowClear` 和远程 `request`。每次请求完成后，将远程结果与本地
选项按 `value` 去重合并。本地选项优先，保证刚创建的名称不会因搜索返回缺少该记录而退化成
ID；远程已经返回同一 ID 时只显示一项。

普通选择继续通过 `fieldProps.onChange` 把当前 option 交给 `onPartnerChange`。清空时传
`undefined`。创建成功时不能依赖程序化 `setFieldValue` 触发 `onChange`，而是依次执行：

1. 生成 `{ value: id, label: legalName, code }`；
2. 合并到本地选项；
3. `form.setFieldValue(name, id)`；
4. 显式调用 `onPartnerChange(newOption)`；
5. 关闭快捷弹窗。

委托单位使用 `onPartnerChange` 更新 `customerCode`；另外两个字段不传该回调。

角色值从 `@/enums.generated` 导入 `PartnerRoleType` 类型和常量，禁止在实现或测试中复制
`1/2/3` 裸数字。字段组件通过 `@@initialState` 读取当前组织 ID，并让弹窗、本地选项与创建请求
都绑定该身份：组织变化时关闭旧下拉和弹窗、清空本地选项；异步搜索和创建完成后再次核对发起
组织，身份不一致时不得设置字段、选项或派生的 `customerCode`。

## 下拉与弹窗交互

选择器使用受控 `open` 和 `onOpenChange`。下拉底部操作行通过 `popupRender` 组合原选项菜单与
分隔边框，点击处理遵循：

1. 阻止按钮鼠标事件触发选择器选项或订单表单提交；
2. 设置下拉 `open=false`；
3. 设置快捷弹窗 `open=true`。

无伙伴创建权限或 `readonly=true` 时不传 `popupRender`。即使父级 ProForm 已按只读方式渲染，
组件也必须独立满足“不渲染入口”的验收要求。

## 创建请求与错误处理

快捷保存对公司抬头执行表单必填、空白和最大 200 字符校验，提交时再次 `trim()`，请求固定为：

```ts
partnerServiceCreatePartner({
  legalName: normalizedLegalName,
  roles: [{ type: PartnerRoleType.PARTNER_ROLE_TYPE_* }],
});
```

响应没有 `data.id` 时抛出明确错误，不返回 `undefined`。网络、业务或响应结构错误均由
`QuickCreateModal` 的既有错误路径展示，弹窗与输入保持不变。成功后才修改订单字段；不做自动
重试、乐观临时 ID 或失败回滚分支。

创建请求发起前捕获当前组织 ID，响应后与当前 `organizationId` ref 比较。组织已变化时丢弃响应
对表单和选项的影响，并显示“当前组织已切换，请重新操作”一类明确提示；不能把旧组织创建结果
作为新组织字段值。远程搜索 Promise 同样在返回前核对组织，不一致时返回空数组。

## 完整档案跳转与预填

附加动作从 QuickCreateModal 提供的 form 读取 `legalName` 并 `trim()`：

- 非空：用 `URLSearchParams` 生成 `?legalName=<encoded>`；
- 空白：只跳转 `createPath`；
- 两种情况都不调用表单校验和伙伴创建接口。

`partner-detail` 从 `location.search` 解析预填值，仅在 `partnerId` 不存在的创建分支，把它与
现有创建默认值放入同一次 `setFieldsValue`。创建初始化 effect 依赖解析后的预填值；编辑分支
完全忽略它。直接通过地址传入超过字段合法范围的内容不截断，仍由伙伴表单自身校验，避免静默
纠错。

## 状态、兼容与生成物

- 本地新建选项只属于当前字段组件，不进入全局状态，也不改写服务端搜索缓存。
- 不修改 `web/src/services/roncin/`、`web/types/`、权限生成文件或 OpenAPI 输入。
- 不改变费用面板 `QuickAddPartnerModal` 的 props、请求或表单字段。
- `QuickCreateModal` 的新属性全部可选；既有调用不需要迁移。
- 跳转完整档案会按现有路由行为离开订单页面，不新增草稿保护或自动返回。

## 测试设计

### 公共弹窗

- 附加动作出现且点击可读取当前表单，但不会触发校验或 `onSubmit`；
- 保存中附加动作不可并发触发；
- 未传附加动作时仍只有既有取消/确认操作；
- 既有“提交失败保留输入并显示错误”测试继续通过。

### 伙伴字段

- 有权限且可编辑时显示对应文案；无权限和只读分别不显示；
- 点击入口关闭下拉并打开居中弹窗，不提交订单表单；
- 三种字段配置分别发送正确角色；
- 成功后设置字段值、调用变更回调并保留可显示选项；重复搜索结果去重；
- API 抛错和缺少 ID 都保留弹窗、输入和原字段值；
- 组织切换会清理本地状态，迟到的搜索与创建响应不写入新组织表单；
- 详情跳转正确编码公司抬头，空白时不携带参数且不触发创建。

### 页面与模板回归

- `partner-detail` 创建模式预填、查询参数变化更新和编辑模式忽略；
- 海运模板三个字段仍存在，委托单位必填，空运模板不受影响；
- 详情页有效 readonly 同时覆盖动作权限和业务锁定；
- 费用面板及 QuickCreateModal 既有用法回归。

## 风险与回滚

- 主要风险是公共 Modal footer 扩展改变既有按钮行为，因此必须复用 Modal 提供的默认按钮并运行
  既有调用回归测试。
- 第二个风险是远程搜索覆盖本地新伙伴选项，必须在每次请求结果上做稳定去重合并。
- 第三个风险是详情页两套 readonly 漂移，必须以单一 `formReadonly` 同时驱动表单与模板。
- 本任务无数据迁移和后端发布顺序要求。若前端回滚，整体回退公共组件扩展、订单字段接入和
  伙伴预填即可，不留下服务端或数据兼容负担。
