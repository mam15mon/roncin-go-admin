# 执行与验证

- [x] 用户批准创建任务，检查干净工作区，完成全站静态盘点和17组真实浏览器研究。
- [x] 完成PRD收敛、设计、实施计划及上下文清单。
- [ ] 用户批准最终规划摘要后task.py start，进入实施；当前只规划。
- [ ] 共享展示能力与用户列表先实施验证，避免将局部补丁复制到全站。
- [ ] 五个公共模板接入，财务列宽/订单固定列/设置小表验证。
- [ ] 按清单分组覆盖admin、partners/enterprise-resources、orders、finance/workbench、features；每组相关定向验证与diff-check后提交。
- [ ] Trellis check独立复核全部差异、覆盖清单和模块边界，机械问题局部自修。
- [ ] 浏览器390/962/1440/1920、展开/折叠侧栏、长表中段、弹窗/抽屉/嵌套/编辑，验证首尾横向访问、固定区不遮挡、条可达、无页面横向溢出。
- [ ] 更新前端规范和能力导航，最终pnpm run check:web（纯前端风险门禁）通过，提交、归档和记录。

局部使用pnpm --dir web exec vitest run <相对web的文件路径>、修改文件Biome、tsc及antd lint --format json。需要新增真正布局回归时使用Playwright真实浏览器，禁止用CSS字符串断言证明滚动成功。无构建/路由生产入口修改时不固定生产build。
