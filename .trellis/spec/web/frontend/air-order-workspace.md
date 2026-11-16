# 航空订单就近工作区

- `air-template.tsx` 沿用六列、4px 行距与五卡结构；AI运输标题为“承运与运输”。
- `TemplateProps.airCargoWorkspace / airTransportMaintenance / airDocumentActions` 由类型扩展贡献；
  货物工作区位于委托货量后，重量历史折叠，普通保存与货物/重量/共同运输独立提交。
- `airTransportSnapshot` 只展示刷新后的共同运输；表单内 `airExport` 与 expectedMasterVersion
  仍为原稿基线。共同运输成功不得用新版本覆盖本票未保存输入。
- 单证卡按 MAWB/HAWB 页签组织，HOUSE默认HAWB，DIRECT仅MAWB；
  详情主单正文用轻量只读摘要，不向本票Form注册第二套字段。
- 常用正文常驻，低频正文懒挂载；页签/折叠共用 `revealSeaFormErrors` 定位错误。
  只有 `airMaster.waybillContent` 错误才切MAWB，运输字段错误不得误切单证。
- SI/AI“进口服务”位于状态摘要后、五卡前；独立保存仍使用原有精确版本采纳规则。
- `documentMenuItems` 贡献页头单证入口；制作/预览与历史的目标取真实主分单身份，DIRECT不虚构HAWB。
- 回归入口：`air-document-workspace.test.tsx`、`air-transport-summary.test.tsx`、
  `AirTransportMaintenance.test.tsx` 以及两类 `detail-*-import-service-save.test.tsx`。

- 航空业务卡嵌入竖向 `Space` 时让内容项伸展到全宽（`alignItems: stretch`），避免页签内表单只占半幅。只读导航、历史和核对操作显式解除父表单禁用继承；确认/更新写入口仍检查各自权限与锁态。

后端列表的两来源确认重量投影见 [订单列表字段与排序](../../server/backend/order-list-fields.md)。

## 订单单证正文布局与显式带入

- `AirWaybillFields(variant="order")` 与只读 MAWB 共用 `AirOrderWaybillLayout`：
  收发货人、通知人/代理、唛头/品名各占一行两列，窄屏一列；抬头4行、货描5行。
  通知人与第二通知人字段常驻，只切可见性，切换本身不标脏。
- 常用文本域须局部设置 `height: auto`，避免全局 `textarea.ant-input` 的240px高度
  覆盖 `rows`。验收用浏览器实际边界框比较海空宽高，不能只断言rows属性。
- 订单低频字段收在“更多运单资料”内，保留原7组。错误揭示同时展开外层、对应内层
  与正确通知人；完整AWB制作页仍走 `variant="full"`，不改变其独立保存协议。
- 只读MAWB复用文本框布局但不注册Form字段；原生 `readOnly` 保留复制，显式解除
  禁用继承仅用于只读文本及导航，不能放开带入或编辑。其正文不进入本票提交。
- `TemplateProps.onAirDocumentImport(prefix, field, value)` 仅接受本票
  `airHouseContent` 或新建主单 `airMaster.waybillContent` 目标，经
  `airDocumentImportPatch` → 模板 `applyValues` 深合并并标脏。禁止只调用
  `form.setFieldValue` 而漏掉草稿及离开保护，也不得替换提交版本基线。
- 三种带入仅写单一目标：委托客户→`shipperText`，国外代理→`agentText`，
  订单货描→`goodsDescription`；不推算运价行或覆盖件重尺。已有文本覆盖前确认；
  空源提示、失败保留，超长值由Form校验报错，不静默截断。
- 客商请求走生成客户端与React Query，按当前公司和客商ID校验响应身份。
  请求/确认期间公司、来源、订单、主单、正文或输入变化均丢弃迟到结果，使用代际
  防止A→B→A身份切换误采纳。没有父级标脏回调时不显示带入入口。
- 回归入口：`AirOrderWaybillFields.*.test.tsx`、
  `air-document-import.integration.test.tsx`、`air-document-readonly.review.test.tsx`、
  `air-document-alignment.e2e.ts`；分别验证字段保留与竞态、真实模板标脏、
  父Form禁用下只读隔离、1440/390实际布局及带入离开保护。
