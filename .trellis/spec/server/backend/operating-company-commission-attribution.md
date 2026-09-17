# 公司经营归属与订单提成责任人契约

## 1. 适用范围 / 触发条件

修改客户责任人、订单人员、开单、草稿更换客户、提成快照或组织工作台时适用。总部仅承担治理；部门/团队仅承担人员 Membership，不是经营数据或提成归属主体。

## 2. 签名

- `PartnerAssignmentInput { role, user_id }`、`OrderPersonnelAssignmentInput { user_id, role }`、`AssignPersonnelRequest { order_id, user_id, role }`：写契约均不接收 `organization_id`。
- `PartnerAssignmentOption` / `OrderPersonnelOption`：只返回 `user_id, display_name`，分页实体为唯一用户。
- 数据库不变量：`partner_assignments.organization_id = partners.organization_id`；`order_personnels.organization_id = orders.organization_id`；上述经营归属组织的 `kind = company`。
- `snapshotOrderCommissionAttributions(ctx, tx, companyID, orderID, customerID, at)` 在订单创建或草稿更换客户事务中运行。

## 3. 请求与落库契约

- 当前工作台新建/导入客户、新建订单必须是启用公司；总部不直接生成自身归属的经营根数据。显式目标公司、已有跨组织权限的治理/财务接口仍按各自权限判定，不以此门禁一刀切。
- 客户/订单人员可仅在公司下的部门或团队拥有启用 Membership，但用户必须启用，成员关系所在组织也必须启用且位于该公司的子树；落库 `organization_id` 一律写经营主体公司，而不是 Membership 所在部门。
- 订单必须使用同公司客户；快照查询继续严格匹配 `partner_assignment.organization_id == order.organization_id`。既有订单提成快照不会因客户责任人变化自动刷新。
- 迁移 `20260917120000_operating_company_personnel_ownership.sql` 先断言总部无客户和订单，再幂等收敛两张人员表的归属；不改历史 `order_commission_attributions`。

## 4. 校验与错误矩阵

| 条件 | 行为 |
|------|------|
| 总部工作台直接创建客户/订单 | `OPERATING_COMPANY_REQUIRED`，事务不落经营数据 |
| 人员无公司子树内启用 Membership 或用户停用 | 客户责任人返回 `PARTNER_INVALID_ARGUMENT`；订单人员返回 `ORDER_PERSONNEL_USER_INVALID` |
| 客户缺销售/操作/客服任一岗位 | `PARTNER_COMMISSION_ASSIGNMENT_MISSING`，中文列出缺失岗位；创建/草稿更换客户回滚 |
| 迁移前总部仍持有客户或订单 | SQL 抛异常，禁止静默改挂至其他公司 |

## 5. Good / Base / Bad

- Good：张三挂靠深圳公司的业务部，深圳客户配置张三为销售；分配记录归属深圳公司，深圳订单提成快照命中张三。
- Base：总部管理员可通过明确目标公司且具有权限的既有跨组织路径管理公司数据，但不会出现总部归属的订单。
- Bad：把部门 Membership 的组织 ID 或用户随手选的另一公司写入责任分配；严格等值快照会漏人，导致提成丢失。

## 6. 必需测试

- 公司子树 Membership（含部门）可保存，子树外/停用拒绝；多部门同用户的候选总数与分页不重复。
- 创建订单三岗位齐备时快照含对应人员；缺岗位时创建和草稿更换客户均报错且订单/号码/快照回滚。
- 总部直接写入失败；迁移在开发库执行两次，查询两张人员表与所属根对象的组织不一致行数均为 0，并核对总部客户/订单数为 0。
- 前端表单只下发用户和岗位；生成的 OpenAPI/客户端与 Proto 同步。

## 7. 错误与正确示例

错误：`SetOrganizationID(memberOrganizationID)`，或快照零命中时 `return nil`。

正确：验证员工在 `companyID` 子树中存在启用 Membership 后 `SetOrganizationID(companyID)`；快照缺岗位返回业务错误并让事务回滚。
