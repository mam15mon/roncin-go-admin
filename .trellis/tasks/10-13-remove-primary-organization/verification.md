# 取消主要组织：审查与验证记录

日期：2026-10-13。范围为本子任务；后续批量成员、公司管理员补齐和二维码登录尚不在本记录的完成结论内。

## 审查结论

本子任务代码可提交。后端、前端与正式迁移中的主要组织配置已移除，未发现尚未处理的本子任务阻塞问题。

- `UpdateUserRequest`、领域用例、仓储接口与服务转换均已删除 `role_ids`；资料更新仓储不再校验或替换成员角色。新建用户仍可分配明确初始组织的角色；已有用户通过明确 `membership_id` 编辑角色。
- `Membership.primary`、凭证 `PrimaryOrganizationID`、默认候选标记及自动归一逻辑已删除。Proto 被删除字段的编号、名称均保留为 `reserved`，Ent/API/OpenAPI/Web 生成物同步。
- 密码与钉钉登录均按组织名称、ID 的稳定顺序取得首项建立初始会话；候选由有效成员关系映射、去重，部门/组归入公司。principal 的组织集合使用相同排序，角色仍限定当前工作台。
- 前端多个工作台仍展示选择页，卡片不标记或突出默认组织。是否直接进入以本次会话的组织 ID 判断；其他组织沿用会话轮转。
- 新迁移 `20261013100000_remove_membership_primary.sql` 仅删除 `memberships.primary`，没有删除用户、组织、成员关系、角色或业务数据。旧基线仍保留原字段，新增迁移与当前 Ent 差异一致；历史迁移未改写。
- bootstrap、开发种子和既有测试调用方同步删除废弃字段。既有测试仅同步接口、夹具、断言及文案，未新增应用测试。

## 审查发现及修复

发现密码登录页与钉钉回调页同步展示组织候选，但用 `startTransition` 延后更新全局 `initialState.currentUser`。若选择器读取旧的全局组织 ID，可能把另一次会话的组织误认成本次当前组织并跳过轮转。

已由实现代理修复：两个页面均将本次认证响应中的 `{ options, currentOrganizationId }` 同步存入一份本地选择状态，选择器直接消费该状态。审查复核了实际代码路径；不依赖全局过渡完成时间。

## 实际执行的检查

以下检查全部退出码为 0：

- `go -C server build -p 32 ./...`：全服务端编译。
- `go -C server vet -p 32 ./internal/biz ./internal/service ./internal/data ./cmd/bootstrap-admin ./cmd/sync-dev`：受影响后端包静态检查。
- `pnpm --dir web lint`：Biome 全量 lint 与 `tsc --noEmit` 均通过。输出 7 条既有信息级建议，无错误或警告；其中用户列表测试的 2 条冗余 Fragment 建议属于既有代码，未为本任务做无关改写。
- `pnpm run check:architecture`：产品边界扫描通过，扫描 431 个产品文件、1592 条本地依赖及 4 条 feature 能力依赖边。
- `pnpm run check:layers:go`：Go 分层扫描通过，检查 1227 个 internal 文件、11570 条 import，违规 0 处。
- `git diff --check`：通过。

实现代理另已记录定向 Biome 通过、6 份 Ant Design lint 均为 0 问题，以及生成命令执行成功；本次复核未重复生成。

## 检查器脚本误执行记录

本轮约束为不新增或运行应用测试。审查代理运行架构门禁时未提前检查脚本内容，实际连带执行了前端架构检查器自身的 18 个 Node 测试，以及 Go 分层检查器自身的 Go 测试，均通过。这是本次检查过程的误执行，不能作为应用业务回归通过的证据。

没有运行服务端业务单元/集成测试、Vitest 或 Playwright。后续仅运行原始扫描入口 `node web/scripts/check-architecture.mjs` 与 `go -C scripts/layer-check-go run .`，避免门禁包装脚本连带执行测试。

## 验证限制与后续事项

- 没有连接或修改实际业务数据库，没有执行迁移、清库或重置数据。空库迁移完整链与旧库升级的数据保留验收未执行，不能把静态审阅记作实际数据库通过。
- 没有运行应用业务测试、真实登录/扫码或浏览器交互验收；Go vet 和 TypeScript 通过仅证明静态检查及既有测试文件的类型链可用。
- `UserMembershipModal` 的既有写请求成功判定不足：请求被统一错误处理消费为 `undefined` 时仍可能成功关闭。已交下一批量成员子任务处理，需同时覆盖新增与编辑分支；本子任务未扩大该流程修改。
- 已同步认证组织选择规范和既有钉钉任务中受替代的局部约定。bootstrap 自动加入公司仍由后续公司管理员子任务实现。
