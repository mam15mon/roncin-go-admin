# 系统审计六类缺陷修复

## 目标

修复本次只读审计确认的六类业务缺陷，保护客商资料、权限边界、启停状态、资金登记唯一性和查询/金额口径。用户于 2026-11-11 指示「写trellis 修复」，授权建立任务并准备修复；随后明确要求「执行计划，独立review，合并回主，安全删除分支，push」，执行与 Git 交付范围已授权。补全规划并独立复核后激活对应子任务。

## 需求与任务地图

| 需求 | 优先级 | 已确认问题与证据 | 所属子任务 |
| --- | --- | --- | --- |
| R1 | P1 | 业务身份弹窗调用全量更新却遗漏资料，导致联系人、别名、档案与非创建人分工被删除，散客标识被清空。`web/src/pages/partners/components/RoleSwitchModal.tsx:67`；`server/internal/data/partner.go:319` | `11-11-partner-role-preservation` |
| R2 | P1 | 覆盖导入仅要求新建权限，缺编辑权限的用户可覆盖既有收发通知人。`server/internal/service/enterprise_resource.go:347`；`server/api/enterprise_resource/v1/enterprise_resource.proto:84` | `11-11-resource-import-permission` |
| R3 | P1 | 非 optional bool 为 false 时 wire 省略，编辑初始化 `?? true` 错误启用已停用数据。`web/src/pages/admin/components/org/OrgEditModal.tsx:43`；`web/src/pages/partners/partner-detail.tsx:393`；`web/src/pages/partners/components/RoleSwitchModal.tsx:75` | `11-11-disabled-state-preservation` |
| R4 | P1 | 同一资金登记失败重试每次生成新幂等键，第一次已落库但响应丢失会重复登记。`web/src/pages/finance/cashflows/index.tsx:543`；`server/internal/biz/finance_cashflow.go:141` | `11-11-cashflow-retry-idempotency` |
| R5 | P1 | 信用额度允许外币却直接同本币欠款比较。`server/internal/data/finance_bill_credit.go:104`；既有 `09-11-partner-terms-credit/prd.md` 明确额度以本币维护、不做查询实时换汇 | `11-11-credit-currency-consistency` |
| R6 | P2 | 空运主单更新机场后显示投影为新机场，列表过滤仍读取订单旧地点。`server/internal/data/order_query.go:148`；`server/internal/data/air_export.go:178`、`:406` | `11-11-air-order-location-filter` |

## 验收标准

- AC1 / R1：仅改变业务身份，其他档案、散客状态、启停、人员与结算资料均保留；非法角色和越公司目标被拒绝。
- AC2 / R2：只有新建权限可导入新资源，覆盖已有资源必须额外具备编辑权限；伪造覆盖请求同样拒绝且无部分写入。
- AC3 / R3：省略 false、显式 false 与 true 在编辑往返后语义一致；新建默认启用仍保留，optional 字段存在性不被抹平。
- AC4 / R4：相同资金登记意图在结果未知后重试只产生同一张流水；不同意图和完成后的新登记可独立创建；失败保留输入。
- AC5 / R5：信用额度与欠款按同一公司本币比较；结算币种独立于信用额度币种，非 CNY 本币公司同样正确；零额度和仅提醒/直接干预模式保持既定语义。
- AC6 / R6：主单机场 A 改 B 后按 B 能查到、按 A 不再查到，起运/目的机场均覆盖，海运与无空运主单路径保持现有口径。
- AC7：各子任务含触发及相邻路径回归；契约与生成物一起交付；最终通过风险匹配的全栈检查，未运行的真实库验收单列。

## 范围与限制

仅修复六项及经实际契约确认的同根启停入口，不扩展为全系统重构；不包含尚未动态复现的空运表单提交基线疑点。保留现有未提交海运排版任务；不重置或清空数据库、不部署、不补猜测历史兼容分支。按本次明确授权合并 main、安全删除已合并分支并推送。禁止 TDD，先实现再补回归。按子任务分组提交，父任务负责集成验收。

## 证据状态

只读审计已用临时组件用例复现业务身份载荷缺失、组织与客商停用状态误启用、流水重试幂等键变化；真实 ProtoJSON 编码确认 false 省略。其余问题由完整代码链证实。上述不等于真实数据库写入验收，也不等于修复已完成。

## 修复验收结论

六项实现及定向验证已完成，独立 review 的结算规则加载失败问题已修复并复查通过。详见 `research/independent-review.md` 与 `research/final-verification.md`；初始“证据状态”仅描述修复前审计。
