# 质量规范

## 包管理器

- 统一使用 `pnpm`；不得新增 `npm`、`npx`、Yarn 入口（含文档与脚本）。
- 历史文档中的其他命令以根目录 `AGENTS.md` 和 `package.json` 为准。

## 分阶段校验规则

### 1. 开发过程中

只验证当前改动，不重复运行完整前端门禁：

```bash
# 单文件/少量文件定向测试（必须相对 web/ 目录）
pnpm --dir web exec vitest run <test-file> [more-test-files...]

# 智能增量测试：仅测试受 Git 当前改动影响的用例
pnpm --dir web test:changed

# 修改文件代码规范检查
pnpm --dir web exec biome check <changed-file> [more-changed-files...]
```

当前 `web` 的 `test` 脚本固定执行 `vitest run`。禁止使用下面的写法作为定向测试，
它会意外运行全量套件：

```bash
pnpm --dir web test -- <test-file>
```

定向测试路径必须相对 `web/` 目录（如 `src/pages/...`）。写成 `web/src/...`
前缀时，vitest 过滤器会静默匹配 0 个测试且退出码为 0，造成「全绿」假通过。
运行后必须确认输出中的测试文件数大于 0，再采信结果：

```bash
# 错误：过滤器不匹配，0 个测试静默通过
pnpm --dir web exec vitest run web/src/pages/foo/bar.test.tsx
# 正确
pnpm --dir web exec vitest run src/pages/foo/bar.test.tsx
```

### 测试并发与长尾效应防护（LPT 调度原则）

- **测试文件长尾约束**：Vitest 采用文件级并发（`fileParallelism: true`）。严禁在单个测试文件中堆砌超过 10 个重型业务组件/深层级树渲染用例（如批量建账、多分节大表单）。
- **拆分规约**：单测试文件耗时超过 20 秒时，必须按功能拆分为多文件（如常规流程为 `*.test.tsx`，超大批量或压测为 `*.scale.test.tsx`），以便让多核 CPU 调度器能分发到不同核心并发跑，杜绝「一核有难、几十核围观」。
- **动画禁用**：`setupTests.ts` 已全局注入零动画样式，彻底消除 300ms 帧延迟，测试用例中禁止添加无意义的 `sleep` 等待动画。
- **极速全量门禁**：全量验收推荐优先执行 `pnpm run check:fast`，自动并行执行前后端门禁，并挂载系统 `/dev/shm` tmpfs 内存虚拟盘消除磁盘 I/O 瓶颈。

### 2. 普通本地提交

- 运行受影响的定向测试与 `git diff --check`。
- 改动公共类型、Hook/组件接口、路由或难以由定向测试覆盖的类型链时，增加
  `pnpm --dir web tsc`。该命令走 package.json 脚本（含 `--noEmit`）；禁止
  直接执行 `pnpm --dir web exec tsc`，缺少 `--noEmit` 会向 `web/src` 输出
  数百个编译 `.js` 产物污染源码树。误执行后按产物清单精确删除并复核
  `git status`，不得混入提交。
- 提交动作本身不触发全量 `pnpm --dir web test`、`pnpm run check:web` 或构建。

### 3. 最终验收

整个任务或批准阶段完成、准备最终验收、Trellis 归档或推送时，才执行一次风险
匹配的完整前端门禁：

```bash
pnpm --dir web lint
pnpm --dir web test
pnpm --dir web tsc
pnpm --dir web biome:lint
```

`pnpm run check:web` 用于完整前端验收，`pnpm run check` 用于前后端全量验收。
`pnpm run build` 只在构建配置、依赖、生产入口等相关变更、发布前验收或用户明确
要求时运行，不是普通 UI 修改的固定门禁。

契约、权限、生成物、跨领域公共组件或大范围状态管理等高风险变更，可以在最终
验收前增加完整检查，但必须说明风险依据。若只是局部页面样式、文案或单组件行为，
最终验证也应避免无依据地扩大到后端全量门禁。

## antd 6 测试交互惯例（vitest + @testing-library）

项目已升级 antd 6（`@rc-component/picker` 1.x）。以下 antd 5 时代的写法在
antd 6 下会静默失效，测试表现为「元素找不到」或「回调未触发」：

- **DatePicker / MonthPicker 弹层只在 `click` 时打开**。rc-picker 1.x 的
  `SinglePicker` 只绑定 `onClick: onSelectorClick`，`fireEvent.mouseDown`
  不再打开弹层：

  ```tsx
  // 错误（antd 5 惯例）：弹层不打开，后续 cell 查询超时
  fireEvent.mouseDown(monthInput);
  // 正确
  fireEvent.click(monthInput);
  ```

- **`htmlType="submit"` 按钮必须点击按钮元素本身**。原生表单提交的激活行为
  只属于按钮节点；`getByText` 取到的是内层 `<span>`，点击它只触发 React
  冒泡事件，不会触发 `onFinish`：

  ```tsx
  // 错误：拿到内层文本 span，onFinish 不触发
  fireEvent.click(screen.getByText('查询'));
  // 正确（仓库既有惯例，见 OrderListTemplate.test.tsx）
  fireEvent.click(screen.getByRole('button', { name: /查询/ }));
  ```

  普通回调按钮（如 `htmlType="button"` + `onClick`）不受影响，点击内层
  文本经 React 事件冒泡仍可触发。

- **`modal.confirm` 已移除 `confirmLoading`**（`ModalFuncProps` 不再包含该
  属性，tsc 直接报错）。加载态改用 `okButtonProps`：

  ```tsx
  // 错误：TS2353 'confirmLoading' does not exist in type 'ModalFuncProps'
  modal.confirm({ okText: '重新提交', confirmLoading: resubmitting, ... });
  // 正确
  modal.confirm({ okText: '重新提交', okButtonProps: { loading: resubmitting }, ... });
  ```

  另注：`modal.confirm` 的配置对象是调用时一次性快照，`loading: someState`
  只反映调用瞬间值；`onOk` 返回 Promise 时 antd 会自动为确认按钮加转圈，
  优先依赖该内置行为而非手工状态镜像。

- **测试输出零噪音口径**（2026-09 起）：全量 vitest 的 stderr 中
  `not wrapped in act`、`is deprecated` 及 antd 运行时警告必须保持 0。
  新增测试如触发异步更新，必须在用例内等待收敛后再结束。

- **act 警告治理模式**（勿用 console 静音、`IS_REACT_ACT_ENVIRONMENT`
  开关或删断言消音）：
  - 挂载期请求/effect 在同步用例结束后落地 → 用例改 async，结尾
    `await act(async () => {})` 冲刷微任务（见 `new.test.tsx` 的
    `flushMountRequests` 惯例）。
  - 测试直接调用异步方法（`formRef.validateFields()`、`onFinish`、
    `onOk`、竞态用例手动 `resolve()`）→ 整体包进
    `await act(async () => { ... })`。
  - 防抖窗口、真实定时器等待 → 包进 act 内 sleep（见
    `BillCreationWorkbench.test.tsx` 的 `sleepInAct`）。
  - antd 静态 `message` 的独立根在微任务后渲染且内部 act 包装为
    no-op → 把触发点击与微任务冲刷包进同一 act 作用域
    （见 `ContainerCargoDelete.test.tsx`）。
  - rc-motion 弹层离开动画在 happy-dom 永不结束（无 transitionend）→
    等进入 leave-active 后在 act 内 `fireEvent.transitionEnd`
    （rc-motion 官方测试同款，见 `TagsView.test.tsx` 的
    `settleDropdownPopup`）。
  - 组件使用 antd 静态 `message`/`modal` 会产生「can not consume
    context」警告，应改 `App.useApp()`（生产链路由 Umi antd 插件
    runtime 统一 `<App>` 包裹，测试渲染需自行补 `<App>`）。

## 禁令

- 页面自行拼接后端主机地址（必须走统一请求配置 / 生成客户端）。
- 站内路由跳转使用 `window.location.href/assign`（必须走 umi `history` /
  `useNavigate`；登录后整页会话重建、iframe 跨 frame 操作等特殊场景除外，
  需注释说明）。
- 手改任何生成文件（见 type-safety.md 清单）。
- 硬编码第二套权限规则或复制后端权限清单。
- 引入无关的大型聚合组件；无关格式化混入功能提交。
- 不为通过检查而关闭 lint 规则、跳过类型错误或提交临时产物。
