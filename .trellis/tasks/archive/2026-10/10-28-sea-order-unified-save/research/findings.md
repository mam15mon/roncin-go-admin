# 代码研究证据

- `web/src/pages/orders/detail.tsx:414`：当前成功由 UpdateOrder 决定，loadData 为后台刷新；新统一保存需要明确单证刷新与基线生命周期。
- `web/src/pages/orders/order-kinds/sea-export/form-adapter.ts:515`：更新仅透传 values.seaDocument，独立表单 seaMasterBillContent/seaHouseBill 未自动组包。
- `web/src/pages/orders/templates/components/sea/SeaDocumentSection.tsx:200`：当前组件自行加载并回填；两个保存按钮各调用专用写端点。
- `web/src/components/ui/order-template/OrderFormTemplate.tsx:173`：onFinish 非 false 即清草稿与脏状态，必须与失败/已保存待重读语义一起审阅。
- `server/api/order/v1/sea_document.proto:225`、`:259`：现有输入已有 HBL ID/expected_version、expected_link_version、expected_mbl_version，无需为普通统一保存新增版本字段。
- `server/internal/biz/sea_document.go:214` 与 `server/internal/data/order_write_sea_document.go:155`：两层均拒绝整单 HBL 更新，必须同步契约与实现。
- `server/internal/data/order_write_draft.go:181`：统一事务与幂等、共享 MBL 前置门禁已存在；调用顺序为主单身份/航程同步、单证同步、订单写入。
- `server/internal/data/sea_document.go:221`：HBL 独立更新固定 Order→MBL→Link→HBL 锁序，校验组织、目标、版本、终态、签发主体与重号；`:351` 附近同时递增 Link。普通更新并非每次创建不可变历史，必须保留既有生命周期语义。
- `.trellis/spec/server/backend/sea-export-document-contract.md:28`：目前规范明确整单不得更新分单，需要与本次需求一起修订。

本轮只读研究与规划，不修改任何产品代码。前一轮全量前端门禁曾有未改动用例偶发失败，定向复跑通过；实施验收不得直接继承旧结果。
