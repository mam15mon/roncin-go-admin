# 财务实施记录

## 范围处置

- BillEditModal：信用/余额、币种、账户为独立 Query；候选首次就绪补默认账户，后台重查不改人工选择。useBillChangeNotice 覆盖前台探测；保存或刷新期间禁止关闭，dirty 刷新明确确认，成功完整重读才更新父级表单及版本。
- bills/index：详情按 ID Query，写后按 finance 前缀失效；显式编辑刷新经当前客户端 fetchQuery，编辑会话保护 A→B→A 与迟到写响应。
- CommissionApplicationDetailDrawer/ApplicationsPanel：申请详情按 ID Query，移除父子 refreshToken，冲突和成功按 commission-applications 前缀失效。
- my-supplement-source：参数 ID Query + silent/skipErrorHandler，保留本人来源不可见统一 404 契约。
- CommissionRulesDrawer：员工关键字 Query，筛选下拉展开才启用；新建/复制/名单子表单只提交关键词和已选 ID，候选从当前组织缓存保活，移除 applySetter 请求协议。
- commissions/index、invoices/index：详情 ID Query；原官方列表与采纳快照不变。
- CustomSettingsPanel：账单费用/信用策略 Query，真实写响应更新缓存，标记 finance 前缀失效且不立即用旧读覆盖写响应；信用干预前缀另行失效。缺用户、缺公司、非公司工作区不查询/写入。
- BillGroupCard：共享 currencyOptionsQueryOptions，删除每个分组的手写请求链。
- CommissionCreateModal：显式预览 mutation + 当前客户端 fetchQuery 获取可取消信号；reset/candidate变化取消旧探测，预览指纹仍约束创建，undefined/失败列表先守卫。
- ExchangeRateSyncModal：抓取 mutation 的状态驱动 loading/error，成功回调才更新原始与人工草稿，关闭 reset 丢弃旧观察。
- ExchangeRateImportModal：预检 mutation 保存 token/batch，确认 mutation 复用该预检成功时创建的幂等键；失败重试不换键，reset/关闭移除旧 mutation 观察。

## 验证

- 相关财务模块（bills、commissions、invoices、CustomSettingsPanel、牌价抓取/导入、bill-creation）一次完整定向回归：48 文件、225 测试通过，日志 /tmp/finance-all-tests-final.log。
- 后续 UI 门禁/申请成功失效/导入同键复验：4 文件、21 测试通过，日志 /tmp/finance-postreview-tests.log。
- 最后增加策略身份缺失/非公司门禁 3 例并单独复验：1 文件、10 测试通过，日志 /tmp/finance-policy-final.log。
- 新增 BillEditModal.change-notice.test.tsx 的 4 例，覆盖人工输入、取消刷新、失败/成功刷新及本人保存两个窗口；已有 use-bill-change-notice 的计时/隐藏/失败回归继续通过。
- 完整 TypeScript 在财务主体完成后通过（/tmp/finance-tsc-final.log）；最后一次 /tmp/finance-tsc-complete.log 仅有其他组在途 SeaOrderChangeHistoryDrawer.tsx:250-251 的 events/total 类型错误，已反馈主会话，财务无类型错误。修改文件 Biome 和 git diff --check 通过。没有运行后端门禁或浏览器。

## 后续检查

由主会话执行跨组检查与最终 check:web；检查时保留正常官方 request 和写入会话 ref，不把这些误报为未迁移查询。未提交 Git，由主会话统一审查提交。
