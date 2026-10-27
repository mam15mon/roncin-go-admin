# 船代公共主数据技术设计

## 目标与变更边界
行为缺口：船代字段已存在，但新建和详情搜索、服务端引用校验、名称投影均使用公司供应商，缺少独立的公共字典和维护入口。本期使其统一引用公共船代主数据，同时接入已核对的 29 条静态初始数据。

复用 `master_data_items`，不引入独立船代平台；不重构订单生命周期、代理经营档案或全站主数据 Hook。系统管理身份沿用现有公共资料维护边界，各公司共享读取。

## 数据模型与初始化
- `biz.MasterDataKindShippingAgent = "shipping_agent"`；Proto 追加 `MASTER_DATA_KIND_SHIPPING_AGENT = 11`，保留既有值和 reserved。
- 船代复用 MasterDataItem 的 UUID、code、name、name_en、source、sort_order、enabled、search_keywords 等字段，不设 organization_id，无公司覆盖。
- 中文名取源 value，英文名忠实保留 enValue；源英文仍为中文时不造译名。code 使用 `HGJ-<源整数ID>`，明确为内部导入标识；本系统 UUID 独立生成。source 标明海管家船代字典快照，排序沿用来源 order 并以 code 稳定兜底。
- 原始无凭据快照和简短来源说明归档 server/seeds/；正式 SQL 迁移按 `(kind,code)` 冲突跳过，保留管理员改名、排序与停用，重复初始化不新增、不覆盖。
- SQL 种子 search_keywords 必须经现有 `searchtext.Build(name,name_en)` 预计算，真实库测试逐行核对。
- 同步 Ent enum、表级稳定名 kind CHECK 与正式迁移；CHECK 集合匹配现行合法种类，包含新船代、不恢复已退役 charge_category。实际开发库缺该稳定名 CHECK，迁移针对已知有/无前态均显式重建；完整正式迁移冷启动必须通过。
- 不新增 version：该字典是低频单人维护主数据，沿用既有 CRUD 行为与唯一索引，避免引入独立并发协议。

## 接口与分层
1. 新增 `ListAllItems` / `GET /api/v1/master-data/items/all`，必填 kind、可选 enabled；无 page/pageSize，返回 MasterDataItem 集合、总数和现有响应包装。沿用 `system.master_data.item.read`。Biz 验证 kind，Data 真正全量查询，无循环分页。
2. 新建/修改/启停沿用通用 CreateItem / UpdateItem；`RequireGlobalMasterDataWrite` 校验系统管理身份和既有 item.create / item.update 权限，审计与 WithTx 同口径。
3. 动态候选复用 `SearchOptions(kind=shipping_agent,keyword,page,pageSize)`，已登录即可使用启用主数据，分页上限复用 200 公共常量。检索复用编码/中文/英文/现有拼音键，无别名字段则不新增别名模型。
4. 只在 service 执行 DTO 转换，biz 持有领域类型和验证，data 负责 Ent 查询。所有生成物通过生成器更新。

## 订单引用与事务
- `shipping_agent_id` UUID 字段和 JSON 名称保持，语义统一切换为 `master_data_items.kind=shipping_agent`。
- 从 Partner supplier/blacklist 校验列表移除船代；订舱代理和国外代理继续走原公司 Partner/角色/黑名单路径。
- 公共船代校验在创建/草稿更新既有事务内经 `Data.client(ctx)` 查询并 ForShare，按 ID+kind 检查存在；新建或变更需 enabled=true。保留同一旧 ID 时允许停用，但仍要求 kind 正确和记录存在。nil 仍代表清空可选引用。
- 名称聚合单独批量查询船代主数据，不加 enabled 过滤，使停用引用仍显示名称；不回退 Partner。
- 普通复制走创建校验，源船代停用则需重选；拆票继承原单船代属于保留既有引用，允许继承已停用的合法原引用，回归 UUID 和显示名不丢失。拆票后若主动更换船代，仍只允许启用项。
- 本机只读统计：订单 0、船代非空引用 0。正式迁移检查非空旧引用若未指向合法公共船代则显式报错并整批回滚，不猜测映射、不清空、不新增双读兼容。
- 本期保持现有逻辑引用和停用机制，不另建 FK/Ent edge；不存在物理删除入口。

## 前端入口与数据流
- 主数据页新增“船代”页签，登记 index.tsx 与 router/workspaceTabs.ts，沿用主数据路由与组织切换可见性。
- ShippingAgentsPanel 复用 MasterDataTemplate 客户端模式、现有字段映射和失败响应守卫；单条 React Query 获取全部含停用船代，本地搜索/筛选/分页/统计不重复请求。mutation 成功后失效列表，ModalForm 由 onFinish Promise 控制关闭。
- 写入口要求 `access.isSystemWorkspace` 且有对应权限；公司工作台只读并提示系统管理统一维护。
- orders/common 定义最小船代搜索 helper，new/detail 统一调用，订舱代理、国外代理查询不变。保留详情 seed 选项显示已有停用引用。
- 远程 keyword 搜索字段设置 `fieldProps.filterOption=false`，避免后端拼音命中后被前端纯文本过滤隐藏。保持原生 request 协议，无自建 data/loading 镜像或全局候选缓存。

## 验证与交付
定向验证覆盖组织写权限、全量读取超过 200、检索五维、种子29条/幂等/检索键、订单正确kind/停用保留/非法引用拒绝/显示及相邻代理路径。真实 PostgreSQL 使用隔离测试数据库与正式迁移；核对 PASS/FAIL/SKIP，不能把跳过计为通过。
契约和订单引用语义跨层，最终执行 check:fast；无需因普通UI固定执行 build。变更和必要生成物同组提交，后续更新公共主数据专项规范并归档任务。

## 风险与停止条件
- 若发现某环境旧供应商船代引用，迁移显式失败；只读汇总实际影响后向用户确认映射，不能重置数据库。
- 29 条仅是已观察字典，不代表全部行业船代。运行时不访问海管家，不保留其凭据。
- 如用户希望某具体公司管理员维护或每公司独立船代，应调整本方案后重新评审；本期推荐现有系统公共主数据边界。
