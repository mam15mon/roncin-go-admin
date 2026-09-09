# 状态管理

## 服务端状态

- 接口数据优先由 React Query 等服务端状态工具管理；不复制到全局可变状态。
- 缓存键按业务领域命名，列表查询携带与后端一致的筛选参数。

## 请求客户端

- 所有后端请求经过统一请求配置或 OpenAPI 生成客户端
  （`src/services/roncin/`）；禁止页面自行拼接后端主机地址。
- 开发期走 Umi 代理，生产同域（Go 服务同时提供 `/api/*` 与静态资源）；
  修改打包或路由时两种路径都要验证。

## 本地状态

- 表单、弹窗开合等 UI 状态留在组件内；跨页面共享的 UI 偏好才考虑全局。
- 不把接口响应镜像进全局 store 再派生——直接消费服务端状态层。

## 场景：跨页签表单草稿

### 1. 适用范围

- 适用于 `OrderFormTemplate` 等在 `sessionStorage` 暂存未提交表单、并由
  `TagsView` 在后台页签关闭时继续判断脏状态的场景。
- 该存储跨组件挂载存在，必须视为用户与组织级业务数据，不能只按路由命名。

### 2. 签名

```ts
getFormDraftScope(userId?: string, organizationId?: string): string | undefined;
getFormDraftKey(tabKey?: string, pathname?: string, draftScope?: string): string;
hasTabDraft(tabKey: string, draftScope?: string): boolean;
clearTabDrafts(tabKey: string, draftScope?: string): void;

// OrderFormTemplate 动作接口：页面外部清理草稿与脏状态只能经由该接口。
interface OrderFormTemplateActions<T> {
  resetTo: (values?: Partial<T>) => void;
}
```

### 3. 契约

- 草稿键必须同时包含用户 ID、当前组织 ID、稳定页签 key 与完整 pathname；缺少用户或组织时
  禁止持久化草稿。
- 草稿身份由页面显式提供：`OrderFormTemplate` 只接收调用方传入的
  `tabKey + draftPathname + draftScope`（如 `/orders/:kind/new`、`/orders/:kind/:id`），
  不读取 `window.location`，也不从当前路由猜测 tabKey；任一身份输入缺失时不生成草稿键、
  不读写持久草稿。
- 完整 `draftKey` 只允许在 `OrderFormTemplate` 内通过 `getFormDraftKey` 计算；页面不得
  自行计算草稿键或直接调用草稿清理函数。
- 脏状态与草稿生命周期由模板独占：模板维护 `internalDirty` 并注册 `useTabCloseGuard`，
  不再提供受控 `dirty/onDirtyChange` props。页面级显式刷新成功、底部「重置修改」等外部
  动作只能调用 `actionsRef.resetTo(values)`（清当前草稿 → 重置 Form store → 回填最新值 →
  清 dirty）。
- 草稿恢复由模板在首次同时满足「非 loading、非 readonly、草稿身份完整」时通过浏览器绘制
  前的 layout effect 执行一次；初始只读或加载中的页面不读取草稿，首次变为可编辑时仍可恢复，
  同身份后续锁状态往返不得覆盖内存中的表单值。
- 身份重建由双层边界保证：用户与组织身份由 `OrganizationWorkspace` key 管理（切换时整体卸载
  页签与页面），页面内资源身份（如业务类型或记录 ID）由调用方在模板边界提供对应 React `key`
  （如 `key={`${config.kind}:${orderId}`}`）。
- 模板挂载期间 `(draftScope, tabKey, draftPathname)` 必须保持同一身份，模板内部不再进行
  多重身份探测；只要页面内资源身份可能变化，调用方就必须在模板边界提供对应 key。
- 新 identity key 挂载新表单实例时，先使用新身份的 `initialValues`，再恢复新身份自己的
  草稿；不得保留旧组织的 Form store 与 dirty 状态。
- 页签脏状态取“实时 guard 或当前身份持久草稿”，防止 React 状态尚未提交时漏掉已同步写入
  的草稿。
- 用户确认关闭后，只清除当前身份、目标页签下的草稿；其他用户或组织的草稿保持不变。
- 订单保存的成功/失败只由订单更新接口决定；更新成功由模板统一清草稿与 dirty，详情与锁状态
  的后台刷新是 best-effort 任务，不得把已落库的保存改判为失败。详情页通过 `<OrderDetailContent key={orderFormIdentity}>`
  进行 Keyed 容器隔离：路由在不同订单间切换时完全卸载旧实例并挂载新实例，从组件树层面物理隔离表单动作 Ref、
  局部状态与异步闭包，阻断 ABA（A→B→回 A 往返）场景下的代际污染。显式刷新在实例内采用单调请求序号门禁，连点
  刷新时只有最后一次请求允许回填，旧刷新的迟到完成直接丢弃；组件卸载后在途的刷新结果静默退出，A 的迟到刷新
  不得清 B 的草稿或重置 B 的表单，旧实例的迟到刷新不得覆盖重新挂载实例中的内容。
- 日期反序列化覆盖订单表单使用的 `*Date`、`*Cutoff`、`*At` 以及 `etd`、`eta` 等字段，普通
  ISO 格式文本不得仅凭值形态被转换。

### 4. 校验与错误矩阵

| 条件 | 行为 |
| --- | --- |
| 用户 ID 或组织 ID 缺失 | 不生成草稿键，不读写草稿；实时关闭 guard 仍工作 |
| 同用户切换组织 | 重建表单，只加载新组织命名空间 |
| 同组织切换用户 | 不读取原用户草稿 |
| 存储不可用、超限或 JSON 非法 | 捕获存储异常；读取返回 `null`，不阻断表单 |
| 实时 guard 为 false、当前身份已有草稿 | 仍判定为 dirty |
| 用户取消关闭 | 保留页签与草稿 |
| 用户确认关闭 | 仅清理当前身份与目标页签草稿 |

### 5. Good / Base / Bad

- Good：用户 A 在组织 1 输入后切到组织 2，页面展示组织 2 默认值或其自身草稿；组织 1 草稿
  留存且不会显示。
- Base：没有草稿时按当前表单 `initialValues` 渲染，关闭页签不提示。
- Bad：使用 `roncin:form-draft:${tabKey}:${pathname}`，导致不同用户或组织共享同一个键。

### 6. 必需测试

- 工具测试：同页签、同路径的两个身份命名空间可以独立保存、判断和清理。
- 模板测试：缺少任一草稿身份输入（tabKey / draftPathname / draftScope）时不持久化，完整
  输入只命中唯一规范路径键；只读挂载不恢复草稿、初始只读在首次解除后只恢复一次且后续
  readonly 往返不覆盖内存编辑值；提交成功清草稿、提交失败保留草稿、原生重置清草稿；
  `resetTo` 清当前草稿、清旧 Form store、回填新值并清 internal dirty。
- 详情生命周期测试：详情 A 原地导航到 B 后模板实例被重建，B 不显示 A 的 Form store 与
  dirty 状态；A 的迟到显式刷新不得重置 B 的表单与 dirty；保存成功立即清草稿，后台刷新
  失败不改判保存结果；底部重置修改与显式刷新成功经由 `actionsRef.resetTo` 生效。
- 新建页测试：页面向模板传入规范 `tabKey + draftPathname + draftScope`，并从规范路径键
  恢复新建草稿。
- 关闭保护测试：实时 guard 暂为 false 但草稿已写入时仍提示；确认后仅清理当前 scope。
- 日期测试：嵌套对象中的 `cargoReadyAt`、截止时间及 `etd`/`eta` 恢复为 Dayjs，普通文本保持
  字符串。

### 7. 错误与正确示例

```ts
// 错误：组织切换后仍命中同一个草稿。
getFormDraftKey(tabKey, pathname);

// 错误：页面自行计算草稿键并直接清理，绕过模板生命周期所有权。
const draftKey = getFormDraftKey(tabKey, pathname, draftScope);
clearFormDraft(draftKey);

// 正确：页面只提供规范身份，模板独占草稿键与清理动作。
<OrderFormTemplate
  tabKey={resolveTabKey(`/orders/${kind}/${orderId}`)}
  draftPathname={`/orders/${kind}/${orderId}`}
  draftScope={getFormDraftScope(user.id, currentOrganization.id)}
  actionsRef={templateActionsRef}
/>;

// 需要外部重置表单时，经由模板动作接口执行。
templateActionsRef.current?.resetTo(initialValues);
```

## 场景：列表筛选驱动的前端 CSV 导出

### 1. 使用范围

- 适用于同一页面中的服务端分页列表与全量导出共用搜索条件的场景。
- 导出接口必须使用 OpenAPI 生成客户端；页面只负责筛选规范化、CSV 序列化与
  浏览器下载，不得自行拼接后端地址或重新推导业务金额。

### 2. 数据签名

- 搜索表单值与请求筛选对象必须使用不同类型；日期控件值只能停留在表单类型中。
- 提交搜索时，通过单一纯函数把表单值转换为后端请求字段，例如月份开始值转为
  当月首日、月份结束值转为当月末日，单边月份允许另一端缺省。
- 列表请求与导出请求必须读取同一个已提交筛选对象；分页字段只在列表请求处追加。

### 3. 正常契约

- 搜索提交顺序固定为：规范化表单值、同步写入已提交筛选引用、触发列表刷新。
  禁止先异步更新 React state 后立即刷新，并期待请求回调读到新值。
- 导出按钮读取已提交筛选，不读取尚未提交的表单编辑值；因此屏幕列表与导出数据
  始终对应同一组查询条件。
- CSV 列必须与导出 DTO 显式逐列映射并固定顺序；空结果不创建 Blob 或下载链接。
- CSV 使用 UTF-8 BOM 与 CRLF 行分隔，包含逗号、双引号、回车或换行的单元格按
  RFC 4180 形式加引号并把双引号写成两个双引号。

### 4. 校验与错误矩阵

| 输入或结果 | 处理方式 |
| --- | --- |
| 关键字只有首尾空格 | 去除首尾空格；结果为空时不发送该筛选字段 |
| 仅选择开始月份 | 只发送开始月首日 |
| 仅选择结束月份 | 只发送结束月末日 |
| 文本字段以 `= + - @` 开头，或前导空白后出现这些字符 | 在原值前加单引号，阻止表格软件执行公式 |
| 后端控制的日期、时间、decimal 字段 | 保持后端字符串原值，不做公式保护，避免改写合法负数 |
| 导出结果为空 | 提示用户无可导出数据，不创建下载对象 |
| 导出请求失败 | 由统一请求错误处理呈现失败，不生成部分文件 |
| 导出成功 | 点击临时下载链接后立即移除链接并回收 Blob URL |

### 5. 基础、正常与边界行为

- 基础：无筛选时，列表和导出均不携带可选筛选字段，文件名使用导出当天日期。
- 正常：双边月份转为首日和末日，列表与导出收到完全相同的业务筛选字段。
- 边界：覆盖同月、跨月、跨年、闰年、仅开始月、仅结束月、中文、双引号、逗号、
  CR/LF、公式前缀、前导空白公式前缀、合法负数与空结果。

### 6. 测试要求

- 纯函数测试必须逐列验证 DTO 到 CSV 的映射、转义、公式保护和文件名规则。
- 页面测试必须实际提交搜索并触发列表与导出，断言两者复用同一规范化筛选；只测
  按钮显隐不能证明该契约。
- 下载测试必须断言空结果不调用 `URL.createObjectURL`，成功结果会移除临时链接并
  调用 `URL.revokeObjectURL`。

### 7. 错误与正确示例

```tsx
// 错误：state 更新异步，reload 可能读取旧筛选；导出又从表单单独计算一遍。
setFilters(normalize(values));
actionRef.current?.reload();
exportCommissions(normalize(form.getFieldsValue()));

// 正确：提交时同步固化一次，列表与导出共同读取。
committedFiltersRef.current = normalize(values);
actionRef.current?.reload();
exportCommissions(committedFiltersRef.current);
```

```ts
// 错误：对所有单元格做公式保护，会把受控 decimal 的合法负数改成文本。
escapeCsvCell(protectFormula(value));

// 正确：仅自由文本字段保护公式；受控日期和 decimal 保持后端原值。
escapeCsvCell(column.kind === 'text' ? protectFormula(value) : value);
```
