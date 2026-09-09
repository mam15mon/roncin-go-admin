# 订单模板草稿生命周期证据

## 当前调用方

- `web/src/pages/orders/new.tsx`：显式传 `tabKey`、`draftScope`，未使用受控 dirty；
- `web/src/pages/orders/detail.tsx`：显式传 `tabKey`、`draftScope`、受控 dirty，并在页面计算
  `draftKey` 处理保存、显式刷新与底部重置。

## 当前运行契约

- `OrderFormTemplate.tsx` 从 `window.location.pathname` 派生 pathname，再与 props 中的
  `tabKey`、`draftScope` 计算 `draftKey`；
- `window.location.pathname` 不包含 search/hash；当前 Umi 配置未启用 hash 路由，
  `config.ts` 的 `hash: true` 是静态资源文件名 hash；
- `useOrderDetailData.loadData` 捕获请求异常、清空当前 order、设置 error 后正常结束；
- `useOrderLockState.refresh` 捕获请求异常、更新错误快照并返回 `null`；
- `OrderFormTemplate` 的 `initialValues` 只负责 Form 初始化，现有 effect 还承担初始
  loading/readonly 解除后恢复草稿的职责。

## 已确认取舍

- 页面提供规范业务身份，不直接构造或理解持久化 key 格式；
- 模板独占 internal dirty、草稿读写、清理和关闭守卫；
- 外部显式重置通过独立 actions ref，不扩展第三方 `ProFormInstance`；
- 草稿恢复使用 layout effect，不定义浅合并或深合并规则；
- 保存成功与后台刷新解耦，不增加通用队列或重试抽象；
- 快速上线阶段不兼容旧 props，不迁移历史 sessionStorage。
