# 技术设计

## 边界与数据流

角色只保存名称、代码、状态、工作台归属与权限关联。删除 Proto 角色 data_scope 与 Ent Role.data_scope，消除可配置入口及存储源。认证收集角色时已有组织树和工作台类型，按公司 organization / 系统 all 推导内部有效范围，不增加逐角色数据库查询。权限画像同样复用工作台解析，保证运行时与防提权一致。内部 DataScope、权限能力投影可继续表达有效边界，不能接受调用方指定。

## 合同变化

CreateRoleRequest/UpdateRoleRequest 删除字段3并 reserved；AdminRole 删除字段5并 reserved。Admin DataScope 枚举无剩余消费者后移除。前端删除对应表单、默认值、列表筛选和展示，并消费重新生成的请求类型。Auth 中角色摘要不再暴露角色级 data_scope；permission_capabilities 的有效范围保留以复用现有权限门禁。

## 迁移与安全

Ent Schema 是结构真相源，生成后提交对应迁移 SQL。迁移先停用 data_scope=self 的角色，保留其关联与记录，再删除列；不修改其他角色启停。系统角色固定 all 的行为是本期退役契约，公司始终 organization，系统工作台不得因此办理经营业务。保留同权限来源匹配、防越权、管理员保护、停用与事务封装。任何持久层的角色范围谓词改为角色归属工作台及成员资格约束，不以删除谓词扩大公司访问。

## 生成与验证

Proto → make -C server api → pnpm run generate:web-client；Ent → go generate 对应 package。后端实现者负责 server 全部及生成前端服务/常量，前端实现者只负责非生成 web 产品文件，生成后再类型校验。正式迁移通过隔离测试数据库验证增量与冷启动，不重置开发库。

## 回滚

本期无生产历史兼容要求。未发布前以 Git revert 回滚代码；删除列的迁移不可仅通过回退二进制恢复原列值，发布需代码与 Schema 配套。不得为了回滚保存第二份隐藏可配置范围。
