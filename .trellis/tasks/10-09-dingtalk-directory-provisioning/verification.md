# 钉钉人员同步与开通验收记录

## 实施范围

用户于 2026-10-09 明确批准实施 D1/D2/D3。分支为 `codex/dingtalk-directory-provisioning`。开发库未重置，真实人员资料未写入仓库。

## 真实钉钉只读核验（2026-10-09）

- 本地配置开启钉钉，应用凭据存在；只在进程内读取，输出不包含凭据、人员标识或资料。
- 应用 accessToken 与 `GET /auth/scopes` 均成功（HTTP 200、errcode 0）。
- 实际范围使用 `auth_org_scopes.authed_dept` / `authed_user`；根部门授权，全企业范围；未配置独立成员范围。
- 实际 `auth_user_field` 包含姓名、头像、部门、工号及部分管理字段；未包含 mobile/email。全企业范围声明不能证明各成员读取接口已获授权。
- `/topapi/v2/department/get` 与 `/department/listsub` 被拒绝：errcode 88、subcode 60011，缺少 `qyapi_get_department_list`。
- `/topapi/v2/user/list` 被拒绝：errcode 88，缺少 `qyapi_get_department_member`。
- **真实全员同步尚不能验收**：钉钉应用管理员需开通部门列表读取和部门成员读取权限，生效后重新核验。无手机号/邮箱权限不阻塞基础人员同步。
- 本次只读核验未执行人员导入、账号开通或任何钉钉写操作。

当前接口文档：[获取通讯录权限范围](https://open.dingtalk.com/document/orgapp-server/obtain-corpsecret-authorization-scope)、[获取部门用户详情](https://open.dingtalk.com/document/orgapp-server/queries-the-complete-information-of-a-department-user)。字段与权限申请口径参考 [阿里云钉钉用户同步官方说明](https://help.aliyun.com/zh/qwenwork/enterprise-ultimate-users-user-sync-dingtalk)。

## 本地验证

使用单独创建的一次性 PostgreSQL 数据库与随机测试 Schema，走正式 SQL 迁移；连接串只存放在权限受限的仓库外临时文件。实际 PASS/FAIL/SKIP 与清理结果在验证完成后补充。
