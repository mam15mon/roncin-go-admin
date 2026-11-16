# 航空表单正文与审核的主会话验证

日期：2026-11-16 UTC。映射/主单工作台及后端的独立报告分别见本目录其他记录。
本记录只列主会话实际执行的检查，不替代正式 PostgreSQL、浏览器或后续 PDF 验收。

## 实施与验证范围

- `AirMasterFields.tsx`：显式 DIRECT/HOUSE；已有主单按服务端 `canAttach` 和固定模式判断，
  不按票数猜；当前身份与模式隔离候选返回；主单正文只出现在新建主单区域。
- `air-template.tsx`：DIRECT 无分单字段，HOUSE 保存真实结构化正文；已有 HAWB 号只读，
  已签发改单原因依真实 `house.status` 显示。
- `AirWaybillFields.tsx`：主分分别绑定路径，折叠分节保留未挂载内容，动态运价行/其他费，
  字符串小数、声明文本/数值、纯日期、长文、签发备注；明确区分印字与实际运输。
- `OrderLockControl.tsx`：航空审核携带打开时单证指纹；缺失指纹或当前指纹改变时阻止确认，
  不给旧弹窗替换新指纹。

## 已执行命令与结果

```bash
pnpm --dir web exec vitest run src/pages/orders/templates/air-create-layout.test.tsx
```

7/7 通过，测试执行 17.22 秒；继续保留既有五卡片、运输分组、危险品及经营字段路径。

```bash
pnpm --dir web exec vitest run src/pages/orders/templates/air-document-mode.test.tsx src/pages/orders/templates/components/air/AirWaybillFields.test.tsx
pnpm --dir web exec vitest run src/pages/orders/templates/air-document-mode.test.tsx
```

正文 4/4 通过；模式 4/4 最终通过（10.42 秒）。首轮失败为测试查询未排除 tooltip 图标的
可访问名及 Select 虚拟列表重复文本，已修正查询并重跑对应文件，未降低业务断言。
覆盖 DIRECT/HOUSE 切换保留输入、保存后模式/HAWB只读、候选模式不符/省略false、
声明 NVD 切换清数值、十进制原值/零、Form.List 嵌套路径、纯日期/换行/10000字正文、
空汇总消息与禁用字段。每文件重型渲染不超过 4 例，测试执行均低于 20 秒。

```bash
pnpm --dir web exec vitest run src/pages/orders/components/detail/OrderLockControl.air-document.test.tsx src/pages/orders/components/detail/OrderLockControl.test.tsx src/pages/orders/components/detail/OrderLockControl.fees.test.tsx
pnpm --dir web exec vitest run src/pages/orders/components/detail/OrderLockControl.air-document.test.tsx
```

两既有文件共 11/11 通过，航空新文件 2/2 最终通过；首轮新例仅 exact 按钮名未匹配图标
而失败，修正为可访问名模式后重跑通过。覆盖航空缺指纹、订单/费用版本不变而主单变化、
普通相邻审核与失败重试。

```bash
pnpm --dir web tsc
pnpm --dir web exec biome check <主会话修改的TS/TSX文件>
git diff --check
```

最终 tsc 完整通过；正文测试曾错误使用 ByRoleOptions.exact，已删除并再次执行成功。
主会话修改的 9 个 TS/TSX 文件 Biome 通过；`AirWaybillFields.tsx`、`AirMasterFields.tsx`、
`air-template.tsx`、`OrderLockControl.tsx` 分别执行离线 `antd lint <单文件> --format json`
均 0 问题。控件 API 使用前通过本地 CLI 核对。

## 后续与限制

- 独立前端检查代理正在复核并补预选缓存/显式模式、审核当前资格等同链路回归，
  最终结果见 `air-web-review.md`，上述早期通过不能代替这些增量修复的验证。
- 本组未应用开发库迁移，未删除或重置任何既有业务数据。方向组记录的孤立海运主单
  仍需明确业务归属或具体处置后才能对该开发库升级；使用隔离 Schema 继续真实 SQL 验证。
- 未把前端 mock 的通过称为真实浏览器/API创建/打印通过。PDF 制作、四类财务链和完整
  CSV 由后续组实施、最终验收，不据此提前标记父任务完成。

## 本组最终门禁

`pnpm run check:fast` 已通过，实际 239.7 秒（WEB 239.7 / SERVER 233.2）。
前端 489 个文件、2625 个测试通过；另有既有 `sea-template.test.tsx` 的
1 个文件、11 个用例跳过，不能算通过。原有 Biome 警告与测试环境
getComputedStyle/localhost:3000 请求噪音保留，没有关闭规则或修改无关代码。
后端完整普通测试/vet/分层/Proto lint/漏洞门禁通过；该命令未注入集成数据库，
真实 SQL 验收采用 `backend-validation.md` 与 `air-server-review.md` 的独立零 SKIP 结果。

全量运行期间后端检查代理补入的显式主单版本复验已独立完成真实 PostgreSQL
`TestAirWaybillOrdinarySaveChecksExplicitMasterVersionPostgres`（3.326秒、0 SKIP）
和全量 Go vet；未改契约或生成物。门禁后未再修改产品代码。
本组未涉及生产打包配置或打印入口，因此未运行 build；按父计划在 PDF/最终组验收。
