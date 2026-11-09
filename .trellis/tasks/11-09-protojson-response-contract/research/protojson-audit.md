# Research: ProtoJSON 零值省略与前端响应映射边界

- Query: 排查主数据同类拒绝、其他布尔误判，确定公共适配与门禁的最小方案。
- Scope: internal（代码、Proto 契约及本机 Go 依赖源码）
- Date: 2026-11-09

## Findings

### 1. 已确认：当前 wire 契约省略非 optional 标量默认值

- `server/internal/server/jsoncodec.go:35`：唯一 Proto 响应编码入口只配置 `UseEnumNumbers: true`，没有设置 Emit 选项。
- `server/api/masterdata/v1/masterdata.proto:325`：通用 MasterDataItem 的 sort_order/enabled 是非 optional int32/bool（336–337 行）；省略分别表示 0/false。
- 同文件 Port（417–418）、Airport（488–489）、Airline（560、562–563）、ShippingLine（634–635）的 sort_order/enabled/cargo_only 同样是非 optional 标量。
- 请求中的 optional bool enabled（如 382、389、396、403 行）表示筛选存在性，不能因响应零值规则而统一将其补成 false；省略表示不筛选。
- `server/go.mod:27` 使用 google.golang.org/protobuf v1.36.11。本机依赖 `/home/vince/go/pkg/mod/google.golang.org/protobuf@v1.36.11/encoding/protojson/encode.go:70` 是 Emit 选项的直接实现/说明依据，无需猜测库行为。

### 2. 已确认：四个专用 mapper 仍会拒绝合法主数据

| 入口 | 错误校验 | 专用字段映射 |
| --- | --- | --- |
| `web/src/pages/master-data/components/PortsPanel.tsx:28` | 34–36 行将 enabled/sortOrder 省略判错 | unLocode→code、nameZh→name、transportModes→modes |
| `web/src/pages/master-data/components/AirportsPanel.tsx:24` | 30–32 行同上 | iataCode→code、名称、城市与国家字段 |
| `web/src/pages/master-data/components/AirlinesPanel.tsx:24` | 30–32 行同上 | iataCode→code、nameZh/nameEn→name、cargoOnly→isCargoOnly |
| `web/src/pages/master-data/components/ShippingLinesPanel.tsx:25` | 32–34 行同上 | scacCode→code、名称、集装箱前缀及跟踪地址 |

- 船代/国家复用的 `masterDataMapper.ts:8` 已在上一修复恢复 enabled=false、sortOrder=0；专用 mapper 只导入 PersistedMasterDataItem 类型（以上文件 14 行），没有复用实现，所以补丁没有覆盖它们。
- 共用字段：id、enabled、source、sortOrder、updatedAt；code/name/nameEn 在实体层做字段重命名后可共享公共基类转换。国家/船代要求 name 非空，而港口/机场允许无中文名称，公共函数不能误加“一律中文名必填”。
- 字符串 source 也为非 optional 标量，但当前业务是否允许空来源应核对服务端写入规则，不能机械地把所有缺失字符串补空；现阶段保留业务必填校验即可。
- `useMasterDataCrud.ts:99` 在 queryFn 中对全页 data 执行 mapItem；任一条抛错会拒绝整页（103、106 行）。创建/更新成功后也调用 mapItem（155、174 行），因此零值记录还可能出现写成功后前端报错。

### 3. 为什么已有公共 UI 组件仍失败

- 四个面板已经共同消费 MasterDataTemplate/useMasterDataCrud，并不是没有共用表格。
- API DTO → UI 行模型在 queryFn / mutationFn 内完成，早于数据交给模板。模板只知道 TItem extends BaseMasterDataItem，不知道 Proto 是否 optional，无法为未知字段安全补值。
- `web/src/components/ui/master-data-template/useMasterDataCrud.ts:28` 的 mapItem 是显式边界。应共用边界的转换代码，而不是向通用表格增加对船司/航司/Proto 的业务判断。
- 最小建议：在现有页面模块内 `masterDataMapper.ts` 形成公共持久化字段适配，四个专用 mapper 只校验/转换各自专有字段并组合该适配。没有真实跨页面模块消费者时不强提取到 features；以后有真实共享再经 features/master-data 公开入口提取。
- 不推荐只建 `protoBool(value) => value ?? false` / `protoNumber(value) => value ?? 0` 小函数：它们既不识别 optional，也不能防止各 mapper 在调用之前继续抛错，容易制造“名义复用”。

### 4. 全仓筛查：另外确认的零值误判（不是整批拒绝）

通过排除生成物和测试的 `=== undefined` / `=== false` 定向筛查，明确的“数据映射拒绝合法记录”目前集中于上述四个主数据 mapper。另有相同根因的 UI 分支问题：

- `SeaOrderReassignmentModal.tsx:182`、`:768` 对 matched===false 分支在实际 wire 上无法命中；`:261` 对 isValid===false 的校验错误展示同样可能失效。契约：`server/api/order/v1/order.proto:962` bool matched，`sea_order_change.proto:476` bool is_valid，均非 optional。恢复时必须先证明查询成功/数据已返回，不能把“尚未加载”也解释为 false。
- `SeaExportDetailFeatures.tsx:378` 将 canReassign===false 作为禁用条件；契约 `sea_order_change.proto:112` 为非 optional bool。是否存在更外层按钮门控使该分支实际不可达，需要进一步核对；故将“局部死分支”列确认，“用户能绕过前置门控”列未确认。
- `response.success === false` 分支大量存在；正常成功信封必为 true，业务错误通常走 HTTP 错误/统一请求处理。仅凭字符串扫描不能认定它们都应替换成 !== true（有无 success 字段的专用计数接口）。这类需要逐接口查验，不做全局替换。
- `expectedVersion/currentStatus === undefined` 等是版本/状态存在性防护；当前未确认其业务允许零，因此不能归类为零值 bug。

### 5. 最小可执行防复发机制

1. AGENTS.md 明确边界：Proto 为真相源；非 optional 标量零值省略是合法协议，optional 缺失保留存在性；不要在页面/模板/全局拦截器按字段名猜默认值。公共实体映射处理共用字段，专用 mapper 只处理专有字段。生成物仍只能由生成器更新。
2. 更新 `type-safety.md:104`，将现有“所有后端 false 一律 !== true”收窄为非 optional 响应字段、已成功加载上下文；补数值/空数组/optional/缺必填业务标识的矩阵。
3. 四类专用 DTO 和通用 MasterDataItem 均用“实际缺 key”的 fixture 验证：零排序、停用、空列表、省略 optional；保留显式非零/true、必要标识缺失拒绝的反向用例。
4. 有意义的服务端编解码契约测试放 `server/internal/server/jsoncodec_test.go`：用真实 Proto 消息验证默认值/optional 的准确 JSON 形态，可将一份真实编码 fixture 供前端消费，避免 mock 总是填 sortOrder=100/enabled=true。
5. 若需强制公共复用，可为主数据专用映射增加窄范围 AST 约束，验证它们消费公共适配且禁止直接重新声明 enabled/sortOrder 的存在性校验；扫描器自身要有“合法 optional 比较不误报”测试。不要全仓禁用 ===undefined/===false，也不要对任意名为 sortOrder 的本地表单变量报警。
6. 公共 mapper 单测 + 各实体 mapper/页面数据路径的用例比仅检索文本更能验证效果；实体专属必填字段、停用行启用载荷仍正确才算验收。

### 6. 服务端统一输出选项的取舍

| 方案 | 收益 | 代价/边界 |
| --- | --- | --- |
| 保留当前省略 wire，前端共用契约适配 | 改动集中、与现有客户端契约一致，零值规则显式 | 消费端需统一入口；其他已确认误判仍须按范围清理 |
| EmitDefaultValues=true | 所有非 optional 标量默认值及空列表/映射显式输出；保留 optional/message 的存在性 | 全站响应形态改变，需跨领域验收空列表/空字符串/零枚举及 fixture；仍不能替代实体映射和业务必填校验 |
| EmitUnpopulated=true | 也输出空值 | 比上项额外产生部分未填消息字段 null，影响嵌套可选对象判定，代价更大；不作为最低成本首选 |

- 若用户明确希望“服务端所有零值显式输出”，优先研究 EmitDefaultValues 而非 EmitUnpopulated；当前仓库库版本支持前者。修改 codec 是全局契约策略，必须在最终规划中明确范围与验收。

## 相关规范与现有测试

- `.trellis/spec/web/frontend/type-safety.md:104`：已记录布尔省略，但缺共享转换入口且表述未限定 optional。
- `.trellis/spec/web/frontend/component-guidelines.md`、`directory-structure.md`：模板复用与模块边界。
- `.trellis/spec/guides/cross-layer-thinking-guide.md`、`code-reuse-thinking-guide.md`：边界和真实复用。
- `web/src/pages/master-data/components/governance.test.tsx:33`：国家/港口 fixture 都显式 enabled=true、sortOrder=100，无法触发省略场景。
- `ShippingAgentsPanel.test.tsx:75`：最近已加入 29 条数据、首条零排序、末条停用省略的真实场景。
- `server/internal/server/jsoncodec_test.go:95`：现有版本 0 测试覆盖的是请求存在性，不等于覆盖响应零值 wire。

## Caveats / Not Found

- 此次是静态审计，没有浏览器或线上数据复现，也没有修改产品代码或运行测试。
- 没有声称全仓无其他缺陷；复杂动态类型、truthy 判断、各种空值比较仍需按具体契约查验。
- 单纯 AST 字符串扫描无法从 TypeScript optional `?` 判断 Proto presence；生成类型的 `?` 反映 JSON 可省略，并不等同 Proto optional。
- 其他订单分支是否纳入本次实施属于范围决策；推荐主任务先固定公共适配、四类 mapper 和契约验证，额外业务分支单独明确验收后再实施。
