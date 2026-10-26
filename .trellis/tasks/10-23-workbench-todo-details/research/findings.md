# 只读调研记录

## 来源与范围

承接友商海管家工作台「预警关联具体订单并提供入口」交互思路。此前浏览器只读确认其按待订舱、待配舱等分组并列出订单/客户，存在规则设置；没有验证规则引擎实际执行。本期只使用本仓库已有可靠事实，不移植未验证业务规则。

账号、密码、Cookie、浏览器存储及原始认证报文不进入本任务/Git。实施不依赖继续登录友商。

## 代码证据

| 能力 | 位置 | 结论 |
| --- | --- | --- |
| 个人协作 | `server/internal/data/workbench.go:447` | OrderPersonnel 去重，当前公司，可读 SE/AE |
| 摘要 | `server/internal/data/workbench.go:548` | 未建账费用笔数与 ACTIVE 异常件数，尚无明细接口 |
| 有效建账 | `server/internal/data/fee_bill_association.go:13` | 活动账单行且账单未取消，包含草稿 |
| 工作台契约 | `server/api/workbench/v1/workbench.proto:11` | 固定会话身份，已有分页 RPC 范本 |
| 权限解析 | `server/internal/service/workbench.go:28` | 当前只有订单 read 品类，需要细分费用/异常读取范围 |
| UI 缺口 | `web/src/pages/workbench/WorkbenchSideCards.tsx` | 费用跳总台账，异常仅静态说明 |
| 订单链接 | `web/src/pages/workbench/display.ts:128` | 支持 SE/AE，未知类型不跳转 |
| 费用页 | `web/src/pages/orders/fees.tsx:90` | fee.read 或费用审核；财务账单跟踪另按权限加载 |
| 异常权限 | `server/api/order/v1/order_abnormal_case.proto:13` | read/create/resolve/delete 已独立，无需新权限码 |
| 异常面板 | `web/src/pages/orders/abnormal-case-panel.tsx`、`detail.tsx:897` | 共用 canManage、类型选项传空，需要修正本次直达所需权限和名称 |
| 异常主数据 | `server/internal/data/order_abnormal_case.go:41` | A 型共享主数据，不能误按组织过滤名称 |
| 工作区隔离 | `.trellis/spec/web/frontend/state-management.md` | 身份实例隔离，不能先展示旧数据再清理 |
| 集成测试 | `server/internal/data/integration_test_helper_test.go:28` | 专用连接，正式迁移，随机 Schema，无 Ent 运行期建表 |

## 方案依据

明细沿用计数粒度，避免笔数/票数歧义；两个显式接口符合当前工作台，不抽象通用任务协议。侧边抽屉承接卡片，处理复用原订单模块，不新增菜单和重复写入口。当前上下文和归档工作台任务已包含历史决策，无需重复检索会话日志。

规划阶段只写本任务目录；用户于 2026-10-26 批准修订及实施。并行列设置改动已由另一窗口独立提交，未覆盖。


## 实施中补充确认（2026-10-26）

- `OrderAbnormalCaseUsecase.Mark/Resolve` 的参数是组织、操作者、订单；原 service 把操作者与订单传反，已修复并新增参数回归。
- 异常登记若调用管理 `ListItems`，只有异常登记权限的员工会被主数据管理权限拒绝。新增登录可用的 `SearchOptions`，复用已有启用项过滤及中文/英文/代码/拼音搜索，管理列表权限保持原契约。
- 只有订单/异常读取的员工不一定有 `personnel.read` 或订单 `create`；详情聚合需分别裁剪人员列表与候选请求，权限进入查询键，并隐藏/只读对应人员区域。不能用额外授权或吞掉403掩盖强依赖。
- 新增 Router Hook 后，相邻详情测试原先依靠 mock useParams，缺真实Router。已改真实 MemoryRouter + Routes，保留原业务断言。
