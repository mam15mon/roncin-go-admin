# 空运进口主会话验收

日期：2026-11-16。实现、独立复核、完整浏览器和最终门禁已完成，待本组提交归档。

## 前置与责任

航空DIRECT/HOUSE及独立正文已在 `df217ccd` 提交，海运进口及固定服务在 `74f86b3e`
提交并完成正式迁移、真实PG/浏览器/生产同域/PDF及门禁，均已归档。父当前完成3/6。
AI任务已start，server/生成物与非生成web分别由实现代理负责；浏览器代理仅准备独立e2e
和一次性库harness，主会话负责规范、整合、门禁和提交。

本组复用已有航空实体/重量/费用和固定进口服务。共享接口保留air-export/air-weights路径，
候选显式AE/AI，按ID目标从实体真实类型授权。没有新增兼容双读、多航段、复杂拆拼改配、
AWB PDF或完整筛选CSV；后两项按已批准依赖在后续组交付。

## 本组必须证明

- AI DIRECT/HOUSE合法IMPORT创建、读取/私有修改、同类型候选、真实主单/成员/独立HAWB、
  审核/锁/终止恢复与五服务完结。DIRECT无HAWB、单成员及历史模式边界不回退。
- HTTP生成路由/Authorization和service/仓储真实品类权限一致；仅AI能力可工作，
  AE借权及跨公司/方向拒绝；ORDER/MASTER重量各用实际目标身份授权。
- 主单权威机场/时刻与列表/日期筛选/详情/经营报表同源；进口ETA中国自然日、
  可更新系统周汇率及缺汇率原子回滚；手工/受保护费用和账单快照保持现有合同。
- 重量ORDER/MASTER与PLANNED/ACTUAL独立、完整成员汇总、变化过期但历史费用不追改。
- 主表与服务各自固定快照、确切成功Order+1及航空Master/Link/House身份版本不变才推进，
  外部变动/迟到/组织切换保持输入，15秒/回焦/隐藏离页符合统一规则。
- AE/SI及共享组件相邻路径；源/生成物幂等、定向测试、类型/静态和风险门禁。

## 真实结果（待补）

前端实现最终合并37文件317项通过（29.54秒），tsc、47文件Biome、3组件Antd及diff通过，
产品/测试写权已释放并进入独立全scope复核。完整最终门禁仍待执行。
服务端实现代理首轮真实PG54.8秒、0SKIP通过，覆盖
AE/AI双模式/正文/重量/审核/终止恢复/机场筛选及费用来源；AI五服务/ETA专项和HTTP
矩阵仍在收尾；最新22.7秒、0SKIP真实PG通过AI服务/完结/锁与恢复、跨身份和不完整成员、
ETA中国午夜/周汇率/缺汇率回滚及AE/AI相邻。产品另补审核完整Order集合锁序和HOUSE
恢复公司复验，修后定向通过。完整浏览器首轮仅AI登录/拒AE权限通过，新建未发POST
因测试操作员未选而停在必填校验；夹具修正后已放行新轮，尚无整例通过结论。

服务端实现已完成并释放写权，报告 `air-import-server-validation.md`。120场真实HTTP绑定/
Authorization/Service/Usecase矩阵通过（仓储stub，不冒称接PG），四包Go、主入口编译、
vet、buf及91份临时生成幂等通过。广相邻PG43.686秒仅一旧SI测试误算public同名表失败，
已改精确current_schema断言并单项2.590秒0SKIP补跑通过；其余AI/六类型审核/报表/费用率
保护矩阵及台账号均通过，不将首轮广批次整体记成功。独立serverchecker已接手全scope。

独立webchecker已修共享主单编辑/恢复弹窗的固定打开快照及迟到会话保护，修正一个
旧测试失效了错误queryKey而未真正重读的问题，正在定向收尾。浏览器第二轮实际DIRECT
UI建票200、AI/IMPORT/无HAWB、私有修改刷新及ORDER计划150通过；后续只因“更多操作”
菜单项被脚本误作按钮而停，已修测试并再次放行最终新轮。阶段证据不当作整例通过。

最终稳定新轮实际 `RONCIN_ACCEPTANCE_WEB_PORT=8002 node /tmp/roncin-ai-browser-acceptance/run.mjs`
退出0，Playwright 1项通过（1.3分钟）。仅AI权限、DIRECT真实UI建票及无HAWB、私有修改刷新、
ORDER计划150/实际180和MASTER210独立确认、本票费源预览150/180、HOUSE两票共享候选/
主单ETA跨午夜/完整汇总/继续建票AI路由、五服务2/5子集及主表草稿、真实15秒和回焦
输入保护均通过，pageerror为空。证据 `/tmp/roncin-ai-browser-5xhpwo/`，
一次性库/角色/进程与端口清理通过，精确结果见browser报告。未将数量预览冒称完整财务链，
本轮不含AWB PDF或生产同域构建，后续按依赖验收。
浏览器准备已通过Biome和Playwright list，
准备报告见 `air-import-browser-validation.md`，不会将准备成功当业务通过。
实现及后续独立检查的精确命令、失败修正与实际PG无跳过结果在各分层research回填。

本组触及公共Authorization中间件、共享航空仓储、权威ETA/费用联动和跨页面基线，
属于权限及状态的跨层高风险修改；最小定向和独立检查后已启动 `pnpm run check:fast`
（日志 `/tmp/hgj-air-import-check-fast.log`），实际退出0、总耗时237.0秒：
WEB237.0秒、SERVER144.0秒；前端510文件2761项通过，既有1文件11项跳过不计通过。
门禁包含类型/静态/架构、普通全仓Go与vet/Proto/漏洞检查；普通Go无数据库环境的跳过
仍不替代上述真实PG专项。已有Biome warning/info、happy-dom焦点TimeoutNaNWarning和
localhost:3000连接噪音均保留，未关规则或改依赖屏蔽。
航空PDF下一组还会增加生产输出入口，本组不固定重复build；真实AI浏览器先验证Vite
与当前Go服务，生产同域及最终包由打印组/父最终验收补齐，不提前宣称本组已测。

开发库已于上一组经用户明确授权保留数据迁移至1600；本组验收仍使用随机Schema或
一次性数据库，不清开发数据、不打印凭据、不发送真实通知。
