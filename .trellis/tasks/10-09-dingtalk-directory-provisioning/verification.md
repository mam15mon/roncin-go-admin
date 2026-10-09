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

## 已完成的本地验证

- Proto、配置、Ent、Wire、OpenAPI、前端客户端、权限键均由生成器更新；最终 `go -C server generate ./...` 前后 SHA 指纹一致。
- `buf lint`、权限/枚举一致性、架构边界、修改文件 Biome、前端 tsc、`git diff --check` 通过。
- 前端全量：181 个文件、1068 个用例 PASS；1 个文件、12 个既有重型订单表单用例 SKIP。跳过项不计为通过。
- biz/service/server/access 全包测试通过；实际 HTTP 路由验证 latest 与 task_id 分发，不能只依据生成文本。
- 真实 PostgreSQL 定向 13 个用例 PASS，无 SKIP；另追加旧/新 lease 交错用例 PASS。覆盖无组织待分配、身份冲突/重复同步/资料授权、同企业任务唯一、过期活动释放、并发开通、9 成功 1 失败、组织/角色停用整人回滚、人工返聘保留 ID、只读资料、来源确证六类迁移与不明候选原子回滚。
- 多级部门遍历/局部 API 失败、独立授权成员补部门名称、分页、字段缺失 vs 明确空值、无范围不查询根部门、跨部门去重均有网关/biz 夹具验证。
- 开发模式与生产构建后浏览器接口夹具验收 PASS：首次同步、同步完成/字段范围、批量部分成功后只重试失败者、公司工作台隔离、未同步/待分配/停用提示；无浏览器脚本异常。浏览器未连接真实钉钉扫码，不将夹具标成实扫验收。
- `pnpm run build` 通过；Sentry 未配置上传凭据，按既有构建行为跳过 Sourcemap 上传；既有大包提示不影响产物生成。
- 首次全栈检查发现删除注册路由后的固定路由数量断言、工作台查询 SQL mock 和用户列变更的旧夹具；同步修正而未删业务断言。前端再次全量通过；修复后 `pnpm run check:server` 完整通过（真实隔离库与迁移开关均注入），包括 data 包 425.044 秒、migration 包 94.449 秒、Go vet 和漏洞扫描。

## 本地开发库迁移

- 只读预检旧占位候选 0、缺注册审计候选 0；没有清空或重置开发数据。
- 严格迁移首次因本地库缺少既有 `background_tasks_kind_check` 失败并完整回滚，users 的 pending 列未残留。
- 只读确认后台任务表为空后，按正式历史迁移 `20260904120000` 的原约束定义补回本地缺失约束，再执行严格迁移；未在迁移代码添加自动容错。
- 正式迁移成功，新权限与角色依赖同步成功；实际确认 pending 列、用户 CHECK 和目录同步权限均已存在，健康检查返回 200。

## 验收限制与后续

- 真实全员导入、真实 OAuth 扫码仍需上述钉钉通讯录接口权限开通后的联调；任务保持待联调，不归档为全部验收完成。
- 执行超时取消及授权撤销/组织停用的真实竞争时序未另外进行故障注入；有代码审阅、lease 交错测试、并发开通与停用目标回滚验证，不宣称全部竞争时序实测。
- 完整门禁结束后隔离 Schema 残留数为 0；已删除本次创建的一次性验证数据库、角色和受限连接文件，停止本次创建的生产预览服务。现有开发服务保留。

## 提交与任务状态

实施提交：`0fe1fa99 feat: 同步钉钉通讯录并统一组织角色开通`。本地代码验收完成，任务因真实钉钉外部权限与实扫联调尚未满足而保留，不误归档为全部 AC 完成。未推送、合并或发布本分支。
