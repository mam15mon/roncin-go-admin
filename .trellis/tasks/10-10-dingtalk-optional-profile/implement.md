# 实施与验证

- [x] 用户已批准最终规则与开始实施，创建独立分支并检查干净工作区。
- [x] Trellis implement实施：网关移除可选字段失败，持久化空串口径统一，实施后更新相关网关/隔离PG回归。
- [x] Trellis check独立复核：真实异常保留、身份/权限/状态/租约边界、前端既有提示与空值消费。
- [x] 主代理同步规范、运行check:fast并通过独立roncin_go_admin_integration实际执行定向测试。
- [x] 主代理开发真实同步、实际页面验收、git diff --check、提交。
- [x] 归档当前任务并记录日志，其他任务状态保持。

定向命令：go -C server test -p 32 ./internal/data ./internal/biz ./internal/service -run TestDingTalkDirectory -count=1 -v；隔离库通过进程内环境注入，不输出连接串或个人资料；go vet相关包；前端既有同步面板/列用例；最终pnpm run check:fast。不涉及构建配置，无需固定生产build。
