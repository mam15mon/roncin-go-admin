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
