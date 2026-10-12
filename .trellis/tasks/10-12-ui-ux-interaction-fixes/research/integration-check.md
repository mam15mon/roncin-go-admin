# 最终门禁集成修正独立审查

日期：2026-10-12。范围仅 `web/src/pages/orders/components/order-commission-summary.test.tsx` 的资源 Hook mock 契约修正。

## 审查结论

通过，无需产品回退或额外兼容处理。真实 `pages/orders/list-resources.ts` 的五个 search 方法均为异步函数，成功或无可用资源时返回候选数组；旧 mock 使用不带返回值的 vi.fn，筛选收起保持挂载后才暴露 undefined 被按候选数组消费的问题。

现 mock 的 searchLocations/searchCustomers/searchOrderPorts/searchOrderCarriers/searchOrderPersonnel 均 resolve 空数组，与真实无候选返回契约一致。补充 serviceTypeOptions/cargoCategoryOptions/locationOptions 为返回对象实际字段，并使用 `ReturnType<typeof useOrderListResources>` 检查完整性。没有修改请求客户端、产品筛选代码或添加 undefined 自动回退。

所有提成断言保持原样：本人/组织可见性、金额/币种、缺省占位、风险提示、下钻内容、台账权限、订单行接线均未删除、放宽或跳过；只调整依赖 mock 结构和异步返回。

## 验证

- 独立 `pnpm --dir web exec vitest run src/pages/orders/components/order-commission-summary.test.tsx`：1 文件 12 例通过；测试执行 7.98 秒，冷启动总耗时20.97秒（转换5.76秒、导入11.77秒），不存在超过20秒的重型组件测试执行长尾。
- 独立单文件 Biome：通过；`git diff --check`：通过。
- 实现者相邻 3 文件 18 例与 tsc 已通过。本次独立审查没有新修改，不重复相同类型全量检查。
- 父任务完整 `check:web` 最终结果由主会话负责记录；本次集成修正已就绪。
