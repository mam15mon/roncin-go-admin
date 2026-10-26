# 技术设计

## 最小实现边界

- `new.tsx`及就近来源辅助模块：读取唯一sourceOrderId、用当前工作区React Query加载来源、组合初始值，规范地址key整个创建内容实例。首次来源加载前不挂可编辑Form；来源失败显示可重试Result，普通新建保持原语义。
- `templates/types.ts`增加最小来源上下文类型及两个可选props：`seaHouseOrderCreateSource`（来源订单/MBL/TE身份）和`seaHouseOrderSourceOrderId`（详情入口使用）；不增加全局状态或通用复制平台。
- `detail.tsx`仅向模板传来源order.id；主单组件使用现有组织和create能力绘制按钮，经history导航。来源锁定不影响创建另一票。
- `useSeaMasterBillCandidate.ts`按源MBL身份+TE ID定位最新候选；来源的版本不作为确认。未命中、MBL不一致或指定TE消失时明确错误，不单航程静默取替代项；用户主动改选身份继续按现有候选流程。
- `HouseBillIdentityFields.tsx`提取唯一号码组件，新建由SeaTransportSection挂载；单证页签不重复注册，详情/模式切换保留原位置与命令。错误揭示按移动后的真实位置调整。

## 来源上下文契约

`SeaHouseOrderCreateSource`字段：`orderId`、可选`orderNo`、`masterBillId`、`masterNo`、`shippingLineId`、可选`shippingLineName`、`transportExecutionId`。来源GET验证同当前组织、海运、HOUSE及完整活动主单/航程后才能生成。只预填船公司、主单号、HOUSE，航程由最新候选回填；不将来源Order整体spread进表单。船公司名称用于初始选项保活。

来源新建URL统一`/orders/sea-export/new?sourceOrderId=<ID>`；重复/空sourceOrderId明确失败。表单身份只取kind和此已知参数，不受无关query或顺序影响。模块tabKey仍由pathname计算，draftPathname为规范资源地址。旧异步创建响应只在当前创建实例存活时反馈/导航。

## 测试与回退

A负责来源读取/新建隔离，B负责字段与入口/候选航程，主会话协调共享类型、审查、规范与Git。定向测试真实Form与独立QueryClient；测试动画按规范关闭，不用sleep掩盖竞态。最终运行check:web与diff检查，真实浏览器只读现有订单与未保存新表单，不写业务数据。无迁移，回退代码提交即可。
