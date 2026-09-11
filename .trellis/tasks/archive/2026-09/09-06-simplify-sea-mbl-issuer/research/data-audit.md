# 海运出口主单身份只读数据审计

## 审计时间与范围

- 日期：2026-09-06
- 数据源：当前本地开发 PostgreSQL，通过 `.env.local` 注入连接配置；文档不记录连接串或凭据。
- 执行方式：显式 `BEGIN READ ONLY`，查询完成后 `ROLLBACK`，未产生任何数据变更。

## 结果

| 检查项 | 数量 |
| --- | ---: |
| SE 订单 `carrier_id` 为空 | 0 |
| 活动 MBL 的运输执行 `carrier_id` 为空 | 0 |
| 活动关系中 Order carrier、TE carrier、MBL issuer 不一致 | 0 |
| 以 TE carrier 统一 issuer 后产生的 MBL 身份碰撞 | 0 |

## 结论

当前开发数据满足新业务不变量，不需要创建数据迁移、回填脚本、兼容分支或静默纠错逻辑。
