# 保存保护审查结果

日期：2026-10-12。范围：A01 / A02 / A06 产品实现及必要公共守卫。当前审查已就绪，提交、spec同步和最终跨任务门禁由主会话负责。

## 已修复发现

1. `tabCloseGuard.ts` 批量关闭只放行首个脏页签。首个为后台订单草稿、当前客商排在后面时，已有确认动作更新页签后仍被路由再次拦截。现改为动作期间记录本次关闭范围内当前脏守卫实例；范围外、重新注册的实例不放行；同步完成或异步结束均撤销。
2. `partner-detail.tsx` 保存请求在途时确认离开 A、在同组件切换 B 后，A 迟到成功会清 B 的 dirty 并导航；A→B→A 也有同样代际风险。经主会话确认纳入必要正确性，使用身份 layout effect setup/cleanup 自增代际；校验、响应、catch/finally 均核对提交代际。新身份清自己的 saving，旧请求不改变新身份输入、dirty、导航或反馈。未增加请求数据链、缓存或通用抽象。
3. `partner-detail.tsx` 创建必要 ID 校验之后仍有冗余存在性分支，已直接按已验证 ID 导航。

## 路径核对

- 客商创建/更新、订单保存、账单编辑、流水登记保留默认统一请求错误消费。`undefined` 返回失败且不叠加局部提示；非空响应还需实体 ID 且 `success !== false`。真实成功才清 dirty/草稿、导航/关闭或刷新；订单刷新仍为 best-effort。
- 主档 GET 用明确外抛查询；首次失败只显示失败页、返回和重试。主档数据存在时重查失败不移除表单；每个身份首次回填，dirty 输入不被后台新值覆盖。
- 客商 dirty 覆盖普通字段、联系人、别名增删以及待提交别名输入。表单以 pathname+search 重建，外部编辑 state 由 layout effect 清理；用户/组织仍走现有 workspace key 隔离。
- 真实 React Router 的 push、POP、replace、同菜单身份变化均经 blocker；关闭/组织异步确认期间复用确认，失败后恢复保护。
- 未修改权限、生成物、服务端契约、写目标或真实业务数据。

## 验证

- 修改文件 Biome：14 文件通过（包括新增加的 `tabCloseGuard.test.ts`）；最终局部修复后的 5 文件再次通过。
- 类型：`pnpm --dir web tsc` 最终通过。
- 守卫/组织/页签相邻回归：`tabCloseGuard.test.ts`、`useRouteLeaveGuard.test.tsx`、`OrganizationSwitcher.test.tsx`、`TagsView.test.tsx`，4 文件 58 例通过。含新增批量关闭真实导航、范围/实例隔离及异步失败撤销。
- 客商最终回归：`partner-detail-save.test.tsx`，1 文件 7 例通过，组件测试耗时 4.63 秒。含同实例 A→B 与 A→B→A 的旧保存迟到成功。
- 未更改的保存失败矩阵不重复全量执行：实现者先前已运行相关订单、账单、流水和模板回归，主会话记录原始结果；本审查完整读取并核对断言与代码路径。
- `git diff --check` 通过。
- 浏览器证据由主会话负责，最终 6 项通过（取消/关页签/侧栏/组织拒绝及失败恢复与成功/GET重试/PUT失败）；全部页面业务写请求拦截，无真实业务写入。浏览器证据文件：`research/browser-verification.json`。

## 未修复发现

无尚待产品决策或任务范围内未修复的具体发现。跨任务最终 `check:web`、spec同步与归档尚由主会话继续执行，不作为本子审查完成的产品缺口。
