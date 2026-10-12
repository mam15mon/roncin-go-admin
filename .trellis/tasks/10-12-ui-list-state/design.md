# 列表筛选刷新与页签状态统一：设计草案

### 筛选与刷新

筛选面板保持挂载/Form实例，隐藏替代卸载；request仍消费已提交条件。开关状态与计时器放共享订单列表编排层，toolbar受控；调用稳定actionRef，清理timer与visibility监听，活动请求未完成不叠加轮询。

### 查询上下文（方案草案）

优先在现有OrganizationWorkspace中保存轻量内存快照，随其用户/组织key卸载销毁；稳定列表ID+页签/业务类型区分查询，页签关闭清理。真实多个领域使用才提取公共查询状态能力，不缓存data/loading，不代理领域请求。页面/筛选模板可接初始化值与提交值，日期按字段定义恢复，远程候选显示名快照只为回显。
无需把过滤条件写入URL，也不改变resolveTabKey语义；重进同一路由保留，关闭重开清理。实现前核对现有列表自身存储，避免形成两个真相源。
快照由当前已认证用户/组织的工作区实例拥有；身份缺失时不跨实例保留，不使用匿名共享键或独立全局Map。快照项按稳定页签key、列表ID和必要业务类型区分，客商三角色不得互相串用。只保已提交条件、对应回显、分页和每页条数；筛选面板收起保留未提交输入，与跨业务页签只恢复已提交查询是两种明确行为。
关闭当前、其它、右侧或全部页签，均只在既有dirty确认通过后清理对应快照；拒绝关闭保留快照。关闭清理发生后，旧页面卸载回调或迟到请求不得再次写回已关闭快照；异步结果仍完全交给ProTable。组织/用户工作区重建销毁全部旧实例快照，完整浏览器刷新不要求保留内存快照。

### 并发及风险

该子任务拥有TagsView查询清理及OrderListTemplate，键盘/窄屏任务可能碰同文件或公共样式，按总任务顺序交付不并发编辑。新上下文能力经公开入口并更新能力导航，禁止pages/features反向导入。

## 影响文件

- `web/src/components/ui/order-list-template/OrderListTemplate.tsx`
- `web/src/components/ui/order-list-template/OrderListToolbar.tsx`
- `web/src/components/ui/order-list-template/OrderListSearchFilter.tsx`
- `web/src/components/ui/search-filter-template/SearchFilterTemplate.tsx`
- `web/src/components/layout/OrganizationWorkspace.tsx`
- `web/src/components/layout/TagsView.tsx`
- `web/src/pages/partners/index.tsx`
- `web/src/pages/finance/fees/index.tsx`
- `web/src/pages/finance/bills/index.tsx`
- `web/src/pages/finance/cashflows/index.tsx`
