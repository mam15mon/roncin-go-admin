# SI 前端独立复核与局部自修

复核角色：`trellis-check`。当前任务 `.trellis/tasks/11-15-hgj-sea-import`，父方案已获批准。
本记录覆盖本组所有非生成前端改动及新增文件；不修改后端、生成客户端、其他代理的
`web/tests/e2e/sea-import.e2e.ts`，不提交或归档。最终生产构建、完整门禁、浏览器验收由主会话统合。

## 范围与合同核对

已读取当前任务 PRD/design/implement、父 PRD/design/字段采纳表，前端 index、类型、状态、
组件、目录及质量规范，领域地图、海运主分单/共享航程/HBL 输出和新的进口服务合同。
按当前全部差异和新增实现核对以下真实代码路径，不仅检查最后一段补丁：

- registry/routes、新建/详情/列表/费用、SI/IMPORT 与真实公司身份，真实海运 MBL/TE/Link/HBL。
- 海运双方向字段映射、经营字段显式清空、付款控件、候选与来源分单创建；SE 原有拆票改配保留。
- 单证历史/模式、实际箱及共享箱、航程、放货/终止恢复入口及真实品类权限传递。
- 固定五服务 wire/表单、未承接与完成存在性、原因/取消确认、候选/附件、事件、轮询和输入基线。
- 服务独立成功后只接受精确 Order+1 且共享实体身份版本不变，保留整票未保存输入。
- 外部 HBL 维护/真实签发方、归档与自签四版区分、读取/指纹/同 Blob 生命周期、字体及真实分页。

没有新增通用平台、兼容分支、依赖或生成物手修。模块仍在同一个 orders 页面领域中复用。

## 已修复

1. **服务保存迟到结果可能关闭新稿**：`ImportHandlingPanel.tsx` 在保存响应后已有会话检查，
   但等待 `onSaved` 或查询失效期间可以重新读取形成新的编辑 serial。现在每段 await 后
   同时核对组件存活、工作区代际和最新 serial；旧提交不会清理/关闭新稿。回归通过真实
   GuardedModalForm 保存等待→重新读取新快照→新输入→旧回调完成的时序验证。
2. **成功读取缺少完整身份约束**：详情/费用补齐目标订单 ID 与当前公司严格相等，缺公司同样拒绝。
   费用 hook 接受已有工作区公司作为必要参数，同步 queryKey、enabled、loadData 与 loading，
   身份缺失不请求、不永久转圈。HBL 初读、显式刷新、写后重读及生成前复核统一经带真实
   type/company 的 `fetchHblBundle`；初读与 Editor 的 hbl-production key 均带 kind。
   未为旧 mock 增加 if-present 放行，测试夹具同步真实公司身份。
3. **归档续页缺少单号与保存版本**：`HblArchivePdfDocument.tsx` 固定页脚加入每页 HBL 号摘要、
   保存版本及页码。固定区域最多 32 字摘要，长多行原号码在正文全文保留，避免固定区域溢出。
4. **输出指纹未覆盖正文和模板修订**：`hbl/model.ts` 复用现有 SHA-256 工具，纳入规范化正文/
   印字的固定长度摘要；`HBL_PDF_TEMPLATE_REVISION` 同时参与变动检查和 PDF 实例 key。
   字段顺序/空投影不误报，正文/印字变化及模板修订会使旧输出失效。
5. **纳秒完成事实被日期控件截断**：进口服务 payload 直接接收打开快照；控件毫秒时刻未变时
   保留原始 RFC3339Nano 字符串，实际修改才输出新 ISO 时刻。事实指纹也保留纳秒尾数，
   不把纳秒级外部变化误认为相同事实。服务时间、控件时间与提交版本没有全局补值或双读。
6. **SI 共同航程误用整票改配权限**：共享航程标题按钮、更多菜单与候选缺项维护链接使用
   SI update；SE 保留 reassign。权限收回关闭旧航程弹窗。SI 不出现整票改配/拆票入口，
   不请求出口改配资格。后端/路由动作对应生成由后端检查代理负责。
7. **服务历史未显示换单代理前后值**：事件展开区补充 before/after 的换单代理姓名，
   让仅换代理的事件可见；不展示内部 UUID 冒充人员名称。
8. **机械问题**：修复 `sea-export/form-adapter.ts` 已知折行及新增检查代码的 Biome 格式问题；
   未进行无关文件格式化。

## 实际定向验证

首轮对详情、费用、HBL 读取、归档 PDF 和新服务会话回归合跑 5 文件：4 文件/47 项通过，
新回归因 Antd 确认框双标题及退出动画定位失败。随后缩小到当前可见编辑对话框，保留
实际确认/新输入断言，不加 setTimeout、不屏蔽可见性，下面重跑全部通过。

```bash
pnpm --dir web exec vitest run \
  src/pages/orders/import-handling/ImportHandlingPanel.change.test.tsx \
  src/pages/orders/hbl/model.fingerprint.test.ts \
  src/pages/orders/hbl/HblEditor.archive.test.tsx \
  src/pages/orders/hbl/HblEditor.save.test.tsx \
  src/pages/orders/hbl/HblEditor.output.test.tsx \
  src/pages/orders/hbl/HblEditor.company-change.test.tsx \
  src/pages/orders/hbl/HblEditor.company.test.tsx \
  src/pages/orders/hbl/HblEditor.draft.test.tsx
```

**8 文件、20 项 PASS，21.53 秒**。覆盖保存/失败/公司变更/草稿/归档及原四版相邻行为。

```bash
pnpm --dir web exec vitest run \
  src/pages/orders/import-handling/model.test.ts \
  src/pages/orders/import-handling/ImportHandlingPanel.test.tsx \
  src/pages/orders/import-handling/ImportHandlingPanel.change.test.tsx \
  src/pages/orders/order-kinds/sea-export/SeaExportDetailFeatures.import.test.tsx \
  src/pages/orders/hbl/model.fingerprint.test.ts \
  src/pages/orders/hbl/pdf/HblArchivePdfDocument.test.tsx
```

**6 文件、22 项 PASS，21.47 秒**。纳秒原样/实际修改/仅纳秒变化、会话迟到、SI/SE 航程权限、
正文/模板指纹、普通/长文/长号实际 PDF 均通过。整个命令耗时不等于单文件耗时；本次没有
单文件超过 20 秒，没有新增超过 10 个重型业务树用例的文件。

```bash
pnpm --dir web exec vitest run \
  src/pages/orders/use-order-fee-options.test.ts \
  src/pages/orders/hbl/use-hbl-data.test.tsx \
  src/pages/orders/hbl/model.fingerprint.test.ts \
  src/pages/orders/order-kinds/sea-export/SeaExportDetailFeatures.import.test.tsx
```

**4 文件、31 项 PASS，5.72 秒**。在最终稳定生成物下复核费用身份/缺公司不请求、HBL 严格
重读/轮询、模板指纹及 SI update/SE reassign/只读门禁。未重复全部已通过的历史验收套件。

## 静态检查与真实 PDF

- `pnpm --dir web tsc`：最终稳定生成完成后 **PASS**，不是仅生成前的旧结果。
- `pnpm --dir web exec biome check <本组69个非生成前端文件>`：**PASS**，最终 69 文件、86ms、
  无待修问题；文件集合由 git diff 与 untracked 合集生成，排除生成文件与独占 e2e。
- 对 `ImportHandlingPanel.tsx`、`SeaExportDetailFeatures.tsx`、`SeaTransportSection.tsx` 各运行
  `pnpm --dir web exec antd lint <file> --format json`：**0 问题、0 skipped**。
- `git diff --check -- web`：**PASS**。
- 使用环境已有 `PYTHONPATH=/tmp/roncin-hbl-pdf-tools python3` 与 PyMuPDF，独立于 React 布局树
  提字、核对逐页坐标并生成图片。三份样张为 `/tmp/roncin-si-hbl-archive.pdf`（1页）、
  `-long.pdf`（7页）和 `-long-number.pdf`（1页）。长地址/货描及末箱/末行保留，所有页都有
  保存版本和单号摘要，长号正文全文与末标记保留，全部文字坐标在 A4 边界内。
- 已实际看图：`/tmp/roncin-si-hbl-archive-long-review-page2.png`、
  `-last.png` 和 `/tmp/roncin-si-hbl-archive-long-number-review.png`。中文、正文、真实外部主体、
  有界页脚可见，未套公司自签条款。临时 PDF/图不进入 Git。

## 未修复项与限制

本轮范围内没有遗留的已确认前端缺陷或待产品决定事项。正式 PostgreSQL、最终浏览器、
生产 build 和完整 check:fast 的结果分别由对应代理/主会话记录；本报告不把 mock 或布局树
通过当成那些实际环境已验收。前序其他定向证据见同目录 front/import-handling 验证记录，
本报告不重复宣称运行过它们。

**写权状态：本代理已释放全部非生成前端产品文件写权；不再修改产品。仅完成本报告。**

## 最终门禁首轮后的测试合同同步

主会话首轮 `check:fast` 的后端已通过；前端为 **499 文件、2694 项通过，5 文件、10 项失败，
1 文件、11 项既有跳过**（`/tmp/hgj-sea-import-check-fast.log`）。首轮整体不能记为通过。
主会话再次授权本代理复核这 5 个文件，本次未修改任何产品代码、生成物或 e2e，生产构建仍有效。

逐项核对结果与修复：

- `router/adaptRoutes.test.tsx`：原固定 42 路由未计新增 SI 列表与归档入口。
  改为按唯一 `config/routes.ts` 中所有组件声明数量核对 lazy 结果，再加唯一布局壳，
  并明确断言 SI 列表与 HBL 路由存在、真实加载每个模块。当前实际 43 页面+1 布局，
  不用单纯改大数字掩盖缺失 loader。
- `components/layout/TagsView.test.tsx`：SI 已正式注册，更新列表/新建标题，补齐 SI 详情、
  费用和 HBL 标题断言；尚未注册的 AI 仍保持中性标题。标题继续来自配置，无另一套产品名称表。
- `orders/order-abnormal-entry-data.test.tsx`：真实详情 hook 需要与当前公司和路由相符的身份。
  由各测试渲染的 SE/AE 明确构造对应生成枚举与 company，不再把缺 businessType/company 的
  非法成功响应当正常订单；人员读取、创建权限与权限回收的原业务断言全部保留。
- `orders/orders-breadcrumbs.test.tsx`：正常详情补齐 SE/org-1。读取失败用实际请求层消费
  错误后的 undefined 验证加载失败页头与重试文案，替换旧的“成功但没有订单身份”的空档案假设。
  公共面包屑、单一返回按钮和费用页相邻断言保留；不为旧 mock 放宽读取边界。
- `orders/components/drawers/SeaTransportExecutionUpdateModal.test.tsx`：目标与影响成员补齐真实
  SE/company，成员链接按生成业务类型解析；保留两个准确 SE 链接及完整影响范围等原断言。

实际执行：

```bash
pnpm --dir web exec vitest run \
  src/router/adaptRoutes.test.tsx \
  src/components/layout/TagsView.test.tsx \
  src/pages/orders/order-abnormal-entry-data.test.tsx \
  src/pages/orders/orders-breadcrumbs.test.tsx \
  src/pages/orders/components/drawers/SeaTransportExecutionUpdateModal.test.tsx
```

**5 文件、57 项 PASS，16.08 秒**。首轮测试启动时本代理夹具编辑产生了两个重复对象键告警，
已直接删除重复字段；最终 Biome、tsc 无误，并单独重跑该航程文件确认最终夹具。

- 这 5 文件 `pnpm --dir web exec biome check <files>`：**PASS，5 文件，无待修问题**。
- `pnpm --dir web tsc`：**PASS**。
- 这 5 文件 `git diff --check -- <files>`：**PASS**。
- 最终前端完整门禁由主会话执行 `check:web`；本补充不把定向通过记作完整门禁通过，
  不重复执行已通过的后端全量检查，不把 11 个跳过测试计为通过。

本补充只涉及上述 5 个测试文件及本报告，未发现需要追加产品修改的缺陷。

航程文件最终独立重跑：**1 文件、6 项 PASS，14.50 秒，无重复键告警**。
本轮再次释放全部非生成前端文件写权，后续完整门禁由主会话接管。
