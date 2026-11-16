# 航空制作编辑与真实主体投影验证

日期：2026-11-16。实施范围为后端必要主体投影、正式生成物、前端制作页/保存模型/版本探测及入口。
航空 PDF 版式、中立字体/Blob 工具和 HBL 相邻改动由 renderer 负责人维护；真实浏览器由 browser
负责人执行，本报告不把组件测试当作实际 PDF 或浏览器验收。

## 交付与冻结合同

- Proto `AirMasterWaybill.organization_name_en=16` 为 OUTPUT_ONLY。biz 纯值、service 转换，
  data `decorateAirMaster` 使用调用方 `Data.client(ctx)` 提供的客户端读取真实
  `master.organization_id` 的 company。无新数据库列、迁移、权限或主体配置平台。
- AE HAWB 的 `SELF_ISSUED` 抬头来自真实公司英文名；正文 `issuedBy` 仍保留独立已保存印字。
  MAWB、AI HAWB 的 `EXTERNAL_ARCHIVE` 只使用各自已保存 `issuedBy`，不猜航司或当前公司。
  缺主体只阻止输出，仍可维护正文；输出不改变签发、运输、审核或费用状态。
- 主单制作路由 `/orders/{air-export|air-import}/masters/:masterId/awb`；
  订单内入口 `/orders/{air-export|air-import}/:id/awb/:target`，target 固定大写 MAWB/HAWB。
  DIRECT 没有 HAWB 入口，直接访问同样明确拒绝。
- 订单内 MAWB 首次验证本票归属后规范为独立主单制作身份；该规范同时下沉 AwbEditor，
  保存、探测和输出不借后续 Order/HAWB 版本。HAWB 始终绑定真实本票/主单/House/Link。
- MAWB 使用打开时完整运输 input，仅改 `waybillContent`，沿 Preview→共享影响确认→Update，
  提交原主单版本、指纹和原因。HAWB 仅提交明确正文、原状态与原 Order/Master/House/Link
  基线，不带旧经营字段或运输副本。保存成功只采纳本次恰当版本和身份。
- 15 秒前台/回焦检查沿真实 React Query；隐藏、离页、blur 暂停。探测独立实例 owner 与
  刷新 session，生产页首次读取也有实例 owner，A→B→A 不复用旧在途或旧缓存基线。
  指纹包含真实公司英文名，因此更名即使不推进 Master.version 也持续提示、使旧预览失效。
- 未保存正文明确阻止输出；生成前重读核验保存快照。保存失败、409、后续版本、保存途中
  新输入与刷新途中新输入均保留草稿，不替换提交基线。覆盖脏输入前确认，取消/失败保留。
  共享确认使用官方 ModalForm.onFinish 生命周期；输出按需加载 renderer。

## 后端与生成验证

1. `make -C server api`、`pnpm run generate:web-client` 完成；再运行同一流程，对全部
   Go Proto 绑定、OpenAPI、web 输入/客户端/枚举做 SHA-256 比较，变化文件数 **0**。
2. `go -C server test -p 32 ./internal/service ./internal/biz -run 'Air' -count=1` 通过。
3. `go -C server test -p 32 ./internal/service -run 'AirWaybillOutputCompanyWire' -count=1`
   通过：真实 ProtoJSON 同时保留公司投影和已保存印发人；空英文名按非 optional 语义省略。
4. `.env.local` 只在进程内加载并将 `DATABASE_SOURCE` 赋给
   `RONCIN_INTEGRATION_DATABASE_SOURCE`；未输出、复制或提交凭据。运行：
   `go -C server test -p 32 ./internal/data -run '^TestAirWaybillOutputCompanyProjectionPostgres$' -count=1 -v`。
   通过，AE/AI 两个子场景，**0 SKIP**。测试辅助入口创建随机隔离 Schema，逐条执行正式 SQL
   迁移，结束清理该 Schema；未清库、未造开发业务票、未使用 Schema.Create。
   断言 Order 内嵌主单与 Get 主单真实 company 投影、系统/其他公司名称不借、跨公司 Get 拒绝、
   公司更名不推主单版本/hash、缺英文名仍能读取且不改主分单已存 issuedBy。

## 前端定向验证

- 首批组合运行以下 **9 文件 / 45 测试** 全通过：`awb/model.test.ts`、
  `awb/use-awb-data.test.tsx`、`awb/AwbEditor.save.test.tsx`、`awb/AwbEditor.output.test.tsx`、
  `air-master.edit.test.tsx`、`air-master.import.test.tsx`、`detail-air-house-save.test.tsx`、
  `detail-air-house-save-direct.test.tsx`、`templates/components/air/AirWaybillFields.test.tsx`。
  命令均为 `pnpm --dir web exec vitest run <以上相对web路径>`。
- 最后补强严格嵌套主单/PDF builder 校验后，`awb/model.test.ts` **8 项通过**；新增主单办理权
  后 `AwbEditor.output.test.tsx` **6 项通过**；刷新改用 QueryClient.fetchQuery 后
  `AwbEditor.refresh.test.tsx` **2 项通过**。
- 新增核心测试总计 27 项：模型 8、hook 6、保存 5、输出 6、刷新 2。组件每文件最多 6 个
  重渲染用例，单文件实际测试耗时均低于 20 秒。
- 覆盖真实 wire 空消息/零/false/optional 未设、缺公司/错类型/缺版本、DIRECT 无 HAWB、
  主体分流与缺名、原基线最薄提交、确切版本/后续版本拒绝、保存失败/409、在途新输入、
  MAWB 共享确认、只读输出和办理权、dirty 输出阻止、刷新确认取消/失败/在途编辑、公司更名、
  15 秒/blur/hidden/回焦/离页、A→B→A 与卸载迟到。
- TypeScript `pnpm --dir web tsc`、改动非生成文件 Biome、`pnpm run check:architecture`
  （18 项脚本测试及实际 AST 扫描）、`git diff --check` 通过。
- 按 antd skill 先查 Form/Modal/Button API 和 Button 示例；制作入口/编辑器分别运行
  `antd lint <文件> --format json`，均 0 issues、partial=false。
- 部分定向组合的 Node 进程显示 `TimeoutNaNWarning`，测试断言全部通过；没有为隐藏该
  非失败提示修改通用计时器或测试基础设施。

## 后续验收责任

本组未执行 check:fast、生产 build、真实浏览器或实际 PDF 提字/逐页图验。
由 Root 与 renderer/browser 负责人统合这些门禁；未提交 Git、未推送，未修改主文档/spec。
