# 设计

## 边界

本次行为差异集中于订单页面：将现有航空能力搬到对应业务区域，海运出口是基线。复用OrderFormTemplate、FormRow/FormCell、TemplateFieldGroup、GuardedModalForm及现有领域服务。后端仅在列表缺少航空重量展示投影时扩展只读合同，不新增持久化事实或迁移。

## 组合接口

- `OrderDetailFeatureContribution`与`TemplateProps`增加可选`airCargoWorkspace`、`airTransportMaintenance`、`airDocumentActions`插槽；前两者为ReactNode，后者为按MAWB/HAWB键控的ReactNode映射。类型扩展持有请求和动作，模板只消费插槽。
- `OrderDetailFeatureContribution`增加可选`documentMenuItems`，独立交给页头“单证制作”按钮。海运可从既有制单动作中分组，不重复手写路由。
- 主会话负责两份类型、detail.tsx组合、公共页头、进口服务位置与最终整合；模板与航空扩展代理只按约定使用插槽，不改这些共享文件。

## 航空资料

- 空运模板复用六列/4px行间距，不改全局FormRow。主单详情完整只读摘要优先使用已加载order.airExport.master，不将其渲染成可以随整票保存的输入。
- 当前订单内主单编辑复用PreviewAirMasterWaybillUpdate→UpdateAirMasterWaybill；独立打开会话绑定主单ID/公司/业务类型/版本，捕获原主单完整内容，仅修改运输字段，保留运单正文。
- 查询用React Query；GuardedModalForm.onFinish管理提交生命周期；输入改动使预览失效，服务端指纹和版本最终兜底。前台15秒/回焦变动检查保留原编辑基线及输入。
- 主单更新成功可刷新普通聚合展示，但不得用刷新结果reset整票表单或自动替换提交基线；提示用户本票输入已保留以及需要刷新时的动作。
- 重量面板提供嵌入模式，放货物区；历史默认折叠。实际货量摘要复用现有接口与真实货物明细查询，不能只统计首200条。重量参考始终采用已保存数据，脏表单显示待保存提示，确认不得误用未保存货量。
- 单证卡按模式切主分单，MAWB详情只读并提供制单入口；HAWB保持本票保存。通用AirWaybillFields提供订单简洁呈现选项，专用AWB编辑/主单工作台保持完整字段。错误展开复用现有事件机制并覆盖页签。

## 列表

品类列与展示映射归pages/orders，通用模板仅消费稳定行模型。列表确认重量如需增加投影，在airExport信息内返回可选只读摘要（当前确认值及是否过期），复用已有确认指纹判定。批量读取确认及货量事实，避免每行HTTP调用或新增数据库事实。Proto→绑定/OpenAPI→前端客户端按正式生成流程更新。

## 保存与风险

现有普通整票保存、共享主单修改、重量确认、进口服务与制单保存保持各自事务和基线。UI不能把它们串成假原子保存。测试重点为本票脏输入、保存失败、后续版本、组织/目标变化、DIRECT缺分单、ProtoJSON省略默认值及海运既有路径。无数据库重建。按组提交便于定位和回退。
