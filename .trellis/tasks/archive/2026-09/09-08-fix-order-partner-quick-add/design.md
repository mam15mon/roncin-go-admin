# 修复订单伙伴快捷新增验收缺口：技术设计

## 设计原则

本任务在现有实现上做窄修复，不重写伙伴字段或弹窗。核心方法是把“当前组织”“是否只读”
和“是否正在保存”各自收敛为单一真相源，再用确定性的测试覆盖异步边界。服务端继续保持
service/biz/data 分层：是否重试属于 biz 用例决策，唯一冲突仍由 data 层映射。

## 1. 详情页有效只读值

详情页保留一个局部 `effectiveReadonly`：

```ts
const effectiveReadonly =
  order?.allowedActions?.includes(
    OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT,
  ) !== true || businessWritesDisabled;
```

该值同时用于 `templateProps.readonly` 和 `OrderFormTemplate.readonly`。不在
`SeaBasicInfoSection` 或 `PartnerQuickAddSelect` 内重新访问订单动作权限；子组件只消费最终的
布尔值。页面测试通过 mock 模板构建器和表单模板捕获两条传参，分别覆盖无 EDIT 动作和锁单。

当前工作区的 `detail.tsx` 已被用户表单草稿 WIP 修改，且包含上述片段。实施时以当前工作树为
基准确认差异：若逻辑已经正确，只补验证并将这几行作为独立 hunk 暂存；不移动、不格式化附近
草稿代码。

## 2. 组织身份与请求世代

`PartnerQuickAddSelect` 使用同步更新的 ref 保存最新组织：

```ts
latestOrganizationIdRef.current = currentOrganizationId;
```

这句在 render 期间执行，因此新组织 render 完成后、effect 尚未运行的窗口中，旧 Promise 也会
看到新身份。搜索和创建各自使用单调递增的请求序号：请求开始时捕获
`requestOrganizationId` 与序号，返回后仅当二者仍分别等于最新组织和最新序号时才处理结果。

组织变化 effect 负责可见状态清理和主动失效：关闭下拉、关闭弹窗、清空本地新增选项，并增加
两个请求序号。它不承担“让 ref 变新”的职责，因此不存在 effect 时序窗口。上下文已失效的
请求安静结束，不展示针对旧页面的成功或失败消息，也不修改表单。

测试使用人工控制的 deferred Promise：先发请求，再 rerender 为新组织，最后 resolve 旧请求；
不能用“先成功再切换”的顺序替代竞态测试。

## 3. 受控下拉与可访问入口

选择器增加 `selectOpen` 状态，并将 `open`、`onOpenChange` 传入字段属性。`popupRender` 底部
使用 `Button type="text"`，显式 `htmlType="button"`。鼠标按下只阻止选择器因失焦提前卸载，
click/keyboard 激活走同一个处理函数：先 `setSelectOpen(false)`，再 `setModalOpen(true)`。

无 `canCreatePartners` 或 `readonly` 时不传 `popupRender`。组件在 readonly 从 false 变 true 时
也关闭已打开的下拉和弹窗，避免权限或业务状态变化后保留旧交互面。

## 4. QuickCreateModal 保存状态机

公共弹窗继续由 `saving` 驱动视觉状态，同时增加同步 `savingRef` 作为重入门：

```text
idle --首次保存--> saving --成功--> close/reset
                         \--失败--> idle + 保留输入
```

- `handleSave` 首行检查 ref；首次进入时同步置 true，再更新 state。
- `finally` 同步释放 ref 和 state；成功路径的关闭与重置只发生一次。
- 保存期间附加动作、取消按钮均 disabled；`handleCancel` 与附加动作处理器还要检查 ref，防止
  DOM 事件在 React 状态更新前穿透。
- Modal 的右上角、遮罩和键盘关闭能力随 saving 禁用；即使 Ant Design 仍触发 `onCancel`，
  guard 也不会关闭。
- 失败仍沿用现有错误展示，并保留字段值。未传 `extraAction` 时布局和行为不变。

## 5. 服务端自动编码重试

`PartnerUsecase.Create` 先判断 normalized code 是否为空：

- 显式代码：直接调用仓储一次，任何错误原样返回。
- 自动代码：每轮调用代码生成器产生一个新候选，再调用仓储；只在
  `errors.Is(err, ErrPartnerCodeExists)` 时继续，最多 3 次。
- 代码生成失败、其他约束错误或仓储错误立即返回。

为避免随机测试，引入仅在 biz 包内部使用、默认指向 `generatePartnerCode` 的生成函数依赖；
测试替换为返回固定序列的函数，精确断言候选值和调用次数。它不是公共接口，不进入 service 或
data 层。仓储 stub 记录每次收到的代码，证明每次重试确实使用新候选。

## 6. 费用面板同源契约

费用面板保留自己的 `QuickAddPartnerModal`，但修正创建边界：

- 表单统一社会信用代码改为必填，normalize 为 trim 后大写，并使用既有 18 位校验规则；
- 请求不提供 code；所选角色转换为 `{ type, enabled: true }`；
- 成功回调和显示名称保持原状。

测试不再只 mock `QuickCreateModal` 后检查 props，而是至少增加一组真实 modal 集成用例，填写
字段、选择角色并点击保存，断言完整请求和成功回调。组件级 mock 测试可保留作为轻量结构验证。

## 7. 直接验收测试

### 订单伙伴字段

- 权限与 readonly 三态；真实 Button 的键盘激活与受控下拉关闭。
- deferred 搜索和创建各一例组织切换竞态。
- 委托单位普通选择、快捷成功、清空三条 `customerCode` 路径。
- 创建后远程搜索返回相同 ID，断言稳定去重和 label 保留。
- 双击保存只调用一次创建 API。

### 公共弹窗与伙伴页面

- 保存 pending 时取消、关闭和附加动作均无效；失败后重新可用。
- 伙伴创建页预填、地址参数变化重建创建默认值、编辑模式忽略参数。
- 三角色详情跳转与特殊字符编码继续覆盖。

### 服务端

- 自动模式按指定三个代码依次冲突/成功；三次冲突后返回冲突。
- 显式重复代码只调用一次。
- 非代码冲突与生成器错误均不重试。

### 既有门禁维护

`orders-breadcrumbs.test.tsx` 只更新与当前规范不符的“订单管理”旧期望，继续断言真实业务实体
面包屑和标题。不得为通过门禁修改页面实现。

## 8. 变更隔离与提交策略

预计分成三组可验证提交：

1. `fix(server): 修正伙伴自动编码冲突重试`
2. `fix(web): 收紧伙伴快捷新增交互边界`
3. `test(web): 补齐伙伴快捷新增验收覆盖`

若费用面板逻辑与测试适合随第二组一并验证，归入第二组。`detail.tsx` 和
`SeaBasicInfoSection.tsx` 必须使用交互式/补丁式暂存，只提交本任务 hunk；每次提交前同时检查
`git diff` 与 `git diff --cached`。不使用 reset、checkout 或自动格式化整个脏文件。

## 9. 验证策略

开发阶段先跑受影响的 Vitest、biz 包测试、修改文件 Biome 和 `git diff --check`；跨层逻辑稳定
后运行前端 tsc、`go test ./...`、`go vet ./...`。任务最终验收只运行一次 `pnpm run check:web`。
追加授权纳入运行时布局配置与静态品牌资源，因此最终验收增加一次 `pnpm run build`。

## 10. 追加授权：草稿隔离与品牌审核

订单表单草稿仍由 `OrderFormTemplate` 统一保存和恢复，但键名增加用户与当前组织组成的
`draftScope`。`TagsView` 使用同一 scope 判断、确认和清理后台页签草稿；实时 guard 与持久
草稿取逻辑或，避免表单变更同步写入 storage 后、React dirty state 尚未提交的窗口漏拦截。
`draftScope` 变化时以草稿键作为 ProForm 的 React key 重建表单，确保旧组织 Form store 不会
残留。日期恢复除既有 `Date`、`Cutoff`、`etd`、`eta` 外覆盖订单使用的 `*At` 时间字段。

详情页不得再对 `initialValues`、`effectiveReadonly` 变化做通用 `setFieldsValue`：锁状态同步会先
关闭写入，再请求同一订单，二者都不是新的表单上下文。首次初始值和草稿由 `OrderFormTemplate`
按草稿 key 接管；详情页只为用户明确点击“刷新数据”设置一次 reset intent，并在 `loadData` 完成
后的下一次渲染用最新 `initialValues` 清草稿、重置 Form。加载失败且订单为空时消费该 intent，
不清除原输入或草稿。

运行时菜单关闭 locale 查找，但菜单头继续使用 Umi `Link`，避免 div 丢失键盘/新标签语义，
也避免内层 click 与 `onMenuHeaderClick` 冒泡造成双重跳转。品牌二进制资源核对文件类型、尺寸
和透明通道；HeaderDropdown 切换到 `popupRender`，兼容参数只在公共包装组件内收口。
