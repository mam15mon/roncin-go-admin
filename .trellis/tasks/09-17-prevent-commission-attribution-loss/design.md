# 技术设计：经营数据与责任人员归属收敛到公司

## 核心口径

1. `partner_assignment.organization_id ≡ partner.organization_id`，且客户档案所属组织必须为 `kind=company`。该列保留在实体与数据库中，继续作为提成快照严格等值匹配的键；但其值在写入时由后端从当前公司派生，**不再由客户端提供**。
2. 部门/团队只承担组织架构展示与人员挂靠（Membership）；人员是否可担任责任人员按「公司子树内存在启用 Membership」判定。
3. 提成快照匹配逻辑保持 `assignment.organization_id == order.organization_id` 严格等值不变；由于两者都被约束为同一公司节点，配置了责任人员的客户必然命中。
4. 快照形成时缺少提成岗位责任人员 → 业务错误阻止订单操作（fail-closed，用户已确认）。
5. 总部 `kind=headquarters` 只承担治理、公共主数据与跨公司管理；没有显式目标公司的客户/订单新建入口在总部工作台拒绝。总部通过已有跨组织权限操作明确目标公司的财务数据时，数据仍归属目标公司。
6. `order_personnel.organization_id ≡ order.organization_id`；人员真实部门只保留在 Membership，订单协作人员写契约同样不再接受组织 ID。

## 契约变更（server/api/partner/v1/partner.proto）

- `PartnerAssignmentInput`：删除 `organization_id`（原 field 3，REQUIRED）。写路径不再接收组织。
- `PartnerAssignmentOption`：删除 `organization_id` / `organization_name` / `membership_enabled`，候选语义收敛为纯人员（`user_id` + `display_name`）。
- `PartnerAssignment`（读消息）：保留 `organization_id`，仅供详情展示，本期不改（最薄闭环）。
- `OrderPersonnelAssignmentInput`：删除 `organization_id`；`OrderPersonnelOption` 删除组织字段并按人员去重。订单人员读模型暂保留 `organization_id`，值统一为订单公司。
- 生成链路：`make -C server api` → 检查差异 → `pnpm run generate:web-client`。

## 服务端变更

### `server/internal/data/partner.go`

- `replacePartnerAssignments`（585-624 行）：
  - 写入 `SetOrganizationID(rootOrganizationID)`（该函数已有 rootOrganizationID 入参；客户档案写入路径严格归属当前组织，等价于档案组织）。
  - 成员关系校验从「人员在该节点持有启用 Membership」改为：先计算当前组织子树内全部启用组织 ID（复用 `organizationWithinRoot` 与现有全量组织投影），再校验 `Membership.OrganizationIDIn(子树) && UserID && Enabled && User.Enabled`。子树外或停用成员关系 → `biz.ErrPartnerInvalidArgument`。
- `ListAssignmentOptions`（99-142 行）：查询范围不变（公司子树内启用成员关系），分页转换前按 `user_id` 去重（同一人员取排序首条），转换不再填充组织字段；关键字检索逻辑保持。

### `server/internal/service`（partner 相关转换）

- 组装 `biz.PartnerAssignment` 时不再读取输入组织字段；`PartnerAssignmentOption` 转换删除组织字段映射。

### `server/internal/biz`

- 增加公司经营工作台门禁，客户创建/导入与订单创建在总部工作台返回明确中文错误；更新路径由既有公司归属实体查询自然限定。
- 新增业务错误（如 `ErrPartnerCommissionAssignmentMissing`），中文消息列出缺失岗位（销售/操作/客服）。
- 现有「除内部联系人外每岗位最多一条」规则与唯一索引不变。

### `server/internal/data/order_write.go`

- `snapshotOrderCommissionAttributions`（159-172 行）：查询结果未覆盖销售/操作/客服三岗位的任一岗位时，返回 `ErrPartnerCommissionAssignmentMissing`（不再静默返回 nil）。错误沿事务回调原样外传，触发 Create / 草稿更换客户两条路径回滚（`order_write.go` 两处调用点需逐一确认错误传播）。
- 既有订单快照不重建、不改写。

### 订单协作人员与列表筛选

- `ListPersonnelOptions` 在公司子树内按唯一用户查询、计数、排序与分页，不按 Membership 行分页后去重。
- `createOrderPersonnel` 校验人员在订单公司子树内存在启用 Membership，写入 `order.organization_id`；客户端不再决定组织。
- 独立 `AssignPersonnelRequest` 同样移除组织输入；`orderPersonnelRepo.Assign` 按订单公司子树校验并写订单公司。订单列表的人员所属组织筛选随之删除，保留人员 ID 筛选。

## 迁移

- 新增 `server/migrations/20260917120000_operating_company_personnel_ownership.sql`：
  幂等 `UPDATE partner_assignments SET organization_id = partners.organization_id FROM partners WHERE partner_assignments.partner_id = partners.id AND partner_assignments.organization_id <> partners.organization_id`。
- 同一迁移把 `order_personnels.organization_id` 收敛为所属订单公司，并在执行前断言不存在总部客户、总部订单；发现即失败，禁止静默迁移经营数据。
- 不触碰 `order_commission_attributions`（历史快照不改写）。

## 前端变更（web/）

- `src/pages/partners/components/BasicInfoSection.tsx`：移除 9 个责任槽位的组织下拉，仅保留人员选择。
- `src/pages/partners/partner-detail.tsx`：删除 `assignXxxOrg` 表单状态与提交字段；`adminServiceListOrganizations` 拉取如无其他消费方一并移除；适配候选选项结构（纯人员）。
- 订单人员表单与适配器删除各岗位组织字段和兜底推导；人员选择后只提交 user + role。
- 生成物：`pnpm run generate:web-client` 更新 `src/services/roncin/`，随同一提交审阅。

## 权衡与边界

- 保留实体列而非删除 `partner_assignment.organization_id`：快照匹配键与审计语义仍在，删除列属于额外重构，不进入本期。
- 候选接口按人员去重放在服务端而非前端：选择器语义属后端契约，避免每个调用方各自去重。
- 去重必须在数据库分页前完成，以用户作为分页实体，确保 total、页大小和跨页稳定性正确。
- 不为「散客/临时客户」开豁免：客户档案写入路径统一，散客同样需要三岗位责任人（当前订单客户校验本就要求同组织档案）。

## 回滚

- 服务端/前端代码按提交粒度回滚即可；迁移 SQL 为幂等收敛，旧代码读回派生后的组织值不受影响。
- 新增报错仅影响「缺配置仍想开单」的场景，回滚后恢复原静默行为。
