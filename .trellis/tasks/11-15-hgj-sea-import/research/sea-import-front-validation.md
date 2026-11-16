# 海运进口前端实施与验证

日期：2026-11-16（UTC）。本报告只记录sea_import_front负责的非生成Web范围，Root负责进口服务组件、后端/API生成、最终门禁与真实浏览器。本代理按分工未提交Git，未修改server、生成物、Root的import-handling目录或任务主文档。产品源码写权已于最终测试期间交回Root；其后只收尾测试与本报告。

## 已实现

- registry注册SI/IMPORT；列表、新建、详情、费用与HBL制作归档沿既有公共页面与OrderFormTemplate。SI分单来源、草稿、候选QueryKey、链接和按钮权限均携带当前类型；与SE同号不会查询SE候选。SI共用真实MBL/TE/HBL、实际箱/共享箱、放货单证引用，SI不显示出口拆票/改配及出口步骤手动流转。
- SI继续海运现有共同卸货港/本票最终目的港、不可变单证/航程版本和审核/终止恢复能力；变动检查同样覆盖SI恢复关系。详情与费用成功响应严格要求真实businessType等于路由定义，缺省/0/错类型均显式拒绝，不为旧mock放行。
- Root的ImportHandlingPanel集成到详情进口服务卡。服务保存只loadData、不reset整票；仅确切Order+1且编辑中的MBL正文/身份/TE/Link/HBL版本全部未变化时推进Form.version并确认本人快照。其他更新保留原版本与输入并提示刷新。普通整票载荷不包含独立进口服务，双方互不清空。
- 两海运共用映射支持显式清空经营文本、代理、金额和操作日期；金额/币种半对输入不再静默吞掉，交由表单及后端明确校验。贸易条款清空发送UNSPECIFIED。业务付款方式由可见下拉唯一注册，移除货描区hidden镜像，预付默认和单证PP/CC独立。
- SI外部HBL只在真实外部签发主体且服务端投影EXTERNAL_ARCHIVE时允许归档。保存保留issuerPartnerId；指纹纳入外部主体ID/名称。归档使用独立A4资料页、真实签发方、保存版本和原单事实，不套公司自签、正本/电放选项、虚构条款或签章；SE自签四版本保留。预览、下载、打印沿同一Blob，缺外部签发方禁止输出但不阻止资料维护。

## 精确验证命令与结果

1. SI主要定向组：

```bash
pnpm --dir web exec vitest run src/pages/orders/order-kinds/sea-export/form-adapter.test.ts src/pages/orders/order-kinds/sea-import/form-adapter.test.ts src/pages/orders/order-kinds/sea-import/service-save-baseline.test.ts src/pages/orders/use-order-detail-data.test.ts src/pages/orders/use-order-fee-options.test.ts src/pages/orders/templates/components/sea/SeaMasterBillFields.import.test.tsx src/pages/orders/detail-import-service-save.test.tsx src/pages/orders/new.test.tsx
```

8文件106测通过（日志`/tmp/si-front-tests4.log`），覆盖SI新建请求/草稿、DIRECT/HOUSE真实载荷、同方向候选及权限、严格响应类型、服务成功与夹带新Order/TE变动的基线和输入保护。

2. HBL定向组：

```bash
pnpm --dir web exec vitest run src/pages/orders/hbl/HblEditor.archive.test.tsx src/pages/orders/hbl/model.test.ts src/pages/orders/hbl/HblEditor.output.test.tsx src/pages/orders/hbl/HblEditor.company-change.test.tsx src/pages/orders/hbl/pdf/HblPdfPreview.test.tsx src/pages/orders/hbl/pdf/HblArchivePdfDocument.test.tsx
```

6文件19测通过（`/tmp/si-front-hbltests.log`）。此首轮布局通过后另发现实际归档PDF正文不可见，不能将该首轮结果单独作为输出验收；最终修复与独立实图见下一节。

3. 相邻海运/路由/状态/实际自签PDF组：

```bash
pnpm --dir web exec vitest run src/pages/orders/order-kinds/registry.test.ts src/pages/orders/sea-house-order-context.test.ts src/pages/orders/list-documents-action.test.ts src/pages/orders/components/detail/OrderStatusSection.import.test.tsx src/constants/statusMeta.test.ts src/pages/orders/hbl/model.archive.test.ts src/pages/orders/hbl/pdf/HblPdfDocument.test.tsx src/pages/orders/hbl/pdf/HblPdfDocument.company.test.tsx src/pages/orders/order-kinds/sea-export/SeaExportDetailFeatures.test.tsx src/pages/orders/order-kinds/sea-export/SeaExportDetailFeatures.qualification.test.tsx
```

10文件64测通过（`/tmp/si-front-finaltests.log`）。更早的详情变动/恢复/终止/主单/单证权限回归组9文件通过，唯一当时失败的共用adapter断言已随明确清空契约更新并在主要组再次通过，未删除相邻回归。

4. 最后F07/F08补齐与回归：

```bash
pnpm --dir web exec vitest run src/pages/orders/templates/sea-payment-term.test.tsx src/pages/orders/templates/sea-create-defaults.test.tsx src/pages/orders/templates/sea-create-layout.test.tsx src/pages/orders/templates/sea-form-labels.test.tsx src/pages/orders/order-kinds/sea-export/form-adapter.test.ts src/pages/orders/order-kinds/sea-import/form-adapter.test.ts src/pages/orders/new.test.tsx
```

首跑其他6文件70测通过，唯一新付款测试将实际选项“到付 (CC)”误写为“到付”导致查询失败；修正测试查询并禁用测试虚拟列表后，以下最终重跑2文件44测通过（含SE/SI恢复参数化回归，日志`/tmp/si-front-last-tests.log`）：

```bash
pnpm --dir web exec vitest run src/pages/orders/templates/sea-payment-term.test.tsx src/pages/orders/use-order-change-notice.test.tsx
```

5. 类型及静态检查：`pnpm --dir web tsc`最终退出0（`/tmp/si-front-tsc-final.log`）；`git diff --check`通过。本地`antd info Form/Button/Drawer/Segmented/Select --format json`已核对API，27个修改TSX文件逐文件`antd lint <file> --format json`均0issues；后补付款涉及BasicInfo/Cargo文件同样0issues（日志`/tmp/si-front-antd-lint.json`、`/tmp/si-front-antd-basic.json`、`/tmp/si-front-antd-cargo.json`）。

对下方58个非生成本人文件执行`pnpm --dir web exec biome check <逐文件相对web路径>`，最后只剩`order-kinds/sea-export/form-adapter.ts:435`的tradeTerm三元表达式折行格式差异，无语义lint问题。已交回产品写权并通知Root，由独立checker修正格式，不再并行修改产品文件。精确文件集合与日志在`/tmp/si-front-owned-files.txt`、`/tmp/si-front-biome-final.log`。

## PDF真实输出检查与首轮漏项修复

首次归档测试读取React PDF实际布局中的TEXT_INSTANCE，因此虽然文字节点存在，显式lineHeight与固定文字容器组合仍使实际字形不可见，首轮测试漏掉了最终绘制。这一缺陷在独立PyMuPDF提字和看图时发现；删除正文lineHeight，沿既有HBL FlowText方式显式fontSize=9和14pt容器高度后修复，并增加实际每行box.height须容纳字形且落入Text容器的检查。此检查同时防止超高行框被容器裁切；只查看TEXT_INSTANCE或PDF文件头不能替代它。

最终命令：

```bash
pnpm --dir web exec vitest run src/pages/orders/hbl/pdf/HblArchivePdfDocument.test.tsx
```

1文件2测通过，1.29秒（`/tmp/si-front-archive-pdf.log`）。普通有值字段样张1页，极值样张7页，缺省可选字段不占无意义空行，显式零保留。使用环境已有`PYTHONPATH=/tmp/roncin-hbl-pdf-tools python3`及`pymupdf.open`对最终PDF逐页get_text、get_pixmap复核：长地址“长/址”各1000、货描“明/细”各3000，与原始正文一致；`ARCHIVE-END-MARKER`及末箱`TEST0000109 / SEAL-109 / 20GP`存在；真实签发方、保存版本7、纯日期2026-11-16可见，所有页包含归档标题与页码。独立span边界约x28～498、y22～819，均在A4页面内。

最终合成资料不含真实客户数据，留在临时目录：

- `/tmp/roncin-si-hbl-archive.pdf`、`/tmp/roncin-si-hbl-archive-long.pdf`
- `/tmp/roncin-si-hbl-archive-page1.png`
- `/tmp/roncin-si-hbl-archive-long-page2.png`、`/tmp/roncin-si-hbl-archive-long-page7.png`

已实际查看普通页、连续中文续页和末页箱号图片；Root也独立看过普通归档图片。

## 未由本代理验收的部分

- 真实服务端写入、权限/正式迁移/事务与完整业务闭环由后端代理和Root负责，不以本报告前端mock证明已完成。
- Root专属浏览器代理验证真实页面路由/交互与同域打印资源；Root统合生产build及本组风险门禁。本代理未运行全量check:fast/build，未改生产配置或依赖。
- 固定五服务组件及全部model/交互测试由Root独占；本代理只验证父详情集成，不替代Root组件自身11测或后端门禁。
- 唯一已知静态收尾是上述一处formatter折行，已明确交给Root/checker。下一子任务的AI、AWB PDF及完整筛选CSV仍按父任务依赖交付，本组未提前实现。

## 本代理文件集合（含测试，58项）

- `web/config/routes.ts`
- `web/src/constants/statusMeta.test.ts`
- `web/src/constants/statusMeta.ts`
- `web/src/pages/orders/components/detail/OrderLockControl.tsx`
- `web/src/pages/orders/components/detail/OrderStatusSection.import.test.tsx`
- `web/src/pages/orders/components/detail/OrderStatusSection.tsx`
- `web/src/pages/orders/components/drawers/ContainerDrawer.tsx`
- `web/src/pages/orders/components/drawers/SeaTransportExecutionUpdateModal.tsx`
- `web/src/pages/orders/detail-import-service-save.test.tsx`
- `web/src/pages/orders/detail.tsx`
- `web/src/pages/orders/fees.tsx`
- `web/src/pages/orders/hbl.tsx`
- `web/src/pages/orders/hbl/HblEditor.archive.test.tsx`
- `web/src/pages/orders/hbl/HblEditor.fixtures.tsx`
- `web/src/pages/orders/hbl/HblEditor.tsx`
- `web/src/pages/orders/hbl/model.archive.test.ts`
- `web/src/pages/orders/hbl/model.ts`
- `web/src/pages/orders/hbl/pdf/HblArchivePdfDocument.test.tsx`
- `web/src/pages/orders/hbl/pdf/HblArchivePdfDocument.tsx`
- `web/src/pages/orders/hbl/pdf/HblPdfPreview.tsx`
- `web/src/pages/orders/hbl/pdf/content.ts`
- `web/src/pages/orders/hbl/pdf/types.ts`
- `web/src/pages/orders/list-documents-action.ts`
- `web/src/pages/orders/list-query.ts`
- `web/src/pages/orders/list.test.tsx`
- `web/src/pages/orders/list.tsx`
- `web/src/pages/orders/new.test.tsx`
- `web/src/pages/orders/new.tsx`
- `web/src/pages/orders/order-detail-transitions.tsx`
- `web/src/pages/orders/order-kinds/registry.test.ts`
- `web/src/pages/orders/order-kinds/registry.ts`
- `web/src/pages/orders/order-kinds/sea-export/SeaExportDetailFeatures.tsx`
- `web/src/pages/orders/order-kinds/sea-export/form-adapter.test.ts`
- `web/src/pages/orders/order-kinds/sea-export/form-adapter.ts`
- `web/src/pages/orders/order-kinds/sea-import/definition.ts`
- `web/src/pages/orders/order-kinds/sea-import/form-adapter.test.ts`
- `web/src/pages/orders/order-kinds/sea-import/service-save-baseline.test.ts`
- `web/src/pages/orders/order-kinds/sea-import/service-save-baseline.ts`
- `web/src/pages/orders/order-kinds/types.ts`
- `web/src/pages/orders/release-pod-panel.tsx`
- `web/src/pages/orders/sea-house-order-context.test.ts`
- `web/src/pages/orders/sea-house-order-context.ts`
- `web/src/pages/orders/templates/components/sea/SeaBasicInfoSection.tsx`
- `web/src/pages/orders/templates/components/sea/SeaCargoSection.tsx`
- `web/src/pages/orders/templates/components/sea/SeaDocumentHistoryActions.tsx`
- `web/src/pages/orders/templates/components/sea/SeaDocumentSection.tsx`
- `web/src/pages/orders/templates/components/sea/SeaMasterBillFields.import.test.tsx`
- `web/src/pages/orders/templates/components/sea/SeaTransportSection.tsx`
- `web/src/pages/orders/templates/components/sea/useSeaMasterBillCandidate.ts`
- `web/src/pages/orders/templates/sea-payment-term.test.tsx`
- `web/src/pages/orders/templates/sea-template.tsx`
- `web/src/pages/orders/templates/types.ts`
- `web/src/pages/orders/use-order-change-notice.test.tsx`
- `web/src/pages/orders/use-order-change-notice.ts`
- `web/src/pages/orders/use-order-detail-data.test.ts`
- `web/src/pages/orders/use-order-detail-data.ts`
- `web/src/pages/orders/use-order-fee-options.test.ts`
- `web/src/pages/orders/use-order-fee-options.ts`
