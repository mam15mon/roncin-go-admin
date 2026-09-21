# 公司经营归属与订单提成责任人契约

## 1. 适用范围 / 触发条件

修改客户责任人、订单人员、开单、草稿更换客户、提成快照或组织工作台时适用。总部仅承担治理；部门/团队仅承担人员 Membership，不是经营数据或提成归属主体。

## 2. 签名

- `PartnerAssignmentInput { role, user_id }`、`OrderPersonnelAssignmentInput { user_id, role }`、`AssignPersonnelRequest { order_id, user_id, role }`：写契约均不接收 `organization_id`。
- `PartnerAssignmentOption` / `OrderPersonnelOption`：只返回 `user_id, display_name`，分页实体为唯一用户。
- 数据库不变量：`partner_assignments.organization_id = partners.organization_id`；`order_personnels.organization_id = orders.organization_id`；上述经营归属组织的 `kind = company`。
- `snapshotOrderCommissionAttributions(ctx, tx, companyID, orderID, customerID, at)` 在订单创建或草稿更换客户事务中运行。
- `access.IsBusinessOperationPermission(key)` 显式标记经营办理权限；`Principal.CanOperateBusiness()` 校验启用公司工作台；`ResolvePermissionOrganizationScope` 将经营写范围收敛到当前公司。
- `Principal.WorkspaceOrganizationID` 保存真实会话工作台；中间件跨组织只读定位时改变的 `Organization.ID` 不代表用户切换工作台。

## 3. 请求与落库契约

- 所有交互经营办理必须处于启用公司工作台，并具备对应操作权限；目标数据必须属于当前公司。总部保留授权查看与治理配置维护，但不能以显式目标公司、组织树范围或初始化管理员身份直接办理分公司业务。总部人员有切换资格和业务授权时，先切换分公司再办理。
- `PermissionKeys` 与后端权限检查使用相同工作台规则；总部不返回纯经营写权限，原始角色授权仍保留，用于角色管理与切换后的权限重算。公司工作台查看他公司单据时，页面写按钮与响应的业务能力均不得开启。
- 客户/订单人员可仅在公司下的部门或团队拥有启用 Membership，但用户必须启用，成员关系所在组织也必须启用且位于该公司的子树；落库 `organization_id` 一律写经营主体公司，而不是 Membership 所在部门。
- 客户 `CREATOR` 是用例根据当前操作者自动追加的创建事实，不是客户端分配的业务岗位，不要求创建人在公司子树内拥有 Membership；例如初始化管理员合法切换到公司后可创建客户。销售、操作、客服等可编辑责任岗位仍严格校验成员资格，客户端不得指定或改写 `CREATOR`。
- 订单必须使用同公司客户；快照查询继续严格匹配 `partner_assignment.organization_id == order.organization_id`。既有订单提成快照不会因客户责任人变化自动刷新。
- 迁移 `20260917120000_operating_company_personnel_ownership.sql` 先断言总部无客户和订单，再幂等收敛两张人员表的归属；不改历史 `order_commission_attributions`。

## 4. 校验与错误矩阵

| 条件 | 行为 |
|------|------|
| 总部工作台直接创建客户/订单 | `OPERATING_COMPANY_REQUIRED`，事务不落经营数据 |
| 总部以跨组织权限办理公司账单、核销、提成等 | 拒绝经营写入；应切换目标公司工作台，初始化管理员同样受限 |
| 公司 A 工作台修改公司 B 经营单据 | 拒绝，读取范围不能转为写入范围 |
| 业务责任人员无公司子树内启用 Membership 或用户停用 | 客户责任人返回 `PARTNER_INVALID_ARGUMENT`；订单人员返回 `ORDER_PERSONNEL_USER_INVALID` |
| 初始化管理员合法切换公司后创建客户，但无该公司子树 Membership | 自动记录 `CREATOR`，不因业务岗位成员校验拒绝建档；公司工作台与操作权限检查仍生效 |
| 客户缺销售/操作/客服任一岗位 | `PARTNER_COMMISSION_ASSIGNMENT_MISSING`，中文列出缺失岗位；创建/草稿更换客户回滚 |
| 迁移前总部仍持有客户或订单 | SQL 抛异常，禁止静默改挂至其他公司 |

## 5. Good / Base / Bad

- Good：张三挂靠深圳公司的业务部，深圳客户配置张三为销售；分配记录归属深圳公司，深圳订单提成快照命中张三。
- Base：总部管理员可按授权查看分公司订单；要确认分公司账单时先切换该公司。组织、角色与公共配置维护仍按治理权限办理。
- Bad：把部门 Membership 的组织 ID 或用户随手选的另一公司写入责任分配；严格等值快照会漏人，导致提成丢失。

## 6. 必需测试

- 公司子树 Membership（含部门）可保存，子树外/停用拒绝；多部门同用户的候选总数与分页不重复。
- 无公司成员关系的合法操作者可创建最小字段客户并保留创建人；同一人员被显式分配为销售等业务责任人时仍应拒绝并回滚。
- 创建订单三岗位齐备时快照含对应人员；缺岗位时创建和草稿更换客户均报错且订单/号码/快照回滚。
- 总部直接写入失败；迁移在开发库执行两次，查询两张人员表与所属根对象的组织不一致行数均为 0，并核对总部客户/订单数为 0。
- 前端表单只下发用户和岗位；生成的 OpenAPI/客户端与 Proto 同步。
- 总部普通用户/初始化管理员持跨组织业务权限均不能办理；切换公司后仅可写当前公司。批量混合公司拒绝且无部分提交；总部治理维护与只读查看不受影响。

## 7. 错误与正确示例

错误：把跨组织只读定位后的 `Organization.ID` 当作真实办理工作台，或给管理员保留总部办理例外。

正确：权限与能力投影保留会话工作台锚点，经营写入必须同时命中公司身份、动作权限、当前公司数据归属。

错误：`SetOrganizationID(memberOrganizationID)`，或快照零命中时 `return nil`。

正确：验证员工在 `companyID` 子树中存在启用 Membership 后 `SetOrganizationID(companyID)`；快照缺岗位返回业务错误并让事务回滚。
