# 单用户多组织成员加入契约

## 1. 适用范围

适用于给已开通用户增加公司、部门或团队成员资格。待分配钉钉人员仍走专用开通流程；多人开通的逐人事务规则见 dingtalk-registration-approval.md。

## 2. 签名

```text
POST /api/v1/admin/users/{user_id}/memberships:batchCreate
CreateUserMembershipsRequest: user_id + items[{organization_id, role_ids[]}]
成功响应：本次新建的成员关系列表
权限：system.user.update
单项编辑：UpdateUserMembership(user_id, id, role_ids, enabled)
```

```go
NewAdminUsecase(repo AdminRepo, transactor Transactor)
AdminUsecase.CreateUserMemberships(ctx, actorOrganizationID, actorID, userID, items)
```

旧单项 CreateUserMembership 接口已替换；一项新增也使用批量接口。

## 3. 契约

- items 非空，组织不可重复；每项角色选填，全部留空可保存。角色属于目标组织的公司或系统工作台，不复制跨公司角色 ID。
- 每个组织检查操作者管理权限及范围，空角色也不能跳过；停用用户、待分配用户不通过新增成员恢复或开通。
- 目标组织存在启用或停用成员关系均冲突。恢复状态和修改角色必须明确指定已有 membership_id，不覆盖或自动恢复。
- 一批全部成员、角色绑定和审计在同一共享事务中提交。用例显式注入 Transactor，仓储使用 Data 的事务上下文，不提供无事务回退；成功后在普通上下文重读响应。
- 固定锁序为用户主键排序、组织主键排序、成员主键排序、角色主键排序。事务内重建操作者 Principal，复核权限、组织状态及领域提权规则。当前权限解析依赖小规模组织全树，使用全树共享锁固定父链及工作台范围；不得在持有角色锁后再升级已有成员锁。
- 前端多选组织后每组织一行角色；按 organizationId 隔离候选和选择。撤销组织清除该行，重新选回重新选角色。ModalForm.onFinish 返回保存 Promise，验证成功响应后才关闭；失败保留输入，新增和编辑都遵循该规则。
- 已通过管理范围校验的组织错误可附组织名称；越权对象不泄漏名称。数据库错误原样上抛并整体回滚，不吞错继续提交。

## 4. 校验与错误矩阵

| 条件 | 结果 |
| --- | --- |
| 空批次、非法 UUID、重复组织 | ADMIN_INVALID_ARGUMENT，零写入 |
| 待分配人员 | ADMIN_USER_AUTHORIZATION_REQUIRED，专用开通入口 |
| 停用账号 | ADMIN_USER_INACTIVE，不自动恢复 |
| 缺少管理权限或超出范围 | 拒绝，零写入，不泄漏越权对象 |
| 已有启用或停用成员 | 409 ADMIN_USER_MEMBERSHIP_EXISTS，整批撤回 |
| 组织停用、角色无效或不属于目标工作台 | 拒绝整批，并保留表单 |
| 任意成员、绑定或审计写入失败 | 本批全回滚，已有成员及授权不变 |

## 5. 正常、基准与错误场景

- 正常：同一员工选择北京公司及成都部门，北京填角色、成都留空，一次建立两条成员资格；成都没有新增角色权限。
- 基准：一组织一角色或一组织空角色都使用同一批量接口。
- 错误：第二组织停用后仍保留第一组织；停用成员通过新增自动恢复；把北京角色 ID 复制给成都。

## 6. 必要验证

- 事务：后续组织失效、绑定写入失败、审计失败，均证明本批零提交；响应重读在提交后。
- 权限：空角色越权、跨公司同名角色、操作者撤权及组织停用竞争。
- 并发：两次同用户批量请求不重复加入、不覆盖；与成员编辑、钉钉开通的锁序无环。
- UI：角色全空、部分空、撤销重选、异步返回隔离、快速提交、失败保留、编辑失败不关闭。

## 7. 错误与正确示例

```text
错误：前端循环多个单项请求，第二个失败时第一个已保存。
正确：一次 batchCreate，锁内复核并在同一事务写入全部成员及审计。

错误：role_ids=[] → 直接跳过组织管理权限；response=undefined → 提示成功并关闭。
正确：空角色仍核验范围；只有有效成功响应才结束 ModalForm。
```
