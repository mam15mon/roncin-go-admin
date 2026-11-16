# 航空主单页实施与验证

## 责任与实现范围

本组仅修改 `web/src/pages/orders/air-master.tsx`，新增主单页专属的
`air-master-change-notice.ts`、对应定向测试与测试 fixture。未修改服务端、生成物、
订单详情映射或 `AirWaybillFields` 的实现；以上由同任务其他工作者负责。

- 「从本主单继续建票」复用创建权限，并要求成功响应中 `canAttach === true`。
  ProtoJSON 省略 false 与显式 false 均禁止操作，不从 `memberCount` 推断资格。
  真实 `attachBlockedReason` 在页面展示；固定 DIRECT/HOUSE 模式随 URL 的
  `documentMode` 传入新建页。尚未固定模式的主单必须显式选择，非法模式不放行。
- DIRECT 成员显示无 HAWB，恢复关联不要求、不提交 `houseNo`；HOUSE 恢复继续要求
  真实分单号及候选订单版本。编辑与流程权限仍分别约束普通修改和恢复关联。
- 打开普通共同资料编辑时克隆服务端快照，预览与正式保存共同使用该快照的
  `expectedVersion`。主查询失效重取、前台探测均不替换表单或提交版本。
  运输字段之外接入 `AirWaybillFields prefix=['input','waybillContent']`，正文经所属
  领域的 `serializeAirMaster` 序列化；页面明确区分运输、印字正文与经营费用，
  运价/其他费用合计交由服务端计算。
- 主单页专属 React Query 探测每 15 秒在前台查询主单版本与阻断原因，恢复前台立即
  查询。使用既有主单详情接口、`pageSize=1` 减少成员载荷，不重复加载主数据。
  弹窗未打开、非当前路径、浏览器隐藏、窗口未聚焦或身份未就绪时不轮询。
- 发现变化持续提示「这个主单已经发生变动，请重新刷新。」检查失败保留提醒。
  显式刷新覆盖脏输入前确认；取消、刷新失败与保存失败保留表单及离开保护。
  刷新成功才重建表单和检查基线。刷新在途新增输入、旧预览迟到均不会覆盖新输入。
  组织/主单/编辑会话切换后旧异步结果不采纳到新会话。
- 保留 `GuardedModalForm` 与官方 `onFinish` 生命周期，不手工维护提交 loading。
  页头不展示内部并发版本，必要的主单历史版本列保持。

## 已运行验证

```bash
pnpm --dir web exec vitest run src/pages/orders/air-master.edit.test.tsx src/pages/orders/air-master.contract.test.tsx src/pages/orders/air-master-change-notice.test.tsx src/pages/orders/air-master.permissions.test.tsx
```

结果：4 个文件、14 项测试全部通过，最终执行约 13 秒；最长单文件低于 20 秒。
覆盖省略/显式 false、DIRECT/HOUSE 允许建票、空模式显式选择、恢复分单必填与直单
无分单、打开版本保持、结构化正文请求、失败保存保护、迟到预览、刷新在途编辑、
取消/失败/成功刷新、15 秒/隐藏/恢复/非当前页检查及原有恢复权限路径。
页面测试保留真实 React Query、官方表单和离开守卫，仅用轻量字段替身验证嵌套绑定；
完整正文控件另由负责该组件的工作者测试。测试 fixture 通过主题关闭弹窗动画，避免
在 happy-dom 中等待 CSS 动画。运行出现依赖侧 `TimeoutNaNWarning`，未产生失败或
未处理异常。

```bash
pnpm --dir web exec biome check src/pages/orders/air-master.tsx src/pages/orders/air-master-change-notice.ts src/pages/orders/air-master-change-notice.test.tsx src/pages/orders/air-master-test.fixtures.tsx src/pages/orders/air-master.edit.test.tsx src/pages/orders/air-master.contract.test.tsx
antd lint web/src/pages/orders/air-master.tsx --format json
antd lint web/src/pages/orders/air-master-change-notice.ts --format json
antd lint web/src/pages/orders/air-master-change-notice.test.tsx --format json
antd lint web/src/pages/orders/air-master-test.fixtures.tsx --format json
antd lint web/src/pages/orders/air-master.edit.test.tsx --format json
antd lint web/src/pages/orders/air-master.contract.test.tsx --format json
git diff --check
```

结果：6 文件 Biome 检查与逐文件 antd lint 通过，`git diff --check` 通过。
开发前已通过离线 `antd info Modal/Radio/Button --format json` 及
`antd demo Modal confirm --format json` 核对控件 API。

```bash
pnpm --dir web tsc
```

首次通过；最后一次在并行新增的
`templates/components/air/AirWaybillFields.test.tsx:57` 报错：
`ByRoleOptions` 不接受 `exact` 属性。已向主代理报告，由该文件责任人处理，
未越界修改。本组文件无类型错误；最终全前端门禁由主代理统一复验。

## 未验收项

- 未进行真实浏览器、后端联调或生产构建，不把 mock 页面测试视为真实事务、锁或数据范围验证。
- 主单正文完整字段、声明值空消息、合计与服务端版本/哈希、正式迁移由其他责任组验证。
- PDF 预览、打印和制作页属于后续输出子任务，本组未实施或宣称验收。
- 未提交 Git；按主代理职责由完整契约组统一提交源文件、生成物与测试。
