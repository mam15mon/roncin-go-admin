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
const [customers, templates, rules] = await Promise.all([
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
assert(customer?.id, '当前组织没有启用的客户');
assert(statusTemplate?.id, '当前组织没有海运出口默认状态模板');
assert(billRule?.enabled, '当前组织没有启用的账单编号规则');
assert(batchRule?.enabled, '当前组织没有启用的账单批次编号规则');

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
const firstKey = `acc-fin-batch-${stamp}-a`;
const secondKey = `acc-fin-batch-${stamp}-b`;
const competing = await Promise.all([
  raw('/api/v1/finance/bill-batches', {
    method: 'POST',
    body: JSON.stringify(createBody(firstKey)),
  }),
  raw('/api/v1/finance/bill-batches', {
    method: 'POST',
    body: JSON.stringify(createBody(secondKey)),
  }),
]);
const successes = competing.filter((item) => item.response.ok);
const conflicts = competing.filter((item) => item.response.status === 409);
assert(
  successes.length === 1 && conflicts.length === 1,
  '并发争抢同一费用时必须且只能有一个批次成功',
);
const batch = successes[0].body.data;
const successfulKey = successes[0] === competing[0] ? firstKey : secondKey;
assert(batch?.id && batch?.batchNo, '批量建单响应缺少批次信息');
assert(batch.bills?.length === 1, '批量建单结果账单数不正确');
assert(batch.bills[0].lines?.length === 2, '账单未固化两条费用明细');

const idempotentRetry = await request('/api/v1/finance/bill-batches', {
  method: 'POST',
  body: JSON.stringify(createBody(successfulKey)),
});
assert(idempotentRetry.data?.id === batch.id, '相同幂等请求未返回原批次');

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

console.log('费用批量转账单与开票快照真实 API 验收通过。');
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
      concurrentConflictVerified: true,
      idempotentRetryVerified: true,
    },
    null,
    2,
  ),
);
