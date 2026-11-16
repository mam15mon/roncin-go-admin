# 表头与现有展示设置跨浏览器保存设计

完整跨层契约以[父技术方案](../11-16-global-filter-settings/design.md)为准，本文件只记录本组边界。

## 实现边界与数据流

- 新增 components/ui/display-preferences 的类型映射、Provider 和 hook；依赖生成客户端、React Query 和认证上下文，不反向依赖 features/pages。
- OrganizationWorkspace keyed QueryClient 和 Antd App 内挂 Provider；区域队列在工作区级持有，在账号／组织切换销毁，写入附预期身份。
- 工作区首次批量读取一次、区域复用快照；刷新／重新登录取得新快照。重试只提交当前区域最新意图，旧失败回调不得复活旧 payload。
- 公共 useColumnSettings 和专有 feeColumnPreference／feeLedgerViewPreference 改用统一 TABLE 快照；保留业务字段定义和默认颜色所在层。
- ResponsiveTable 三类组件显式接收区域标识或受控宽度／更新接口；统一叶子列稳定 key 编码，不能持久化测量快照和辅助列宽。
- 密度合成官方 size/onSizeChange 并保留调用方回调；整体恢复删除 TABLE 配置，局部颜色恢复合并当前完整快照。
- 旧 localStorage 不双读／迁移／回退；配置未就绪先使用区域加载表现，读失败明确可重试，不新增全屏启动画面。

## 验证与回退

按 implement.md 实施后补定向验证；已验证分组提交。出现差距回到父方案明确修改，不靠本地存储或旧契约回退掩盖；保留已有配置及业务数据。
