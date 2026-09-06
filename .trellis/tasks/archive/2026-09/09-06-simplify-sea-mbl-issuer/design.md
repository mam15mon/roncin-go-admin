# 技术设计：简化海运出口主单签发方录入

## 1. 设计目标与边界

本次只改变 MBL 的录入责任：业务人员维护船公司和 MBL 主单号，系统维护
`SeaMasterBill.issuer_partner_id`。不改变 MBL 的持久化身份、共享关系、版本快照和
锁定机制，也不触碰 HBL 签发主体。

采用“保留现有契约字段、前端派生、后端建立不变量”的最小方案，不删除 Proto 字段，
因此不需要 API/OpenAPI/Web Client 生成。这样既能立即减负，又避免为内部持久化字段做
无业务收益的大范围契约迁移。

## 2. 权威数据与不变量

新写入的海运出口数据必须满足：

```text
Order.carrier_id
  = SeaTransportExecution.carrier_id
  = SeaMasterBill.issuer_partner_id
```

- 业务权威输入是订单或目标运输执行的 `carrier_id`。
- MBL `issuer_partner_id` 继续保存，用于现有唯一索引、候选查询、版本和锁定快照，但不再
  是用户独立输入。
- 前端仍可在现有请求 DTO 中发送 `issuer_partner_id = carrier_id`，以避免本阶段修改契约；
  后端必须自行从 carrier 规范化或验证相等，不能信任客户端派生值。
- HBL 的 issuer 不满足上述等式，继续按 SELF/CUSTOMER/OTHER 独立校验。

## 3. 跨层数据流

### 3.1 订单新建与详情保存

```text
用户选择船公司 + 输入 MBL 号
→ 前端 payload 将 MBL issuer 设为 carrier
→ biz 要求 SE carrier 非空并规范化 MBL issuer=carrier
→ data 锁内验证 carrier 角色、共享 MBL 身份和航程
→ 写入 Order、TransportExecution、SeaMasterBill
```

- `SeaBasicInfoSection` 给 SE 的船公司增加必填规则。
- `SeaTransportSection` 只展示 MBL 主单号，删除 issuer 选择器；候选请求用 carrier 同时填入
  现有 `issuerPartnerId` 与 `carrierId`。
- 详情页不再回填 issuer 表单字段。单成员时，MBL 号或 carrier 相对现有共享 MBL 发生变化，
  继续要求更正原因；多成员时，MBL 号和 carrier 都在 UI 禁用，后端仍锁内拒绝绕过。

### 3.2 共享 MBL 候选

候选查询的外部契约暂不改变：现有请求仍含 `issuer_partner_id` 和可选 `carrier_id`。前端将
当前船公司同时用于两字段；服务端继续按组织、内部 issuer、规范化主单号定位，并核对候选
运输执行 carrier 与当前船公司及其他航程字段。命中只返回候选，写入仍要求用户明确确认并
携带候选版本。

### 3.3 拆票

- CURRENT 目标不增加输入，继续沿用当前共享 MBL。
- NEW 目标删除 issuer 输入，船公司默认取来源主单 carrier，并允许修改；提交前必须非空。
- CANDIDATE 目标使用当前选择/默认的船公司执行候选匹配；匹配成功后提交候选 MBL 与 TE 的
  ID/版本，内部 issuer 取候选权威事实，不让用户再选择。
- 为保持现有 DTO，前端对 NEW/CANDIDATE 仍派生 `issuerPartnerId = carrierId`；biz/data 对
  NEW 强制 carrier 非空并以其维护 issuer，对 CANDIDATE 锁内验证候选 MBL issuer 与 TE
  carrier 一致。

### 3.4 整票改配

- 删除“发单人 / 船代”，将现有“承运人 / 船东”统一显示为“船公司”并设为必填。
- 默认带入当前 MBL 的 carrier，允许为 NEW 或匹配前的 CANDIDATE 修改。
- NEW 内部使用同一 carrier 创建运输执行和 MBL issuer；CANDIDATE 使用候选权威身份与版本。

## 4. UI 展示边界

- 新建、编辑、拆票、改配不再显示 MBL 签发方输入。
- MBL 摘要中与船公司重复的“签发主体/实际签发主体/发单人”展示删除或改为现有船公司展示，
  避免业务人员误以为仍需维护两个概念。
- HBL 区域的“签发主体”及其选择控件保留，测试必须明确区分 MBL 与 HBL。
- 本次不移动 MBL 主单号，不重排整个海运表单。

## 5. 服务端校验与错误行为

| 条件 | 处理 |
| --- | --- |
| SE 新建/更新缺少 carrier | 400，复用订单或 MBL 参数错误，不创建部分数据 |
| MBL 号为空或格式非法 | 保持现有 400 `SEA_MASTER_BILL_INVALID_ARGUMENT` |
| 请求 issuer 缺失或与 carrier 不同 | 服务端以 carrier 规范化；禁止将不同 issuer 写入数据库 |
| NEW 拆票/改配缺少 carrier | 对应 400 InvalidArgument |
| 候选 MBL issuer 与候选 TE carrier 不一致 | 视为非法/冲突数据并阻断，不静默修复 |
| 共享 MBL 单票尝试修改 carrier 或 MBL 号 | 保持现有共享结构/冲突错误并整体回滚 |
| HBL issuer 与 carrier 不同 | 正常，继续按 HBL 自身来源规则处理 |

选择“服务端以 carrier 规范化请求 issuer”是因为 issuer 已不再是用户输入；客户端旧值不应成为
第二个真相。但数据库中既有不一致数据不得在普通保存时静默批量修复，实施前先只读审计。

## 6. 现有数据只读审计

实施阶段使用当前开发数据库做只读查询，统计：

- SE 订单 `carrier_id` 为空；
- 活动 MBL 的运输执行 `carrier_id` 为空；
- Order carrier、TE carrier、MBL issuer 三者不一致；
- 若把 issuer 统一为 carrier 后，是否会碰撞现有
  `(organization_id, issuer_partner_id, normalized_master_no)` 唯一身份。

若均为零，记录结果后不创建迁移。若存在任一异常，停止数据修复并向用户报告；代码仍不得
加入自动回退或兼容分支。

## 7. 并发、事务与兼容性

- 沿用现有 Order → MBL → Link → TransportExecution 固定锁序、候选版本和订单乐观版本；
  不新增事务实现。
- 多成员共享 MBL 的 carrier 是共享事实，必须锁定全部活动成员并重验，不能只更新当前订单。
- MBL issuer 字段、索引、版本和响应字段保留，因此既有只读消费者不受影响。
- 非 SE 订单和 HBL 写入路径不应用 carrier=issuer 规则。

## 8. 验证策略

- 前端：海运模板、创建 payload、详情 payload、拆票、改配弹窗定向测试。
- Biz：SE carrier 必填、issuer 规范化、NEW/CANDIDATE 目标校验和非 SE/HBL 相邻路径。
- Data/PostgreSQL：创建/更新、共享候选、单成员改船公司、共享 MBL 阻断、拆票和改配新 MBL
  的三方 carrier/issuer 一致性。
- 开发阶段只跑受影响最小集；任务最终验收时按风险各执行一次完整 Go/Web 门禁。无契约和
  Schema 变化时不运行生成命令或制造生成物差异。
