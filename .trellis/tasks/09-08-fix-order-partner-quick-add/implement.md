# 修复订单伙伴快捷新增验收缺口：实施计划

## 开始前检查

- [ ] 运行 `git status --short`，保存 `detail.tsx`、`SeaBasicInfoSection.tsx` 和相关测试的初始
      diff；确认所有用户 WIP 的来源与边界。
- [ ] 完整读取适用 `AGENTS.md`、本任务 `prd.md`、`design.md`、`implement.md` 及
      `implement.jsonl` 引用的规范。
- [ ] 运行 `task.py current`，确认当前任务就是本目录，并在用户明确批准后执行
      `task.py start`；批准前不修改产品代码。
- [ ] 复核归档任务实现和当前 HEAD，确认 `CreatePartnerRequest.code` 已是可选且生成物一致，
      本任务无需再次修改 Proto 或生成客户端。
- [ ] 对脏文件逐 hunk 操作；任何来源不明或无法隔离的重叠改动立即停止并报告。

## 第一组：修正服务端自动编码重试

- [ ] 在伙伴用例中区分显式代码与自动代码路径；显式代码只调用一次仓储。
- [ ] 自动代码每次尝试都生成新候选，仅代码唯一冲突重试，最多 3 次。
- [ ] 为用例提供包内可替换的代码生成函数，生产默认使用加密随机生成器。
- [ ] 扩展伙伴 biz 测试，记录候选代码并覆盖不同候选、三次耗尽、显式冲突、非冲突错误和生成
      失败。
- [ ] 实现完成后运行相关包测试、格式检查和 diff 检查；通过后提交
      `fix(server): 修正伙伴自动编码冲突重试`。

建议验证：

```bash
gofmt -w server/internal/biz/partner.go server/internal/biz/partner_test.go
go -C server test ./internal/biz -run 'TestPartnerUsecase_Create'
git diff --check
```

## 第二组：收紧公共弹窗和订单伙伴交互

- [ ] 在 `QuickCreateModal` 增加同步保存互斥；保存中禁用取消、关闭、遮罩/ESC 和附加动作，
      失败后恢复。
- [ ] 把订单伙伴选择器改为受控 open；用真实 Button 渲染底部入口，显式先关下拉再开弹窗。
- [ ] 在 render 阶段同步最新组织 ref；搜索/创建分别用组织快照和请求序号拒绝迟到响应。
- [ ] 组织或 readonly 变化时清理旧交互状态；保持本地/远程按 ID 去重和标签显示。
- [ ] 将订单详情的同一 `effectiveReadonly` 同时传给表单模板和分节 props；只处理目标 hunk，
      不带入表单草稿 WIP。
- [ ] 修正费用面板统一社会信用代码必填及 normalize，创建角色发送 `enabled: true`，代码留空由
      服务端生成。
- [ ] 实现完成后补相邻功能测试并运行定向检查；通过后提交
      `fix(web): 收紧伙伴快捷新增交互边界`。

建议验证（以实际测试文件为准）：

```bash
pnpm --dir web exec vitest run \
  src/components/ui/quick-create-modal/QuickCreateModal.test.tsx \
  src/pages/orders/components/PartnerQuickAddSelect.test.tsx \
  src/pages/orders/components/fees/QuickAddPartnerModal.test.tsx \
  src/pages/orders/templates/sea-template.test.tsx
pnpm --dir web exec biome check <本组修改的前端文件>
pnpm --dir web tsc
git diff --check
```

## 第三组：补齐直接验收证据

- [ ] 增加详情页完整 readonly 的页面或等价集成测试，分别验证无 EDIT 动作与锁单。
- [ ] 用 deferred Promise 覆盖组织切换后的迟到搜索和迟到创建。
- [ ] 覆盖委托单位正常选择、快捷创建、清空三条 `customerCode` 路径。
- [ ] 覆盖远程返回本地同 ID 时只显示一项且当前值保留名称。
- [ ] 覆盖保存 pending 时重复保存、取消和添加公司详情均不会产生第二条流程。
- [ ] 增加费用面板真实 `QuickCreateModal` 集成测试，断言税号与角色载荷。
- [ ] 增加伙伴创建页预填、查询参数变化、编辑模式忽略参数测试。
- [ ] 更新过期的订单面包屑测试期望，使其匹配当前已存在规范，不修改页面产品行为。
- [ ] 运行所有相关定向测试、修改文件 Biome、tsc 与 diff 检查；通过后提交
      `test(web): 补齐伙伴快捷新增验收覆盖`。

## 第四组：跨层与最终验收

- [ ] 逐层复核 API 请求：订单/费用入口 → 生成客户端 → service 转换 → biz normalize → data 唯一
      错误映射；确认角色 enabled、税号和自动 code 均无丢失。
- [ ] 运行后端全量测试与 vet：

```bash
go -C server test ./...
go -C server vet ./...
```

- [ ] 因涉及公共弹窗、跨层创建契约和共享订单模板，运行一次最终前端门禁：

```bash
pnpm run check:web
git diff --check
```

- [ ] 若 `.agents` 权限产生 Biome 环境错误，记录完整输出并先定位目录所有权；禁止修改 lint
      配置或排除规则掩盖。若产品代码出现新的失败，按归属修复后只重跑相关定向检查，最终状态
      改变后再运行一次必要全量门禁。
- [ ] 浏览器抽验：可编辑订单成功创建并回填；无编辑权限/锁单无入口；保存中无法退出；组织切换
      不回填旧结果；费用面板至少创建一种角色。
- [ ] 使用 `trellis-check` 对照 AC1–AC12 独立检查真实差异、测试有效性和脏文件隔离。
- [ ] 所有检查通过后归档任务；最终提交与归档遵循 Conventional Commits，不执行 push、merge、
      rebase 或 reset。

## 不执行

- 不运行 Proto/OpenAPI/权限生成命令，除非复核发现生成物与当前 Proto 不一致；若必须修改契约，
  先退回规划并向用户说明。
- 不运行 `pnpm run build`，因为任务不涉及依赖、构建配置或生产入口。
- 不处理 TagsView、表单草稿、开发脚本、贸易条款或其他用户 WIP。

## 回滚点

1. 服务端自动代码重试为独立提交，可单独 revert，不影响可选 code 契约。
2. Web 交互和费用面板修复为完整业务提交，回滚时整体 revert，避免只恢复一条创建入口。
3. 测试补强提交不改变产品行为，可独立保留；回滚不得覆盖用户未提交文件。
