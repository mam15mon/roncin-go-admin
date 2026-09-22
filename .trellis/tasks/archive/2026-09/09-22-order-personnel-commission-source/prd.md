# 订单人员为提成归属唯一真相：开单三岗必填与快照源切换

## Goal

把「订单提成归属（销售/操作/客服三岗）」的唯一真相源从**客户档案责任人**切换为**订单人员分工**：

- 订单新建表单的「内部信息」区，操作人员/业务人员/客服人员三个字段变为必填（交互对齐「请选择委托单位」），选委托单位后仍自动带入客户档案责任人作为默认值，可手动改选。
- 后端开单校验改为「订单人员三岗齐全」，提成归属快照从订单人员取数；客户档案责任人缺配不再阻断开单。
- 客户档案/订单页快建中的三岗字段从必填降级为选填（仅承担带入默认值职责）。
- 同步修订 `.trellis/spec` 规格文档与 `sync:dev` 种子。

## Background（问题与证据）

- 用户在 SE 订单新建页选中客户 CUST-HY-001 后提交，被后端 `PARTNER_COMMISSION_ASSIGNMENT_MISSING`（"客户档案缺少操作、客服责任人…"）拦截。该客户档案只配了销售（`partner_assignments` 仅 SALES 一行），`sync:dev` 注入的全部 5 个客户均只有 SALES。
- 前端订单表单人员字段全部无必填规则（`web/src/pages/orders/templates/components/sea/SeaPersonnelSection.tsx:41`），带入只有销售、操作/客服空着也能提交；而委托单位是必填的（`SeaBasicInfoSection.tsx` 的 `PartnerQuickAddSelect required`）。
- 后端开单事务 `snapshotOrderCommissionAttributions`（`server/internal/data/order_write.go:146`）只读客户档案 `partner_assignments` 并要求三岗齐全，完全不看订单自身的 `personnel_assignments`——用户在订单页手选齐人员也无法通过。
- 用户确认产品方向：**订单人员为唯一真相**（2026-09-22 会话中通过选项确认），客户档案责任人仅作为带入默认值。

## Requirements

### R1 前端：订单新建表单三岗必填

- `SeaPersonnelSection` 在**新建模式**下，操作人员（`operatorUserId`）、业务人员（`salesUserId`）、客服人员（`customerServiceUserId`）为必填，校验消息形如「请选择操作人员」「请选择业务人员」「请选择客服人员」；关联/单证/商务人员保持选填。
- 选中委托单位后从客户档案带入三岗默认值的既有行为不变（`SeaBasicInfoSection.tsx` `extractPersonnelFromPartnerAssignments`）；客户档案缺配的岗位留空，由表单校验引导用户手动补选。
- 详情/编辑页（`isDetail`）渲染同一区块时**不施加**必填规则，保持现状，避免存量订单缺岗时卡死其它字段保存。

### R2 后端：开单三岗校验切换到订单人员

- `OrderUsecase` 创建路径校验请求 `PersonnelAssignments` 必含销售/操作/客服三岗（创建者 CREATOR 岗位不计入）；缺失时返回 BadRequest，错误码 `ORDER_COMMISSION_PERSONNEL_MISSING`，中文消息按固定顺序列出缺失岗位（例："订单缺少操作、客服人员，请在内部信息区补全后再开单"）。
- 更新草稿路径（`UpdateDraft`）不施加该校验：`UpdateOrderRequest` 不携带人员字段，人员维护走独立接口。

### R3 后端：提成归属快照从订单人员取数

- `snapshotOrderCommissionAttributions` 数据源改为同事务内已写入的 `order_personnels`（角色限 SALES/OPERATOR/CUSTOMER_SERVICE，联查 `user` 取显示名）；`source_assignment_id` 写入订单人员行 ID（列无外键约束，无需迁移）；`customer_id` 仍写订单客户。
- 三岗齐全由 R2 的 biz 层门禁保证，data 层不再承担缺配校验职责。
- 创建事务内既有调用顺序（先 `createOrderPersonnel` 后快照，`order_write.go:127-133`）已满足新数据源要求，不调整。

### R4 后端：草稿换客户不再重拍快照

- `UpdateDraft` 换客户分支（`order_write_draft.go:284-303`）删除「删除归属行 + 按新客户档案重拍」，改为把现有归属行的 `customer_id` 更新为新客户（人员未变，归属不应重算）；无归属行的存量草稿为无操作。

### R5 清理旧门禁与降级档案必填

- 删除 `NewPartnerCommissionAssignmentMissing`、`partnerCommissionRoleLabels`、`partnerCommissionAssignmentRoleOrder`（`server/internal/biz/partner.go:33-62`）及其测试。
- 客户档案表单（`web/src/pages/partners/components/BasicInfoSection.tsx:127-137`）客户角色的操作/业务/客服改为选填；订单页快建客户（`web/src/pages/orders/components/PartnerQuickAddSelect.tsx:86-88`）同步放宽；相关注释一并修正。
- 订单创建仍要求客户具有 customer 角色（既有校验，不在本任务范围）。

### R6 规格文档与种子同步

- 修订 `.trellis/spec/server/backend/operating-company-commission-attribution.md`：错误矩阵、快照取数源、草稿换客户行为、客户档案职责描述。
- `.trellis/spec/domain/` 中涉及提成归属数据流的描述如有同样口径一并修订。
- `sync:dev` 种子客户补齐操作/客服责任人（幂等补挂），使「选客户带入默认值」演示完整；种子订单既有归属行不受影响。

## Acceptance Criteria

- [ ] AC1 新建 SE 订单：不选操作/客服人员时表单无法提交，三字段出现与「请选择委托单位」同级的必填提示；手选齐三岗后可提交成功。
- [ ] AC2 客户档案只有销售（如种子客户 CUST-HY-001）时，订单三岗手选齐全即可成功开单，不再出现 `PARTNER_COMMISSION_ASSIGNMENT_MISSING`。
- [ ] AC3 开单成功后 `order_commission_attributions` 恰好生成销售/操作/客服三行，`employee_id`/`employee_name` 与订单人员一致，`source_assignment_id` 指向 `order_personnels` 行，`customer_id` 为订单客户。
- [ ] AC4 创建请求缺任一提成岗位时接口返回 400，错误码 `ORDER_COMMISSION_PERSONNEL_MISSING`，消息按 销售、操作、客服 固定顺序列出缺失岗位且去重。
- [ ] AC5 草稿换客户后，归属三行保持原人员不变，仅 `customer_id` 变为新客户；不再因新客户档案缺配而报错。
- [ ] AC6 客户档案新建/编辑页与订单页快建客户时，操作/业务/客服可以留空保存。
- [ ] AC7 `pnpm run sync:dev` 重复执行后，种子客户三岗齐全（幂等补挂），再次开单带入完整默认值。
- [ ] AC8 受影响的后端测试（usecase 校验、快照、草稿换客户）与前端测试（人员区必填、带入默认值）更新并通过；`pnpm run check:fast` 全绿。

## Out of Scope

- 详情/编辑页人员分工的编辑落库缺口（当前详情页人员字段可改但 `UpdateOrderRequest` 不携带人员、改动不落库）——既有行为，另行立项。
- 订单创建后人员 assign/remove 接口不加三岗门禁：归属快照在创建时固化，后续人员调整不回写归属（维持既有"快照固化"口径）。
- 改配/拆单（`sea_order_change_split_execute`）流程：其人员与归属行均为从源订单复制，口径不受本任务影响。
- 历史订单/草稿的归属行回填或修补（项目无生产历史契约，不做兼容迁移）。

## Key Decisions

- D1 订单人员为提成归属唯一真相（用户 2026-09-22 确认）；客户档案责任人降级为带入默认值，随之档案表单三岗必填降为选填。
- D2 错误提示职责前移到前端表单校验；后端保留同口径硬校验作为数据完整性兜底。
- D3 `source_assignment_id` 列语义从"客户档案责任人行 ID"改为"订单人员行 ID"，无外键约束、无需 Schema 迁移，仅修订规格文档描述。
- D4 草稿换客户不重拍归属：人员未变则归属不变，仅同步 `customer_id` 冗余列。
