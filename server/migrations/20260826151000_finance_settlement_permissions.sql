-- 货代业务结算子系统的独立权限；管理员角色默认补齐，其他角色按职责授权。
INSERT INTO "permissions" ("id", "created_at", "updated_at", "key", "name", "group", "description") VALUES
  (md5('system.finance.fee.read')::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system.finance.fee.read', '查看费用总台账', '费用管理 · 集运费用明细', '查看当前组织全部业务线的应收应付费用'),
  (md5('system.finance.bill.read')::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system.finance.bill.read', '查看账单', '费用管理 · 账单', '查看应收应付账单及明细'),
  (md5('system.finance.bill.create')::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system.finance.bill.create', '创建账单', '费用管理 · 账单', '按结算单位聚合已确认费用创建账单'),
  (md5('system.finance.bill.update')::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system.finance.bill.update', '编辑账单', '费用管理 · 账单', '编辑、撤回或作废未结清账单'),
  (md5('system.finance.bill.confirm')::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system.finance.bill.confirm', '确认账单', '费用管理 · 账单', '确认账单并锁定账单费用'),
  (md5('system.finance.invoice.read')::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system.finance.invoice.read', '查看开票记录', '费用管理 · 开票', '查看销项和进项发票台账'),
  (md5('system.finance.invoice.create')::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system.finance.invoice.create', '登记发票', '费用管理 · 开票', '登记发票并向账单分配开票金额'),
  (md5('system.finance.invoice.update')::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system.finance.invoice.update', '处理发票', '费用管理 · 开票', '开具、作废或红冲发票'),
  (md5('system.finance.cashflow.read')::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system.finance.cashflow.read', '查看收付', '费用管理 · 收付', '查看银行流水和收付款单'),
  (md5('system.finance.cashflow.create')::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system.finance.cashflow.create', '登记收付', '费用管理 · 收付', '登记银行流水和收付款单'),
  (md5('system.finance.cashflow.update')::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system.finance.cashflow.update', '处理收付', '费用管理 · 收付', '认领、确认或冲销收付款单'),
  (md5('system.finance.verification.read')::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system.finance.verification.read', '查看核销', '费用管理 · 核销', '查看账单与收付款核销记录'),
  (md5('system.finance.verification.create')::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system.finance.verification.create', '执行核销', '费用管理 · 核销', '将收付款金额分配到应收应付账单'),
  (md5('system.finance.verification.reverse')::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system.finance.verification.reverse', '反核销', '费用管理 · 核销', '按原因撤销有效核销分配'),
  (md5('system.finance.commission.read')::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system.finance.commission.read', '查看提成', '费用管理 · 提成', '查看单票毛利和人员提成结果'),
  (md5('system.finance.commission.manage')::uuid, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system.finance.commission.manage', '管理提成', '费用管理 · 提成', '维护提成规则并计算、确认提成')
ON CONFLICT ("key") DO UPDATE SET
  "updated_at" = EXCLUDED."updated_at",
  "name" = EXCLUDED."name",
  "group" = EXCLUDED."group",
  "description" = EXCLUDED."description";

INSERT INTO "role_permissions" ("role_id", "permission_id")
SELECT role."id", permission."id"
FROM "roles" AS role
JOIN "permissions" AS permission ON permission."key" LIKE 'system.finance.%'
WHERE role."code" = 'administrator'
ON CONFLICT ("role_id", "permission_id") DO NOTHING;
