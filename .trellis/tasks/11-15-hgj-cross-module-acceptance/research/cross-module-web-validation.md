# 四类订单横切与完整CSV：前端实现验证

## 变更边界

本组只维护非生成Web产品与针对性测试；服务端/API/全部生成物由server负责人维护，最终E2E与实际下载由独立验收负责人执行，规范和任务主文档由Root统合。未提交Git、未推送、未改生成客户端、未新增依赖。

### 完整导出

- `OrderListTemplate.tsx`继续使用官方ProTable `request/actionRef`，每次request把本次筛选/排序/总数与返回行数组用WeakMap关联；只有官方`onLoad`采纳后才建立导出快照。空成功结果也有有效快照，失败结果不能伪装为成功空列表。真实ProTable测试已证明官方引擎保留该数组引用且迟到结果不采纳。
- 提交筛选、排序和分页事件同步清除旧快照；新request也独立推进代际。导出完成前检查快照对象与挂载状态，布局清理立即撤销资格，防止条件变化或卸载后旧响应触发下载。订单模板按用户/当前公司/kind重建，切公司不会复用旧导出会话。
- `list-query.ts`抽出`buildOrderListRequestFilters`及`mapOrderListItems`供列表和完整导出共用；只有列表补page/pageSize。主/分单号码、模式及航空航司名称来自真实订单资源，表格未改用外层data/loading架空。失败封套/undefined和真实ID缺失明确失败。
- `list-export.ts`只调用一次真实生成的`orderServiceExportOrders`，不分页拼接。验证响应total与完整行数、生成时刻、每行ID/订单号/公司/真实业务类型；旧会话响应在映射可能触发主数据读取前即取消，避免切公司后继续加载旧候选。
- 按稳定业务行模型显式映射固定CSV列，实际箱号和附件数分别消费真实`container.read`、`attachment.read`权限；未授权列不出现，optional未知不补零。金额`csvAmount`两位无千分位、数量`csvDecimal`保精度；时间和文件名复用本地`formatDate`，BOM/CRLF/中文引号/多行和公式防护复用`utils/csv`。没有导出内部ID、版本或业务标签。
- 按钮文案为「导出完整筛选结果」。当前公司/用户未就绪或缺该类read，不请求列表/候选/导出。仅read权限即可导出已有授权列，无新增权限。

### 四类横切

- `features/orders/navigation/index.ts`公开最小的`ORDER_BUSINESS_ROUTES`、`OrderRouteKind`、`orderListPath/orderDetailPath`。四个订单定义和类型、工作台详情/费用深链/快捷入口以及通知动作白名单共同消费这份业务类型→路由关系；未知/未开放类型无链接，名称继续复用现有`statusMeta`。
- 工作台费用/异常入口按四类各自order read与对应fee/abnormal read取交集，只有SI或AI权限即可看到本类待办和快捷入口，不借出口权限。近期订单/财务摘要复用统一path，进口进度显示现有中文状态。
- 通知动作白名单支持四类合法UUID费用深链，仍拒绝参数/片段/错误路径；系统通知测试页的示例能力与`canSelectOrder=false`未改变，没有放开系统工作台读取真实订单。
- 财务费用详情原来只以`orderId`启用查询，缺少公司/权限门控和响应目标验证。本组最小补齐当前公司与`canReadFinanceFees`、用户/公司/订单query key、响应`orderId/organizationId`严格等值及失败封套拒绝；通过校验后才挂费用更正区。SI/AI类型显示复用业务中文元数据。未改变费用、核销、更正、报表计算或授予新权限。

### 最终字段核查的相邻修复

Root按已批准F24/F25指出，危险品取消后表单字段虽清undefined，航空更新仍省略optional，服务端会保留旧UN/CLASS/申报截止/包装等级。只修改航空公共适配入口`commonFields(clearEmpty)`：更新用已有text/time入口显式空串，包装等级空/0显式UNSPECIFIED=0；创建未填仍省略。表单类型明确接纳日期/选择控件清空null，AE/AI共用，没有增加第二套adapter或修改后端业务规则。

## 实际验证

所有Vitest均以`pnpm --dir web exec vitest run <相对web测试路径>`定向执行；没有用test脚本伪装单文件，也没有TDD。以下批次存在相邻交叉，不能直接累加成唯一样本数。

| 批次 | 实际结果 | 重点 |
| --- | --- | --- |
| `OrderListTemplate.export`、`list-export-page`、`list-export`、`fees/detail`、`form-adapter-dangerous-clear` | 5文件31项通过，11.36秒；其后fees/detail补失败封套再单独7项通过，6.85秒 | 官方采纳/迟到/空失败/筛选排序/卸载；pending切公司；四类各251行一次导出；CSV真实wire与identity；DG清空 |
| `list-query`、`list`、`MyTodosDrawer`、`workbench/index`、`notification-test-action` | 5文件96项通过，20.92秒 | 既有列表、待办与工作台相邻 |
| 公共模板默认列/主分单/审核列/拼箱/筛选恢复及registry、AI/SI适配器 | 9文件36项通过，18.87秒 | 公共接口与四定义注册无回退 |
| `four-kind-navigation`、`WorkbenchSideCards`、`notification-test-action` | 3文件56项通过，4.42秒 | 只有SI/AI权限、四类深链及通知白名单 |
| `todo-workspace`、`system-workspace`、通知access/navigation/page | 5文件16项通过，11.76秒 | 当前工作区/迟到与system示例边界 |
| 航空旧adapter、新DG clear和真实DG取消组件 | 3文件30项通过，8.38秒 | AE/AI显式空串/0、null创建省略、邻近填写保持/取消联动 |

新增重组件文件每个最多7个用例，单文件均低于20秒；没有加超时、静音console或关闭规则。

- 最终`pnpm --dir web tsc`通过（源和测试）；最后复验日志`/tmp/cross-web-final-tsc-complete.log`。
- 本组27个非生成Web文件Biome通过，无修复遗留；最终`git diff --check`通过。
- `pnpm run check:architecture`：18项规则测试通过，577个产品文件/2366本地依赖实际AST扫描通过，新增navigation经index公开，无页面反向互引。
- 使用离线antd CLI查询Button API与demo、Alert API；Toolbar及五个实际页面运行antd lint，六文件均0 issues、partial=false。

## 首轮修正与未完成项

- 首轮真实ProTable新测试使用精确按钮名忽略图标的accessible name，改为匹配实际按钮；初始sorter可为null，产品现只采纳ascend/descend，避免把未排序当有效排序。没有降低采纳断言。
- 页面导出测试最初候选mock返回undefined引发RemoteSelect失败，修夹具为异步空数组；跨公司rerender最初漏QueryClientProvider，按现有测试工具返回client重包同一provider。均为测试装配问题，无产品兜底。
- 首轮tsc定位测试ref的null/undefined和DG夹具漏paymentTerm，按真实类型补齐，没有类型压制。
- 本报告不声称真实服务器>200结果、实际CSV下载、四类财务链/通知内容或最终全栈门禁已通过；由server/验收及Root另记。未运行check:fast/build，也不借前五组结果替本组。

产品和测试已冻结，可交独立Web全scope复核；后续发现必要变更由Root分配写权。
