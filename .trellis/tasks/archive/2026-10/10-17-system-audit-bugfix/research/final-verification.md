# 最终验收记录

## 交付范围

本任务修复系统体检确认的三类缺陷及同根因路径：写入失败误报成功、认证工作区缓存与迟到响应串身份、财务台账导出语义缺失；同时补齐真实 PostgreSQL CI 门禁及启用后暴露的失效集成测试夹具。未修改生成接口或产品授权、经济投影和提成计算规则。

## 验证方法与证据

- R1：扫描 214 处直接非 GET 生成客户端调用，逐项按真实 DTO 守卫成功分支；共享 CRUD、ModalForm、React Query mutation、版本动作、带原因确认和异步气泡确认同步传播失败。真实请求层代表回归覆盖重置密码、角色/客商角色、账单与对冲；独立检查再补齐本地 catch 与前置校验的 false 传播。详见 `write-call-audit.md` 和 `independent-review.md`。
- R2：真实 Provider 与请求适配器验证用户/公司切换、A→B→A、匿名、StrictMode、旧响应迟到成功/失败/401；回归公开钉钉预热和换码。浏览器真实个人提成抽屉切换公司时旧记录在新响应前已消失。
- R3：六类台账逐列核对真实 DTO，验证币种、两位金额、有效小数、中文状态、本地时间、公式及特殊字符、空列表和下载资源释放。真实 ProTable 验证只从 onLoad 采纳数据更新导出，迟到与失败结果不覆盖当前导出。范围为当前页或已选中行，未增加全量接口或循环翻页。
- 浏览器使用真实组件、统一请求层和真实领域映射，只拦截网络；重置密码 409 后输入不丢、不关窗、不刷新，同窗成功重试正常。未向开发库提交密码或财务写入。浏览器结果见 `browser-verification.md`。
- 后端使用本次独立数据库和独立角色，连接仅保留在 `/tmp` 的 0600 私有状态文件。显式绑定集成连接、`DATABASE_SOURCE`、迁移开关和真实时间后台任务开关，执行 `go -C server test -p 32 -timeout 25m -count=1 -json ./...`。没有用 `Postgres$` 筛选冒充整包验证。
- CI 同步执行完整真实库测试并检查带 `Test` 的 PASS/FAIL/SKIP；无测试文件的生成包不计为测试跳过。数据库名遵循实时验收要求的临时库前缀，公开 Schema 与各随机 Schema 使用正式 SQL 迁移。

## 根因与防复发

| 根因 | 本次落地措施 |
| --- | --- |
| 把 Promise resolve 当作业务成功，忽略统一错误处理返回 undefined | 最薄写响应守卫；失败不执行成功副作用；真实请求与组件联合回归 |
| 假设确认组件与 ModalForm 对失败的处理相同 | 读取安装版本实现；共享 AsyncPopconfirm 保留官方 loading 并拦截失败关闭；catch/前置校验显式返回 false；原生 hook Modal 消费确认结果启用 await 模式 |
| 认证身份与 QueryClient 生命周期不一致 | 工作区按用户及公司 keyed 创建独立业务客户端；layout 清理旧缓存与请求响应作用域；公开入口独立 |
| 从可见列或原始 dataIndex 猜测财务导出 | 六类显式类型化业务列；公共 CSV 序列化和下载；官方表格采纳后保存导出快照 |
| 普通 Go 测试 SKIP 被当作真实库通过，夹具落后于当前契约 | 完整真实库门禁统计；正式迁移构造准确前态；补齐真实清偿来源与守恒数据，保留业务断言 |

规范同步至前端状态管理、能力导航及后端质量规范，防止同类调用继续复制旧写法。

## 最终结果

### 完整真实 PostgreSQL

- 完整 `./...`：2168 个带 Test 的运行事件全部 PASS，0 FAIL、0 SKIP；21 个含测试包全部 PASS。134 个无测试文件包的包级 skip 不代表跳过测试。
- 主要包：`internal/data` 947、`internal/platform/migration` 39、`internal/server` 106、`internal/biz` 792、`internal/service` 163，均为 PASS 事件数（含子用例）。后台任务真实时间验收、bootstrap-admin 和 sync-dev 均实际通过。
- data 包耗时 724.786 秒，无 panic、超时或遗漏运行事件。原始 JSON 日志 `/tmp/roncin-audit-postgres-final.jsonl`，无凭据汇总 `/tmp/roncin-audit-postgres-final-summary.json`。
- 测试结束后以创建记录及数据库所有者校验，只删除本次拥有的独立数据库和角色；两项残留查询均为 0，0600 凭据状态文件已删除。未重置或清理开发库。

### 前端门禁与构建

- 最终 `pnpm run check:fast` 退出码 0，前端与后端全门禁通过，总耗时 65.6 秒。日志 `/tmp/roncin-audit-final-gate.log`；没有通过忽略未处理拒绝、关闭规则或屏蔽类型检查获得通过。
- 前端 251 个测试文件、1355 个用例 PASS，无 FAIL、无 Vitest 未处理错误；1 个既有跳过文件、12 个既有跳过用例来自 `web/src/pages/orders/templates/sea-template.test.tsx` 的重型海运表单套件，本次没有新增 SKIP。Biome、类型、架构扫描和后端静态/漏洞检查均进入最终门禁并通过。
- 真实 Modal、AsyncPopconfirm、请求层和业务组件分别验证失败及成功重试；全量测试中保留相邻测试输出的 localhost:3000 连接控制台信息，没有将该输出藏匿或将其误记为 Vitest 未处理拒绝。
- 完整生产构建已通过（前端 Vite 与服务端二进制），日志 `/tmp/roncin-audit-final-build.log`；本地没有配置 Sentry 上传令牌，按既有构建流程跳过 SourceMap 上传。
- `git diff --check` 通过；生成客户端、Protobuf 及权限生成文件没有手工修改。

### 主线交付

- 工作提交依次为 `30bad387`（工作区隔离）、`10a4f76f`（真实库门禁与迁移前态）、
  `844be35e`（提成清偿夹具与实时验收）、`81b5dabb`（写反馈及台账导出）、
  `6df676dd`（完整验收记录）；均已由任务分支快进合并到本地 `main`。
- 切换前工作树干净，`git merge --ff-only codex/system-audit-bugfix` 成功；
  `git merge-base --is-ancestor codex/system-audit-bugfix main` 通过后，
  `git branch -d codex/system-audit-bugfix` 成功删除本次分支。随后复核分支不存在，
  `6df676dd` 为 `main` 祖先，工作树干净。
- 本任务按 Trellis 归档至当前目录；其他在办任务不属于本次归档范围。
