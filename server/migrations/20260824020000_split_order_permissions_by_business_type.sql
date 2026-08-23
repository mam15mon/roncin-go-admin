-- 将已展开的通用订单权限进一步拆分为海运出口、海运进口、空运出口和空运进口权限。
CREATE TEMP TABLE order_business_types (
  "code" varchar NOT NULL,
  "name" varchar NOT NULL
) ON COMMIT DROP;

INSERT INTO order_business_types ("code", "name") VALUES
  ('se', '海运出口（SE）'), ('si', '海运进口（SI）'),
  ('ae', '空运出口（AE）'), ('ai', '空运进口（AI）');

CREATE TEMP TABLE order_permission_definitions (
  "operation" varchar NOT NULL,
  "name" varchar NOT NULL,
  "resource" varchar NOT NULL,
  "description" varchar NOT NULL
) ON COMMIT DROP;

INSERT INTO order_permission_definitions ("operation", "name", "resource", "description") VALUES
  ('read', '查看订单', '订单', '查看当前数据范围内的订单'), ('create', '新建订单', '订单', '新建订单并检查业务编号'), ('update', '编辑订单', '订单', '修改订单基础与业务资料'), ('transition', '流转订单状态', '订单', '执行订单状态流转'),
  ('milestone.read', '查看里程碑', '里程碑', '查看订单里程碑'), ('milestone.set', '设置里程碑', '里程碑', '完成、跳过或重置订单里程碑'),
  ('attachment.read', '查看附件', '附件', '查看订单附件'), ('attachment.register', '登记附件', '附件', '登记订单附件元数据'),
  ('personnel.read', '查看协作人员', '协作人员', '查看订单协作人员'), ('personnel.assign', '指派协作人员', '协作人员', '指派订单协作人员'), ('personnel.remove', '移除协作人员', '协作人员', '移除订单协作人员'),
  ('container.read', '查看集装箱', '集装箱', '查看订单集装箱'), ('container.create', '新增集装箱', '集装箱', '新增订单集装箱'), ('container.update', '编辑集装箱', '集装箱', '修改订单集装箱'), ('container.delete', '删除集装箱', '集装箱', '删除订单集装箱'),
  ('cargo_item.read', '查看货物明细', '货物', '查看订单货物明细'), ('cargo_item.create', '新增货物明细', '货物', '新增订单货物明细'), ('cargo_item.update', '编辑货物明细', '货物', '修改订单货物明细'), ('cargo_item.delete', '删除货物明细', '货物', '删除订单货物明细'),
  ('shipping_document.read', '查看提单', '提单', '查看订单提单'), ('shipping_document.create', '新增提单', '提单', '新增订单提单'), ('shipping_document.update', '编辑提单', '提单', '修改订单提单'), ('shipping_document.transition', '流转提单状态', '提单', '执行提单状态流转'), ('shipping_document.delete', '删除提单', '提单', '删除订单提单'),
  ('abnormal_case.read', '查看异常事件', '异常', '查看订单异常事件'), ('abnormal_case.create', '登记异常事件', '异常', '登记订单异常事件'), ('abnormal_case.resolve', '处理异常事件', '异常', '解决或重新打开订单异常事件'), ('abnormal_case.delete', '删除异常事件', '异常', '删除订单异常事件'),
  ('release_pod.read', '查看放货凭证', '放货', '查看订单放货凭证'), ('release_pod.create', '新增放货凭证', '放货', '新增订单放货凭证'), ('release_pod.update', '编辑放货凭证', '放货', '修改订单放货凭证'), ('release_pod.transition', '流转放货状态', '放货', '执行放货状态流转'), ('release_pod.delete', '删除放货凭证', '放货', '删除订单放货凭证');

INSERT INTO "permissions" ("id", "created_at", "updated_at", "key", "name", "group", "description")
SELECT md5(format('business.order.%s.%s', business_type."code", definition."operation"))::uuid,
       CURRENT_TIMESTAMP, CURRENT_TIMESTAMP,
       format('business.order.%s.%s', business_type."code", definition."operation"),
       definition."name",
       format('订单管理 · %s · %s', business_type."name", definition."resource"),
       definition."description"
FROM order_business_types AS business_type
CROSS JOIN order_permission_definitions AS definition
ON CONFLICT ("key") DO UPDATE SET
  "updated_at" = EXCLUDED."updated_at",
  "name" = EXCLUDED."name",
  "group" = EXCLUDED."group",
  "description" = EXCLUDED."description";

INSERT INTO "role_permissions" ("role_id", "permission_id")
SELECT current_grant."role_id", new_permission."id"
FROM "role_permissions" AS current_grant
JOIN "permissions" AS old_permission ON old_permission."id" = current_grant."permission_id"
JOIN order_permission_definitions AS definition ON old_permission."key" = format('business.order.%s', definition."operation")
CROSS JOIN order_business_types AS business_type
JOIN "permissions" AS new_permission ON new_permission."key" = format('business.order.%s.%s', business_type."code", definition."operation")
ON CONFLICT ("role_id", "permission_id") DO NOTHING;

INSERT INTO "role_permissions" ("role_id", "permission_id")
SELECT administrator_role."id", permission."id"
FROM "roles" AS administrator_role
JOIN "permissions" AS permission ON permission."key" ~ '^business[.]order[.](se|si|ae|ai)[.]'
WHERE administrator_role."code" = 'administrator'
ON CONFLICT ("role_id", "permission_id") DO NOTHING;

DELETE FROM "permissions"
WHERE "key" LIKE 'business.order.%'
  AND "key" !~ '^business[.]order[.](se|si|ae|ai)[.]';
