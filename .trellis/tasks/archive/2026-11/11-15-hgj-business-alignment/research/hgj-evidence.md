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
- 首轮列表网络采集未读取请求体/响应体/敏感头，只能证明已发起请求及可见方法、路径、状态。后续运行DTO采样已读取选定请求/响应体，见 runtime-contract-observations.md；两阶段均不能证明内部数据库关系。
- 海管家列表含标签入口，与本仓库退役约束冲突，不采纳。

## 覆盖进度

- 四类列表和新增表单首轮已完成，字段对照见 `form-field-matrix.md`，控件出现不代表所有角色可用或固定必填。
- 四类真实已有详情已取证；内部ID与显示号纠正及DTO样本见 `runtime-contract-observations.md`。早期空详情记录已排除，不算有效完成。
- MAWB/HAWB 空制作表单已取证，五类单证目录及打印未知项见 `air-waybill-observations.md`；没有生成PDF。
- 已交付前端代码已作 REA 静态分析，规则/字段映射/端点及字节范围见 `frontend-bundle-rules.md`；语义图部分覆盖，不代表后端实现验证。
- 另五类列表及新增表单均已取证，当前详情样本覆盖陆运、铁路；详见 `other-business-matrix.md`。其他三类缺已有样本，不能声称完整业务流已验证。
- 未验证：服务端写入校验、数据库关系/锁/事务/并发、PDF输出、真实状态流转、跨角色/公司组合；这些均不得由前端或单样本推断为已验证。
- 原始证据部分存在网络事件覆盖不足或体积限制，具体报告均应保留局限；本轮没有将没有捕获某接口写成接口不存在。
