# 保存保护与详情失败恢复：技术设计

### 保存判定

保持现有请求客户端契约；目标写调用显式取得可判定的成功响应，失败走既有错误处理。可采用生成客户端 options.skipErrorHandler 让原始错误外抛，再由调用方复用统一错误反馈；避免重复notification和吞异常。实现时逐一检查接口成功报文（不能统一假定每个接口都有data.id）。订单写成功后背景刷新仍为best-effort，不重新把成功改成失败。

### 工作保护

复用useTabCloseGuard，dirty判定覆盖外部state；路由离开需要与React Router blocker配合，避免仅注册关闭guard却在点击侧栏时丢输入。确认只出现一次，保存成功导航和已确认离开有明确放行。先实现保护，不添加持久业务副本。
现有 `web/src/router/index.ts` 使用createBrowserRouter，路由阻断应基于这一真实路由入口。客商身份变更可能复用组件，不能只依赖卸载触发保护；dirty及确认放行绑定当前客商身份。关闭页签和组织切换已有确认后，不再由路由阻断重复询问；用户拒绝时不得提前清状态或改变组织。

### 失败恢复

客户主档查询不得读取undefined.data产生额外异常。首次失败分支复用订单Result/重试样式，辅助查询失败不冒充主档失败；refetch不得覆盖已有未保存编辑。

### 边界与风险

仅前端目标页面/公共guard所需局部能力；生成物和后端不变。利息任务也涉及partner-detail，须在本任务完成并提交后开始，避免同文件并发。

## 影响文件

- `web/src/pages/partners/partner-detail.tsx`
- `web/src/pages/orders/detail.tsx`
- `web/src/pages/finance/bills/index.tsx`
- `web/src/pages/finance/cashflows/index.tsx`
- `web/src/components/layout/tabCloseGuard.ts`
