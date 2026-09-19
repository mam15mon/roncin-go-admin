# 清理 antd6 弃用 API 与测试 act 警告噪音

## Goal

迁移 web 端 109 处 antd6 弃用 API 到 v6 新契约，并清理测试套件中的 React `act(...)` 警告与
vitest 配置弃用，使全栈门禁输出不再包含任何弃用警告与 act 噪音。

## 背景与现状（2026-09-19 实测）

- `pnpm run check:fast` 全绿，但 WEB 输出充满警告噪音：
  - 运行时 antd 弃用警告约 1000 行：Drawer `width`（407 次）、Modal `destroyOnClose`（351）、
    Alert `message`（77）、Space `direction`（63）、Modal `maskClosable`（52）、
    Input/InputNumber `addonAfter`（53）、Select `onDropdownVisibleChange`（11）、
    Spin `tip`（3）、`List` 组件弃用（3）、Timeline `items.children`（1）。
  - React `act(...)` 警告 304 次，分布在约 48 个测试文件。
- `antd lint ./src --only deprecated` 精确扫出源码侧 109 处、47 个文件（清单见 design.md）。
- `web/vitest.config.ts:40` 使用已弃用的 `cache.dir` 字段。

## Requirements

1. 源码弃用 API 清零：按 antd 6.6.0 官方契约迁移全部 109 处用法，包括 lint 未覆盖但
   运行时会警告的 `maskClosable`（4 文件）与 Timeline `items.children`。
2. 测试噪音清零：约 48 个测试文件的 304 处 act 警告全部消除；修复方式必须是让测试
   正确等待异步收敛，禁止通过关闭 act 环境、静默 console 或删除断言来"消音"。
3. `vitest.config.ts` 弃用字段 `cache.dir` 改为新写法（内存虚拟盘提速行为保持不变）。
4. 不改变业务行为：迁移是等价替换；`List → Listy` 与 `addonAfter → Space.Compact`
   两类结构性迁移必须保持现有视觉与交互语义（行内容 JSX 原样复用）。
5. 分阶段提交：阶段一（弃用 API 迁移）、阶段二（act 噪音治理）各自独立成 commit，
   任一阶段可独立回滚。

## Acceptance Criteria

- [x] `cd web && antd lint ./src --only deprecated --format json` 输出 0 issues。
- [x] 全量前端测试运行后，stderr 中 `deprecated` 与 `not wrapped in act` 计数均为 0。
- [x] 测试结果不劣化：全部用例通过（基线 853 passed / 12 skipped），无用例被删除或跳过。
- [x] `pnpm run check:fast` 整体通过。
- [x] `web/vitest.config.ts` 无弃用字段，测试缓存仍写入 `/dev/shm` 内存盘。
- [x] 两阶段提交信息符合 Conventional Commits，生成物与源码同提交（本次无契约生成物）。

## 验收结果（2026-09-19）

- 提交：`b119bfe2`（弃用 API 迁移，67 文件）、`966c8c88`（act 治理，21 文件）、
  `6d48b5e9`（残余警告清理，6 文件）。
- 追加清理了 4 类非弃用残余警告（静态 `message`、Descriptions span、initialValues
  覆盖、测试替身 FormContext/缺 key），全量输出上述警告归零。
- 遗留（范围外）：`useMasterDataCrud.test.tsx` 存量「Maximum update depth exceeded」
  （改动前已存在），建议后续单独排查共享 Hook effect 依赖。

## Notes

- 项目处于上线前阶段，无历史兼容负担（AGENTS.md），一次性迁移到位，不留兼容分支。
- 本次不涉及 `.proto`、权限码、OpenAPI 生成物变更。
