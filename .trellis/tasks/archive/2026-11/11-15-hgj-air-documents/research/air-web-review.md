# 航空单证前端复核与自修

日期：2026-11-16（UTC）。角色：本任务 `trellis-check`。只复核及自修本组非生成
`web/` 文件；未修改服务端、生成物或规范，未提交、归档或覆盖其他工作者改动。
规范与任务文件由主会话同步。

## 已修复发现

### 1. 主单预选覆盖显式单证模式

- 文件：`web/src/pages/orders/templates/components/air/AirMasterFields.tsx`。
- 问题：`masterId` 预选成功后优先采用主单模式；URL 明确选择 DIRECT、主单固定
  HOUSE 时会静默变成 HOUSE，掩盖选择与主单身份不匹配。
- 修复：合法显式 `documentMode` 优先；未显式指定时采用主单固定模式。
  现有 `canAttach === true` 与模式校验继续阻止不匹配关联，不猜测票数资格。
- 验证：真实表单验证明确 DIRECT 保留且 HOUSE 主单校验失败，以及未指定模式时
  采用固定 DIRECT。没有新增历史合同或兼容分支。

### 2. 主单页与建票预选缓存形态冲突

- 文件：同上。
- 问题：主单页与预选使用相同 `['air-export', 'master', id, {organizationId}]`，
  前者缓存 `GetAirMasterWaybillResponse`，后者缓存 `response.data`。主单页进入
  继续建票、再返回的实际导航路径会消费另一种形态并污染其缓存。
- 修复：建票预选使用独立 `preselected-master` 查询键；身份和组织维度保留，
  继续由 React Query 管理请求，不引入额外数据镜像。
- 验证：预填主单页信封缓存后挂载建票，预选正确采用返回的独立 ID/版本，
  且主单页原信封及成员资料保持原形态。

### 3. 审核弹窗忽略已知资格撤销或重查失败

- 文件：`web/src/pages/orders/components/detail/OrderLockControl.tsx`。
- 问题：确认按钮只比较费用/订单/航空指纹，打开后若版本指纹未变、但最新
  `canLock=false`，或锁状态正在重查/重查失败，旧弹窗仍可能确认。
- 修复：确认同时要求当前非 loading、无 error、明确 `canLock === true` 且未锁定。
  保留打开时快照、输入和幂等键，不借新版本或指纹替换提交基线；服务端仍最终复验。
- 验证：资格撤销禁用；在途/失败禁用；恢复成功后提交原 Order/费用版本及航空指纹。

## 其余审阅结论

- `AirWaybillFields` 使用独立主/分正文路径；动态 `Form.List` 的运价行和其他费用
  均落在正文内，未产生顶层旁路字段。增加实际 `validateFields()` 验证：只编辑一个
  分区仍提交未挂载长文、声明值与运价；其他费用原始八位小数及明确金额 `'0'` 保留。
- 声明空消息与空费用汇总在领域适配器中清理；NVD/NCV/XXX 与 AMOUNT 区分，
  纯日期保持日期字符串。没有把 ProtoJSON 省略的 false 当成可关联 true。
- 详情保存仍仅采纳同次确切版本：Order +1，正文/状态改变时 House 与 Link 各 +1，
  Master 版本保持。主分身份及模式取编辑基线，未使用迟到重读替换提交预期版本。
  仅同步元数据，不回填在途编辑正文。后台变化检查增加航空主单/分单/Link 指纹，
  既有显式刷新确认及路由离开保护继续复用。
- 主单共同资料编辑克隆打开快照；预览/保存固定版本，15 秒前台及回焦探测不回填。
  在途编辑的 revision/session/scope 校验保留；DIRECT 恢复不提交分单，HOUSE 恢复
  保持分单和原订单版本。历史正文使用共用控件只读展示，时间继续本地格式化。
- 两份 E2E 夹具已同步 `house` 新消息，未保留旧扁平 house 字段调用。

## 验证命令与结果

最终增量测试：

```bash
pnpm --dir web exec vitest run \
  src/pages/orders/templates/air-preselected-master.test.tsx \
  src/pages/orders/templates/components/air/AirWaybillFields.test.tsx \
  src/pages/orders/templates/air-document-mode.test.tsx \
  src/pages/orders/components/detail/OrderLockControl.air-document.test.tsx
```

4 文件、16 项全部通过，总耗时 16.05 秒，未出现超过 20 秒文件。
此前局部审核门禁改动后另运行下列相邻回归，3 文件、14 项全部通过：

```bash
pnpm --dir web exec vitest run \
  src/pages/orders/components/detail/OrderLockControl.air-document.test.tsx \
  src/pages/orders/components/detail/OrderLockControl.fees.test.tsx \
  src/pages/orders/components/detail/OrderLockControl.test.tsx
```

```bash
pnpm --dir web tsc
pnpm --dir web exec biome check <本组所有修改及新增的32个非生成TS/TSX文件>
pnpm --dir web exec antd lint src/pages/orders/templates/components/air/AirMasterFields.tsx --format json
pnpm --dir web exec antd lint src/pages/orders/components/detail/OrderLockControl.tsx --format json
git diff --check
```

最终 tsc 通过；32 文件 Biome 通过；两个实现文件 antd lint 均 0 问题；
`git diff --check` 通过。Biome 文件集通过 `git diff --name-only -- web` 与
`git ls-files --others --exclude-standard -- web` 的并集选择，排除生成 services，
使用相对 web 的 TS/TSX 路径执行。复核前先查询离线 `antd info Form --format json`。

已有映射、详情竞态、主单工作台及 15 秒探测验证沿用
`air-front-mapping-validation.md` 和 `air-master-front-validation.md` 记录；本次未重复
运行所有已通过的套件。最终类型检查已覆盖本次所有新增回归及当前生成类型。

## 未修复问题与验证限制

- 本复核范围内未留下已确认而未修复的前端问题。
- 跨层 EXISTING 仅 ID/预期版本关联的后端旧副本要求已由主会话协调给后端责任人，
  本复核没有改回前端重复发送主单资料。该后端及真实 PostgreSQL 验证不以本报告
  的前端通过代替，须由主会话确认。
- 未执行真实浏览器、API 联调或生产构建；不把 mock 表单测试称为实际联调。
  PDF、AI 业务入口、进口服务及最终全量门禁归后续子任务或主会话，本组不宣称验收。
