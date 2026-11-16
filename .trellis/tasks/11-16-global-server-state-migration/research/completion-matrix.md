# 原始清单完成对照

2026-11-16。按原始盘点逐项回查，原始记录不回写为完成；这里记录最终处置。查询缓存属于认证工作区独立 QueryClient，纯取数助手不持有认证结果缓存。

## 31 个查询消费文件

| 领域 | 入口（相对 web/src） | 最终处置 |
| --- | --- | --- |
| 公共 | `components/ui/searchable-select/RemoteSearchSelect.tsx` | 资源身份、requestKey 与防抖关键词驱动 Query；signal 透传，已选名称由缓存渲染保活。 |
| 公共 | `pages/orders/components/PartnerQuickAddSelect.tsx` | 关键词 Query 替代 useLatestAsync；合并已选和新建候选；快捷创建保留写入保护。 |
| 订单 | `pages/orders/list-resources.ts` | 主聚合使用 Query，原生搜索通过当前客户端 fetchQuery；组织、业务类型和运输方式参与身份。 |
| 订单 | `pages/orders/use-fee-exchange-preview.ts` | 按订单、方向、币种和日期等参数查询参考汇率；手工覆盖独立保留，查询失败不伪造结果。 |
| 订单 | `pages/orders/components/fees/OrderFeeTableTabs.tsx` | 行内汇率按完整身份 useQueries 去重；保存后清旧基线，保留官方表格 request 和人工行内容。 |
| 订单 | `pages/orders/components/detail/SameBatchOrdersSection.tsx` | 订单/品类 Query，成功才上报计数；换单和失败不沿用旧订单数据。 |
| 订单 | `pages/orders/components/detail/UnlockRequestHistoryDrawer.tsx` | 历史 Query 与原 5 秒前台活动轮询；隐藏/关闭停止，恢复立即检查，open 参与取消身份。 |
| 订单 | `pages/orders/components/drawers/SeaOrderChangeHistoryDrawer.tsx` | 预览、分页和详情分别 Query；关闭取消，重开隔离，首次失败显示错误，同单打开分页可占位。 |
| 订单 | `pages/orders/order-kinds/sea-export/SeaExportDetailFeatures.tsx` | 动作资格 Query 按订单及采纳版本隔离；失败禁用，手工刷新和本人同步后失效均保留。 |
| 订单 | `pages/orders/templates/components/sea/SeaDocumentSection.tsx` | 缺省单证补读和关联放货记录 Query；首载采纳正文基线，后台重查不覆盖；模式预览 mutation。 |
| 订单 | `pages/orders/templates/components/sea/SeaExternalConfirmationFields.tsx` | 附件候选 Query；失败就地反馈，取消信号传输，请求层和全局不重复提示。 |
| 订单 | `pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx` | 版本/事件 useInfiniteQuery；按原始分页判断下一页，重开第一页；修订/作废预览 mutation 绑定单证及版本。 |
| 订单 | `pages/orders/release-pod-panel.tsx` | 关联提单 Query 跟随当前订单/品类，放货记录表格继续官方 request。 |
| 财务 | `pages/finance/bills/components/BillEditModal.tsx` | 信用/未核销余额/账户 Query，共享币种 Query；接入已有账单变动 Hook，显式刷新 mutation 保护输入。 |
| 财务 | `features/finance/bill-creation/BillGroupCard.tsx` | 共享 currencyOptionsQueryOptions，删除每组重复货币 state。 |
| 财务 | `pages/finance/commissions/components/CommissionApplicationDetailDrawer.tsx` | 申请 ID 驱动详情 Query；去除 refreshToken，失败就地重试并校验明细信封。 |
| 财务 | `pages/finance/commissions/my-supplement-source.tsx` | 来源 ID Query，明确缺失/失败状态，移除手工 requestSequenceRef。 |
| 财务 | `pages/finance/commissions/components/CommissionRulesDrawer.tsx` | 编辑与筛选候选各自 Query，关键词/组织/打开身份隔离；缓存仅保活当前已选人员。 |
| 财务 | `pages/finance/bills/index.tsx` | 账单 ID 详情 Query；显式编辑刷新 fetchQuery，取消前影响读取保留命令校验；保存后失效详情。 |
| 财务 | `pages/finance/commissions/index.tsx` | 提成 ID 详情 Query；批准/驳回等写后失效，删除手工详情 loading/refresh token。 |
| 财务 | `pages/finance/invoices/index.tsx` | 发票 ID 详情 Query，当前公司和选中 ID 门禁；关闭清详情身份。 |
| 财务 | `pages/finance/fee-settings/components/CustomSettingsPanel.tsx` | 两类策略独立 Query，当前公司门禁；真实写成功后按策略/关联前缀失效。 |
| 客商 | `pages/partners/components/AuditLogSection.tsx` | 客商 ID、页码、页大小 Query；保留原有分页交互与就地失败显示。 |
| 客商 | `pages/partners/components/ContractCardList.tsx` | 客商合同 Query；写后失效，后台失败保留已有合同与编辑内容。 |
| 客商 | `pages/partners/components/ShippingPresetSection.tsx` | 客商预设 Query；写后失效，编辑生命周期独立。 |
| 企业资源 | `pages/enterprise-resources/index.tsx` | 客商/负责人关键词 Query 和 capability Query；选择/表单草稿独立，表格继续官方 request。 |
| 企业资源 | `pages/enterprise-resources/useRegionOptions.ts` | 层级与父编码构成 Query key；父级缺失不请求，旧城市/地区候选不串入新父级。 |
| 管理 | `pages/admin/organizations.tsx` | 整树 Query 与选择节点解耦；选择节点不重取，真实写成功才失效。 |
| 管理 | `pages/admin/components/NumberRulesPanel.tsx` | 编号规则 Query；保存/启停后失效，不再手工 fetchRules。 |
| 主数据 | `pages/master-data/components/CurrenciesPanel.tsx` | 全量 Query 和本地分页；启停同时失效直接消费者和 meta 标记聚合，写失败返回 false。 |
| 主数据 | `pages/master-data/components/CitiesPanel.tsx` | 全量 Query 和本地分页；首载错误显式展示，非 optional enabled 省略映射 false。 |

## 两套模块缓存

| 原入口 | 最终处置 |
| --- | --- |
| `features/orders/options/orderOptionsCache.ts` | 删除字典、港口、机场、人员四个结果 Map 与 clear 调用；公开薄取数签名保留并支持 RequestOptions。 |
| `features/master-data/currencies/currencies.ts` | 删除 Promise 和 forceRefresh 双重缓存；统一 currencyQueryOptions/key/select，原生 ProForm 零参助手与信号取数助手分开。 |

## 七个命令补充入口

| 入口 | 最终处置 |
| --- | --- |
| CommissionCreateModal | 计算预览 mutation，显式计算使用当前客户端 fetchQuery，载荷签名/输入修订和幂等意图保留。 |
| SeaTransportExecutionUpdateModal | 航程预览 mutation；旧编辑修订结果失效，正式提交仍使用预览对应基线。 |
| SeaBillContentFormFields | 人工带入使用当前客户端 fetchQuery；按客商/目标输入修订保护迟到结果。 |
| air-master | onFinish 影响预览 mutation，保留人工确认、共享主单版本和写入保护。 |
| AirTransportMaintenance | 同类影响预览 mutation，独立保留打开基线和正文。 |
| ExchangeRateSyncModal | 抓取 mutation，人工草稿保留；发布在途冻结输入及关闭，未知结果沿用原意图。 |
| ExchangeRateImportModal | 预检 mutation，业务 token/批次不作自动重查；提交在途冻结关闭与文件输入，失败保留原批次。 |

## 相关修复与豁免

- useMasterDataCrud、公共候选、货币和行政区划在 unwrap 前统一验证信封；支持成功省略 repeated，拒绝 undefined、success:false、非法 data 及真正业务必填缺失。
- BillEditModal 已覆盖前台 15 秒/回焦、隐藏/关闭停止、外部变化持续提醒、输入保留、刷新取消/失败/成功及本人保存两个窗口。浏览器使用合成 API 响应验证代表性交互，未连接真实后端写入。
- FeeItemsPanel、ExchangeRatesPanel、VerificationWorkbench、FeeTemplatesPanel、客商详情和订单聚合同步共享货币 key/meta，币种失效不再命中旧模块 Promise。
- 官方 ProTable/ProForm request、认证 AppProvider 初始化、本地输入/dirty/保存基线/写入代际及幂等键保留。useLatestAsync 无业务消费者，工具定义保留；其余 Map 为本地派生索引、选择或提交意图，不是认证结果缓存。
- 交叉审阅与定向结果见 review-core.md、review-finance.md、review-orders.md；最终门禁与提交状态见 integration-progress.md。
