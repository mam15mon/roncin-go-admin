const baseURL = (
  process.env.RONCIN_ACCEPTANCE_BASE_URL || 'http://127.0.0.1:8000'
).replace(/\/$/, '');
const apply = process.argv.includes('--apply');

function requireEnvironment(name) {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`缺少环境变量 ${name}`);
  return value;
}

async function readJSON(response) {
  const text = await response.text();
  if (!text) return {};
  try {
    return JSON.parse(text);
  } catch {
    throw new Error(`${response.url} 返回了非 JSON 响应`);
  }
}

async function login() {
  const response = await fetch(`${baseURL}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({
      username: requireEnvironment('BOOTSTRAP_ADMIN_USERNAME'),
      password: requireEnvironment('BOOTSTRAP_ADMIN_PASSWORD'),
    }),
  });
  const body = await readJSON(response);
  if (!response.ok) {
    throw new Error(
      `管理员登录失败（HTTP ${response.status}）：${body.message || '未知错误'}`,
    );
  }
  const cookie = response.headers
    .getSetCookie()
    .map((value) => value.split(';', 1)[0])
    .join('; ');
  if (!cookie) throw new Error('管理员登录成功但响应未设置会话 Cookie');
  return cookie;
}

function createClient(cookie) {
  async function raw(path, options = {}) {
    const response = await fetch(`${baseURL}${path}`, {
      ...options,
      headers: {
        cookie,
        ...(options.body ? { 'content-type': 'application/json' } : {}),
        ...options.headers,
      },
    });
    return { response, body: await readJSON(response) };
  }
  return {
    raw,
    async request(path, options = {}) {
      const result = await raw(path, options);
      if (!result.response.ok) {
        throw new Error(
          `${options.method || 'GET'} ${path} 失败（HTTP ${result.response.status}）：${result.body.message || '未知错误'}`,
        );
      }
      return result.body;
    },
  };
}

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

const cookie = await login();
const client = createClient(cookie);
const { request, raw } = client;
const [me, customers, templates, rules] = await Promise.all([
  request('/api/v1/auth/me'),
  request('/api/v1/partners?page=1&pageSize=100&role=1&enabled=true'),
  request('/api/v1/master-data/status-templates?businessType=1&published=true'),
  request('/api/v1/master-data/number-rules'),
]);
const customer = customers.data?.[0];
const statusTemplate = templates.data?.find(
  (item) => item.isDefault && item.enabled,
);
const billRule = rules.data?.find(
  (item) =>
    item.documentType === 2 || item.documentType === 'DOCUMENT_TYPE_BILL',
);
const batchRule = rules.data?.find(
  (item) =>
    item.documentType === 14 ||
    item.documentType === 'DOCUMENT_TYPE_BILL_BATCH',
);
const writeOffRule = rules.data?.find(
  (item) =>
    item.documentType === 4 || item.documentType === 'DOCUMENT_TYPE_WRITE_OFF',
);
const receiptPaymentRule = rules.data?.find(
  (item) =>
    item.documentType === 5 ||
    item.documentType === 'DOCUMENT_TYPE_RECEIPT_PAYMENT',
);
const commissionNumberRule = rules.data?.find(
  (item) =>
    item.documentType === 13 ||
    item.documentType === 'DOCUMENT_TYPE_COMMISSION',
);
assert(me.data?.id, '当前登录用户缺少用户编号');
assert(me.data?.currentOrganization?.id, '当前登录用户没有可用组织');
assert(customer?.id, '当前组织没有启用的客户');
assert(statusTemplate?.id, '当前组织没有海运出口默认状态模板');
assert(billRule?.enabled, '当前组织没有启用的账单编号规则');
assert(batchRule?.enabled, '当前组织没有启用的账单批次编号规则');
assert(writeOffRule?.enabled, '当前组织没有启用的核销编号规则');
assert(receiptPaymentRule?.enabled, '当前组织没有启用的收付编号规则');
assert(commissionNumberRule?.enabled, '当前组织没有启用的提成编号规则');

if (!apply) {
  console.log(
    '财务批量转账单验收前置条件检查通过。追加 --apply 执行真实闭环。',
  );
  process.exit(0);
}

const stamp = new Date()
  .toISOString()
  .replace(/[-:.TZ]/g, '')
  .slice(0, 14);
const today = new Date().toISOString().slice(0, 10);
const orderResponse = await request('/api/v1/orders', {
  method: 'POST',
  body: JSON.stringify({
    customerId: customer.id,
    businessType: 1,
    tradeDirection: 1,
    tradeTerm: 3,
    paymentTerm: 1,
    statusTemplateId: statusTemplate.id,
    shipmentType: 2,
    shipmentMode: 1,
    loadingTerms: 'CFS-CFS',
    goodsDescription: '财务批量转账单自动验收货物',
    totalPackages: 2,
    totalGrossWeightKg: 200,
    totalVolumeCbm: 2,
    customerReferenceNo: `ACC-FIN-${stamp}`,
    orderDate: new Date().toISOString(),
  }),
});
const order = orderResponse.data;
assert(order?.id && order?.orderNo, '验收订单创建失败');

const options = await request(`/api/v1/orders/${order.id}/fee-options`);
const feeSetting = options.feeSettings?.find(
  (item) => item.id && item.defaultBillingUnitId && item.taxRate != null,
);
const settlementParty = options.settlementParties?.find(
  (item) => item.id === customer.id,
);
assert(feeSetting?.id, '没有可用于验收的启用费用设置');
assert(feeSetting.defaultBillingUnitId, '费用设置缺少默认计费单位');
assert(settlementParty?.id, '费用选项中未返回验收客户');
assert(options.baseCurrency, '费用选项缺少本位币');

const createdFees = [];
for (const [index, price] of ['100.00', '25.00'].entries()) {
  const response = await request(`/api/v1/orders/${order.id}/fees`, {
    method: 'POST',
    body: JSON.stringify({
      orderId: order.id,
      direction: 1,
      settlementPartyId: settlementParty.id,
      quantity: '1',
      unitPrice: price,
      currency: options.baseCurrency,
      expenseDate: today,
      feeSettingId: feeSetting.id,
      billingUnitId: feeSetting.defaultBillingUnitId,
      idempotencyKey: `acc-fin-fee-${stamp}-${index}`,
      taxInclusive: true,
      note: '财务批量转账单自动验收',
    }),
  });
  const confirmed = await request(
    `/api/v1/orders/${order.id}/fees/${response.data.id}/confirm`,
    {
      method: 'POST',
      body: JSON.stringify({
        orderId: order.id,
        id: response.data.id,
        expectedVersion: response.data.version,
      }),
    },
  );
  assert(
    confirmed.data?.status === 2 ||
      confirmed.data?.status === 'ORDER_FEE_STATUS_CONFIRMED',
    '费用确认失败',
  );
  createdFees.push(confirmed.data);
}

const feeIds = createdFees.map((fee) => fee.id);
const preview = await request('/api/v1/finance/bill-batches/preview', {
  method: 'POST',
  body: JSON.stringify({
    feeIds,
    groupingPolicy: { splitByOrder: true, splitByTaxRate: true },
  }),
});
assert(preview.previewToken?.length === 64, '批量建单预览未返回有效快照令牌');
assert(
  preview.data?.length === 1,
  `同订单同税率费用应拆成 1 组，实际 ${preview.data?.length || 0}`,
);
assert(preview.data[0].fees?.length === 2, '预览分组未包含全部费用');

const createBody = (idempotencyKey) => ({
  feeIds,
  groupingPolicy: { splitByOrder: true, splitByTaxRate: true },
  previewToken: preview.previewToken,
  idempotencyKey,
  groups: preview.data.map((group) => ({
    groupKey: group.groupKey,
    statementTitle: customer.legalName,
    billDate: today,
    paymentTermsDays: 30,
    note: '财务批量转账单自动验收',
  })),
});
const sharedKey = `acc-fin-batch-${stamp}-shared`;
const concurrentRetries = await Promise.all([
  raw('/api/v1/finance/bill-batches', {
    method: 'POST',
    body: JSON.stringify(createBody(sharedKey)),
  }),
  raw('/api/v1/finance/bill-batches', {
    method: 'POST',
    body: JSON.stringify(createBody(sharedKey)),
  }),
]);
assert(
  concurrentRetries.every((item) => item.response.ok),
  '相同幂等键的并发请求必须全部重放为成功响应',
);
const batch = concurrentRetries[0].body.data;
assert(
  concurrentRetries[1].body.data?.id === batch?.id,
  '相同幂等键的并发请求未返回同一个批次',
);
assert(batch?.id && batch?.batchNo, '批量建单响应缺少批次信息');
assert(batch.bills?.length === 1, '批量建单结果账单数不正确');
assert(batch.bills[0].lines?.length === 2, '账单未固化两条费用明细');

const idempotentRetry = await request('/api/v1/finance/bill-batches', {
  method: 'POST',
  body: JSON.stringify(createBody(sharedKey)),
});
assert(idempotentRetry.data?.id === batch.id, '相同幂等请求未返回原批次');

const competingKey = await raw('/api/v1/finance/bill-batches', {
  method: 'POST',
  body: JSON.stringify(createBody(`acc-fin-batch-${stamp}-competing`)),
});
assert(
  competingKey.response.status === 409,
  '不同幂等键重复占用同一费用必须返回冲突',
);

const confirmedBatch = await request(
  `/api/v1/finance/bill-batches/${batch.id}/confirm`,
  {
    method: 'POST',
    body: JSON.stringify({
      id: batch.id,
      bills: batch.bills.map((bill) => ({
        billId: bill.id,
        expectedVersion: bill.version,
      })),
    }),
  },
);
assert(
  confirmedBatch.data?.bills?.every((bill) => bill.status === 'CONFIRMED'),
  '批次账单未全部确认',
);

let profile;
const profileResponse = await request(
  `/api/v1/partners/${customer.id}/invoice-profiles`,
);
profile = (profileResponse.data || []).find((item) => item.enabled);
if (!profile) {
  const saved = await request(
    `/api/v1/partners/${customer.id}/invoice-profiles`,
    {
      method: 'POST',
      body: JSON.stringify({
        partnerId: customer.id,
        invoiceTitle: customer.legalName,
        taxpayerIdentificationNo:
          customer.unifiedSocialCreditCode || `91310000ACC${stamp}`,
        registeredAddress: customer.registeredAddress || '',
        registeredPhone: '',
        bankName: '',
        bankAccount: '',
        defaultInvoiceType: 'NORMAL',
        isDefault: true,
      }),
    },
  );
  profile = saved.data;
}
assert(
  profile?.invoiceTitle && profile?.taxpayerIdentificationNo,
  '开票资料不完整',
);
const primaryProfile = profile;

const secondaryProfileResponse = await request(
  `/api/v1/partners/${customer.id}/invoice-profiles`,
  {
    method: 'POST',
    body: JSON.stringify({
      partnerId: customer.id,
      invoiceTitle: `${customer.legalName} 验收抬头 ${stamp}`,
      taxpayerIdentificationNo: profile.taxpayerIdentificationNo,
      registeredAddress: customer.registeredAddress || '',
      registeredPhone: '',
      bankName: '',
      bankAccount: '',
      defaultInvoiceType: 'NORMAL',
      isDefault: false,
    }),
  },
);
const createdSecondaryProfile = secondaryProfileResponse.data;
const updatedSecondaryProfileResponse = await request(
  `/api/v1/partners/${customer.id}/invoice-profiles/${createdSecondaryProfile.id}`,
  {
    method: 'PUT',
    body: JSON.stringify({
      partnerId: customer.id,
      id: createdSecondaryProfile.id,
      invoiceTitle: createdSecondaryProfile.invoiceTitle,
      taxpayerIdentificationNo:
        createdSecondaryProfile.taxpayerIdentificationNo,
      registeredAddress: createdSecondaryProfile.registeredAddress || '',
      registeredPhone: createdSecondaryProfile.registeredPhone || '',
      bankName: createdSecondaryProfile.bankName || '',
      bankAccount: createdSecondaryProfile.bankAccount || '',
      defaultInvoiceType: createdSecondaryProfile.defaultInvoiceType,
      isDefault: true,
      enabled: true,
      expectedVersion: createdSecondaryProfile.version,
    }),
  },
);
const secondaryProfile = updatedSecondaryProfileResponse.data;
const profilesAfterCreate = await request(
  `/api/v1/partners/${customer.id}/invoice-profiles`,
);
const enabledProfiles = (profilesAfterCreate.data || []).filter(
  (item) => item.enabled,
);
assert(enabledProfiles.length >= 2, '同一客户未能保存多套开票抬头');
assert(
  enabledProfiles.filter((item) => item.isDefault).length === 1,
  '同一客户的默认开票抬头不是唯一项',
);
assert(secondaryProfile.isDefault, '新指定的默认开票抬头未生效');
const removeDefaultResult = await raw(
  `/api/v1/partners/${customer.id}/invoice-profiles/${secondaryProfile.id}`,
  {
    method: 'PUT',
    body: JSON.stringify({
      partnerId: customer.id,
      id: secondaryProfile.id,
      invoiceTitle: secondaryProfile.invoiceTitle,
      taxpayerIdentificationNo: secondaryProfile.taxpayerIdentificationNo,
      registeredAddress: secondaryProfile.registeredAddress || '',
      registeredPhone: secondaryProfile.registeredPhone || '',
      bankName: secondaryProfile.bankName || '',
      bankAccount: secondaryProfile.bankAccount || '',
      defaultInvoiceType: secondaryProfile.defaultInvoiceType,
      isDefault: false,
      enabled: true,
      expectedVersion: secondaryProfile.version,
    }),
  },
);
assert(
  removeDefaultResult.response.status === 409,
  `存在启用抬头时不应允许取消唯一默认抬头，实际 HTTP ${removeDefaultResult.response.status}：${removeDefaultResult.body.message || '无错误消息'}`,
);
assert(
  removeDefaultResult.body.reason ===
    'PARTNER_INVOICE_PROFILE_DEFAULT_REQUIRED',
  '取消唯一默认抬头未返回明确业务错误',
);
profile = enabledProfiles.find((item) => item.id === primaryProfile.id);
assert(profile && !profile.isDefault, '原默认抬头未正确切换为非默认抬头');

const invoiceResponse = await request('/api/v1/finance/invoices', {
  method: 'POST',
  body: JSON.stringify({
    billIds: confirmedBatch.data.bills.map((bill) => bill.id),
    invoiceProfileId: profile.id,
    invoiceType: 'NORMAL',
    note: '财务批量转账单自动验收',
    idempotencyKey: `acc-fin-invoice-${stamp}`,
  }),
});
const invoice = invoiceResponse.data;
assert(invoice?.id && invoice?.recordNo, '创建开票记录失败');
assert(invoice.invoiceTitle === profile.invoiceTitle, '发票未固化开票抬头快照');
assert(
  invoice.taxpayerIdentificationNo === profile.taxpayerIdentificationNo,
  '发票未固化纳税人识别号快照',
);
assert(invoice.lines?.length >= 1, '发票未按费用项目和税率生成税务明细');
assert(
  invoice.lines.every(
    (line) => line.taxRate != null && Number(line.sourceLineCount) > 0,
  ),
  '发票税务明细缺少税率或来源行数',
);

const detail = await request(`/api/v1/finance/invoices/${invoice.id}`);
assert(
  detail.data?.lines?.length === invoice.lines.length,
  '开票详情未回显税务明细',
);

const confirmedBill = confirmedBatch.data.bills[0];
assert(
  confirmedBill.totalAmount === '125.00000000',
  `验收账单金额应为 125.00000000，实际 ${confirmedBill.totalAmount}`,
);
const cashflowBody = (amount, idempotencyKey) => ({
  direction: 'RECEIVABLE',
  settlementPartyId: customer.id,
  currency: confirmedBill.currency,
  amount,
  exchangeRate: '1',
  baseCurrency: confirmedBill.baseCurrency,
  transactionDate: today,
  ourAccount: '验收基本户',
  counterpartyAccount: '验收客户账户',
  paymentMethod: '银行转账',
  bankReferenceNo: `ACC-BANK-${stamp}-${amount}`,
  note: '财务资金核销自动验收',
  idempotencyKey,
});
const sharedCashflowKey = `acc-fin-cashflow-${stamp}-shared`;
const concurrentCashflows = await Promise.all([
  raw('/api/v1/finance/cashflows', {
    method: 'POST',
    body: JSON.stringify(cashflowBody('40.00000000', sharedCashflowKey)),
  }),
  raw('/api/v1/finance/cashflows', {
    method: 'POST',
    body: JSON.stringify(cashflowBody('40.00000000', sharedCashflowKey)),
  }),
]);
assert(
  concurrentCashflows.every((item) => item.response.ok),
  '相同幂等键的并发收款请求必须全部重放为成功响应',
);
assert(
  concurrentCashflows[0].body.data?.id === concurrentCashflows[1].body.data?.id,
  '相同幂等键的并发收款请求未返回同一资金流水',
);
const cashflowA = concurrentCashflows[0].body.data;
const cashflowBResponse = await request('/api/v1/finance/cashflows', {
  method: 'POST',
  body: JSON.stringify(
    cashflowBody('85.00000000', `acc-fin-cashflow-${stamp}-b`),
  ),
});
const cashflowB = cashflowBResponse.data;
const [confirmedCashflowAResponse, confirmedCashflowBResponse] =
  await Promise.all([
    request(`/api/v1/finance/cashflows/${cashflowA.id}/confirm`, {
      method: 'POST',
      body: JSON.stringify({
        id: cashflowA.id,
        expectedVersion: cashflowA.version,
      }),
    }),
    request(`/api/v1/finance/cashflows/${cashflowB.id}/confirm`, {
      method: 'POST',
      body: JSON.stringify({
        id: cashflowB.id,
        expectedVersion: cashflowB.version,
      }),
    }),
  ]);
const confirmedCashflowA = confirmedCashflowAResponse.data;
const confirmedCashflowB = confirmedCashflowBResponse.data;
const filteredCashflows = await request(
  `/api/v1/finance/cashflows?page=1&pageSize=200&status=CONFIRMED&direction=RECEIVABLE&settlementPartyId=${customer.id}&currency=${confirmedBill.currency}`,
);
assert(
  [confirmedCashflowA.id, confirmedCashflowB.id].every((id) =>
    filteredCashflows.data.some((item) => item.id === id),
  ),
  '核销候选条件未返回刚确认的资金流水',
);
assert(
  filteredCashflows.data.every(
    (item) =>
      item.direction === 'RECEIVABLE' &&
      item.settlementPartyId === customer.id &&
      item.currency === confirmedBill.currency,
  ),
  '资金流水候选接口未按方向、结算单位和币种隔离',
);

const verificationBody = (cashflowId, amount, idempotencyKey) => ({
  allocations: [{ cashflowId, billId: confirmedBill.id, amount }],
  verificationDate: today,
  note: '财务资金核销自动验收',
  idempotencyKey,
});
const sharedVerificationKey = `acc-fin-verification-${stamp}-shared`;
const concurrentVerifications = await Promise.all([
  raw('/api/v1/finance/verifications', {
    method: 'POST',
    body: JSON.stringify(
      verificationBody(
        confirmedCashflowA.id,
        '40.00000000',
        sharedVerificationKey,
      ),
    ),
  }),
  raw('/api/v1/finance/verifications', {
    method: 'POST',
    body: JSON.stringify(
      verificationBody(
        confirmedCashflowA.id,
        '40.00000000',
        sharedVerificationKey,
      ),
    ),
  }),
]);
assert(
  concurrentVerifications.every((item) => item.response.ok),
  '相同幂等键的并发核销请求必须全部重放为成功响应',
);
assert(
  concurrentVerifications[0].body.data?.id ===
    concurrentVerifications[1].body.data?.id,
  '相同幂等键的并发核销请求未返回同一核销记录',
);
const verificationA = concurrentVerifications[0].body.data;

await request(`/api/v1/orders/${order.id}/personnel`, {
  method: 'POST',
  body: JSON.stringify({
    orderId: order.id,
    userId: me.data.id,
    organizationId: me.data.currentOrganization.id,
    role: 3,
  }),
});
const commissionRuleResponse = await request(
  '/api/v1/finance/commission-rules',
  {
    method: 'POST',
    body: JSON.stringify({
      rule: {
        name: `ACC-COMMISSION-${stamp}`,
        personnelRole: 'SALES',
        calculationBasis: 'REALIZED_PROFIT',
        ratePercent: '10',
        effectiveFrom: today,
        effectiveTo: today,
        enabled: true,
        note: '提成可解释化自动验收',
      },
    }),
  },
);
let commissionRule = commissionRuleResponse.data;
const commissionPreview = await request('/api/v1/finance/commissions/preview', {
  method: 'POST',
  body: JSON.stringify({
    verificationId: verificationA.id,
    employeeId: me.data.id,
    ruleId: commissionRule.id,
  }),
});
assert(
  commissionPreview.data?.lines?.length === 1,
  '提成预览未返回逐订单计算明细',
);
assert(
  commissionPreview.data.lines[0].orderId === order.id,
  '提成预览明细未归属到验收订单',
);
assert(
  commissionPreview.data.commissionAmount === '4.00000000',
  `40 元回款按 10% 毛利提成应为 4 元，实际 ${commissionPreview.data.commissionAmount}`,
);
const firstCommissionBody = {
  verificationId: verificationA.id,
  employeeId: me.data.id,
  ruleId: commissionRule.id,
  note: '验证来源变化拦截',
  idempotencyKey: `acc-fin-commission-${stamp}-stale`,
};
const concurrentCommissions = await Promise.all([
  raw('/api/v1/finance/commissions', {
    method: 'POST',
    body: JSON.stringify(firstCommissionBody),
  }),
  raw('/api/v1/finance/commissions', {
    method: 'POST',
    body: JSON.stringify(firstCommissionBody),
  }),
]);
assert(
  concurrentCommissions.every((item) => item.response.ok),
  '相同幂等键的并发提成请求必须全部重放为成功响应',
);
assert(
  concurrentCommissions[0].body.data?.id ===
    concurrentCommissions[1].body.data?.id,
  '相同幂等键的并发提成请求未返回同一提成草稿',
);
const firstCommission = concurrentCommissions[0].body.data;
const firstCommissionDetail = await request(
  `/api/v1/finance/commissions/${firstCommission.id}`,
);
assert(
  firstCommissionDetail.data?.lines?.length === 1 &&
    firstCommissionDetail.data.lines[0].commissionAmount === '4.00000000',
  '提成草稿未固化逐订单计算快照',
);
assert(
  firstCommissionDetail.data.lines[0].personnelOrganizationId ===
    me.data.currentOrganization.id &&
    firstCommissionDetail.data.lines[0].personnelAssignedAt,
  '提成草稿未固化人员组织和指派时间快照',
);

const updatedRuleResponse = await request(
  `/api/v1/finance/commission-rules/${commissionRule.id}`,
  {
    method: 'PUT',
    body: JSON.stringify({
      id: commissionRule.id,
      expectedVersion: commissionRule.version,
      rule: {
        name: commissionRule.name,
        personnelRole: commissionRule.personnelRole,
        calculationBasis: commissionRule.calculationBasis,
        ratePercent: commissionRule.ratePercent,
        effectiveFrom: commissionRule.effectiveFrom,
        effectiveTo: commissionRule.effectiveTo,
        enabled: commissionRule.enabled,
        note: '修改规则版本以验证草稿来源指纹',
      },
    }),
  },
);
commissionRule = updatedRuleResponse.data;
const staleConfirmation = await raw(
  `/api/v1/finance/commissions/${firstCommission.id}/confirm`,
  {
    method: 'POST',
    body: JSON.stringify({
      id: firstCommission.id,
      expectedVersion: firstCommission.version,
    }),
  },
);
assert(
  staleConfirmation.response.status === 409 &&
    staleConfirmation.body.reason === 'FINANCE_COMMISSION_SOURCE_CHANGED',
  '规则变化后确认旧提成草稿必须返回来源变化冲突',
);
await request(`/api/v1/finance/commissions/${firstCommission.id}/cancel`, {
  method: 'POST',
  body: JSON.stringify({
    id: firstCommission.id,
    expectedVersion: firstCommission.version,
    reason: '来源变化后取消旧草稿',
  }),
});

const refreshedPreview = await request('/api/v1/finance/commissions/preview', {
  method: 'POST',
  body: JSON.stringify({
    verificationId: verificationA.id,
    employeeId: me.data.id,
    ruleId: commissionRule.id,
  }),
});
assert(
  refreshedPreview.data?.ruleVersion === commissionRule.version,
  '重新预览未使用最新提成规则版本',
);
const refreshedCommissionResponse = await request(
  '/api/v1/finance/commissions',
  {
    method: 'POST',
    body: JSON.stringify({
      verificationId: verificationA.id,
      employeeId: me.data.id,
      ruleId: commissionRule.id,
      note: '验证来源一致时正常确认',
      idempotencyKey: `acc-fin-commission-${stamp}-confirmed`,
    }),
  },
);
const refreshedCommission = refreshedCommissionResponse.data;
const draftFeeBeforeCloseResponse = await request(
  `/api/v1/orders/${order.id}/fees`,
  {
    method: 'POST',
    body: JSON.stringify({
      orderId: order.id,
      direction: 1,
      settlementPartyId: settlementParty.id,
      quantity: '1',
      unitPrice: '1.00',
      currency: options.baseCurrency,
      expenseDate: today,
      feeSettingId: feeSetting.id,
      billingUnitId: feeSetting.defaultBillingUnitId,
      idempotencyKey: `acc-fin-fee-${stamp}-draft-before-close`,
      taxInclusive: true,
      note: '验证存在草稿费用时不得关账',
    }),
  },
);
const draftFeeBeforeClose = draftFeeBeforeCloseResponse.data;
const closeWithDraftFee = await raw(
  `/api/v1/finance/commissions/${refreshedCommission.id}/confirm`,
  {
    method: 'POST',
    body: JSON.stringify({
      id: refreshedCommission.id,
      expectedVersion: refreshedCommission.version,
    }),
  },
);
assert(
  closeWithDraftFee.response.status === 409 &&
    closeWithDraftFee.body.reason === 'FINANCE_COMMISSION_UNCONFIRMED_FEES',
  '关联订单存在草稿费用时必须拒绝确认提成',
);
await request(
  `/api/v1/orders/${order.id}/fees/${draftFeeBeforeClose.id}?expectedVersion=${draftFeeBeforeClose.version}&reason=${encodeURIComponent('完成草稿费用关账拦截验收')}`,
  {
    method: 'DELETE',
  },
);
const confirmedCommissionResponse = await request(
  `/api/v1/finance/commissions/${refreshedCommission.id}/confirm`,
  {
    method: 'POST',
    body: JSON.stringify({
      id: refreshedCommission.id,
      expectedVersion: refreshedCommission.version,
    }),
  },
);
const confirmedCommission = confirmedCommissionResponse.data;
assert(confirmedCommission.status === 'CONFIRMED', '提成草稿确认失败');
const lockedFeeOptions = await request(
  `/api/v1/orders/${order.id}/fee-options`,
);
assert(
  lockedFeeOptions.financeLocked === true &&
    lockedFeeOptions.financeLockCommissionNos?.includes(
      confirmedCommission.commissionNo,
    ),
  '提成确认后关联订单未进入财务锁定',
);
const addFeeAfterClose = await raw(`/api/v1/orders/${order.id}/fees`, {
  method: 'POST',
  body: JSON.stringify({
    orderId: order.id,
    direction: 1,
    settlementPartyId: settlementParty.id,
    quantity: '1',
    unitPrice: '1.00',
    currency: options.baseCurrency,
    expenseDate: today,
    feeSettingId: feeSetting.id,
    billingUnitId: feeSetting.defaultBillingUnitId,
    idempotencyKey: `acc-fin-fee-${stamp}-after-close`,
    taxInclusive: true,
    note: '验证提成确认后禁止静默追加费用',
  }),
});
assert(
  addFeeAfterClose.response.status === 409 &&
    addFeeAfterClose.body.reason === 'ORDER_FEE_FINANCE_LOCKED',
  '提成确认后新增费用必须返回财务锁定冲突',
);

const adjustmentBody = {
  commissionId: confirmedCommission.id,
  orderId: order.id,
  direction: 'INCREASE',
  amount: '1.25000000',
  reason: '验收确认后补提',
  note: '原始提成保持不变',
  idempotencyKey: `acc-fin-commission-adjustment-${stamp}-shared`,
};
const concurrentAdjustments = await Promise.all([
  raw(`/api/v1/finance/commissions/${confirmedCommission.id}/adjustments`, {
    method: 'POST',
    body: JSON.stringify(adjustmentBody),
  }),
  raw(`/api/v1/finance/commissions/${confirmedCommission.id}/adjustments`, {
    method: 'POST',
    body: JSON.stringify(adjustmentBody),
  }),
]);
assert(
  concurrentAdjustments.every((item) => item.response.ok) &&
    concurrentAdjustments[0].body.data?.id ===
      concurrentAdjustments[1].body.data?.id,
  '相同幂等键的并发提成调整未重放为同一记录',
);
const adjustmentDraft = concurrentAdjustments[0].body.data;
const confirmedAdjustmentResponse = await request(
  `/api/v1/finance/commission-adjustments/${adjustmentDraft.id}/confirm`,
  {
    method: 'POST',
    body: JSON.stringify({
      id: adjustmentDraft.id,
      expectedVersion: adjustmentDraft.version,
    }),
  },
);
const confirmedAdjustment = confirmedAdjustmentResponse.data;
const commissionAfterAdjustment = await request(
  `/api/v1/finance/commissions/${confirmedCommission.id}`,
);
assert(
  commissionAfterAdjustment.data.commissionAmount === '4.00000000' &&
    commissionAfterAdjustment.data.adjustmentAmount === '1.25000000' &&
    commissionAfterAdjustment.data.effectiveCommissionAmount === '5.25000000',
  '提成调整不得改写原始金额，且必须正确形成有效提成',
);
const cancelParentWithAdjustment = await raw(
  `/api/v1/finance/commissions/${confirmedCommission.id}/cancel`,
  {
    method: 'POST',
    body: JSON.stringify({
      id: confirmedCommission.id,
      expectedVersion: confirmedCommission.version,
      reason: '验证存在有效调整时禁止取消原提成',
    }),
  },
);
assert(
  cancelParentWithAdjustment.response.status === 409 &&
    cancelParentWithAdjustment.body.reason === 'FINANCE_COMMISSION_TRANSITION',
  '存在有效调整时必须禁止取消原提成',
);
await request(
  `/api/v1/finance/commission-adjustments/${confirmedAdjustment.id}/cancel`,
  {
    method: 'POST',
    body: JSON.stringify({
      id: confirmedAdjustment.id,
      expectedVersion: confirmedAdjustment.version,
      reason: '验收结束后取消调整',
    }),
  },
);
const excessiveDecreaseResponse = await request(
  `/api/v1/finance/commissions/${confirmedCommission.id}/adjustments`,
  {
    method: 'POST',
    body: JSON.stringify({
      commissionId: confirmedCommission.id,
      orderId: order.id,
      direction: 'DECREASE',
      amount: '100.00000000',
      reason: '验证冲减不得超过有效提成',
      idempotencyKey: `acc-fin-commission-adjustment-${stamp}-excessive`,
    }),
  },
);
const excessiveDecrease = excessiveDecreaseResponse.data;
const excessiveDecreaseConfirmation = await raw(
  `/api/v1/finance/commission-adjustments/${excessiveDecrease.id}/confirm`,
  {
    method: 'POST',
    body: JSON.stringify({
      id: excessiveDecrease.id,
      expectedVersion: excessiveDecrease.version,
    }),
  },
);
assert(
  excessiveDecreaseConfirmation.response.status === 409 &&
    excessiveDecreaseConfirmation.body.reason ===
      'FINANCE_COMMISSION_ADJUSTMENT_EXCEEDS',
  '冲减后有效提成小于零时必须返回明确冲突',
);
await request(
  `/api/v1/finance/commission-adjustments/${excessiveDecrease.id}/cancel`,
  {
    method: 'POST',
    body: JSON.stringify({
      id: excessiveDecrease.id,
      expectedVersion: excessiveDecrease.version,
      reason: '验收结束后取消超额冲减草稿',
    }),
  },
);
const reverseWithCommission = await raw(
  `/api/v1/finance/verifications/${verificationA.id}/reverse`,
  {
    method: 'POST',
    body: JSON.stringify({
      id: verificationA.id,
      expectedVersion: verificationA.version,
      reason: '验证提成存在时禁止反核销',
    }),
  },
);
assert(
  reverseWithCommission.response.status === 409 &&
    reverseWithCommission.body.reason === 'FINANCE_VERIFICATION_HAS_COMMISSION',
  '存在已确认提成时必须禁止反核销',
);
await request(`/api/v1/finance/commissions/${confirmedCommission.id}/cancel`, {
  method: 'POST',
  body: JSON.stringify({
    id: confirmedCommission.id,
    expectedVersion: confirmedCommission.version,
    reason: '完成提成验收后释放核销',
  }),
});
const unlockedFeeOptions = await request(
  `/api/v1/orders/${order.id}/fee-options`,
);
assert(
  unlockedFeeOptions.financeLocked !== true,
  '提成及调整全部取消后订单财务锁未释放',
);

const overAllocated = await raw('/api/v1/finance/verifications', {
  method: 'POST',
  body: JSON.stringify(
    verificationBody(
      confirmedCashflowB.id,
      '85.00000001',
      `acc-fin-verification-${stamp}-over`,
    ),
  ),
});
assert(
  overAllocated.response.status === 409 &&
    overAllocated.body.reason === 'FINANCE_VERIFICATION_BALANCE',
  '超过资金余额的核销必须返回明确余额冲突',
);

const verificationBResponse = await request('/api/v1/finance/verifications', {
  method: 'POST',
  body: JSON.stringify(
    verificationBody(
      confirmedCashflowB.id,
      '85.00000000',
      `acc-fin-verification-${stamp}-b`,
    ),
  ),
});
const verificationB = verificationBResponse.data;

const balancesAfterFullVerification = await Promise.all([
  request('/api/v1/finance/cashflows?page=1&pageSize=200&status=CONFIRMED'),
  request('/api/v1/finance/bills?page=1&pageSize=200&status=CONFIRMED'),
]);
const cashflowsAfterFullVerification = balancesAfterFullVerification[0].data;
const billsAfterFullVerification = balancesAfterFullVerification[1].data;
assert(
  [confirmedCashflowA.id, confirmedCashflowB.id].every(
    (id) =>
      cashflowsAfterFullVerification.find((item) => item.id === id)
        ?.unverifiedAmount === '0.00000000',
  ),
  '两笔资金完成核销后仍存在未核销余额',
);
assert(
  billsAfterFullVerification.find((item) => item.id === confirmedBill.id)
    ?.unverifiedAmount === '0.00000000',
  '账单完成核销后仍存在未核销余额',
);

const cancelAllocatedCashflow = await raw(
  `/api/v1/finance/cashflows/${confirmedCashflowB.id}/cancel`,
  {
    method: 'POST',
    body: JSON.stringify({
      id: confirmedCashflowB.id,
      expectedVersion: confirmedCashflowB.version,
      reason: '有效核销期间取消测试',
    }),
  },
);
assert(
  cancelAllocatedCashflow.response.status === 409 &&
    cancelAllocatedCashflow.body.reason ===
      'FINANCE_CASHFLOW_INVALID_TRANSITION',
  '存在有效核销时必须禁止取消资金流水',
);

await request(`/api/v1/finance/verifications/${verificationB.id}/reverse`, {
  method: 'POST',
  body: JSON.stringify({
    id: verificationB.id,
    expectedVersion: verificationB.version,
    reason: '验证反核销释放余额',
  }),
});
const cancelledCashflowBResponse = await request(
  `/api/v1/finance/cashflows/${confirmedCashflowB.id}/cancel`,
  {
    method: 'POST',
    body: JSON.stringify({
      id: confirmedCashflowB.id,
      expectedVersion: confirmedCashflowB.version,
      reason: '反核销后取消资金流水验收',
    }),
  },
);
assert(
  cancelledCashflowBResponse.data?.status === 'CANCELLED',
  '反核销后资金流水仍无法取消',
);
await request(`/api/v1/finance/verifications/${verificationA.id}/reverse`, {
  method: 'POST',
  body: JSON.stringify({
    id: verificationA.id,
    expectedVersion: verificationA.version,
    reason: '验证全部反核销恢复余额',
  }),
});
const balancesAfterReverse = await Promise.all([
  request('/api/v1/finance/cashflows?page=1&pageSize=200'),
  request('/api/v1/finance/bills?page=1&pageSize=200&status=CONFIRMED'),
]);
assert(
  balancesAfterReverse[0].data.find((item) => item.id === confirmedCashflowA.id)
    ?.unverifiedAmount === '40.00000000',
  '反核销后资金余额未恢复',
);
assert(
  balancesAfterReverse[1].data.find((item) => item.id === confirmedBill.id)
    ?.unverifiedAmount === '125.00000000',
  '全部反核销后账单余额未恢复',
);

console.log('费用、账单、开票、收付、核销、提成关账与调整真实 API 闭环验收通过。');
console.log(
  JSON.stringify(
    {
      orderNo: order.orderNo,
      feeCount: feeIds.length,
      previewGroupCount: preview.data.length,
      batchNo: batch.batchNo,
      billNos: confirmedBatch.data.bills.map((bill) => bill.billNo),
      invoiceRecordNo: invoice.recordNo,
      invoiceLineCount: invoice.lines.length,
      invoiceProfileCount: enabledProfiles.length,
      selectedNonDefaultProfile: !profile.isDefault,
      concurrentIdempotentReplayVerified: true,
      competingKeyConflictVerified: true,
      idempotentRetryVerified: true,
      defaultProfileProtectionVerified: true,
      cashflowNos: [confirmedCashflowA.flowNo, confirmedCashflowB.flowNo],
      verificationNos: [
        verificationA.verificationNo,
        verificationB.verificationNo,
      ],
      cashflowConcurrentIdempotencyVerified: true,
      verificationConcurrentIdempotencyVerified: true,
      commissionNo: refreshedCommission.commissionNo,
      commissionPreviewLineCount: refreshedPreview.data.lines.length,
      commissionSourceChangeRejected: true,
      commissionConcurrentIdempotencyVerified: true,
      commissionDetailSnapshotVerified: true,
      commissionReverseProtectionVerified: true,
      commissionFinanceLockVerified: true,
      commissionDraftFeeCloseRejected: true,
      commissionAdjustmentIdempotencyVerified: true,
      commissionAdjustmentEffectiveAmountVerified: true,
      commissionAdjustmentCancellationProtectionVerified: true,
      commissionAdjustmentOverDecreaseRejected: true,
      overAllocationRejected: true,
      reverseBalanceRecoveryVerified: true,
      cashflowCandidateFilterVerified: true,
    },
    null,
    2,
  ),
);
