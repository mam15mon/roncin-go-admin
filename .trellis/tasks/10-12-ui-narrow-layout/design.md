# 窄屏导航入口与订单布局修复：设计草案

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
