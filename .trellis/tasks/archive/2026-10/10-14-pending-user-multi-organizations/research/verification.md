# 验收证据

## 产品与根因

待分配的单组织表单、单人/批量契约与单目标仓储是完整根因链路，见 root-cause.md。现已统一为 items，每项独立组织和公司角色；两个前端入口共用 OrganizationRolesField。首次/人工重新开通整体至少一个角色，各组织可空；已开通新增成员仍可全部空角色。

## 需求映射

| 需求 | 实际证据 | 结果 |
| --- | --- | --- |
| R1 多组织开通 | 单人/批量 Proto items；真实 HTTP 测试正确解码两组织；真实 PostgreSQL 多公司/同公司两部门 | PASS |
| R2 逐组织角色 | 真实 ModalForm 测试候选隔离、删除重选清空/其他行保留、部分空角色及全空拒绝后改正；PostgreSQL 验证实际角色 ID | PASS |
| R3 每人原子提交 | PostgreSQL 第二成员、第二角色、通知任务、通知投递及审计错误注入均零残留；返聘失败保留旧关系/角色，成功明确恢复所选 | PASS |
| R4 人员独立结果 | 真实 PostgreSQL 和前端部分成功仅重试失败者、保留组织输入；实际 HTTP 证明失败 false 被省略，前端接受带 reason 的合法失败；未知结果禁止盲重试 | PASS |
| R5 权限与并发 | 真实 PostgreSQL 跨公司同名角色/系统身份/停用祖先或角色/陈旧 Principal 撤权拒绝；并发一成功一冲突 | PASS |
| 相邻已开通路径 | UserMembershipModal 真实新增全空角色、保存失败保留/重选清空与编辑失败重试；真实系统公司边界和员工生命周期 | PASS |

## 实际运行记录

- 前端定向：两个修改组件测试文件，10 例 PASS，联合约 13.9 秒，单文件均不超过 20 秒，无 stderr。本次五文件 Biome、三个产品组件 antd lint、tsc 均 PASS。
- biz/service/data 最终定向测试和三包 vet PASS。新增 service 验证单人 items 真实透传、200 人合法、201 个原始重复人员仍拒绝；items 200/201、非零 UUID、重复组织及空项边界覆盖。
- 根会话新增 TestDingTalkMultiOrganizationAuthorizationHTTPContract，以实际 Kratos HTTP 注册、RequiredFieldsValidator 和 encodeResponse 验证单人/批量 items、空角色项和省略 false 的逐人失败响应，PASS。
- 显式专用 PostgreSQL -v 日志：/tmp/roncin-pending-organizations-pg-targets.log。以下 5 个顶层用例实际 PASS，无 SKIP，约 10.9 秒：
  - TestDingTalkMultiOrganizationAssignmentPostgres
  - TestDingTalkDirectoryAssignmentPostgresConcurrencyAndPartialSuccess
  - TestAdminEmployeeLifecyclePostgres
  - TestSystemCompanyAdministratorAuthorizationBoundaries
  - TestSystemDirectoryAssignmentCannotGrantSystemIdentity
- 相邻系统/公司测试原夹具重复创建 initializeCompanyAdministrator 已生成的 system.user.update，导致唯一键冲突；改为查询正式初始化权限，重跑实际 PostgreSQL PASS。没有生成历史回退。
- make -C server api、pnpm run generate:web-client PASS，二次生成 SHA256 幂等 PASS；最终差异只涉及开通 payload 的必要生成物。
- 全量 pnpm run check:fast 已返回 exit 0，日志 /tmp/roncin-pending-organizations-check-fast.log，实际 69.9 秒（WEB 66.5 秒 / SERVER 69.9 秒）。默认 Go 全包测试、vet、漏洞扫描、Proto lint、前后端架构与重复扫描检查、前端 lint/tsc/test 均通过。
- 全量前端：228 文件、1228 用例 PASS，另有 1 文件、12 用例 SKIP，不计为通过。默认 Go 未注入真实库的 SKIP 同样不充当集成验证，真实库证据单独列于上项。
- 完整门禁后补充的服务端边界测试与相邻权限夹具调整已再次通过最终定向测试/vet，产品实现未再更改。
- CI YAML 已解析并断言现有 PostgreSQL 服务、专用环境变量、两个开通用例正则绑定；新增步骤将实际执行新增多组织及原并发/部分成功事务测试。
- 独立 trellis-check 审查无本任务内产品缺陷、未作额外编辑。范围外 7 个 Biome info 与 localhost:3000 测试请求噪音未造成门禁失败，未扩大修改范围。
- git diff --check PASS。

## 测试资源清理

本任务唯一命名临时 PostgreSQL 数据库/角色中，隔离测试 Schema 残留为 0；确认活动连接 0 后删除本次数据库和角色，查询残留数 0；权限 0600 的临时凭据文件已删除。未重置、删除或清理开发业务数据库。

## 交付状态

- 产品修复、规范同步、生成、定向与全量门禁、独立审查及测试资源清理均完成。
- 工作提交：96b75125832b69917bc3c9243ac8f91cffc864a7（fix: 统一待分配人员的多组织开通与角色选择），35 文件，按本任务路径显式暂存；未重写或覆盖已有登录页提交。
- 已 git switch main + git merge --ff-only codex/pending-user-multi-organizations，本地 main 包含修复与其他会话的既有 f1552321/97f1b8d2 提交；分别以 git merge-base --is-ancestor 验证可达。
- 已 git branch -d codex/pending-user-multi-organizations，分支列表确认不存在，未使用强制删除。
- 最后按 trellis-finish-work 归档本任务并记录会话；其他任务和分支保持其原有状态。
