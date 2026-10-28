# 实施计划

状态：规划完成，等待用户批准最终方案；未运行 task.py start，未修改产品代码。

## 顺序
1. 用户批准后激活任务；按 trellis-implement 分配服务端契约/事务与前端表单职责，交代共享工作区与文件所有权；最终由 trellis-check 检查。
2. 服务端：更新普通整单 HBL 编辑校验；复用 data 内单证写入规则；保持固定锁序、共享成员门禁、版本推进、幂等和审计原子性。重点审阅同包 MBL 多种修改的版本比较顺序。
3. 契约生成：更新 proto 注释与相关说明，按 `make -C server api` → `pnpm run generate:web-client` 检查源与生成物差异，不手改生成物。
4. 前端：详情统一取得单证基线；接入现有整页载荷与提交；移除两个独立保存按钮、处理器及提示；完成冻结编辑、成功刷新、失败草稿和未知结果重试语义。
5. 功能后补测试（禁止 TDD）：覆盖 PRD A1–A7；同步上一轮独立保存测试到统一入口，保持真实表单交互。
6. 更新海运契约、共享航程/表单保存规范中的“整单禁止 HBL”及独立保存描述。相关测试与 diff/type 检查通过后按组提交。
7. 最终执行全栈门禁；记录真实库回滚验证、生成物、UI 浏览器验收与剩余限制，然后归档任务和记录日志。

## 预计影响文件
- `server/api/order/v1/sea_document.proto`：更新输入语义注释及生成契约。
- `server/internal/biz/sea_document.go`、相关领域测试：区分创建、普通更新和专用模式切换。
- `server/internal/data/order_write_draft.go`、`order_write_sea_document.go`、`order_write_sea_master_bill.go`、`sea_document.go`：事务内目标身份、版本、共享门禁与可复用写入规则；按实际修改收敛。
- `server/internal/service/order_convert.go`：确认现有 ID/version 透传完整，仅必要时调整。
- `web/src/pages/orders/detail.tsx`、详情数据 Hook、海运 form-adapter/types 与模板 props：完整基线与统一保存编排。
- `web/src/pages/orders/templates/components/sea/SeaDocumentSection.tsx`、`SeaTransportSection.tsx`：移除分散保存、统一字段提示及状态。
- `web/src/components/ui/order-template/OrderFormTemplate.tsx` 仅在现有提交/草稿生命周期无法支持上述需求时作最小修改，并增加公共模板相邻回归。
- 相关生成物、定向测试、`.trellis/spec/server/backend/sea-export-document-contract.md`、前端海运规范。

## 验证
- 前端：适配器单测、整页真实表单一次提交、失败保留、连续保存、草稿恢复、只读/DIRECT、离开/切换身份；每个重型 suite 不超过 10 项或 20 秒。
- 后端：领域参数校验、服务转换和 PostgreSQL 集成事务测试；每种冲突均断言订单/MBL/Link/HBL/审计整体未改变，验证幂等重放及共享锁顺序。
- 集成测试仅用 `RONCIN_INTEGRATION_DATABASE_SOURCE` 指定实例的隔离 Schema 和正式迁移，不使用开发库兜底、不并发 Schema.Create、不删除用户数据。
- 定向命令按实际测试文件运行 `pnpm --dir web exec vitest run ...`、`go -C server test -p 32 ./internal/biz ./internal/service ./internal/data`；集成用例须确认未因缺少环境而跳过。
- 提交前：修改文件 Biome、`pnpm --dir web tsc`、`git diff --check`。
- 最终：`pnpm run check:fast`，浏览器验证新建与详情一次保存及失败定位。无构建配置修改时不固定执行 build。

## 回滚检查点
- 不单独交付取消按钮而后端尚未支持 HBL 的中间状态。
- 不将上次有缺陷的仅回填 HBL 响应当作版本同步范本：独立 HBL 更新还会推进 Link。
- 开始前保持用户工作区改动；当前规划基于干净 main，规划提交不代表实施批准。
