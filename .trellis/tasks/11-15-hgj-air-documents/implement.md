# 航空直单与结构化主分运单：实施步骤

## 前置

.trellis/tasks/11-15-hgj-order-direction

父最终复核批准之后才运行task.py start。

## 顺序

- [ ] 核对前置验收、当前Git状态与角色注入规范；指定本组文件责任。
- [ ] 修改领域/Proto/Ent源与正式迁移（如适用），审阅生成差异；保留用户改动。
- [ ] 完成service/biz/data和前端调用/映射/交互；功能实现后再补风险匹配验证，禁止TDD。
- [ ] 跑受影响Go和正式迁移集成测试、前端定向Vitest/Biome/必要tsc及git diff --check。
- [ ] 派检查代理复核当前合同、失败/越权/并发/相邻路径；本组验证通过后Conventional Commit。
- [ ] 将具体命令、结果/缺口与父验收对应，不以测试文件存在或SKIP当作通过。

最终门禁在父整体验收组执行check:fast；打印生产配置相关组运行build及实际页面/PDF检查。详见父validation-matrix.md。

## 回退

不自动清空/重置数据库；若迁移/原子性验证失败，停止后续依赖并修正源合同和迁移，不用隐藏兼容分支掩盖。
