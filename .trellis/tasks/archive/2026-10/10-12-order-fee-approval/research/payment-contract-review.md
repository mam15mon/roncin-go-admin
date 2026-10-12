# Research: 第 7 / 10 节付款契约复核

- Query: 禁止父提成未发时单独执行调整，是否足以最薄收敛真实付款与反转敞口。
- Scope: internal
- Date: 2026-10-12
- 审阅对象: 父任务 `design.md` 第 7 / 10 节；本文不修改该文档。

## Findings

### 最小修正建议

1. 第 7 节补明新契约：父提成 CONFIRMED 时，任何来源的 CONFIRMED 调整只能随父单一次净额结清；独立调整 PAID 接口对父未 PAID 一律拒绝。覆盖人工 INCREASE / DECREASE、补录冲减、反转冲减，不能只限制补录来源。父单 PAID 后才允许独立执行。父付款与调整执行互锁，抵扣记录与父付款快照一起写入。
2. 第 7 节“过去已单独执行的调整不再次加减”改为新运行期此前提不应出现；遇到此旧组合明确阻断，按第 10 节整理，而非增加长期旧组合付款公式。净额零允许抵扣结清；净额负应报错，不能静默置零。
3. 第 10 节增加明确预检：所有父 status != PAID 且任一调整 PAID 的记录，部署前阻断；按旧系统真实可执行动作及线下凭据核对/结清父单，重新预检。不能自动标 PAID、伪造金额或调整 execution mode。历史父 PAID 无实际金额字段仍展示未知，不拿当前有效金额填历史事实。
4. 新限制充分消除“父未付但调整已付”新数据，使未付父来源反转可整体取消未付权益；已付父来源反转保留历史 PAID，并复用既有冲减流程。但是还须补一个保护：**父已 PAID，尚有 CONFIRMED INCREASE 未实际增付时，当前反转按权益净额生成冲减可能追回未付增提。** 最薄方案是来源反转阻断并明确“先执行或取消未决增提，再重试”；不扩张现金冲销算法。若选择自动取消增提，应额外复核它是否曾支撑 CONFIRMED DECREASE 的限额，不能取消后变成超额追回。
5. 历史 PAID 缺实际付款快照不自动禁止所有独立登记，但任何新动作确实依赖历史现金额而无法证明时必须人工核对后处理。不能因为旧 paid_at 存在就断言旧实付等于原额或当前净额。

### 代码依据

- `server/internal/data/finance_commission_adjustment.go:298`：目前独立执行只要求调整 CONFIRMED，不检查父 PAID，应添加父状态门禁。
- `server/internal/data/finance_commission_write.go:155`：父付款目前只改父状态；需原子标记参与净额的调整并写实际金额快照。
- `server/internal/biz/finance_commission_types.go:228`：任一 PAID 调整即可触发“有已付敞口”分支，新父未付禁止独立调整后新记录不再出现这一组合。
- `server/internal/biz/finance_commission_types.go:255`：有敞口时按原额和 CONFIRMED/PAID 全部调整计算追回；未执行 INCREASE 会被包含，这是需要显式阻断的剩余组合。
- `server/internal/data/finance_verification.go:393`：反核销/反对冲共用助手，可在单个入口集中加未决增提保护，避免两侧漂移。
- `server/internal/data/finance_commission_adjustment.go:303`：人工 CONFIRMED 调整允许取消，补录 CONFIRMED 冲减不可取消；迁移整理不得随手忽略补录调整。

### 必要验证

| 场景 | 预期 |
| --- | --- |
| 父 CONFIRMED，人工增/减、补录减独立标 PAID | 所有入口拒绝，状态/审计零变更 |
| 父 CONFIRMED 200 + CONFIRMED 减 50 | 一次净付 150，减同时 PAID 并关联父付款；不能再独立扣回 |
| 父 CONFIRMED 200 + CONFIRMED 增 20 + 减 50 | 一次净付 170，两项原子执行 |
| 净额 0 / 非法负净额 | 零额结清留痕；负额明确拒绝，无静默截零 |
| 父 PAID 后新增 CONFIRMED 减 20 | 旧付款快照不变，可以独立实际扣回一次 |
| 父付款和调整执行竞争 | 父未付独立接口拒绝；父付款先成功后已抵扣调整不能再执行 |
| 补录提交/批准与父付款竞争 | 待审先成立则付款阻断；付款先成立则补录按已付上下文产生独立待执行冲减 |
| 新未付父单来源反转 | 原父/未付调整取消，不生成现金追回 |
| 新已付净额 150，后有待执行减 20，再来源反转 | 已有减 20 保留，新反转减按剩余权益 130；合计追回不重复扣减 |
| 父已付，有未执行 CONFIRMED 增 20，再来源反转 | 明确阻断先处理增提，不生成未实际增付的追回 |
| 旧父未 PAID + 调整 PAID | 预检阻断并列出；无自动升级/删除/回填，人工旧流程处理后重跑 |
| 历史父 PAID 无净额证据 | 展示未知，必要新现金敞口动作要求核对，禁止 effectiveAmount 填 paidAmount |
| 反核销及反对冲同场景 | 共享助手行为相同，已有 PAID 快照及执行关联保持不变 |

## External References

无，依据仓库代码。

## Related Specs

- `.trellis/spec/server/backend/finance-commission-lock.md`：PAID 不改写、反核销/反对冲一致。
- `.trellis/spec/server/backend/database-guidelines.md`：订单→父单→调整锁内事实复核与共享事务。

## Caveats / Not Found

- 未修改父设计、代码、数据或运行测试。
- “独立执行必须父已付”是主会话提出的新契约建议，应在最终设计审阅呈现。
- 原自动 CONFIRMED 补录不可撤销限制保持，不新增成本红冲。
- 新限制减少组合，但不能单独解决父已付、未执行增提与来源反转的现金敞口问题；需上述最薄阻断或经审阅选择明确处理。
