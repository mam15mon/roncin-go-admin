# 执行计划：经营数据与责任人员归属收敛到公司

## 顺序清单

1. **契约先行**
   - [x] 修改 `server/api/partner/v1/partner.proto`：`PartnerAssignmentInput` 删 `organization_id`；`PartnerAssignmentOption` 删 `organization_id` / `organization_name` / `membership_enabled`。
   - [x] 修改 `server/api/order/v1/order.proto`：订单人员输入与候选删除组织字段。
   - [x] `make -C server api`，检查生成差异；`pnpm run generate:web-client` 更新前端客户端。

2. **服务端保存链路**
   - [x] 增加公司经营工作台门禁；客户创建/导入与订单创建拒绝总部工作台直接落经营数据。
   - [x] `server/internal/service` 责任分配组装不再读取输入组织；候选转换删除组织字段。
   - [x] `server/internal/data/partner.go` `replacePartnerAssignments`：`SetOrganizationID(rootOrganizationID)`；成员关系校验改为公司子树内启用 Membership。
   - [x] `server/internal/data/partner.go` `ListAssignmentOptions`：按 `user_id` 去重。
   - [x] 订单人员候选按唯一用户分页；订单人员保存由后端派生订单公司并按公司子树校验 Membership。

3. **开单拦截**
   - [x] `server/internal/biz` 新增 `ErrPartnerCommissionAssignmentMissing`（中文消息列出缺失岗位）。
   - [x] `server/internal/data/order_write.go` `snapshotOrderCommissionAttributions`：三岗位任一缺失即报错；确认 Create 与草稿更换客户两个调用点错误原样外传、事务回滚。

4. **存量收敛迁移**
   - [x] 新增迁移：断言无总部客户/订单；幂等收敛客户责任人员与订单协作人员组织，不改历史提成快照。

5. **前端表单**
   - [x] `web/src/pages/partners/components/BasicInfoSection.tsx`：移除 9 个组织下拉。
   - [x] `web/src/pages/partners/partner-detail.tsx`：删除 `assignXxxOrg` 状态与提交字段；`organizations` 拉取如无其他消费方一并移除；适配纯人员候选结构。
   - [x] 订单表单、适配器与测试删除人员组织字段。

6. **验证**（按下方命令）

7. **提交拆分建议**
   - 提交 1：契约 + 服务端 + 迁移（`refactor(partner): 责任人员归属组织收敛到公司并拦截缺配开单`，或按 feat/fix 拆分）。
   - 提交 2：前端表单与生成物（随契约同一组审阅，可并入提交 1，保证生成物与源同提交）。

## 验证命令

```bash
# 服务端定向
go -C server vet ./...
go -C server test ./internal/data/... ./internal/service/... ./internal/biz/...

# 迁移（开发库，允许校验和修复）
pnpm run migrate:dev
# 幂等性：连续执行两次，第二次零行变更

# 前端定向
pnpm --dir web exec vitest run <受影响测试文件>
pnpm --dir web exec biome check <改动文件>
pnpm --dir web tsc
```

## 针对性验证场景

- 客户档案保存：人员挂靠部门（子树内 Membership）可保存，`organization_id` 落为公司节点；子树外/停用成员关系拒绝。
- 总部工作台直接创建/导入客户或创建订单被拒绝；公司工作台正常创建。
- 订单协作人员仅挂靠部门时可选择，落库组织为订单公司；候选多 Membership 仍只出现一次且分页 total 正确。
- 开单：三岗位齐全 → 快照生成；缺任一岗位 → 创建订单、草稿更换客户均报中文错误且回滚。
- 候选：同一人多部门成员关系只返回一条。
- 迁移：存量部门归属行收敛为档案组织；重复执行无副作用。
- 历史快照：`order_commission_attributions` 无任何改写。

## 风险与回滚点

- 高风险文件：`server/internal/data/order_write.go`（开单主链路，错误传播必须回归两条路径）、`server/internal/data/partner.go`（档案写入主链路）。
- 回滚：代码按提交回滚；迁移幂等可保留，旧代码兼容派生后的组织值。
- 完成后、`task.py start` 前：确认本清单全部勾选，pr/design/implement 三件套与最新结论一致。
