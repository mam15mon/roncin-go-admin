# 海管家观察证据索引

北京时间 2026-11-16。原始记录含业务数据，仅保留本机，不提交仓库。

原始目录：`/home/vince/.local/share/rea/hgj/order-research`。已登录同一公司工作区；本轮仅导航和读取，未提交业务写入。

| 模块 | 实测页面 | REA 证据 | 原始文件 SHA-256 |
| --- | --- | --- | --- |
| 海运出口 | `https://eyun.hgj.com/order/sea_export` | `ev_9c30968c89b2c1543bbaf9ed6120eb21851f2479ae9bcbd8ad743c4a86b063bf` | `13faad079df3244f271c67601c49621717bb7bac3b78b8d62feaaedbcfe524ff` |
| 海运进口 | `https://eyun.hgj.com/order/sea_import` | `ev_d3e52191a3677c1c789c3d247296e5168ca74d4de6d7c324ca603b7227c376b4` | `de629179d62463934e8be70e955c265ef90e80a6177f6136eb5575fa344fab29` |
| 空运出口 | `https://eyun.hgj.com/order/air_export` | `ev_f25a61c1a6c8113ade0c1acab01affe6d9cebb031b880b66af4520b858fb7397` | `d30ded025e6b59c107deea40090d768ef47a600697faac7083ec67956b382657` |
| 空运进口 | `https://eyun.hgj.com/order/air_import` | `ev_42d52cedc3d762c6eebf5ce151f82d5efe7941d0d64038e2b0853dd83ad97fe2` | `e61a9b3ad1566b59309d60c1cd62b818b9c7b030228e2b3ba121d565405c466e` |

## 已观察事实

- 四类列表均可打开；菜单另有陆运、铁路、内贸水运、自拼和物流园报关。
- 列表出现新增、复制、导入、导出单证、完结、退关、锁定等入口；入口存在不证明权限、写入效果或完整状态机。
- 观察到跨域 API 主机 `anoingress.hgj.com`；列表使用 `/saas-service/order/index`，异常使用 `/saas-service/order/getAbnormalList`；海运出口还出现 `/saas-service/order/orderTotal`。
- 查询设置使用 `/saas-service/custom/query/list`，流程候选使用 `/saas-service/order/status/child-list/...`。
- 当前网络采集未读取请求体/响应体/敏感头，只能证明已发起请求及其可见方法、路径、状态，尚不能证明 DTO 或内部数据库关系。
- 海管家列表含标签入口，与本仓库退役约束冲突，不采纳。

## 覆盖进度

- 已完成：四类列表首轮导航与控制项、网络元数据取证；现有代码能力盘点。
- 待完成：四类新建表单首轮已完成（见 create-form-observations.md）；仍待编辑/默认值、主分单与格式、运输与状态、费用联动、打印导出和权限。
- 待完成：另外五类业务的完整调研。
- 未验证：服务端写入校验、事务/并发内部机制、真实单据提交后的规则；这些均不得从前端字符串推断为已验证。
