# 窄屏导航入口与订单布局修复：技术设计

复用HeaderMenus现有权限过滤的资源/设置数据，抽取最薄菜单数据入口供桌面与移动使用；窄屏优先加入原Drawer菜单，避免把整排桌面按钮挤入头部。保持业务详情hideInMenu语义，不直接取消全站hideInMenu造成桌面菜单重复。
实际浏览器定位溢出元素及父容器，核对FormRow单列媒体规则、ProForm字段默认宽度、min-width/跨列嵌套；只修根因约束，禁止全页overflow-x:hidden。测量必须等待响应式稳定；原双RAF采样只能当覆盖记录。
在键盘任务提交后修改HeaderMenus/global.less，保证同一菜单数据语义；公共组件调整要验证订单各品类真实消费者。

## 影响文件

- `web/src/components/layout/HeaderMenus.tsx`
- `web/src/app/AppLayout.tsx`
- `web/src/router/adaptRoutes.tsx`
- `web/src/global.less`
- `web/src/components/ui/order-template/OrderFormTemplate.tsx`
- `web/src/components/ui/form-row/FormRow.tsx`
- `web/src/pages/orders/templates/components/sea/SeaBasicInfoSection.tsx`

## 实施前定位收敛

research/narrow-preflight.md及几何证据已锁定真实根因：SeaBillContentFormFields标题/操作flex与order-plan-fields箱计划inline-flex不可换行，FormRow降列已正确，不需要猜测修改模板网格。正式最薄修改首先仅两处具名窄屏wrap；没有新根因证据不修改FormRow、SeaBasicInfoSection或OrderFormTemplate。

实际产品文件范围收敛为HeaderMenus/AppLayout、layout范围共享资源设置菜单数据、global.less，以及`web/src/pages/orders/templates/components/sea/SeaBillContentFormFields.tsx`和`web/src/pages/orders/order-plan-fields.tsx`。桌面与移动使用同一权限构造；导航隐藏断点严格<768，与ProLayout移动判定对齐。详见预研验收限制，不以临时CSS探针代替正式详情/有值箱计划验证。
