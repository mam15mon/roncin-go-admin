# 企业资源停用状态分支

`web/src/pages/enterprise-resources/index.tsx` 编辑成功实体的 enabled 回填由 `record.enabled ?? true` 改为 `Boolean(record.enabled)`；创建分支仍显式 true，optional 过滤不变。

新增 `resource-enabled.test.tsx`：真实省略 enabled、显式 false、true 三种响应，验证 Switch 及最终 update 载荷一致。该文件与 resource-types 拆分，避免超过20秒。定向三文件11项通过，resource-enabled 13.22秒；真实省略字段调整后的单文件复跑3项通过。相关创建保存/类型隔离相邻用例已通过。
