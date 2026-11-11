# 企业资源覆盖导入权限

## 1. 适用范围

资源备忘的收货人、发货人、通知人等批量导入；按名称/代码匹配已有记录并覆盖时属于编辑行为。

## 2. 签名

`PreviewEnterpriseResourceImport` 返回 `conflictCount`、`overwriteAllowed`；`CommitEnterpriseResourceImport` 接收 `overwriteConflicts`。两接口基础权限为 `business.enterprise_resource.create`，覆盖额外要求 `business.enterprise_resource.update`，均限定当前公司。

## 3. 契约

- create 只依赖 read，不能为了覆盖导入把 update 变成所有新建用户的隐式依赖。
- service 入口复用当前公司权限判定；提交 `overwriteConflicts=true` 时必须额外校验编辑能力，不以客户端预览值作为鉴权依据。
- 预览 `overwriteAllowed` 同时满足匹配无歧义及拥有编辑能力。前端从同一 access 能力控制覆盖按钮和说明；无冲突的新建导入正常开放。
- 拒绝必须在写入前，混合新建/覆盖不能只写入有权限部分；沿用原事务与匹配规则。

## 4. 校验矩阵

| 权限/请求 | 结果 |
| --- | --- |
| create-only，无冲突 | 允许新建 |
| create-only，预览有冲突 | 不允许覆盖，提示缺编辑权限 |
| create-only，直接伪造 overwriteConflicts=true | 权限错误，仓储不执行 |
| create+update，合法单目标冲突 | 可覆盖 |
| update-only 或非公司工作区 | 不能绕过基础新建/工作区权限 |
| 一行匹配多个资源 | 仍按原歧义规则拒绝，有编辑权限也不放行 |

## 5. 正常、基础与错误场景

正常：具备新建和编辑权限的操作者确认覆盖已有收货人。基础：只有新建权限可导入全新记录。错误：以 RPC 只标记 create 为由，无条件透传覆盖参数。

## 6. 必需验证

service 权限矩阵必须覆盖直接调用，断言无权限时仓储零调用；UI 覆盖纯新建、无权限冲突、伪造旧预览能力与歧义说明；数据库导入既有原子性和公司边界相邻用例保留。

## 7. 错误与正确

错误：`CommitImport(..., request.GetOverwriteConflicts())` 前不判断编辑权限。

正确：验证当前公司 create，再在覆盖意图为 true 时验证 update，之后才调用用例与事务写入。
