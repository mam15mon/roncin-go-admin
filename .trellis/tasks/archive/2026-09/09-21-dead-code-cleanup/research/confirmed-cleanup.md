# 已确认清理汇总

- 前端：见 [web-cleanup.md](web-cleanup.md)，72 文件删除、5 文件精确修改。locale 实为8语言64文件。
- 后端：见 [server-cleanup.md](server-cleanup.md)，14手写Go文件清理，保留在用批量/Scoped入口与事务断言。
- 依赖：见 [dependencies.md](dependencies.md)，移除5个直接依赖、更新锁文件；go.mod不变，tidy复查无差异。
- 提交：745b7320（依赖）；effd92f3（代码）。
- 保留：Sentry、回填工具、仅测试使用符号、有意测试钩子、重复实现、忽略缓存。
- 后续记录：Sentry初始化缺入口、web/CLAUDE.md旧Umi描述；不纳入本次功能改动。
