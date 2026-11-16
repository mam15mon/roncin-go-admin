# 空运进口真实浏览器验收

## 最终结果

2026-11-16：正式SQL迁移后的真实PostgreSQL隔离库、真实后端和浏览器整例通过。仅AI业务权限身份完成DIRECT真实表单建票、刷新编辑、ORDER/MASTER及PLANNED/ACTUAL重量确认、费用数量来源、HOUSE共享双票与跨中国午夜ETA、进口服务和外部变动输入保护。

实际命令：

```bash
RONCIN_ACCEPTANCE_WEB_PORT=8002 node /tmp/roncin-ai-browser-acceptance/run.mjs
```

退出码0，内部执行 `pnpm --dir web exec playwright test tests/e2e/air-import.e2e.ts`，结果 `1 passed (1.3m)`。最终证据目录 `/tmp/roncin-ai-browser-5xhpwo` 为仓库外0700目录，截图与日志为0600。最终稳定轮未遇产品文件HMR，未修改生成物，没有写外站或发送员工通知。

## 已执行覆盖

| 场景 | 真实结果 |
| --- | --- |
| 仅AI身份 | 新建公司角色仅授真实权限目录的AI业务族及必要公共读取；真实账号登录并选择公司；`/auth/me`含AI create且无任何AE业务能力，AE主单候选请求403 |
| DIRECT真实UI建票 | 在`/orders/air-import/new`明确选择DIRECT，填写真实航司/机场/时刻/客户/操作人员和货量，表单POST200；返回businessType=AI、tradeDirection=IMPORT、master.businessType=AI，house消息不存在 |
| 直单详情 | 路由为AI详情，刷新无HAWB；操作备注真实编辑保存200、刷新原值重读，后端仍不存在HAWB |
| ORDER PLANNED | 真实重量面板选择订单委托货量，填写明确人工理由并确认150KG；服务端预览重读当前确认150且未过期 |
| ORDER ACTUAL | 真实货物抽屉新增1件100KG货物，尺寸120×100×100cm，显式带入1.2m³；切实际货物来源，填写理由确认180KG，重读当前确认180且未过期，PLANNED150独立保留 |
| MASTER与直单成员 | 通过更多操作进入`/orders/air-import/masters/:id`；共同资料入口可用、DIRECT继续加票按钮禁用；主单PLANNED填写理由确认210KG，ORDER PLANNED仍150 |
| 费用数量来源 | KG数量预览按本票PLANNED/ACTUAL分别返回150/180、各自真实confirmationId和source；未误用主单210；不创建经营费用，也不引入汇率回退 |
| HOUSE真实共享 | API创建两票同一AI MAWB，每票独立HAWB；仅AI账号候选查询仅返回AI，目标主单存在 |
| 共享资料修改 | 主单真实Modal修改航班并把ETA改为中国11月18日00:30；预览显示2票影响，确认PUT200；两票权威ETA均为`2026-11-17T16:30:00Z`、航班均QX299 |
| 全成员汇总 | 主单请求pageSize1只返回1成员但total=2，完整合计200KG；从主单继续建票进入AI新建路由、正确预选共享主单和HOUSE输入 |
| 进口服务子集 | 仅承接换单与派送，真实控件分别登记完成，保存200；API精确2项承接/2项完成，其余不适用 |
| 主表输入保护 | 服务保存前主表有未保存操作备注，服务保存不覆盖；主表随后保存200并刷新原文重读，版本基线可继续提交 |
| 真实15秒轮询 | 独立管理员Cookie会话更正已完成派送参考号且填写原因；不模拟时钟，前台25秒等待窗内出现持续变动提示，原输入保留、服务保存按钮禁用 |
| 回焦检查 | 刷新建立新基线后再次打开服务稿，独立管理员会话修改；blur/focus触发检查，10秒等待窗内出现提示，输入保留、提交禁用 |
| 页面异常 | 最终整例收集pageerror为空 |

本组覆盖父矩阵V1/V3/V4/V7/V8/V9/V11/V16的代表浏览器风险，实际后端权限、审核/恢复、并发和财务专项另由各验证记录承担。

## 截图与证据核对

最终场景子目录包含：

- `ai-service-polling-input-preserved.png`，SHA-256 `2786e874d095434699a7bb0d5ee32943288b4c8d8248f961342c7461b4f1b285`。
- `ai-service-focus-input-preserved.png`，SHA-256 `bc67828b5437de212ad18d409f356b86f98d6671131c3f21c78133620bfc7993`。

已实际查看两张实图：页面为AI主分单详情，主表备注保持，原服务变更原因输入分别保留polling/focus后缀，保存服务按钮禁用，页面持续显示变动提示；没有将新版本回填旧输入。

## 隔离编排与独立清理核验

`/tmp/roncin-ai-browser-acceptance/run.mjs`复用已验收SI的最薄编排：随机本地管理员、随机AI前缀数据库/角色、正式有序SQL迁移、真实Go验收配置与Vite、指定Playwright单文件、finally/信号进程组回收。只管理本次追踪的资源，本机postgres经已授权sudo创建/删除随机库与角色，不改开发角色。通知和对象存储关闭，不做AWB PDF。

最终随机资源 `roncin_acc_ai_browser_1794796008549_db1e12d3`：

- harness按owner校验后清库/角色，数据库、角色、活跃进程三个追踪集合归零。
- 结束后独立SQL核对：指定pg_database计数0、指定pg_roles计数0。
- `ss`独立核对8010、9010、8002均无监听；8001既有未知进程未停止。
- 最终证据目录0700，截图与日志0600；凭据未写入仓库文档。

## 检查及中间失败分类

- `pnpm --dir web exec biome check tests/e2e/air-import.e2e.ts`：最终通过。
- `pnpm --dir web tsc`：最终通过。
- `node --check /tmp/roncin-ai-browser-acceptance/run.mjs`：通过。
- `git diff --check`（本组两文件）：通过。

中间试运行不计整例通过：首轮测试漏选必填操作人员、选项快速提交导致航司未保留，新增明确选人及控件本体选中值等待后解决；第二轮沿旧航空例子误把“实际货物明细”当button，当前实际入口为更多操作menuitem，按真实DOM修正。两轮均完整回收本次库/角色/进程/端口，无半票或产品校验放宽。实现/检查代理期间必要主单会话、审核锁序和恢复归属修正完成后，才启动最终稳定轮。

## 验收边界

本组未做AWB制作/PDF、物理打印、生产构建/同域静态入口、工作台/通知/报表/完整CSV及完整费用财务链；分别属于后续依赖与父最终验收。重量均为有明确理由的人工确认值，不假定算法自动产生150/180/210。共享ETA仅验证权威日期投影，不将未创建费用的场景声称为已验证跨周系统率刷新。服务只选代表子集，不穷举全部32种组合；HOUSE建票由真实API辅助，DIRECT建票确由真实浏览器提交。
