# Research: 船代公共主数据前端与海运订单引用

- Query: 核查新增公共船代字典的主数据入口、小数据全量模式、系统工作台权限、海运新建/详情候选链路、停用引用显示、缓存和定向验证。
- Scope: internal；仅前端定向研究，未实现产品代码、未修改任务状态、未执行 Git 操作。
- Date: 2026-10-27

## Findings

### 1. 公共字典的维护主体已有明确实现

- `web/src/access.ts:163`：`isSystemWorkspace` 只根据 `/auth/me.currentOrganization.kind === ORGANIZATION_KIND_SYSTEM` 判定；公司工作台是 COMPANY（`:169`），不按组织名称“总公司”放行。
- `web/src/access.ts:338`：通用主数据复用 `canReadMasterDataItems`、`canCreateMasterDataItems`、`canUpdateMasterDataItems`；权限键来自生成物及后端有效能力。扩展 kind 无须在前端发明另一套权限。
- `web/src/pages/master-data/components/CountriesPanel.tsx:34`：公共资料采用 `access.isSystemWorkspace && access.canCreateMasterDataItems/canUpdateMasterDataItems`，公司只读；`:229` 附近传入 notice “由系统管理员统一维护与共享”。船代适合沿用此门控。
- `web/src/pages/master-data/components/governance.test.tsx:104`、`:118`、`:134`：已有公司即使具备同名写权限仍隐藏写入口、系统有权限可写、系统无权限不可写三组范本。
- 币种是特殊的“公司启用”模式，`web/src/pages/master-data/components/CurrenciesPanel.tsx:85` 的启停规则不能照搬为船代组织私有开关。

结论：本期最薄实现是系统管理工作台统一维护公共船代、公司共享消费。用户所称“总公司”可在产品解释中澄清为系统管理维护入口，代码不应新增集团层级或给名为总公司的 COMPANY 提权。

### 2. 菜单和子页签必须一起登记

| 文件锚点 | 当前职责 | 船代影响 |
| --- | --- | --- |
| `web/src/components/layout/workspaceMenus.tsx:70` | 设置中心显示“主数据”入口，条件 `canReadMasterData` | 原入口可复用，无需另建顶层菜单 |
| `web/config/routes.ts:279` | `/master-data` 路由，access 为 `canReadMasterData`，组织切换保留，原始侧栏隐藏 | 原路由可复用 |
| `web/src/pages/master-data/index.tsx:26` | 目前 7 个 `MultiTabCenterTemplate` 子页签 | 新增船代 panel/`shipping-agents` 页签；同步副标题 |
| `web/src/router/workspaceTabs.ts:41` | 页签可见性与组织切换落点共用真相源 | 新增 `shipping-agents`，可见性复用 `canReadMasterDataItems` |
| `web/src/pages/master-data/index.test.tsx:68` | 断言 7 个 tab、标题与副标题，子 panel 均被 mock | 同步新增 panel mock、标签与副标题预期 |
| `web/src/router/organizationSwitchDestination.test.ts:162` | 主数据组织切换保留有权限页签 | 增加船代新页签有权保留、无权退到首个可见页签 |

`workspaceTabs.ts` 不同步会导致页面可见与组织切换行为不一致；禁止仅在页面数组里新增。

### 3. 29 条维护页使用现成模板客户端模式，数据层不能直接复制旧 CountriesPanel

- `web/src/components/ui/master-data-template/MasterDataTemplate.tsx:92`：存在 request、server、client 三种模式；不传 `request` 且不传 `query/onQueryChange` 即客户端模式。
- 同文件 `:133`：客户端搜索代码、中英文名称及其它字符串字段，筛选 enabled 和动态条件；`:177` 本地 slice 分页；`:184` 在本地计算统计。输入与翻页不会发起列表请求。
- `web/src/pages/master-data/components/CurrenciesPanel.tsx:31`、`:85`：已有“无分页全量接口 + items + onRefresh”消费示例。但它仍用手写 useState/useEffect 请求，属于存量，新增船代应遵循当前 React Query 规范。
- `web/src/components/ui/master-data-template/useMasterDataCrud.ts:97`、`:116`：该 hook 固定为服务端分页，列表一条 query、启停统计再发两次请求，不具备全量客户端模式。无需为本期把整个 hook 改造成多模式平台；船代 panel 可就近使用单条 `useQuery` 加必要的 `useMutation`。
- `web/src/pages/master-data/components/masterDataMapper.ts:8`：可复用 `mapPersistedMasterDataItem`（严格校验 id/code/name/enabled/source/sortOrder）及 `requireMasterDataResponse`；新增 alias/search 字段需按契约映射。
- `web/src/services/roncin/masterDataService.ts:125` 与 `web/src/services/roncin/typings.d.ts:4831`：目前 `ListItems` 是 page/pageSize/kind/keyword/enabled 分页接口；不能省略分页参数就当作全量，也不能循环翻页。需要后端提供明确无分页船代全量接口/契约，再生成客户端。
- create/update/启停 Promise 必须先 `isWriteResponse(response, true)` 验证实际结果，失败返回 false，成功失效当前 panel query；模板负责模态框生命周期。沿用模板已有字段配置、编码/中英文名称、启停操作即可。

### 4. 船代当前错走供应商的完整调用链

1. `web/src/pages/orders/new.tsx:160`：`searchShippingAgents(keyword)` 实际调用 `searchPartnersByRole(PARTNER_ROLES.SUPPLIER, keyword)`。
2. `web/src/pages/orders/detail.tsx:369`：同样调用供应商，并外包 `withSeededOptions`。
3. `web/src/pages/orders/common.ts:181`、`:187`：SUPPLIER 枚举传入 `searchPartnerOptions(keyword, { role, enabled: true })`。
4. `web/src/features/partners/partnerOptions.ts:17`：请求 `partnerServiceListPartners({ page:1,pageSize:50,keyword,role,enabled })`，把 partner legalName/code/id 映射为 SelectOption。
5. `web/src/pages/orders/templates/types.ts:33`：模板只依赖 `(keyword?) => Promise<SelectOption[]>`，无需为来源变化改动全部模板 props。
6. `web/src/pages/orders/templates/components/sea/SeaBasicInfoSection.tsx:604`：`ProFormSearchableSelect name="shippingAgentId"` 的原生 request 调用上述函数。
7. `web/src/pages/orders/templates/sea-template.tsx:79`：船代显示在“订舱与运输”分节。
8. `web/src/pages/orders/order-kinds/sea-export/form-adapter.ts:174`、`:300`、`:404`：创建提交、详情初始值、更新提交均直接携带 `shippingAgentId`；字段名和 DTO 可以保留，但服务端必须同步切换 ID 语义/名称聚合/校验。

建议搜索复用 `masterDataServiceSearchOptions`（`web/src/services/roncin/masterDataService.ts:239`）+ 新生成 `MasterDataKind.MASTER_DATA_KIND_SHIPPING_AGENT`，第一页 pageSize 50，传 keyword，服务端筛选启用。`web/src/pages/orders/abnormal-case-panel.tsx:337` 有完全相同的通用主数据搜索范例。

现有 MasterDataKind 枚举（`web/src/enums.generated.ts:242`）尚无船代；应改 proto 并重新生成，禁止手改。若搜索函数只由 new/detail 消费，可就近收敛在 orders/common；如维护页也需要真实共用的映射/请求，再按能力导航提取到 `features/master-data/shipping-agents/index.ts`，不从 orders 反向深引主数据页面。

相邻路径：订舱代理应继续 SUPPLIER（new `:156`、detail `:359`），国外代理继续 FOREIGN_AGENT（new `:158`、detail `:364`）；不能为修船代而整体替换 `searchPartnersByRole`。

### 5. 服务端拼音搜索结果会被当前前端默认过滤再次筛掉

- `web/src/components/ui/searchable-select/SearchableSelect.tsx:38`：默认 `fieldProps.filterOption = defaultSelectFilterOption`。
- `web/src/components/ui/searchable-select/utils.ts:7`：仅在 label/value/code/name/title/key/search/keywords 直接做字符串 contains，并不计算中文拼音。
- 当前船代字段没有覆盖 filterOption；若服务端通过“shanghaichuandai”命中纯中文 label，客户端还会把它过滤掉。
- 使用服务端 keyword 搜索时，船代字段应显式 `fieldProps={{ filterOption: false }}`，沿用异常字典选择器 `abnormal-case-panel.tsx:341` 的既有范式。必须实际断言拼音输入后选项可见，不能仅测试搜索 helper 返回值。

### 6. 停用既有引用显示与详情 seed

- `web/src/pages/orders/detail.tsx:289`：说明聚合返回当前引用展示名；`:302` 以 `shippingAgentId` + `shippingAgentName` 构造 currentOption；`:338` 形成 seed。
- `detail.tsx:321` 的 `withSeededOptions` 等搜索 Promise 完成后，将不在结果中的当前 seed 合并到首部；`:369` 专门包裹船代查询。切换来源时应保留 seed 合并，否则停用或首批以外记录会显示原始 ID。
- seed 能保住显示不等于授权新增停用引用。前端只保留当前已选项；后端应区分“原 ID 保持”和“切换到停用 ID”。聚合查询不得因为 enabled=false 丢弃名称。
- 新建无详情 seed，正常搜索只返回 enabled 船代；已有订单保留当前名称。原供应商 ID 若存在，需主会话/后端研究明确处置，不添加前端双读兼容。

### 7. 缓存与刷新影响

- 当前船代候选调用是原生 ProForm request，**未进入**订单字典缓存；最小修复继续直接使用 SearchOptions，不必为 29 条新增全局 Map。
- `web/src/features/orders/options/orderOptionsCache.ts:20`：`getMasterDataOptions(organizationId)` 缓存 `ListOptions` 返回的 Promise，按组织区分，无 TTL；`:154` 负责定向/全部清除。
- `web/src/pages/orders/common.ts:275`：`fetchOrderMasterData` 同时读取该缓存、港口/机场和币种；目前只从 masterOptions 派生货物类别、地区等，并未派生船代。
- `web/src/pages/orders/use-order-create-options.ts:43`、`:137`：create-options 的 React Query 缓存包裹该 Map，重试才显式清 Map。
- `web/src/pages/orders/use-order-detail-data.ts:66`、`:87`：详情 query 同样加载该 masterData bundle。
- `web/src/app/AppLayout.tsx:119` 和 `web/src/components/RightContent/AvatarDropdown.tsx:32`：组织切换、登出清订单 Map；`web/src/components/layout/OrganizationWorkspace.tsx:27`、`:44`：用户+工作台更换重建 QueryClient 并同步销毁旧查询。
- 因此建议 panel 独立全量 query、订单选择器直接 SearchOptions。若改为复用 ListOptions 全量候选，新增/改名/启停后除了 invalidate panel/query，还要清 Map 及对应 create/detail 数据，否则已缓存候选不会更新；这会明显扩大影响范围。

### 8. 最小验证范围（实施后运行，研究阶段未执行）

新增针对性用例：

- 船代 panel：系统管理+写权限能新增/编辑/启停；公司即使持权限仍只读；写接口 resolve undefined 不关闭模态、不显示成功；列表一次全量获取后输入/筛选/翻页不再次请求。
- 船代搜索：调用新 kind 的 SearchOptions，传 keyword/page/pageSize；不调用 partnerService；中文、英文、编码及拼音/首字母结果可见（字段关闭本地二次过滤）。
- new/detail 编排：都使用船代搜索；订舱代理仍供应商、国外代理仍外代；详情当前停用项 seed 被合并且 label 正常，不被候选缺失清空；提交 ID 与名称显示一致。
- 主数据页签与切换：船代有读权限可见，无权限不见；有权组织切换保留该 tab，无权移到可见 tab。

推荐复用/影响测试：

| 测试 | 影响 |
| --- | --- |
| `web/src/pages/master-data/index.test.tsx:68` | 新 panel mock、tab 数与副标题 |
| `web/src/pages/master-data/components/governance.test.tsx:104` | 可参照治理三态；专用船代测试独立文件避免重组件堆积 |
| `web/src/components/ui/master-data-template/MasterDataTemplate.test.tsx:90` | 已有客户端搜索；模板不改则无需增加镜像测试 |
| `web/src/pages/orders/common.test.ts:26` | 如搜索 helper 放 common，可追加接口参数和来源断言 |
| `web/src/pages/orders/new.test.tsx:66` | 当前 masterDataService mock 仅有 ListShippingLines，新增搜索需同步 mock，真实表单挂载即请求 |
| `web/src/pages/orders/new-source.test.tsx`、`new-create-idempotency.test.tsx` | 真实挂载场景注意新增服务 mock，不改创建生命周期 |
| `web/src/pages/orders/detail-draft-lifecycle.test.tsx:91` | 已有详情编排 fixture 范本；建议停用 seed 单独轻量测试，勿继续向长文件叠重组件 |
| `web/src/pages/orders/templates/components/sea/SeaBasicInfoSection.partner-mapping.test.tsx:65` | 相邻三种 partner 角色映射保持不变 |
| `web/src/pages/orders/order-kinds/sea-export/form-adapter.test.ts:489` 等 | 已覆盖 shippingAgentId 序列化，字段不变可跑现有用例 |
| `web/src/router/organizationSwitchDestination.test.ts:162` | 新船代 tab 组织切换 |
| `web/src/features/orders/options/orderOptionsCache.test.ts` | 仅当修改该缓存才需要扩大执行 |

定向命令（新增测试文件就位后调整文件名）：

```bash
pnpm --dir web exec vitest run src/pages/master-data/index.test.tsx src/pages/master-data/components/ShippingAgentsPanel.test.tsx src/router/organizationSwitchDestination.test.ts
pnpm --dir web exec vitest run src/pages/orders/common.test.ts src/pages/orders/new.test.tsx src/pages/orders/templates/components/sea/SeaBasicInfoSection.partner-mapping.test.tsx src/pages/orders/order-kinds/sea-export/form-adapter.test.ts
pnpm --dir web test:changed
pnpm --dir web exec biome check src/pages/master-data/components/ShippingAgentsPanel.tsx src/pages/master-data/index.tsx src/router/workspaceTabs.ts src/pages/orders/new.tsx src/pages/orders/detail.tsx src/pages/orders/templates/components/sea/SeaBasicInfoSection.tsx
pnpm --dir web tsc
pnpm run check:architecture
```

合同、生成物与订单来源语义跨层，最终集成适用 `pnpm run check:fast`；普通 UI 不必追加 build。本文件仅提供验证计划，未宣称检查已经通过。

### 9. 相关规范与外部参考

- 根目录 `AGENTS.md`：公共字典初始化、动态候选拼音检索、小数据全量+客户端分页、生成物禁止手改、风险匹配检查。
- `.trellis/spec/domain/glossary.md`、`architecture-map.md`：公司与系统工作台、主数据、订单缓存领域边界。
- `.trellis/spec/web/frontend/capability-navigation.md`：公开 UI 模板、主数据候选、订单缓存、跨模块提取约束。
- `.trellis/spec/web/frontend/state-management.md`：新增服务端状态用 React Query；原生 ProForm request 属于框架自管异步；失败响应守卫。
- `.trellis/spec/web/frontend/component-guidelines.md`、`organization-switch.md`、`quality-guidelines.md`：模板、组织切换落点和定向测试。
- 外部参考：本研究全部取自仓库现有代码与规范，未进行外部资料检索；运行依赖见 `web/package.json`（Ant Design 6、React Query 5、ProComponents 3）。

## Caveats / Not Found

- 前端无法自行决定旧供应商语义 shipping_agent_id 的迁移、安全引用校验与全量接口形状；须与后端方案一起明确。本研究没有读取真实业务数据库。
- 当前不存在船代 kind、船代 panel 或明确的船代全量维护接口；新增契约后应重新生成 enum/OpenAPI 客户端，再消费生成物。
- 用户“总公司”在现有模型中不等于 SYSTEM；建议沿用已存在系统管理治理，避免误把公司名称当权限来源。这是已有代码边界，产品如坚持公司维护才需要另行决策。
- 29 条是否允许英文名为空、是否对外显示编码、别名是否纳入可编辑字段，需根据种子实际字段与后端约束定稿，不能从当前通用表单推断成强制必填。
