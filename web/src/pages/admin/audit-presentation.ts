type AuditActionPresentation = {
  title: string;
  category: string;
  color: string;
  objectType?: string;
};

const actionPresentations: Record<string, AuditActionPresentation> = {
  'auth.login': { title: '登录系统', category: '登录安全', color: 'blue' },
  'auth.logout': { title: '退出系统', category: '登录安全', color: 'default' },
  'auth.getuserinfo': {
    title: '获取当前账号信息',
    category: '登录安全',
    color: 'default',
  },
  'auth.organization.switch': {
    title: '切换当前企业',
    category: '登录安全',
    color: 'blue',
    objectType: '企业',
  },
  'auth.dingtalk.login': {
    title: '使用钉钉登录',
    category: '登录安全',
    color: 'blue',
  },
  'auth.dingtalk.register': {
    title: '提交钉钉注册',
    category: '账号注册',
    color: 'cyan',
  },
  'auth.wecom.login': {
    title: '使用企微登录',
    category: '登录安全',
    color: 'blue',
  },
  'auth.wecom.register': {
    title: '提交企微注册',
    category: '账号注册',
    color: 'cyan',
  },

  'admin.organization.create': {
    title: '创建企业',
    category: '系统管理',
    color: 'purple',
    objectType: '企业',
  },
  'admin.organization.update': {
    title: '修改企业资料',
    category: '系统管理',
    color: 'purple',
    objectType: '企业',
  },
  'admin.role.create': {
    title: '创建角色',
    category: '权限管理',
    color: 'geekblue',
    objectType: '角色',
  },
  'admin.role.update': {
    title: '修改角色权限',
    category: '权限管理',
    color: 'geekblue',
    objectType: '角色',
  },
  'admin.user.create': {
    title: '创建人员账号',
    category: '人员管理',
    color: 'purple',
    objectType: '人员',
  },
  'admin.user.update': {
    title: '修改人员资料',
    category: '人员管理',
    color: 'purple',
    objectType: '人员',
  },
  'admin.user.terminate': {
    title: '办理人员离职',
    category: '人员管理',
    color: 'volcano',
    objectType: '人员',
  },
  'admin.user.membership.create': {
    title: '添加企业成员',
    category: '人员管理',
    color: 'purple',
    objectType: '人员',
  },
  'admin.user.membership.update': {
    title: '调整组织和角色',
    category: '权限管理',
    color: 'geekblue',
    objectType: '人员',
  },
  'admin.user.membership.delete': {
    title: '移出企业成员',
    category: '人员管理',
    color: 'volcano',
    objectType: '人员',
  },
  'admin.user.password.reset': {
    title: '重置人员密码',
    category: '登录安全',
    color: 'orange',
    objectType: '人员',
  },
  'admin.user.dingtalk.authorize': {
    title: '完成钉钉账号授权',
    category: '权限管理',
    color: 'geekblue',
    objectType: '人员',
  },
  'admin.user.wecom.authorize': {
    title: '完成企微账号授权',
    category: '权限管理',
    color: 'geekblue',
    objectType: '人员',
  },

  'background_task.requeue': {
    title: '重新执行后台任务',
    category: '系统任务',
    color: 'gold',
    objectType: '后台任务',
  },

  'master_data.create': {
    title: '新增基础资料',
    category: '基础资料',
    color: 'cyan',
    objectType: '基础资料',
  },
  'master_data.update': {
    title: '修改基础资料',
    category: '基础资料',
    color: 'cyan',
    objectType: '基础资料',
  },
  'master_data.import': {
    title: '导入基础资料',
    category: '基础资料',
    color: 'cyan',
    objectType: '基础资料',
  },
  'number_rule.create': {
    title: '新增编号规则',
    category: '基础资料',
    color: 'cyan',
    objectType: '编号规则',
  },
  'number_rule.update': {
    title: '修改编号规则',
    category: '基础资料',
    color: 'cyan',
    objectType: '编号规则',
  },
  'milestone_template.create': {
    title: '新增里程碑模板',
    category: '基础资料',
    color: 'cyan',
    objectType: '里程碑模板',
  },
  'milestone_template.publish': {
    title: '发布里程碑模板',
    category: '基础资料',
    color: 'cyan',
    objectType: '里程碑模板',
  },
  'milestone_template.set_default': {
    title: '设置默认里程碑模板',
    category: '基础资料',
    color: 'cyan',
    objectType: '里程碑模板',
  },
  'port.create': {
    title: '新增港口',
    category: '基础资料',
    color: 'cyan',
    objectType: '港口',
  },
  'port.update': {
    title: '修改港口',
    category: '基础资料',
    color: 'cyan',
    objectType: '港口',
  },
  'airport.create': {
    title: '新增机场',
    category: '基础资料',
    color: 'cyan',
    objectType: '机场',
  },
  'airport.update': {
    title: '修改机场',
    category: '基础资料',
    color: 'cyan',
    objectType: '机场',
  },
  'airline.create': {
    title: '新增航空公司',
    category: '基础资料',
    color: 'cyan',
    objectType: '航空公司',
  },
  'airline.update': {
    title: '修改航空公司',
    category: '基础资料',
    color: 'cyan',
    objectType: '航空公司',
  },
  'shipping_line.create': {
    title: '新增船公司',
    category: '基础资料',
    color: 'cyan',
    objectType: '船公司',
  },
  'shipping_line.update': {
    title: '修改船公司',
    category: '基础资料',
    color: 'cyan',
    objectType: '船公司',
  },
  'milestone.set': {
    title: '设置里程碑',
    category: '基础资料',
    color: 'cyan',
    objectType: '里程碑',
  },

  'order.create': {
    title: '创建订单',
    category: '订单管理',
    color: 'blue',
    objectType: '订单',
  },
  'order.update': {
    title: '修改订单',
    category: '订单管理',
    color: 'blue',
    objectType: '订单',
  },
  'order.flow.transition': {
    title: '变更订单状态',
    category: '订单管理',
    color: 'blue',
    objectType: '订单',
  },
  'order.closure.transition': {
    title: '变更订单关账状态',
    category: '订单管理',
    color: 'blue',
    objectType: '订单',
  },
  'order.termination.transition': {
    title: '变更订单终止状态',
    category: '订单管理',
    color: 'orange',
    objectType: '订单',
  },
  'order.personnel.assign': {
    title: '分配订单人员',
    category: '订单管理',
    color: 'blue',
    objectType: '订单',
  },
  'order.personnel.remove': {
    title: '移除订单人员',
    category: '订单管理',
    color: 'blue',
    objectType: '订单',
  },
  'order.milestone.set': {
    title: '更新订单节点',
    category: '订单管理',
    color: 'blue',
    objectType: '订单',
  },
  'order.fee.add': {
    title: '新增订单费用',
    category: '订单费用',
    color: 'gold',
    objectType: '订单',
  },
  'order.fee.update': {
    title: '修改订单费用',
    category: '订单费用',
    color: 'gold',
    objectType: '订单',
  },
  'order.fee.remove': {
    title: '删除订单费用',
    category: '订单费用',
    color: 'volcano',
    objectType: '订单',
  },
  'order.fee.confirm': {
    title: '确认订单费用',
    category: '订单费用',
    color: 'gold',
    objectType: '订单',
  },
  'order.fee.reopen': {
    title: '重新打开订单费用',
    category: '订单费用',
    color: 'gold',
    objectType: '订单',
  },
  'order.container.add': {
    title: '新增订单箱量',
    category: '订单管理',
    color: 'blue',
    objectType: '订单',
  },
  'order.container.update': {
    title: '修改订单箱量',
    category: '订单管理',
    color: 'blue',
    objectType: '订单',
  },
  'order.container.remove': {
    title: '删除订单箱量',
    category: '订单管理',
    color: 'volcano',
    objectType: '订单',
  },
  'order.cargo_item.add': {
    title: '新增货物信息',
    category: '订单管理',
    color: 'blue',
    objectType: '订单',
  },
  'order.cargo_item.update': {
    title: '修改货物信息',
    category: '订单管理',
    color: 'blue',
    objectType: '订单',
  },
  'order.cargo_item.remove': {
    title: '删除货物信息',
    category: '订单管理',
    color: 'volcano',
    objectType: '订单',
  },
  'order.attachment.register': {
    title: '上传订单附件',
    category: '订单管理',
    color: 'blue',
    objectType: '订单',
  },
  'order.abnormal_case.mark': {
    title: '标记订单异常',
    category: '订单异常',
    color: 'orange',
    objectType: '订单',
  },
  'order.abnormal_case.resolve': {
    title: '解决订单异常',
    category: '订单异常',
    color: 'orange',
    objectType: '订单',
  },
  'order.abnormal_case.remove': {
    title: '删除订单异常',
    category: '订单异常',
    color: 'volcano',
    objectType: '订单',
  },
  'order.shipping_document.add': {
    title: '新增航运单证',
    category: '订单单证',
    color: 'cyan',
    objectType: '订单',
  },
  'order.shipping_document.update': {
    title: '修改航运单证',
    category: '订单单证',
    color: 'cyan',
    objectType: '订单',
  },
  'order.shipping_document.transition': {
    title: '变更航运单证状态',
    category: '订单单证',
    color: 'cyan',
    objectType: '订单',
  },
  'order.shipping_document.remove': {
    title: '删除航运单证',
    category: '订单单证',
    color: 'volcano',
    objectType: '订单',
  },
  'order.release_pod.add': {
    title: '新增放货单证',
    category: '订单单证',
    color: 'cyan',
    objectType: '订单',
  },
  'order.release_pod.update': {
    title: '修改放货单证',
    category: '订单单证',
    color: 'cyan',
    objectType: '订单',
  },
  'order.release_pod.transition': {
    title: '变更放货状态',
    category: '订单单证',
    color: 'cyan',
    objectType: '订单',
  },
  'order.release_pod.remove': {
    title: '删除放货单证',
    category: '订单单证',
    color: 'volcano',
    objectType: '订单',
  },

  'partner.create': {
    title: '新增往来单位',
    category: '往来单位',
    color: 'green',
    objectType: '往来单位',
  },
  'partner.update': {
    title: '修改往来单位',
    category: '往来单位',
    color: 'green',
    objectType: '往来单位',
  },
  'partner.import': {
    title: '导入往来单位',
    category: '往来单位',
    color: 'green',
    objectType: '往来单位',
  },
  'partner.account.create': {
    title: '新增单位账户',
    category: '往来单位',
    color: 'green',
    objectType: '往来单位',
  },
  'partner.account.update': {
    title: '修改单位账户',
    category: '往来单位',
    color: 'green',
    objectType: '往来单位',
  },
  'partner.attachment.register': {
    title: '上传单位附件',
    category: '往来单位',
    color: 'green',
    objectType: '往来单位',
  },
  'partner.contract.create': {
    title: '新增单位合同',
    category: '往来单位',
    color: 'green',
    objectType: '往来单位',
  },
  'partner.contract.update': {
    title: '修改单位合同',
    category: '往来单位',
    color: 'green',
    objectType: '往来单位',
  },
  'partner.invoice_profile.create': {
    title: '新增开票资料',
    category: '往来单位',
    color: 'green',
    objectType: '往来单位',
  },
  'partner.invoice_profile.update': {
    title: '修改开票资料',
    category: '往来单位',
    color: 'green',
    objectType: '往来单位',
  },
  'partner.settlement_rule.create': {
    title: '新增结算规则',
    category: '往来单位',
    color: 'green',
    objectType: '往来单位',
  },
  'partner.settlement_rule.update': {
    title: '修改结算规则',
    category: '往来单位',
    color: 'green',
    objectType: '往来单位',
  },
  'partner.shipping_preset.create': {
    title: '新增运输预设',
    category: '往来单位',
    color: 'green',
    objectType: '往来单位',
  },
  'partner.shipping_preset.update': {
    title: '修改运输预设',
    category: '往来单位',
    color: 'green',
    objectType: '往来单位',
  },
  'partner.supplier_blacklist.set': {
    title: '设置供应商黑名单',
    category: '往来单位',
    color: 'orange',
    objectType: '往来单位',
  },

  'finance.bill.create': {
    title: '创建账单',
    category: '财务管理',
    color: 'gold',
    objectType: '账单',
  },
  'finance.bill.update': {
    title: '修改账单',
    category: '财务管理',
    color: 'gold',
    objectType: '账单',
  },
  'finance.bill.confirm': {
    title: '确认账单',
    category: '财务管理',
    color: 'gold',
    objectType: '账单',
  },
  'finance.bill.cancel': {
    title: '取消账单',
    category: '财务管理',
    color: 'volcano',
    objectType: '账单',
  },
  'finance.bill_batch.create': {
    title: '创建账单批次',
    category: '财务管理',
    color: 'gold',
    objectType: '账单批次',
  },
  'finance.bill_batch.confirm': {
    title: '确认账单批次',
    category: '财务管理',
    color: 'gold',
    objectType: '账单批次',
  },
  'finance.cashflow.create': {
    title: '登记收付款',
    category: '财务管理',
    color: 'gold',
    objectType: '收付款记录',
  },
  'finance.cashflow.confirm': {
    title: '确认收付款',
    category: '财务管理',
    color: 'gold',
    objectType: '收付款记录',
  },
  'finance.cashflow.cancel': {
    title: '取消收付款',
    category: '财务管理',
    color: 'volcano',
    objectType: '收付款记录',
  },
  'finance.invoice.create': {
    title: '创建发票',
    category: '财务管理',
    color: 'gold',
    objectType: '发票',
  },
  'finance.invoice.issue': {
    title: '开具发票',
    category: '财务管理',
    color: 'gold',
    objectType: '发票',
  },
  'finance.invoice.cancel': {
    title: '作废发票',
    category: '财务管理',
    color: 'volcano',
    objectType: '发票',
  },
  'finance.invoice.red_flush': {
    title: '红冲发票',
    category: '财务管理',
    color: 'volcano',
    objectType: '发票',
  },
  'finance.verification.create': {
    title: '新增核销',
    category: '财务管理',
    color: 'gold',
    objectType: '核销记录',
  },
  'finance.verification.reverse': {
    title: '反核销',
    category: '财务管理',
    color: 'volcano',
    objectType: '核销记录',
  },
  'finance.commission.create': {
    title: '生成业务提成',
    category: '财务管理',
    color: 'gold',
    objectType: '业务提成',
  },
  'finance.commission_adjustment.create': {
    title: '调整业务提成',
    category: '财务管理',
    color: 'gold',
    objectType: '业务提成',
  },
  'finance.commission_rule.create': {
    title: '新增提成规则',
    category: '财务设置',
    color: 'purple',
    objectType: '提成规则',
  },
  'finance.commission_rule.update': {
    title: '修改提成规则',
    category: '财务设置',
    color: 'purple',
    objectType: '提成规则',
  },
  'finance.billing_unit.create': {
    title: '新增开票单位',
    category: '财务设置',
    color: 'purple',
    objectType: '开票单位',
  },
  'finance.billing_unit.update': {
    title: '修改开票单位',
    category: '财务设置',
    color: 'purple',
    objectType: '开票单位',
  },
  'finance.taxable_service.create': {
    title: '新增应税服务',
    category: '财务设置',
    color: 'purple',
    objectType: '应税服务',
  },
  'finance.taxable_service.update': {
    title: '修改应税服务',
    category: '财务设置',
    color: 'purple',
    objectType: '应税服务',
  },
  'finance.fee_setting.create': {
    title: '新增费用设置',
    category: '财务设置',
    color: 'purple',
    objectType: '费用设置',
  },
  'finance.fee_setting.update': {
    title: '修改费用设置',
    category: '财务设置',
    color: 'purple',
    objectType: '费用设置',
  },
  'finance.exchange_rate.create': {
    title: '新增汇率',
    category: '财务设置',
    color: 'purple',
    objectType: '汇率',
  },
  'finance.exchange_rate.update': {
    title: '修改汇率',
    category: '财务设置',
    color: 'purple',
    objectType: '汇率',
  },
  'finance.exchange_rate.disable': {
    title: '停用汇率',
    category: '财务设置',
    color: 'volcano',
    objectType: '汇率',
  },
  'finance.exchange_rate.import.preview': {
    title: '预览汇率导入',
    category: '财务设置',
    color: 'purple',
    objectType: '汇率',
  },
  'finance.exchange_rate.import.confirm': {
    title: '确认导入汇率',
    category: '财务设置',
    color: 'purple',
    objectType: '汇率',
  },
  'finance.exchange_rate.time_standard.update': {
    title: '修改汇率时间标准',
    category: '财务设置',
    color: 'purple',
    objectType: '汇率设置',
  },
  'finance.exchange_rate.custom_setting.update': {
    title: '修改自定义汇率设置',
    category: '财务设置',
    color: 'purple',
    objectType: '汇率设置',
  },
  'finance.custom_setting.billed_fee_edit.update': {
    title: '修改已开票费用设置',
    category: '财务设置',
    color: 'purple',
    objectType: '财务设置',
  },
};

const resourceTypeLabels: Record<string, string> = {
  organization: '企业',
  user: '人员',
  role: '角色',
  order: '订单',
  partner: '往来单位',
  background_task: '后台任务',
};

const detailLabels: Record<string, string> = {
  'order.no': '订单编号',
  'partner.code': '单位编号',
  'master_data.code': '资料编号',
  'master_data.kind': '资料类型',
  'milestone_template.code': '模板编号',
  'fee.code': '费用编号',
  standard_code: '业务代码',
  resource_id: '业务对象 ID',
  role_id: '角色 ID',
  organization_id: '企业 ID',
  kind: '资料类型',
  code: '业务编号',
  status: '状态',
  reason: '原因',
};

export function auditActionPresentation(
  action?: string,
): AuditActionPresentation {
  if (action && actionPresentations[action]) return actionPresentations[action];
  return { title: '未识别的系统操作', category: '其他', color: 'default' };
}

export function auditActorName(record: API.AdminAuditLog): string {
  if (record.actorDisplayName) return record.actorDisplayName;
  if (record.userId) return '未知人员';
  if (record.action?.startsWith('auth.')) return '未登录访问者';
  return '系统自动执行';
}

export function auditBusinessObject(record: API.AdminAuditLog): {
  name: string;
  type?: string;
} {
  const presentation = auditActionPresentation(record.action);
  if (record.targetDisplayName)
    return {
      name: record.targetDisplayName,
      type: presentation.objectType ?? '人员',
    };

  const details = record.details ?? {};
  const businessCode =
    details['order.no'] ??
    details['partner.code'] ??
    details['master_data.code'] ??
    details['milestone_template.code'] ??
    details['fee.code'] ??
    details.standard_code ??
    details.value;
  if (businessCode)
    return { name: businessCode, type: presentation.objectType };

  if (record.action?.startsWith('auth.')) return { name: '当前账号' };

  const resourceType = record.resourceType
    ? resourceTypeLabels[record.resourceType]
    : undefined;
  const objectType = resourceType ?? presentation.objectType;
  return objectType ? { name: objectType } : { name: '—' };
}

export function auditDetailLabel(key: string): string {
  return detailLabels[key] ?? key;
}
