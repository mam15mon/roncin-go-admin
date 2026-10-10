# 首尾冻结独立复核

## 范围与依据

已读取当前 PRD、设计、实施计划、check.jsonl 所列规范和研究证据，核对公共列设置、认证上下文、响应式 Pro/Editable 适配、订单费用专有偏好及实际调用点。实现代理并行补测试期间先只读复核，缺陷反馈主代理和对应文件负责人，避免覆盖未完成改动。浏览器验收与最终 check:web 由主代理统一执行。

## 已修复的问题

问题经独立复核及主代理真实浏览器反馈发现，由对应实施代理完成修复；本复核再次检查最终代码与定向回归，没有覆盖他人产品改动。

1. **ProComponents 内部 fixed 缓存**：官方 `genProColumnToColumn.getColumnConfig` 优先使用 `columnsMap`；无 persistenceKey/type 时列变更不会自动更新内部 map，开关与最终 props 正确仍可能没有实际固定效果。公共 Pro/Editable 包装应通过官方受控列状态保持当前列真相，不能重挂整个表格损失编辑或分页状态。
2. **合法 false 被另一侧非法值丢弃**：公共存储解析原先要求两侧同时合法，`{first:false,last:'bad'}` 会丢弃整个 freeze，重载时首列回到默认 true。读取结果改为 `ColumnSettingsPreference` 的逐侧部分配置，生效前 `resolve` 补默认，保留合法 false，并覆盖存取后的回归。
3. **禁用期间仍可操作已开的浮层**：原入口只禁用齿轮按钮，已打开后进入禁用状态仍能修改开关。最终同步关闭展示、阻止禁用时打开并保护变更/重置/高级入口回调；`disabledRef` 防止离场动画缓存旧回调，测试覆盖已打开→禁用→离场缓存按钮→解除禁用不重开。
4. **序号结构锚点被官方排序改变**：`columnSort` 会把 left 列移到非固定列之前；未固定序号位于业务列前时，冻结业务首列可能将其移到序号前。主代理与实现代理选用官方更薄方案：Responsive Pro/Editable 的输出列将 left/right 转换为 Antd 6 正式支持的 start/end，绕过仅识别 left/right 的 Pro 内部排序；输入顺序、辅助结构规则和普通 Table 行为保持。rc table 的 `flatColumns` 本身也将 left/right 规范为 start/end，实际展示语义一致。
5. **已打开浮层缩窄后超出视口**：主代理稳定浏览器触发 1440→390，bottomRight 边缘对齐浮层左右翻转均无法容纳，浮层右边缘到 578px。最终采用官方 `placement="bottom"` 单方向自动平移；保持浮层打开缩窄实测浮层 x=66、right=390、pageWidth=390，消除溢出，没有增加整页裁切。

## 已核对的数据流

- 首尾在权限、显隐、顺序之后按实际可见叶列计算；操作/option/显式结构标识跳过首列目标，末尾操作保留必显与位置；选择/展开官方 sentinel 保持对象身份。
- 开关显式 false、首尾重合左优先、搜索不改变目标、原始列与回调不变，均在公共实现路径处理。
- 认证上下文拆分为独立模块，AppProvider 与读取 Hook 使用同一 Context；产品订单费用唯一调用点 `fees.tsx` 已传实际用户与组织。
- 费用应收/应付共享一份偏好，存取、默认相等、权限剪裁、排序/显隐和最终冻结全部接入；option 独立归位末尾。实际费用列没有 colSpan/rowSpan 合并配置；外部自定义列模式禁用设置。
- 财务列宽与 onHeaderCell 在冻结映射时保留；Responsive 层负责临时解冻且不写偏好。
- 产品调用点没有显式 columnsState；统一原生 setting:false，受控空 map 不覆盖另一套产品列偏好。

## 验证

- **独立定向测试通过**：14 个实际文件、77 个用例全部通过。包含公共浮层/偏好/Hook/scope、真实 Pro/Editable 动态固定、响应式布局及订单费用专有链。临时 JSON 报告 `/tmp/column-freeze-independent-review-tests.json` 不提交；JSON 套件计数含 describe 层级，文件数按 testResults 统计。
- **重型套件符合规范**：本次与主代理完整门禁并行运行，费用真实组件套件 19.291 秒，其余单文件均低于 4 秒，未超过 20 秒；没有拆弱断言或静音警告。
- **Biome 通过**：25 个修改/新增 TypeScript 文件检查通过，没有执行自动修复。
- **TypeCheck 通过**：独立运行 `pnpm --dir web tsc`（`--noEmit`）退出码 0。
- **差异检查通过**：`git diff --check` 退出码 0。
- **实际路径断言**：同一 Pro 实例切换首尾固定后表头 class 变化、序号与官方选择/展开锚点不变、request 仍仅调用一次；窄宽切换恢复原固定；Editable 输入节点、焦点与未提交内容保持。
- **规范已同步**：公共读取/生效类型、真实认证 scope、首尾目标、受控列状态及 start/end 适配均记录在 component-guidelines 与能力导航。
- **浏览器与最终门禁**：主代理负责真实几何验收和最终 check:web，证据在同目录 `browser-notes.md`、`browser-freeze-verification.json`、`browser-freeze-fixture.json`；本复核没有重复全量门禁。

## 未修复边界

`deriveColumnFreezeDefaults` 从可见叶列自身 fixed 推导默认，尚未继承仅写在父分组上的 fixed。当前产品没有父分组固定的调用，显式叶列首尾应用和分组递归仍有效。此项未修是因为实际产品尚无该默认输入，本期不扩展分组默认策略；将来增加这种分组调用时，应同步处理继承和子列显式 false，并补默认解析回归。

## 结论

本期实际产品路径没有尚未修复的阻断缺陷，公共能力、费用适配、认证单一真相和结构锚点与批准方案一致。没有自行提交或更改业务数据。最终完整门禁和归档由主代理完成。

## 完整门禁后的定向修复

首轮 check:web 中 `sea-document-visibility.test.tsx` 首个用例失败，套件耗时 25.601 秒。失败位置是第 145 行：HOUSE 必填校验后切回 DIRECT 时 HBL 尚在，初始 DIRECT 的断言已通过。原文件单独运行 5/5 通过、测试耗时 10.18 秒，没有 localhost:3000 连接错误，因此未证实本次公共产品代码引入提单模式缺陷，也不能把全量日志的连接错误归因于这个套件。

按既有长尾规则与异步交互规范，只修改测试：

- 保留全部 5 个用例，将页签可见性 2 例与校验定位 3 例分为 `sea-document-visibility.test.tsx`、`sea-document-validation.test.tsx`。
- 提取 `sea-document-visibility.fixtures.tsx` 共享原服务 mock、真实模板设置与查询辅助，未替换成假的业务组件。
- 首个用例 HOUSE/DIRECT 点击分别用异步 act 等待交互收敛；同时断言 Form store 模式值、Radio checked 和 HBL 页签状态，补强测试对实际状态的验证。没有加长超时、删断言或静音错误。
- AST 对比证明 4 个未改用例正文完全相同；首个用例原断言全部保留，另增加 4 个 store/checked 断言。

拆后定向 2 文件、5 用例全部通过，套件分别 7.139 秒和 4.609 秒，测试日志无连接错误。3 文件 Biome、`pnpm --dir web tsc` 及 diff-check 通过。原始/拆后临时日志分别为 `/tmp/column-freeze-sea-visibility-reproduce.log`、`/tmp/column-freeze-sea-visibility-fixed.log`，不提交。没有修改提单业务产品代码或公共表格实现；最终全量高负载回归由主代理确认。
