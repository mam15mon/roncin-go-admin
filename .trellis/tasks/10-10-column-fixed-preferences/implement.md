# 实施与验证

- [x] 用户同意创建任务；干净工作区、前置分支确认，公共/专有偏好与结构约束研究完成。
- [x] PRD收敛、设计、计划及spec/research清单完成。
- [ ] 最新最终摘要后获明确批准，再task.py start。
- [ ] Trellis implement实施共享偏好/元数据/应用与浮层，先查antd Table/Select官方API；先实现后定向验证，不接管业务数据。
- [ ] 操作列稳定key与必要元数据接入，补真实用户/组织隔离，避免各页面重复逻辑。
- [ ] 订单费用专有保存/应用全链同步，验证应收/应付、编辑禁用、默认相等、权限和合并行；定向回归后分组提交。
- [ ] Trellis check独立复核列区域、结构约束、scope切换与调用方覆盖。
- [ ] 主代理浏览器用户、财务、订单费用、弹窗/抽屉：固定左右/取消、隐藏/排序、刷新、窄宽切换、键盘、存储失败、首尾与吸底条。
- [ ] 更新规范/导航，最终check:web、提交、归档、日志。

实施默认native trellis-implement/check注入，缺失子侧加载，不再嵌套派生。主代理协调、浏览器、规范、提交和收尾。批准前不派发实现或start。

局部：pnpm --dir web exec vitest run具体column-settings/responsive-table/feeColumnPreference/feeColumns测试；修改文件Biome、antd lint、tsc。最终pnpm run check:web。无后端/构建变更，不固定server全量/build；超过20秒的重型测试按功能拆分，禁止削弱断言。

依赖：完成分支codex/table-responsive-scroll提供统一Responsive能力，base_branch指向该分支；不擅自合并main或删除前置分支。
