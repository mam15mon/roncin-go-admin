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
