# 技术设计

## 1. 数据口径
顶部业务指标与底部列表明细汇总并列但不混用：顶部沿用现有有效/已确认/已开票业务金额规则，应用完整筛选；底栏 matchedSummary 完全使用列表分页前同一谓词，不额外加业务状态。对冲顶部改为完整列表谓词再 AND CONFIRMED，与其余页面保持筛选行为一致。

## 2. 服务端正式读取契约
在 server/api/finance/v1/settlement.proto 为六个列表响应新增同语义 matched_summary，建议复用 FinanceLedgerMatchedSummary / FinanceLedgerMatchedAmount 消息。
- matched_summary.count：全部匹配行数，必须等于 total，不能拿有效记录数代替。
- amounts：按领域方向、原币、可选本币分桶；每桶提供 direction、currency、amount、optional base_currency、optional base_amount、unconverted_count。方向值沿用实体既有领域码，由专用投影解释，不借用数值恰好一致的兄弟枚举。
- 金额为 decimal 字符串，数据库聚合/纯 Go decimal 合并，不用浮点数计算业务合计。不新增无用户场景的通用报表参数。
- 无折本币的发票仍进入原币总额并计入未折算数量；不能把 null 当零；部分未折算需显式提示，完整折算为零与未折算可区分。
- data 在分页前对已含组织范围的列表查询克隆聚合；各实体只写必要字段映射。biz 持有纯领域汇总类型，service 做 DTO 投影，Ent 不越层。使用 Data.client(ctx)，不引入新事务/锁或额外写入。
- 追加合法字段号，不手改生成物；make -C server api 后 pnpm run generate:web-client，检查生成差异。

## 3. 前端公共边界
FinanceLedgerTemplate 保留官方 request/actionRef/onLoad。扩展已存在 WeakMap，将返回数组绑定类型化 rows/total/业务summary/matchedSummary 快照；只有官方 onLoad 采纳后更新公共快照，并回调页面生成顶部指标。页面 request 只返回数据，删除完成即 setMetricStats/setSummary 的副作用，不另建请求序号令牌。

领域适配器将六类行与新汇总投影为稳定展示模型，公共底栏不导入页面/领域 Proto 或枚举；明确收款/付款/核销/抵销，只有费用可显示费用利润。原币与本币按各自币种分组，缺失值不猜 CNY。删除旧平铺 summary 依赖，不保留兼容分支。

查询失败保留已显示的上次成功快照并提示错误；尚无成功快照时不显示伪造成功零值。顶部、底栏与当前页导出消费同一采纳结果。选择中的跨页记录来自框架缓存的实际已选行；不假装它们已经做后台实时刷新。

## 4. 跨页选择与条件切换
- 已选键和记录复用 AntD preserveSelectedRowKeys 及其记录缓存，禁止只数当前页 rows。组合调用方回调而不覆盖公共选择更新。
- 有选择时底栏显示选中记录投影；零选择时显示最近采纳的 matchedSummary。
- 以已提交的业务筛选（关键字、条件、标签、日期、公司等）构成稳定 selection scope；翻页、页大小、排序不属于筛选范围。筛选或公司变更立即清空键与记录，不能等新请求成功才清空；未提交的表单输入不清空。
- 普通同条件刷新不新增选择持久化/重读服务，保持现有框架机制；新采纳页中已选记录对应的展示数据应与当前行一致，跨页未重载记录保留已选快照。已有业务写入的选择清理随现有成功路径处理，不扩大写入逻辑。

## 5. 路由标题
集中 routes.ts 增加最小展示标题元数据，默认取 name；菜单短名与页面标题不同的真实场景显式声明。解析器从集中配置构造规则，静态优先动态，redirect 也来自配置，删除 layout 手写标题目录与重定向副本。
SE/AE 新建和列表保留真实标题，未知 kind 保持中性回落。hideInMenu 不排除标题，layout:false 不新增业务页签。动态单号事件及 tab key 身份规则保持原状。

## 6. 改配
只在已成功取得预览的边界归一化 isValid 或以成功状态解释省略 false，展示 errors 或现有中文阻断兜底；确认仍要求明确有效及合法版本。matched 已归一化，不做无意义全局替换。资格真值在菜单与已打开弹窗间共享派生判断，未加载、失效与失败禁止继续操作，但保留已有写成功后的 synchronizing 保护和提示。

## 7. 风险与回退
金额精度、发票未折算、不同本币和筛选状态交叉为主要风险；跨页选择不能泄漏旧公司记录；生成结果需审阅；框架迟到隔离需确定性验证。本任务无数据库迁移，按交付包提交可独立回退。公共模板不扩为报表或异步状态平台。
