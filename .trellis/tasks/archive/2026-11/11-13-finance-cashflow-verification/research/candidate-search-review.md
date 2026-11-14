# 核销候选分页检索独立窄复审（2026-11-14）

## 范围与结论

以7addc1f6之后的候选组未提交差异为范围，审查两个候选RPC、领域校验、服务权限、仓储过滤、Partner关键词、官方ProTable请求、跨页选择快照及严格正数分配。保留所有其他任务改动，本轮没有修改产品代码或生成物，未提交Git。

未确认新增缺陷。本组已补齐上一轮发现的固定第一页200条问题；建账阶段F的只写编排仍属后续集成，不在本轮候选组宣称完成。

## 逐项核对

- Proto中资金/账单候选分别使用独立请求响应和静态路径，旧合并RPC已移除，未添加兼容分支。
- Service按verification.create的可写公司范围限定请求组织；Biz对两类候选共用明确page/pageSize、1～200及keyword长度校验，规范化币种与关键词，错误原样返回。
- 仓储同时保留公司、方向、结算对象、币种、确认状态与未结余额谓词；关键词OR是额外AND条件，Partner英文/拼音/别名子查询不绕过外层范围。Count与LIMIT使用同一筛选，资金排序补ID，账单已有ID末级排序。
- 工作台使用两个官方ProTable request，分页或搜索只请求当前表页；没有循环翻页或拼全量。选择快照仅包含明确勾选来源，换页/搜索后仍可分配，移除后不从历史缓存复活。
- 原先的范围/选择同步提交会话失效、空分配保护及按无序分配规范化幂等指纹均保留。自动分配/提交金额与公司判断从已选快照读取，服务端仍做最终余额复核。
- Decimal.js的正零判定已改为严格gt(0)，空、零与负值不能生成分摊或通过金额正数校验，多个资金覆盖多个账单不产生0金额行。
- 候选报文辅助明确拒绝undefined、success:false、非法列表与缺少ID/公司；success:true的ProtoJSON空列表可合法省略data/total，设计说明与根AGENTS及实现一致。

## 验证证据

本轮直接执行TypeScript、候选组7文件Biome及git diff --check，均PASS。核对实施代理最终日志：4个前端文件22例PASS，含真实ProTable跨页/搜索与提交、原16例输入会话、候选报文及严格正数回归；PG205条分页/关键词、余额过滤与SQL范围4项PASS，无FAIL/SKIP。未重复执行已通过的这些测试；根会话正在执行完整Web门禁。

日志：`/tmp/verification-candidate-independent-tsc.log`、`/tmp/verification-candidate-independent-biome.log`；已核对`/tmp/verification-candidates-web-pagination-tests.log`与`/tmp/verification-candidates-pg-final.log`。
