# ADR 0013: 订单业务锁与不可变单证历史

- 状态：已采纳
- 日期：2026-09-04（订单锁定与单证不可变版本上线）
- 主题：业务设计（订单域）

## 背景

货代业务中订单单证定稿后（出单、结算触发），继续随意编辑会造成财务与
单证事实脱节；而出错后的追责与换单历史必须可审计。只在订单行加一个
布尔「已锁定」不够：锁定的涟漪要波及费用、单证版本、共享 MBL 成员订单；
解锁也需要审批留痕而非口头解锁。

## 决策

- 业务锁事实只追加不覆盖：`order_lock_records(order_id, generation)` 唯一，
  每次锁定产生新代次；订单行保存当前锁状态（`locked_at/locked_by/
  lock_generation`），成功锁定与解锁都递增 `version`（配合 ADR 0005）。
- 统一内容写门禁：所有订单业务资料及子资源写入口在 Order `FOR UPDATE`
  后经 `ensureOrderBusinessContentEditable` 校验（活跃 + 未结案 + 未业务
  锁定），禁止另写第二套重复门禁；预览/按钮接口与执行同口径感知门禁，
  避免「预览通过、提交才被拒」。
- 门禁与生命周期分离：终止、结案等生命周期命令走各自专属状态校验，
  已锁定的 `DOCUMENT_RELEASED` 订单允许结案，不被内容门禁误杀。
- 解锁三分流：bootstrap admin → 管理员紧急；持锁权限用户 → 角色直解；
  普通编辑人 → 钉钉审批；审批候选人快照保存实际成员/角色与 DingTalk ID，
  回调实时复核资格。
- 不可变历史：MBL/HBL 版本快照、变更事件、拆票事件只追加；
  `current_version_id` 只能指向自身版本；SE 退关在同一事务按主键序锁定
  并结束活动 Link，超一条活动 Link 显式冲突回滚，不静默修复。

## 理由

- 锁定事实不可变 + 代次化使「锁过什么、谁解锁、何时」永久可审计，
  覆盖追责场景。
- 单一内容门禁杜绝「某个子资源忘了检查锁状态」的绕过路径；这是把
  安全检查做成公共谓词而非散落 if 的决策。
- 「先预览后执行 + 预读同口径门禁」让用户在提交前看到阻断原因，而不是
  填完表单才吃 409。

## 后果

- 新增订单写入口必须复用统一门禁与固定锁序；共享 MBL 写入前逐单校验
  全部活动成员订单。
- 历史外键保持 `ON DELETE NO ACTION`，配合 ADR 0003 的 Ent 同源声明。
- 契约全文见
  [../server/backend/order-lock-and-document-version.md](../server/backend/order-lock-and-document-version.md)，
  实现位于 [../../../server/internal/data/order_lock.go](../../../server/internal/data/order_lock.go)。
