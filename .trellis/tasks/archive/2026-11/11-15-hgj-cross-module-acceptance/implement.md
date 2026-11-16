# 四类订单横切接通与完整筛选导出：实施步骤

## 前置

.trellis/tasks/archive/2026-11/11-15-hgj-order-direction、.trellis/tasks/archive/2026-11/11-15-hgj-air-documents、.trellis/tasks/archive/2026-11/11-15-hgj-sea-import、.trellis/tasks/archive/2026-11/11-15-hgj-air-import、.trellis/tasks/archive/2026-11/11-15-hgj-air-waybill-pdf

父最终方案已批准；确认前置交付验收通过后运行task.py start。

## 本组最薄实施边界

当前四类创建/详情/单证/服务/PDF已经前五组验收；缺口位于订单列表完整导出及工作台、通知等横切出口硬编码。

- 服务端专用`GET /api/v1/orders/export`，独立无page请求、明确四类businessType、其余筛选排序与列表同内核；返回`Order[]/total/generatedAt`及标准封套。完整资源投影在`withReadSnapshot`，当前公司和真实类型read权限/子资源裁剪复用列表。普通列表200上限保持。
- 前端公共订单模板官方`onLoad`采纳筛选/排序快照后允许导出，消费专用客户端一次取得完整结果；切公司/条件/排序使迟到下载失效。CSV仅渲染稳定授权行模型，复用既有工具。
- 工作台/本人待办/异常、正式通知深链和台账报表只按真实四类身份接通；不改变财务事实、固定进口服务、锁/事务或另五类范围。无新Schema、锁、权限平台；若实际发现必要变化先说明证据。
- 系统通知测试页仍只预览示例/首页动作，不借治理权限读取真实订单；四类正式通知内容与链接以无网络受控模板验证，禁止向员工发送。
- 本组验证脚本独占新的`acceptance-hgj-*`及必要e2e，真实四类业务/应收应付/建账/发票或免票/核销/提成、超过200结果CSV、两公司及按类角色在随机库执行，清理本次资源。

责任：`cross_module_server`所有server及生成物，`cross_module_web`所有非生成web排除验收e2e，复用browser代理只维护本组验收scripts/e2e和私有编排，Root负责spec/任务/最终门禁/提交。各代理不互抢写权、不commit或派子代理。

## 顺序

- [x] 核对前置验收、当前Git状态与角色注入规范；指定本组文件责任。
- [x] 修改领域/Proto/Ent源与正式迁移（如适用），审阅生成差异；保留用户改动。
- [x] 完成service/biz/data和前端调用/映射/交互；功能实现后再补风险匹配验证，禁止TDD。
- [x] 跑受影响Go和正式迁移集成测试、前端定向Vitest/Biome/必要tsc及git diff --check。
- [x] 派检查代理复核当前合同、失败/越权/并发/相邻路径；本组验证通过后Conventional Commit。
- [x] 将具体命令、结果/缺口与父验收对应，不以测试文件存在或SKIP当作通过。

最终门禁在父整体验收组执行check:fast；打印生产配置相关组运行build及实际页面/PDF检查。详见父validation-matrix.md。

## 回退

不自动清空/重置数据库；若迁移/原子性验证失败，停止后续依赖并修正源合同和迁移，不用隐藏兼容分支掩盖。

实施授权同步：父最终方案已获用户明确批准，按前置依赖执行；具体完成状态以实际验证结果为准。
