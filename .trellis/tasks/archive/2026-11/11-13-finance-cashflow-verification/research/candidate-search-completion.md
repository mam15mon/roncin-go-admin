# 核销候选分页检索复审补修验收（2026-11-14）

## 已完成

- 删除固定第一页200条的合并候选RPC与独占消息，替换为 `ListVerificationCashflowCandidates`、`ListVerificationBillCandidates`（各自独立Request/Response），地址为 `/api/v1/finance/verifications/cashflow-candidates`、`/api/v1/finance/verifications/bill-candidates`。
- 两端明确传page/pageSize/keyword；biz统一 `ValidListPagination` 校验1至200和关键字长度。data分别复用现有流水/账单的未核销/未结清SQL过滤、分页和total；保留同公司、方向、结算对象及币种。
- 金融列表共用Partner代码、中文、英文档案名称、别名与已有拼音检索键谓词；没有新增Schema、历史双读或分页拼全量。资金排序补ID作为最终稳定顺序。
- 两个候选表使用ProTable官方request独立分页/搜索，页大小20/50/100/200；跨页保留明确所选来源快照，搜索为空后仍能使用此前所选记录生成分配。不额外自动选择当前页外记录。
- 保留上轮review补的同步会话失效、勾选变化/全清分配保护、按资金+账单排序的幂等指纹；失败报文与真实必填标识缺失显式中文报错，ProtoJSON合法空列表零值保持正确。
- 真实跨页测试额外发现并修复旧问题：Decimal.js `isPositive()` 对正零返回true，导致多笔资金跨账单生成0金额分摊、0及空金额误通过校验；改用严格 `gt(0)`，补回归。

## 验证结果

- `go -C server test -p 32 ./internal/biz ./internal/service -run 'Verification.*Candidate'` PASS：公共分页0/1/200/201边界、筛选规范化与仓储错误原样返回、资金及账单创建权限范围、跨公司拒绝。
- 真实专用PostgreSQL `TestVerificationCandidatesSearchPaginationPostgres` PASS：各205条有效候选，第一页200、第二页5，total始终205、跨页无重复；业务编号单条定位；单位代码、中文、英文、别名、全拼及首字母检索；公司与结算对象隔离。
- 相邻 `TestVerificationCreationCandidatesBeyondSettledLimitPostgres` 及两项SQL范围/余额谓词测试PASS；`-v`日志无FAIL/SKIP。日志 `/tmp/verification-candidates-pg-final.log`。
- Vitest共4文件22例PASS：真实ProTable第二页与关键字请求、两个表跨页所选快照、搜索清空当前页后仍准确分配100、仅明确所选来源提交；原16例输入/迟到/幂等守卫；报文与快照契约、严格正数分配。日志 `/tmp/verification-candidates-web-pagination-tests.log`。
- `pnpm --dir web tsc`、改动文件Biome、`pnpm run check:architecture`、`git diff --check` PASS。
- API/OpenAPI/客户端均由 `make -C server api`、`pnpm run generate:web-client` 正式生成；生成器锁已释放。

## 文件边界与交付

产品变更仅候选Proto/生成物、核销biz/data/service方法、两类列表关键词与稳定排序、Workbench与本地候选/分配助手。相关旧调用方与测试一次性同步，无旧RPC兼容分支。未改独立核销/差额组合提交载荷，也未扩展建账组合事务。

未Git提交，由根会话统一独立review、完整web门禁及提交。专用库经 `/tmp/roncin-finance-20261114-pg/run-integration.py` 注入，未触碰开发库数据。
