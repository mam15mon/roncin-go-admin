# 实施清单

- [x] 创建任务并研究现有管理员入口、8类模板、发送载荷和订单范围。
- [x] 收敛PRD/设计/验收标准与研究记录；用户批准并明确改为独立Admin页面。
- [x] 用户已批准方案并纠正入口位置，同步后start任务，按Trellis要求派实施代理。
- [x] 增加Proto、权限清单与同步生成物，接入最薄通知预览/自发测试用例。
- [x] 复用模板及最终链接准备，固定示例/当前收件人/订单读取校验，保持零业务副作用。
- [x] 新增独立Admin通知测试页面、预览/源码、受控订单候选与发送反馈。
- [x] 功能后补针对性用例（禁止TDD），定向Go/Vitest、Biome/tsc、架构与生成幂等校验。
- [x] 独立trellis-check完整review并修复，更新可执行规范。
- [x] 契约/权限与依赖跨层变更执行check:fast、生产build及真实浏览器协议模拟，记录真机限制。
- [x] 提交功能工作，完成规范与验收记录，准备按授权归档/日志及Git交付（执行结果见最终说明）；不得擅自发送真实测试消息。

## 最小验证

Go定向覆盖新用例、Admin HTTP实际路由/权限及data消息准备/发送，回归8类模板和password-reset相邻路径；前端定向运行新增面板、admin/index及access相关测试，路径相对web。git diff --check；pnpm --dir web tsc；pnpm run check:architecture。

## 最终门禁依据

新增管理端公开契约、权限及消息副作用入口，需pnpm run check:fast；新增Markdown依赖和懒加载入口，需SENTRY_AUTH_TOKEN= pnpm run build并核对产物无测试辅助。真实钉钉仍以用户手动点击验证，不把mock和默认Skip当作真机/真实数据库通过。
