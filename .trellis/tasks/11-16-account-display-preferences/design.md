# 账号展示偏好契约与服务端持久化设计

完整跨层契约以[父技术方案](../11-16-global-filter-settings/design.md)为准，本文件只记录本组边界。

## 实现边界与数据流

- 新增 server/api/auth/v1/display_preference.proto，按现有 AUTHENTICATED 本人接口注册 HTTP／gRPC／访问规则；不增加业务权限码。
- 新增 biz 领域结构／用例／仓储接口，service 只 DTO 转换，data 使用 Data.client(ctx) 的 Ent 原子 upsert；独立 DisplayPreference 表以 user_id＋organization_id＋kind＋region_key 唯一。
- FILTER 载荷包含顺序、隐藏字段、各字段 span、收起数量；TABLE 包含顺序、隐藏列、冻结、叶子列宽、密度及可选高级配色。Proto 用存在性清晰的消息／optional 处理未配置和显式空。
- 删除仅作用目标区域／类型，幂等恢复当前业务默认。任何多步事务改用统一 WithTx，不能手写事务模板。
- 部署新增表迁移后上线同时更新契约；不操作真实数据库清空，不额外建设通用配置平台。

## 验证与回退

按 implement.md 实施后补定向验证；已验证分组提交。出现差距回到父方案明确修改，不靠本地存储或旧契约回退掩盖；保留已有配置及业务数据。
