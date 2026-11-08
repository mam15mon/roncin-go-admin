# 海运退关恢复与显式补链

## 1. 适用范围

SE订单退关完成后恢复、已被旧恢复流程写成ACTIVE无活动关联的受控修复，以及恢复后详情/变动检测。历史Link始终不可变，恢复不等于复活旧行。

## 2. 入口

- 普通恢复沿用订单终止维度流转命令：TERMINATED→ACTIVE，expectedVersion及原因必填。
- 专用修复命令`server/cmd/repair-sea-restore-link`：显式`--organization-id`、`--order-id`、`--expected-version`、`--ended-link-id`、`--actor-id`、`--reason`；缺省只预检，加`--apply`才写业务数据。
- `Data.RepairSeaOrderRestoreLink`复用普通恢复关系校验，不对常规接口增加ACTIVE→ACTIVE兼容。

## 3. 原子契约与锁序

- 普通恢复唯一匹配同公司/订单、本次terminated_at、ENDED、ended_reason=订单退关的原关系；保留原MBL/TE/结构，创建新ID/started_at/version=1的ACTIVE Link。
- 完整成员Order按UUID排序加锁，再锁MBL、排序Link、TE和HBL；MBL锁后重验成员集合与原关系，变化返回冲突，不能先锁本票后补成员。
- 原主单不VOIDED，MBL/TE/Order船公司一致，原TE共享航程字段与订单事实一致；HOUSE存在有效唯一原HBL，DIRECT无有效HBL。批次DIRECT独占或全员HOUSE。
- 本票和原主单活动成员须未审核锁定；其他活动成员终止维度必须ACTIVE，CLOSED但未锁仍允许，不能把完结当成审核锁；目标订单仍须OPEN。恢复不覆盖本票最终目的港、共享正文或审核快照。
- 新ACTIVE、Order恢复/version+1、生命周期事件（引用新Link）和审计（原/新Link）同事务提交；缺一项整体回滚。MBL/TE/HBL实体及内容版本不因仅新增关联而改写，保持与现行加票契约一致。
- 取消退关沿用现有ACTIVE关联，不新建；空运仍从原主单显式恢复，不借此放宽。

## 4. 显式异常订单修复与错误矩阵

| 条件 | 行为 |
| --- | --- |
| 非同公司SE、非OPEN、已有ACTIVE关系、expectedVersion不符 | 拒绝，无部分写入 |
| 原主单作废/分单结构变化/共享成员审核锁/运输差异 | 明确拒绝，普通订单保持退关 |
| 同时恢复两次 | 最多一次建立新关系，另一次冲突 |
| 专用命令操作者没有实际有效read/transition授权 | 拒绝，不能仅用actorID做审计假冒授权 |
| 专用命令当前非ACTIVE无Link | 拒绝，不能代替普通状态流转 |
| 最新终止事件不是旧transition动作的TERMINATED→ACTIVE（不得已有关联reference）、前一事件不是完成退关、时间与指定Link不匹配或非唯一 | 拒绝，不能选任意最近Link |
| 恢复事件后已有其他新运输关系 | 拒绝补链 |
| 默认预检 | 可加锁验证，但业务行、版本、事件和审计均无写入 |
| --apply通过 | 新Link、Order version+1、独立修复审计；不伪造ACTIVE→ACTIVE事件、不改旧历史 |

## 5. 正常与异常示例

正确：退关后点恢复，原主单/航程仍有效，事务创建新关系并恢复；立即读取详情可用。独占最后一票退关后没有其他活动TE成员也可恢复，历史Link证明真实MBL/TE配对。

错误：只改状态、清terminated_at却不新建关联；或GET遇到无Link时自动修补；或修改旧Link为ACTIVE来抹除退关事实。

## 6. 必需验证

真实PostgreSQL退关→恢复→订单/单证/历史重读，DIRECT/HOUSE、新旧Link、取消/AE相邻路径、版本/并发/回滚、结构/作废/锁/差异门禁。专用修复证据、授权和默认预检无写入。前端明确原主单/原航程，精确识别本人恢复，不重置未保存输入或提交基线，不接纳更晚外部变化。

## 7. 现场执行约束

仅对明确授权的订单和版本运行，先保存预检报告、独立review及门禁，再--apply；之后正式API与只读SQL验证。不得直接SQL改业务状态、重造退关事件或批量修复。失败停下据实处理，不自动回退其他历史关联。
