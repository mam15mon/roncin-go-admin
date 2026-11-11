# 六项修复执行计划

## 状态

- [x] 六项静态交叉复核完成，修复前记录提交 `5c4bff3a`。
- [x] 用户要求修复并授权独立 review、合并 main、安全删本次分支、push，之后继续审计。
- [x] 收敛具体设计、文件责任、验收范围和真实库隔离方式。
- [x] 非实现者独立预审三份规划，未发现必要阻断；权限入口、版本更新、普通保存与 Git 交付边界完整。
- [x] 用户于 2026-11-11 回复“是的”，批准最终方案及全部实施/交付流程。

## 执行顺序

1. 检查 Git 状态与远端，保留海运排版目录。新建 `codex/system-audit-round2-fixes` 工作分支；如其他并行改动出现，使用独立工作树。记录基线，不重置他人内容。
2. 配置上下文并分发三个 `trellis-implement`：A 组 B1/B2 客商前后端及专属测试；B 组 B3/B4 财务服务端及专属测试；C 组 B5/B6 订单前后端、表单模板及专属测试。每次分发以当前任务路径开头，注明文件所有权、不得回退他人改动。
C 组实现及审阅前须直接分段读取 `.trellis/spec/web/frontend/state-management.md` 的保存基线、未保存输入、轮询/外部变动相关章节；该文件超过上下文自动注入大小，不依赖被截断的全文注入。

3. 先实现，再补针对性回归。各组完成局部验证、`git diff --check` 后交主协调者分组提交；不把产品改动与未执行的验证混称完成。
4. 派未实现对应改动的 `trellis-check` 交叉 review，重点检查 design.md 风险及六项验收表。修复 review 阻断项，按改动重跑相关测试。
5. 主协调者同步必要规范、修复状态及验收证据。最终执行全栈门禁，确认所有必要集成用例实际 PASS。
6. fetch 最新 origin/main，必要时合并远端变化并复核受影响路径；合并修复分支回 main，核对 `merge-base --is-ancestor` 后用 `git branch -d` 删除本次已合并分支。若存在本次远端临时分支且确认可删才删除，不触碰其他分支。
7. 普通 push main，核对本地 HEAD 与 origin/main SHA 一致。失败如实记录并继续解决，不强推。
8. 回到父任务继续只读找问题，不运行前端测试，不修改业务数据；新问题按触发、后果、锚点、反证和独立复核入清单。用户明确“结束”前不归档父审计。

## 最小验证与验收门禁

| 组 | 局部验证 | 必须覆盖 |
| --- | --- | --- |
| A | Go biz/service/data 定向测试；客商页面/结算面板 Vitest；改动文件 Biome、web tsc | 聚合七入口读裁剪；Create/Update/Import 独立写权限及原子性；仅主档权限的正常保存；空规则/读取失败分离；两条切默认入口与旧表单冲突 |
| B | Go biz/data 定向测试；正式迁移 PostgreSQL 提成与对冲夹具 | 100+100−150 再取消增提被拒；订单行/父单两层余额；安全取消/系统来源/旧版本；500 已清+2 未清及真正 500/501 未清，撤销分摊重新入选 |
| C | 订单详情及 OrderFormTemplate Vitest；订单 biz/data 定向测试；正式迁移 PostgreSQL 并发 | 首签→同页改单；签发改回草稿；失败/迟到/等待期间输入与 dirty；两种实际箱竞态、正常 FCL、无箱切换、共享箱及删除边界 |

建议前端新增测试拆为 `detail-air-house-save.test.tsx`、`detail-air-house-save-race.test.tsx`，模板邻接补入 `OrderFormTemplate.unsaved.test.tsx`；重型用例单文件不超过十个，超过 20 秒按功能拆分。

具体执行命令采用实际新增测试名与文件，不用会触发全量的伪定向入口：

```bash
pnpm --dir web exec vitest run <本组受影响测试文件>
pnpm --dir web exec biome check <本组改动文件>
pnpm --dir web tsc
go -C server test -p 32 ./internal/biz ./internal/service ./internal/data -run '<本组测试名>' -count=1
git diff --check
pnpm run check:fast
```

B6 建议新测试名 `TestOrderContainerShipmentConcurrencyPostgres`，同时覆盖既有 `TestOrderMasterBillDownstreamFactsPostgres` 与 `TestOrderCreateTransactionPostgres`。数据库连接凭据只经环境变量注入；优先复用可用专用测试库，否则创建一次性测试环境，禁止读取并打印秘密或对业务库清空/重建。环境不能提供真实验收时明确阻断合并，不把 SKIP 当 PASS。

全栈门禁的风险依据为独立权限、多个仓储事务、跨层接口内部签名及公共表单状态同时变化。无构建/依赖/生产入口变更，不固定执行 build。

## 交付记录

实施时在 `research/fix-verification.md` 逐项记录实现提交、定向验证、真实库结果、独立 review、最终门禁、合并与远端 SHA。原 cross-review.md 保留审计时点的事实，不将历史“未运行”改写成新验收记录。

## 本轮验收进度

- [x] 六项已分组实现、定向回归、独立 review、规范同步。
- [x] 全量前端门禁与带隔离库后端测试通过；新 Go 标准库漏洞升级至 1.26.11 后，后端门禁组成项、关键真实库回归、构建与漏洞扫描均通过。
- [ ] 合并 main、安全删除本次分支、push 核对远端，随后继续父审计。
