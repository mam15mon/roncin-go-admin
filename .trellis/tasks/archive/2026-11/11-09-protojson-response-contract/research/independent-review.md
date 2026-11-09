# 独立审查记录

## 范围与结论

2026-11-09，以 `trellis-check` 独立角色审查本任务全部未提交变更，包括公共 mapper、四类专用主数据适配器、Go codec 契约测试、前端测试、复用门禁、根规范和任务文档。未修改实际 codec、Proto、生成客户端或订单业务代码。未发现剩余阻塞问题。

## 已修复发现

1. `web/scripts/check-master-data-mappers.mjs` 原实现把首参为指定 DTO 的所有函数视作 mapper，普通展示/查询 helper 也会被强制调用公共入口。现限定 `map*` 命名的 DTO 适配函数；内层函数按独立作用域处理，避免同名形参误报，也避免内层未执行函数的公共调用被误认为外层已经复用。
2. `web/scripts/check-master-data-mappers.test.mjs` 增加普通 DTO helper/内层同名形参不误报，以及内层未执行调用不能满足复用要求两项测试，门禁共 7 项通过。
3. `web/src/pages/master-data/components/governance.test.tsx` 现有国家和港口页面 fixture 仅使用显式 true/100。增加缺少 enabled/sortOrder 的实际 wire 形态页面用例，验证记录仍展示停用状态；保留权限入口验证。最终治理文件 6 项通过。

## 审查确认

- `mapPersistedMasterDataFields` 在共享模块统一 id/source 校验、enabled=false 和 sortOrder=0，四个专用 mapper 与通用 MasterDataItem mapper 均真实组合此入口。
- 港口和机场的可选中文名保持原有展示规则；船司中文名、各实体代码/英文名/国家等专用必填仍校验。航司 non-optional cargoOnly 省略还原 false，optional awbPrefix 未被补值。
- 国家继续经 `mapPersistedMasterDataItem`，船代既有 29 条省略零值回归保留。读取和写响应最终使用同一 mapper，不新增吞错空列表分支。
- Go 测试经实际注册 JSON codec 对五类消息验证省略零值、输出非零值；optional bool 筛选保留未设置/显式 false/true 的存在性，optional 响应空字符串区分未设置。
- 门禁以 AST 识别实际公共导入/调用，不把注释、字符串、请求 optional 字段和其他 DTO 类型当作缺陷。检查已接入 check:web。
- AGENTS.md、类型安全文档和能力导航指向实际公共入口；当前服务端编码策略保持不变。

## 未修复与范围边界

- research/protojson-audit.md 中订单改配 matched/isValid/canReassign 的相邻问题属于已批准延后范围，未混入本次实现。
- 复用门禁是对本模块 `map*` + 已登记 DTO 的窄约束，不承诺进行全仓类型推导或识别任意未类型化/别名映射；新增类型或改命名时须同步登记与规范。
- 机场既有 optional 字符串转空展示文本属于既有明确 UI 投影，本次保留；协议存在性与业务判断不得盲补的规范需要与此区别，已提醒主会话微调表述。

## 独立验证

- 修改产品、测试与门禁的 Biome 检查通过，无新增警告；新增治理用例最终版本再次检查通过。
- `pnpm --dir web tsc` 通过，审查最后修改后再次运行通过。
- `pnpm run check:master-data-mappers`：7 项测试及真实目录扫描通过。
- `pnpm --dir web exec vitest run src/pages/master-data/components/masterDataMapper.test.ts src/pages/master-data/components/masterDataEntityMappers.test.ts src/pages/master-data/components/ShippingAgentsPanel.test.tsx src/pages/master-data/components/governance.test.tsx`：4 个文件 40 项通过（新增治理用例前）。
- 新增治理用例最终版本单独运行 `pnpm --dir web exec vitest run src/pages/master-data/components/governance.test.tsx`：1 文件 6 项通过。
- `go -C server test ./internal/server -run TestProtoJSONCodec -count=1`：通过。
- `git diff --check` 通过。
- 主会话负责本任务完整 check:fast、提交、归档、合并、分支清理与推送；本审查未进行提交或 Git 发布动作。
