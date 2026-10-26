# 连续分单入口与字段研究

## 路由与草稿

- `router/history.ts:21` 只支持字符串导航；采用 `?sourceOrderId=`，只传来源ID，重新读取服务端，不在URL塞业务快照。
- `routeUtils.ts:75` 海运按模块归组；`TagsView.tsx:108` 用 pathname 归组，完整地址存在 tag.path。保持模块页签，不改全站规则。
- `OrderFormTemplate.tsx:139` 每实例仅恢复一次；`new.tsx:26` 内的表单ref、loading与创建幂等键属于资源身份。按规范化来源地址key整个创建内容组件。
- 普通新建草稿路径保持原样，来源A/B各用带唯一sourceOrderId的规范地址；模板继续独占草稿key与清理。首次来源加载成功前不挂可编辑表单，不用后续effect覆盖草稿。
- 模板只有tab close guard与onValuesChange即存草稿，无普通路由离开确认。本期复用此语义，不另加确认框；关闭模块会一起处理当前工作区全部来源草稿。
- 来源查询用React Query、key包含当前组织和来源ID，权限门控。来源切换key重建后，旧创建响应不能提示或导航新表单。

## HBL与详情入口

- `HouseBillIdentityFields.tsx:45` 当前唯一注册 `['seaHouseBill','houseNo']`，含批次排重与清单晚到重验。拆独立号码组件，新建移至MBL旁，其他信息仍在HBL页签；详情和模式变更弹窗保留原编辑位置。
- `SeaTransportSection.tsx:320` 当前关联分单号只是展示；新建HOUSE替换成正式号码组件，DIRECT不挂载规则。
- `SeaDocumentSection.tsx:315` 当前所有HBL错误都切页签；新建houseNo移动后应排除，签发方/内容错误及详情仍切页签。
- detail.tsx向TemplateProps传来源订单ID；字段组件复用create与organization能力判断，在主单区显示入口，单票HOUSE也可用。不能依赖Form id，当前海运初始值没有此字段。
- 来源订单的审核锁只限制修改来源本身，不限制独立新建；不用readonly或memberCount>1判定新票入口。

## 来源航程与候选复验

- Summary含transportExecutionId与transportExecutionVersion；summary.version属于MBL。
- 新票仅预填船公司、主单号和HOUSE；指定来源TE在最新候选中按ID定位，再带入当前航程与当前版本。
- 当前hook多TE仅内存selection；增加来源上下文提示，不把来源ID/version直接当候选确认。指定TE失效不得自动切换到另一条，即使最新候选只剩一条。
- 客户、HBL、目的地、货物、费用、单证内容不复制；同来源草稿恢复后不得被异步默认值覆盖。
