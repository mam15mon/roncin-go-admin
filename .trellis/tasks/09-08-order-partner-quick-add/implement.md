# 订单表单伙伴字段快捷新增实施计划

## 开始前检查

- [ ] 运行 `git status --short`，确认除本任务规划外没有来源不明或与目标文件重叠的改动。
- [ ] 完整读取根目录及目标目录适用的 `AGENTS.md`、本任务 `prd.md`、`design.md`、
      `implement.md` 和 `implement.jsonl` 全部引用。
- [ ] 运行 `task.py current` 确认活动任务路径就是本任务；没有活动任务时不得直接修改产品代码。
- [ ] 核对 Ant Design、ProForm 和 Umi 当前 API 类型，以仓库已安装版本为准，不引入新依赖。
- [ ] 记录目标文件初始差异，后续提交只包含本任务文件。

## 第一组：扩展公共快捷创建弹窗

- [ ] 在 `QuickCreateModal` 增加带泛型表单实例的可选 `extraAction` 和默认关闭的 `centered`。
- [ ] 使用 Ant Design Modal footer render 保留默认 `CancelBtn`、`OkBtn`，仅在配置时插入
      `htmlType="button"` 的附加动作。
- [ ] 保存期间禁用附加动作；点击附加动作只调用其回调，不校验、不提交、不重置表单。
- [ ] 扩充 `QuickCreateModal.test.tsx`：覆盖附加动作、无提交/无校验、保存中禁用、无配置兼容
      和既有失败保留输入行为。
- [ ] 运行公共组件定向测试和该文件 Biome；差异稳定后提交：
      `feat(web): 扩展快捷创建弹窗附加操作`。

定向验证：

```bash
pnpm --dir web exec vitest run src/components/ui/quick-create-modal/QuickCreateModal.test.tsx
pnpm --dir web exec biome check src/components/ui/quick-create-modal/QuickCreateModal.tsx src/components/ui/quick-create-modal/QuickCreateModal.test.tsx
git diff --check
```

## 第二组：实现并接入订单伙伴快捷新增字段

- [ ] 在海运模板组件目录新增 `PartnerQuickCreateField` 及其定向测试，业务配置保留在订单域。
- [ ] 实现受控下拉、`popupRender` 固定入口、权限与 readonly 门、本地新伙伴选项和远程结果按
      ID 去重合并。
- [ ] 使用生成的 `PartnerRoleType` 常量；绑定当前组织身份，组织切换时清理旧状态，并阻止迟到
      的旧组织搜索或创建响应回填新组织页面。
- [ ] 实现只含公司抬头的居中快捷弹窗；提交前 trim，固定发送单一角色；响应缺少 ID 时抛错。
- [ ] 创建成功后按设计顺序缓存 option、设置当前表单字段、显式触发伙伴变更回调并关闭弹窗。
- [ ] 在 `SeaBasicInfoSection` 用该组件替换委托单位、订舱代理和国外代理三个字段，保留原必填
      规则、检索函数以及委托单位 `customerCode` 同步。
- [ ] 在订单详情页提取单一 `formReadonly`，同时传给 `OrderFormTemplate` 和
      `TemplateProps.readonly`；更新依赖数组并验证锁定及无编辑动作权限两条路径。
- [ ] 更新海运模板定向测试，证明三字段接入、有效只读传递及既有字段行为没有漂移。

建议定向验证：

```bash
pnpm --dir web exec vitest run \
  src/pages/orders/templates/components/sea/PartnerQuickCreateField.test.tsx \
  src/pages/orders/templates/sea-template.test.tsx
pnpm --dir web exec biome check \
  src/pages/orders/templates/components/sea/PartnerQuickCreateField.tsx \
  src/pages/orders/templates/components/sea/PartnerQuickCreateField.test.tsx \
  src/pages/orders/templates/components/sea/SeaBasicInfoSection.tsx \
  src/pages/orders/templates/types.ts \
  src/pages/orders/detail.tsx
git diff --check
```

## 第三组：实现完整伙伴档案预填

- [ ] 在字段附加动作中读取当前公司抬头：trim 后非空则通过 `URLSearchParams` 携带
      `legalName`，空白则直接进入对应新建路由。
- [ ] 在 `partner-detail` 创建模式解析并设置 `legalName`；将预填值纳入创建初始化依赖，编辑
      模式忽略该参数。
- [ ] 增加伙伴页面定向测试：三个路由映射、特殊字符编码与解码、空白不携参、创建预填、同组件
      查询参数切换，以及编辑模式不覆盖。
- [ ] 联合运行第二、三组定向测试；执行 TypeScript 检查。全部通过后提交：
      `feat(orders): 支持伙伴字段快捷新增`。

建议定向验证：

```bash
pnpm --dir web exec vitest run \
  src/pages/orders/templates/components/sea/PartnerQuickCreateField.test.tsx \
  src/pages/orders/templates/sea-template.test.tsx \
  src/pages/partners/partner-detail.test.tsx \
  src/components/ui/quick-create-modal/QuickCreateModal.test.tsx
pnpm --dir web exec biome check \
  src/pages/orders/templates/components/sea/PartnerQuickCreateField.tsx \
  src/pages/orders/templates/components/sea/PartnerQuickCreateField.test.tsx \
  src/pages/orders/templates/components/sea/SeaBasicInfoSection.tsx \
  src/pages/orders/templates/types.ts \
  src/pages/orders/detail.tsx \
  src/pages/partners/partner-detail.tsx \
  src/pages/partners/partner-detail.test.tsx
pnpm --dir web tsc
git diff --check
```

实际测试文件名可根据既有测试组织方式调整，但不得用 `pnpm --dir web test -- <file>` 冒充
定向测试；新增辅助模块时必须把其源文件与测试加入同一组 Biome 检查。

## 第四组：最终验收与独立检查

- [ ] 复核真实差异：没有修改生成客户端、权限生成物、后端、空运模板或费用面板业务逻辑。
- [ ] 运行费用面板受影响回归测试；根据实际 import 链补充最小测试文件，不改测试脚本。
- [ ] 因变更涉及公共组件接口、共享海运模板和路由预填，执行一次完整前端门禁：

```bash
pnpm run check:web
git diff --check
```

- [ ] 浏览器抽验海运新建订单的三个字段：打开/关闭、校验失败、保存回填、再次搜索、三个详情
      跳转；再抽验一个只读详情场景、一个无伙伴创建权限场景和一次组织切换隔离。
- [ ] 独立 `trellis-check` 按 PRD 逐项检查权限、有效 readonly、角色载荷、失败保留、选项去重、
      路由预填、公共组件兼容和测试有效性。
- [ ] 修正检查发现的问题并只重跑受影响的定向检查；最终状态变化后再重跑一次必要的完整前端
      门禁，避免每轮重复执行全量测试。
- [ ] 暂存前逐文件确认只包含本任务变更；若验收修正产生独立可验证修改，使用准确的
      Conventional Commit 提交。

## 不执行的检查

- 本任务不改 Go、Proto、OpenAPI、权限清单、数据库或构建配置，不运行 `check:server`、
  `go test ./...`、生成命令或 `pnpm run build`。
- 如果实施中发现必须修改上述范围，停止实施并退回规划阶段，说明原因和新增风险，不能自行
  扩大任务范围。

## 回滚点

1. 公共弹窗扩展独立提交，可在未接入订单字段时单独回退。
2. 订单字段、有效 readonly 与伙伴页面预填作为完整功能提交共同回退，避免保留半条用户路径。
3. 回滚不得使用 `git reset --hard` 或覆盖其他未提交文件；已提交变更使用新的 revert 流程，
   未提交变更只针对本任务明确文件处理。
