# 统一 ProtoJSON 响应映射与主数据契约约束

## 目标与用户价值
消除合法 ProtoJSON 零值响应被前端误判为损坏的重复缺陷。统一公共映射入口、架构约束及自动验证，使合法主数据可查看、编辑、启停，并避免页面继续复制有缺陷的字段校验。

## 已确认事实
- 用户实测船代接口 HTTP 200、29 条数据，首条未输出 sortOrder；页面加载失败。局部修复已提交 f61ca1e8。
- server/internal/server/jsoncodec.go:35-36 只开启 UseEnumNumbers；server/api/masterdata/v1/masterdata.proto:336-337 的非 optional sort_order/enabled 在 wire 省略时分别表示 0/false。
- PortsPanel.tsx:28、AirportsPanel.tsx:24、AirlinesPanel.tsx:24、ShippingLinesPanel.tsx:25（均在 web/src/pages/master-data/components/）仍将这两个字段省略当作响应缺损。通用 mapper 只被国家/船代复用，局部修复未传播。
- useMasterDataCrud.ts:99 在调用 UI 模板之前 mapItem，列表和写成功后的转换都可能失败。MasterDataTemplate 本身已共用，不应承担 Proto 字段语义。
- .trellis/spec/web/frontend/type-safety.md:104 已有布尔零值说明，但未限定 optional/加载状态、没有数值规则及统一映射入口。governance.test.tsx:33 等 fixture 显式填 true/100，未覆盖 wire 省略形态。
- 完整确认锚点、其他领域筛查及方案比较见 research/protojson-audit.md。

## 需求与验收
| ID | 需求 | 可观察验收 |
| --- | --- | --- |
| R1 | 主数据模块内收敛公共身份/来源/状态/排序/更新时间适配，国家/船代/港口/机场/航司/船司共用 | 零排序和停用行不会整批加载失败，显式非零和启用值保留，29 条船代可翻页和检索 |
| R2 | 专用 mapper 只负责实体字段及业务校验，区分 Proto presence 和业务必填 | 港口/机场中文名仍可缺省；缺少必填身份/代码等仍报错；optional 未设置不被改成 false/0 |
| R3 | AGENTS.md、类型安全规范及能力导航约定统一入口和复用方式 | 文档能准确指向映射入口；禁止模板和全局拦截器按字段名盲补默认值；保留生成物规则 |
| R4 | 使用真实 codec 与 wire 省略形态做契约验证，增加窄范围复用门禁 | Go 编码测试、前端缺 key/显式零值/非零/缺必填反向用例通过；门禁拒绝逐页重复公共逻辑，合法 optional 判断不误报 |
| R5 | 保护既有权限、编辑/启停载荷与模块边界 | 相邻国家及专用主数据回归通过，前后端风险匹配门禁通过，中文 Conventional Commit 提交 |

## 技术决策与范围
- 保留服务端现有 ProtoJSON 输出，领域共享适配器还原有明确契约的零值。UI 模板继续只消费稳定行模型。
- 复用范围仍在主数据模块，扩展已有 masterDataMapper.ts，不无依据创建跨领域框架或微型 bool/number 包装函数。
- 先实现后按风险补充验证，遵循禁止 TDD 的仓库要求。
- 本任务不修改 Proto、数据库、全站编码策略或生成客户端，不增加历史兼容、不重建 UI 模板。

## 风险与延后项
- 共享改动可能影响国家及实体专有字段，回归必须保护现有必填差异。
- 审计另确认订单改配 matched/isValid 的 false 比较存在同根因死分支；canReassign 局部判断也需结合外层门控确认影响。这些订单业务修复本期先记录证据，后续单独明确范围与验收。
- 全仓 success===false 不能机械替换，计数型接口/缺失响应与业务失败必须按具体契约判定。

## 规划状态
PRD 已按目标、事实、验收、边界和风险收敛；design.md、implement.md 与两份真实 JSONL 上下文齐备。当前保持 planning，等待用户对最终方案的后续明确批准后再启动实施。
