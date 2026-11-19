# Journal - mam15mon (Part 4)

> Continuation from `journal-3.md` (archived at ~2000 lines)
> Started: 2026-11-17

---



## Session 218: 退役订单集运跨境模式与自拼汇总
<!-- trellis-session: v=2 fp=a46451b943a68ef0 -->

**Date**: 2026-11-17
**Task**: 退役订单集运跨境模式与自拼汇总
**Branch**: `codex/retire-shipment-mode`

### Summary

删除四类订单模式字段/UI/API/存储及自拼汇总专用投影，费用标题统一；保留普通运输与主分单。标准门禁和定向PG通过，扩展真库20项基线失败已逐项复现留档。

### Main Changes

- 模式字段、默认值、读写映射、拆票复制和幂等指纹项全链路删除；Proto/Ent/OpenAPI/客户端同步生成并新增正式删列迁移
- 移除自拼汇总入口/RPC/专用类型，保留普通拼箱、共享主单和财务；费用统一显示费用明细
- 历史迁移快照按被测正式前缀验证后再升级完整链，保留原断言

### Git Commits

| Hash | Message |
|------|---------|
| `34216cbd` | refactor: 退役订单集运跨境模式与自拼汇总 |
| `f6f40f1c` | test: 按历史时点验证退役迁移并记录基线失败 |

### Testing

- [OK] check:web分支通过：3009 PASS、11既有SKIP；check:server标准配置通过（默认集成SKIP不计真库验收）
- [OK] 定向真实库：新迁移2 PASS、历史升级修正2 PASS、生命周期/幂等20项含子用例PASS，0 SKIP
- [OK] 扩展PG整包未通过：本次相关2项已修，另20顶层失败全部在实现前4c05c13a基线复现，详任务research/baseline-failures.md
- [OK] 临时验证数据库/角色/凭据均清理，开发库未改；未执行真实HTTP脚本或浏览器业务验收

### Status

[OK] **Completed**

### Next Steps

- 随正常迁移流程应用20261117100000_retire_order_shipment_mode.sql；基线既有测试问题留档待后续处理


## Session 219: 账单五态筛选与无票核销限制交付
<!-- trellis-session: v=2 fp=68cd8bf42f01eb70 -->

**Date**: 2026-11-18
**Task**: 账单五态筛选与无票核销限制交付
**Branch**: `codex/bill-business-status-alignment`

### Summary

完成账单五态服务端投影与多选筛选，列表、详情、汇总、CSV同源；候选及锁内禁止完全未开票核销，部分票和免票仍可，建账同次输入保护与原子回滚一致。相关PG专项、类型检查、独立审查、真实应收应付与USD/EUR API闭环、浏览器多选和完整海运财务E2E通过。check:fast服务端通过，前端20例失败已在修改前基线4文件复现；全data PG扫描有范围外旧夹具失败并15分钟超时，均未宣称全量通过。临时隔离运行区、服务和凭据缓存已清理。

### Git Commits

| Hash | Message |
|------|---------|
| `1c054567` | feat(finance): 对齐账单五态筛选并限制无票核销 |
| `89d37d68` | fix(finance): 同步账单详情五态并完成流程验收 |

### Status

[OK] **Completed**


## Session 220: 统一 TagsView 页签草稿生命周期
<!-- trellis-session: v=2 fp=dfdf053980d91a00 -->

**Date**: 2026-11-18
**Task**: 统一 TagsView 页签草稿生命周期
**Branch**: `codex/tagsview-draft-lifecycle`

### Summary

业务页签切换保留草稿，关闭经确认清理，Chrome 前后台草稿使用原生离开提示。真实 Chrome 验收及 188 项定向测试通过，全量提成台账 5 项在改动基线复现。

### Git Commits

| Hash | Message |
|------|---------|
| `f5bbb02d` | feat(web): 按 TagsView 页签统一管理表单草稿 |

### Status

[OK] **Completed**


## Session 221: 账期与到期日收口：规划、复核与归档
<!-- trellis-session: v=2 fp=d33b8315808db84f -->

**Date**: 2026-11-19
**Task**: 账期与到期日收口：规划、复核与归档
**Branch**: `codex/tagsview-draft-lifecycle`

### Summary

规划父任务与两个子任务：账单汇率与到期日在建账后固定，增减费用和编辑都不再改动；客户结算规则收口为每个客户角色一条收款约定（未约定/现结/N 天 + 本币信用额度），附额度币种守卫迁移与只读盘点脚本。按评审补齐编辑沿用汇率、散客按最终到期日预警、迁移币种守卫，并按往来单位只属于公司简化守卫。实现由另一会话完成；复核到期日实现的代码与测试覆盖，按实现报告记录验收；check:fast 的既有失败与本任务无关。三个任务以 --skip-branch-validation 归档（直接提交 main，无 PR）。

### Git Commits

| Hash | Message |
|------|---------|
| `2f5568ad` | docs: 规划账期与到期日收口任务 |
| `24fef7ea` | docs: 补充账期与到期日收口规划的三处遗漏 |
| `072260be` | docs: 按组织约束调整收款约定迁移测试场景 |
| `53cb70e0` | docs: 收款约定迁移守卫直接取往来单位所属公司本币 |
| `33dfdc75` | fix(finance): 账单汇率与到期日建账后固定，增减费用和编辑不再改动 |
| `1f0b52ea` | feat(web): 建账与编辑账单可确认并直接指定到期日 |
| `f2806a9f` | refactor(partner): 客户结算规则收口为单条收款约定 |
| `8bc3d29d` | refactor(web): 客户档案只维护收款期限与信用额度 |
| `92ce5a61` | test(partner): 记录收款约定验收并清理旧测试命名 |

### Status

[OK] **Completed**
