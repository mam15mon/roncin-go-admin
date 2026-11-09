# 独立审查记录

## 第一阶段：导航与海运改配（2026-11-09）

### 范围
集中路由 pageTitle/redirect/layout、routeUtils、TagsView、改配预览归一化与动作资格门控，以及相关回归。财务前后端仍在实施，本阶段不修改其文件。

### 结论
未发现剩余阻塞问题，本阶段未修改产品代码。已独立追踪真实调用路径：
- 标题与重定向从 routes.ts 构建，隐藏菜单仍能获得标题；具体航空共享主单路由优先于泛型详情；SE/AE 新建解析与未知 kind 中性回落保留。
- 稳定页签归组与动态单号事件保持；/settings 根据配置解析至费用设置；layout:false 继承对子路径生效。
- 改配仅在成功预览返回时归一化 isValid，不将加载误当无效；matched 已有归一化未机械替换。
- 菜单与已打开弹窗共用明确允许判断；最终确认守卫再次检查 disabled；写成功后的 synchronizing 阶段不因临时资格变化提前关闭。

### 独立验证
- 11 个导航/改配文件 Biome check 通过。
- 7 文件 Vitest，68 项测试通过（14.57 秒总耗时，单文件未超过20秒）。
- pnpm --dir web tsc 通过（当前树全量类型检查，财务仍可能后续更新）。
- git diff --check 通过。

## 第二阶段：财务前后端及规范全范围

### 基线与范围
以任务产品基线 `20f14205` 对比当前 HEAD 与工作树，包含已提交导航 `3495b19f`、后端/生成物 `611c8523`、财务/UI/规范 `425d1c52`。后续合入分页任务 `062ea382` 的合并 `4b7316cc` 已重点检查公共财务模板：`paginationSizeChanger` 与新增快照/选择逻辑均保留。第一阶段已验证且未变化的导航逻辑不重复审查。

### 已发现并修复
1. 六个财务列表仍可能将请求层消费后的 `undefined` 交给 `unwrapPage` 或直接读取 summary，新增公共 onRequestError 又会把 TypeError 输出给用户并重复请求层通知。已对这些列表显式使用 `skipErrorHandler:true`，由公共模板单点提取服务端中文报文或显示中文兜底；各页通过现有 ensureListResponse 守卫缺失响应。不修改其他请求。测试覆盖消费后的缺失响应拒绝、首次失败不冒成功零值、真实 Error/Axios 形态的服务端原因。
2. 新增前端金额累加使用 Decimal 默认20位有效数，不足以覆盖数据库 numeric(28,8) 的合法范围；大额正负相抵会先丢失小数。已在金额聚合模块局部 clone precision48，覆盖28位金额与int64记录数累加位数，不改变全站 Decimal 配置；回归验证大额相抵后仍精确保留0.12。
3. AGENTS 新增公共列表规则的 Markdown 嵌套缩进机械错误已修复。
4. 新增服务端错误提示用例初始模拟原始 object，ProTable 会包装非 Error 而丢弃 response 属性；最终改为真实 Error 携带 response，消息可见性使用 waitFor，不以 sleep 规避调度。

### 产品路径结论
- 六类 matchedSummary 使用各自完整列表谓词的分页前克隆聚合；组织边界不另建、不放宽，count来自同一分桶记录数且作为total。金额字段按实体映射，数据库 numeric 求和后用Go decimal解析，未加载全量实体。
- 发票null本币在SUM/COUNT中保持缺失，显式本币0仍存在；按方向/原币/本币分桶，不跨币种混加；对冲顶部先完整筛选再AND已确认，草稿筛选不显示已确认金额。
- DTO仅由service投影，biz为纯领域类型；新增Proto字段、Go绑定、OpenAPI与web生成类型同步，未见手改生成物或写入/锁/Schema变更。
- 前端request不再提交指标状态；模板在官方onLoad按返回数组读取同一快照，迟到/取消/失败结果不覆盖。底栏无选消费完整matchedSummary，跨页勾选优先，非费用没有利润；费用台账明确显示明细差额，订单有效集合才显示费用利润。
- 跨页记录使用框架原生preserveSelectedRowKeys，外部onChange与公共更新组合；采纳同ID新行时同步已选显示值；提交筛选/公司立即清空，分页/页大小/排序不在scope内；未提交草稿不清选。
- 财务公共UI不依赖API枚举/页面，领域投影走真实共享feature公开入口；唯一额外正式调用方orders/fees同步迁移且保持整单有效集合，不新增请求。
- 规范与实际签名、范围及原币/本币/未折算语义一致。剩余独立阻塞问题：无。

### 第二阶段独立验证
- 5文件11项定向Vitest通过，覆盖三页查询恢复、undefined中文拒绝、真实原生搜索草稿保留/提交清选、跨页选择与乱序快照。
- 最终3文件8项通过：快照4项（含中文失败/服务端原因）、金额精度1项、合并后分页3项。旧fixture失败已在本轮修正，最终验证通过。
- pnpm --dir web tsc通过；前端架构边界检查18项及真实509文件扫描通过。
- 14个最后修改产品/测试文件Biome check通过，git diff --check通过。
- 独立Go定向测试 `./internal/data ./internal/service ./internal/biz -run TestFinanceLedgerMatchedSummary -count=1` 通过（该命令未注入专用库，Postgres项不记作独立运行通过）。已核对后端代理在任务专用真实库的 `/tmp/shared-finance-postgres4.log`：六实体真实PostgreSQL集成均PASS，正式迁移隔离Schema，未触及开发库。
- 最终完整check:fast由主会话在合并分页提交后运行，本记录不提前宣称其结果。

### 限制
本审查以真实组件/框架定向测试、源码分层与专用数据库结果为依据，没有进行浏览器逐页人工点击；不据此声称所有其他未触及模块均无缺陷。任务范围内无剩余阻塞，后续提交、归档、合并与推送由主会话执行。
