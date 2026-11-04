# 现状与影响定位

## 公司维护链路

- `server/api/admin/v1/admin.proto`：CreateOrganizationRequest、UpdateOrganizationRequest、AdminOrganization 追加可选 name_en；原字段号不变。
- `server/internal/biz/admin_organization.go`：AdminOrganization 和新建/编辑校验；UpdateOrganization 的全部仓库内调用方一次性同步。
- `server/internal/data/admin_organization.go`：沿用已有 WithTx、公司范围和原子审计，不新增事务边界；organizationToBiz 等映射需完整传递。
- `server/internal/data/ent/schema/organization.go`：可空 name_en；现有关键词 Hook 仅包含 name，新英文名同步进入可搜索键。
- `server/internal/data/organization_seed.go`：默认 BJ/CD 独立公司；仅新建北京写入已确认英文名，已存在的公司不被种子覆盖。
- `web/src/pages/admin/components/org/OrgCreateModal.tsx`、`OrgEditModal.tsx`、`OrgDetailCard.tsx`、`types.ts`：现有公司表单及详情扩展。
- `web/src/pages/admin/organization-tree.ts`：全量组织列表的本地匹配可加入 nameEn，不引入新请求。

## HBL 链路

- `biz/sea_house_bill_print.go` 当前模板由 SELF_ORGANIZATION + BJ 决定，改为自签公司模板，不再靠 code 白名单。
- `data/sea_document.go` 的聚合读取与 getSeaHouseBillByID 两条路径都读取真实签发组织；新增英文名应沿此查询投影，不另调管理端接口。
- `api/order/v1/sea_document.proto` 的当前 SeaHouseBill 返回只读 issuer_organization_name_en；此字段不是 PrintDetails 输入，也不进入正文写入/历史内容哈希。
- `web/src/pages/orders/hbl/model.ts` 的不可用原因、资料指纹，`HblEditor.tsx` 的输出检查，以及 pdf/types/content/document 共同消费签发英文名。
- PDF 中 COMPANY 常量用于页眉、作者和多处签发/固定条款，必须全部移除运行时固定北京引用，只有正式种子、迁移和测试可保留已确认北京名字。

## 验证资料

- 上一任务已归档在 `.trellis/tasks/archive/2026-11/11-04-sea-export-hbl-production/`，其四版本、审核独立性、FlowTable 极值分页和字体授权继续成立。
- 已知正式英文名：`RONCIN (BEIJING) SUPPLY CHAIN MANAGEMENT CO., LTD`，来自用户参考 PDF；成都名不能根据中文替换城市来推测。
- 工作区原有未跟踪任务 `.trellis/tasks/10-30-sea-order-page-layout/`，本次不编辑、不提交。
