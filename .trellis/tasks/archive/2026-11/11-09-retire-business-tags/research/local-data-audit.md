# 本地标签数据只读审计

- 日期：2026-11-09。
- 连接：仅从原开发checkout的私密环境读取，不记录连接串、凭据或资源内容。
- 执行：psql 事务 BEGIN READ ONLY；只读 COUNT；最后 ROLLBACK。未运行DDL/DML/迁移。

| 对象 | 数量 |
| --- | ---: |
| enterprise_tag_groups | 0 |
| enterprise_tags | 0 |
| order_enterprise_tags | 0 |
| order_fee_enterprise_tags | 0 |
| finance_bill_enterprise_tags | 0 |
| enterprise_resources WHERE resource_type = TAG | 0 |
| enterprise_resource_partners WHERE resource_type = TAG | 0 |

这是本次本地快照，不代表其他数据库。全站取消功能不自动等于永久删除数据授权；正式迁移删除标签专用表/元数据/关联的范围须在最终规划摘要明确，待用户后续批准。核心单据和历史审计、其他资源不在删除范围。
