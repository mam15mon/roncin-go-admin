# 取证与决策依据

- 上一任务：`.trellis/tasks/archive/2026-09/09-20-ai-friendly-architecture/implement.md` 明确延期订单候选缓存、utils/options 领域化和重复语义发现。本次不包含所有页面内部业务逻辑外提。
- `web/src/utils/options.ts:1` 聚合币种、船公司、往来单位请求；`SelectOption` 混合通用字段与 isCasual/creditExceeded；`disableCreditExceededOptions` 是信用控制展示逻辑。
- `web/src/utils/options.ts` 的 `getCurrencies(forceRefresh)` 在失败回调中无条件清空 currenciesRequest；旧请求失败可能清除刷新建立的新缓存。已有 options.test.ts 只覆盖普通失败重试，需补迟到失败场景。
- `web/src/utils/order-options-cache.ts:1` 实现字典、首批港口、首批机场、订单人员的缓存；人员 key 含组织与业务类型。其失败回调以 Promise 身份对比保护新缓存。
- `web/src/utils/order-options-cache.test.ts:95` 已覆盖旧失败迟到，其他用例覆盖并发去重、组织隔离、按组织与全量清理；需将验证覆盖四类缓存，不仅单一字典。
- `web/src/components/OrganizationSwitcher.tsx:39`、`web/src/components/RightContent/AvatarDropdown.tsx:30` 清理订单缓存，是必须同步迁移的真实跨模块调用方。
- `web/src/pages/master-data/components/CurrenciesPanel.tsx:73` 启停币种后调用 getCurrencies(true)；该刷新入口须保留。
- 财务建账、组织编辑、往来单位详情、财务核销/收付、订单单证等调用 utils/options。迁移包含 vi.mock 和动态 import 类型路径。
- 现有 `features/partners/index.ts` 与 `features/finance/credit-control/index.ts` 可扩展，避免重复领域入口。
- 架构规则禁止 utils 导入 features，因此旧 utils 文件不能以转导出方式保留。基础候选类型应脱离领域字段。
- `web/scripts/check-architecture.mjs` 已使用 @babel/parser；可复用现有直接依赖做前端函数级解析，不必新增扫描框架。
- 仓库未发现现成 duplicate/jscpd/clone/semgrep 实现。Go protobuf 和 Ent 生成文件具有 Code generated 标记；Ent schema 为手写输入，不能把整个 ent 目录一律排除。

## 扫描能力边界
自动化可可靠报告同语言函数体的相同/规范化结构，不能证明任意算法业务等价，也不能凭变量同名判定重复。明确覆盖函数与方法（前端含箭头函数），不承诺跨语言语义推理。扫描报告包含覆盖范围、排除项与解析错误；错误不得当作“无重复”。
