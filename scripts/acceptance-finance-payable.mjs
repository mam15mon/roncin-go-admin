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

function isEnumValue(value, code, name) {
  return value === code || value === name || String(value).endsWith(`_${name}`);
}

const cookie = await login();
const { request, raw } = createClient(cookie);
const [customers, rules] = await Promise.all([
  request('/api/v1/partners?page=1&pageSize=200&role=1&enabled=true'),
  request('/api/v1/master-data/number-rules'),
]);
const customer = customers.data?.[0];
const requiredDocumentTypes = [
  [2, 'DOCUMENT_TYPE_BILL'],
  [4, 'DOCUMENT_TYPE_WRITE_OFF'],
  [5, 'DOCUMENT_TYPE_RECEIPT_PAYMENT'],
  [14, 'DOCUMENT_TYPE_BILL_BATCH'],
];
assert(customer?.id, '当前组织没有启用的客户，无法创建应付验收订单');
for (const [code, name] of requiredDocumentTypes) {
  assert(
    rules.data?.some(
      (item) => item.enabled && isEnumValue(item.documentType, code, name),
    ),
    `当前组织缺少启用的财务编号规则：${name}`,
  );
}

if (!apply) {
  console.log('应付费用闭环验收前置条件检查通过。追加 --apply 执行真实闭环。');
  process.exit(0);
}

const stamp = new Date()
  .toISOString()
  .replace(/[-:.TZ]/g, '')
  .slice(0, 14);
const today = new Date().toISOString().slice(0, 10);
const supplierName = `应付验收供应商 ${stamp}`;
const supplierResponse = await request('/api/v1/partners', {
  method: 'POST',
  body: JSON.stringify({
    code: `ACC-SUP-${stamp}`,
    legalName: supplierName,
    unifiedSocialCreditCode: `91310000PAY${stamp.slice(-7)}`,
    registeredAddress: '应付自动验收地址',
    roles: [{ type: 2, enabled: true }],
    contacts: [
      {
        name: '应付验收联系人',
        phone: '13800000000',
        isPrimary: true,
      },
    ],
    aliases: [{ aliasName: `AP SUPPLIER ${stamp}`, sortOrder: 1 }],
    profile: {
      nameEn: `AP Acceptance Supplier ${stamp}`,
      countryCode: 'CN',
      businessTypes: [1],
      remark: '应付费用端到端自动验收创建',
    },
  }),
});
const supplier = supplierResponse.data;
assert(
  supplier?.id &&
    supplier.roles?.some(
      (role) => isEnumValue(role.type, 2, 'SUPPLIER') && role.enabled,
    ),
  '验收供应商创建失败或未启用供应商角色',
);

const orderResponse = await request('/api/v1/orders', {
  method: 'POST',
  body: JSON.stringify({
    customerId: customer.id,
    businessType: 1,
    tradeDirection: 1,
    tradeTerm: 3,
    paymentTerm: 1,
    shipmentType: 2,
    shipmentMode: 1,
    loadingTerms: 'CFS-CFS',
    goodsDescription: '应付费用自动验收货物',
    totalPackages: 1,
    totalGrossWeightKg: 100,
    totalVolumeCbm: 1,
    customerReferenceNo: `ACC-AP-${stamp}`,
    orderDate: new Date().toISOString(),
  }),
});
const order = orderResponse.data;
assert(order?.id && order?.orderNo, '应付验收订单创建失败');

let createdFeeSetting = null;
if (apply) {
  const createdUnit = await request('/api/v1/finance/billing-units', {
    method: 'POST',
    body: JSON.stringify({
      code: `UNIT_AP_${stamp.slice(-6)}`,
      name: '票',
      sortOrder: 1,
      isContainerUnit: false,
    }),
  });
  const billingUnit = createdUnit.data;
  assert(billingUnit?.id, '应付验收计费单位创建失败');

  const createdService = await request('/api/v1/finance/taxable-services', {
    method: 'POST',
    body: JSON.stringify({
      name: `基础物流服务_${stamp.slice(-6)}`,
      defaultTaxRate: '0.00',
    }),
  });
  const taxableService = createdService.data;
  assert(taxableService?.id, '应付验收应税服务创建失败');

  const createdSetting = await request('/api/v1/finance/fee-settings', {
    method: 'POST',
    body: JSON.stringify({
      feeCode: `FEE_AP_${stamp.slice(-6)}`,
      nameZh: '海运应付运费',
      defaultCurrency: 'CNY',
      billingUnitId: billingUnit.id,
      taxableServiceId: taxableService.id,
      taxRate: '0.00',
      sortOrder: 1,
    }),
  });
  createdFeeSetting = createdSetting.data;
  assert(createdFeeSetting?.id, '应付验收费用科目创建失败');

  await request('/api/v1/finance/exchange-rate-time-standards', {
    method: 'PUT',
    body: JSON.stringify({
      data: [
        { rateType: 'BASE_CURRENCY', timeStandards: ['ORDER_CREATED_AT'] },
        { rateType: 'BILL', timeStandards: ['BILL_DATE'] },
        { rateType: 'INVOICE', timeStandards: ['INVOICE_DATE'] },
        { rateType: 'SETTLEMENT', timeStandards: ['TRANSACTION_DATE'] },
        { rateType: 'WRITE_OFF', timeStandards: ['WRITE_OFF_TIME'] },
      ],
    }),
  });
}

const options = await request(`/api/v1/orders/${order.id}/fee-options`);
let feeSetting;
if (apply) {
  feeSetting = options.feeSettings?.find(
    (item) => item.id === createdFeeSetting.id,
  );
  assert(feeSetting?.id, `费用选项中未返回新创建的专用应付费用科目: ${createdFeeSetting?.id}`);
} else {
  feeSetting = options.feeSettings?.find(
    (item) => item.id && item.defaultBillingUnitId && item.taxRate != null,
  );
  assert(feeSetting?.id, '没有可用于应付验收的启用费用设置');
}
const settlementParty = options.settlementParties?.find(
  (item) => item.id === supplier.id,
);
assert(feeSetting.defaultBillingUnitId, '应付验收费用设置缺少默认计费单位');
assert(settlementParty?.id, '费用选项没有返回刚创建的供应商');
assert(options.baseCurrency, '费用选项缺少本位币');

const feeResponse = await request(`/api/v1/orders/${order.id}/fees`, {
  method: 'POST',
  body: JSON.stringify({
    orderId: order.id,
    direction: 2,
    settlementPartyId: supplier.id,
    quantity: '1',
    unitPrice: '88.00',
    currency: options.baseCurrency,
    expenseDate: today,
    feeSettingId: feeSetting.id,
    billingUnitId: feeSetting.defaultBillingUnitId,
    idempotencyKey: `acc-ap-fee-${stamp}`,
    taxInclusive: true,
    note: '应付费用端到端自动验收',
  }),
});
const draftFee = feeResponse.data;
assert(isEnumValue(draftFee?.direction, 2, 'PAYABLE'), '新建费用未保存为应付方向');
assert(draftFee.settlementPartyId === supplier.id, '应付费用未关联供应商');
const confirmedFeeResponse = await request(
  `/api/v1/orders/${order.id}/fees/${draftFee.id}/confirm`,
  {
    method: 'POST',
    body: JSON.stringify({
      orderId: order.id,
      id: draftFee.id,
      expectedVersion: draftFee.version,
    }),
  },
);
const confirmedFee = confirmedFeeResponse.data;
assert(confirmedFee?.status === 2, `费用确认后状态应为 CONFIRMED(2)，实际 ${confirmedFee?.status}`);
assert(isEnumValue(confirmedFee?.direction, 2, 'PAYABLE'), '费用确认后应付方向丢失');
const orderFees = await request(`/api/v1/orders/${order.id}/fees`);
const persistedFee = orderFees.data?.find((item) => item.id === confirmedFee.id);
assert(
  persistedFee && isEnumValue(persistedFee.direction, 2, 'PAYABLE'),
  '订单费用列表未按应付方向回显费用',
);
const payableLedger = await request(
  `/api/v1/finance/fees?page=1&pageSize=200&keyword=${encodeURIComponent(order.orderNo)}&direction=PAYABLE`,
);
const ledgerFee = payableLedger.data?.find((item) => item.id === confirmedFee.id);
assert(
  ledgerFee?.financialProgress === 1,
  '应付费用建账单前状态不是 UNBILLED',
);
assert(
  ledgerFee.customerId === customer.id &&
    ledgerFee.settlementPartyId === supplier.id &&
    ledgerFee.settlementPartyName === supplierName,
  '应付费用台账未正确区分委托单位与供应商结算单位',
);

const preview = await request('/api/v1/finance/bill-batches/preview', {
  method: 'POST',
  body: JSON.stringify({
    feeIds: [confirmedFee.id],
    groupingPolicy: { splitByOrder: true, splitByTaxRate: true },
  }),
});
assert(preview.previewToken?.length === 64, '应付账单预览缺少有效快照令牌');
assert(preview.data?.length === 1, '单笔应付费用应只生成一个预览分组');
const previewGroup = preview.data[0];
assert(previewGroup.direction === 'PAYABLE', '应付账单预览方向错误');
assert(previewGroup.settlementPartyId === supplier.id, '应付账单预览结算单位错误');

const batchResponse = await request('/api/v1/finance/bill-batches', {
  method: 'POST',
  body: JSON.stringify({
    feeIds: [confirmedFee.id],
    groupingPolicy: { splitByOrder: true, splitByTaxRate: true },
    previewToken: preview.previewToken,
    idempotencyKey: `acc-ap-batch-${stamp}`,
    groups: [
      {
        groupKey: previewGroup.groupKey,
        statementTitle: supplierName,
        billDate: today,
        paymentTermsDays: 30,
        note: '应付账单端到端自动验收',
      },
    ],
  }),
});
const batch = batchResponse.data;
assert(batch?.id && batch?.bills?.length === 1, '应付账单批次创建失败');
const draftBill = batch.bills[0];
assert(draftBill.direction === 'PAYABLE', '生成的账单不是应付方向');
assert(draftBill.settlementPartyId === supplier.id, '应付账单未关联供应商');
assert(draftBill.totalAmount === '88.00000000', '应付账单金额快照错误');

const confirmedBatchResponse = await request(
  `/api/v1/finance/bill-batches/${batch.id}/confirm`,
  {
    method: 'POST',
    body: JSON.stringify({
      id: batch.id,
      bills: [{ billId: draftBill.id, expectedVersion: draftBill.version }],
    }),
  },
);
const confirmedBill = confirmedBatchResponse.data?.bills?.[0];
assert(confirmedBill?.status === 2, '应付账单确认失败');
assert(confirmedBill.direction === 'PAYABLE', '确认账单时应付方向丢失');

const wrongCashflowResponse = await request('/api/v1/finance/cashflows', {
  method: 'POST',
  body: JSON.stringify({
    direction: 'RECEIVABLE',
    settlementPartyId: supplier.id,
    currency: confirmedBill.currency,
    amount: '88.00000000',
    exchangeRate: '1',
    baseCurrency: confirmedBill.baseCurrency,
    transactionDate: today,
    ourAccount: '应付验收基本户',
    counterpartyAccount: '验收供应商账户',
    paymentMethod: '银行转账',
    bankReferenceNo: `ACC-AP-WRONG-${stamp}`,
    note: '验证收付方向不能混合核销',
    idempotencyKey: `acc-ap-wrong-cashflow-${stamp}`,
  }),
});
const confirmedWrongCashflowResponse = await request(
  `/api/v1/finance/cashflows/${wrongCashflowResponse.data.id}/confirm`,
  {
    method: 'POST',
    body: JSON.stringify({
      id: wrongCashflowResponse.data.id,
      expectedVersion: wrongCashflowResponse.data.version,
    }),
  },
);
const confirmedWrongCashflow = confirmedWrongCashflowResponse.data;
const mismatchResult = await raw('/api/v1/finance/verifications', {
  method: 'POST',
  body: JSON.stringify({
    allocations: [
      {
        cashflowId: confirmedWrongCashflow.id,
        billId: confirmedBill.id,
        amount: '88.00000000',
      },
    ],
    verificationDate: today,
    note: '应收流水不得核销应付账单',
    idempotencyKey: `acc-ap-mismatch-${stamp}`,
  }),
});
assert(
  mismatchResult.response.status === 400 &&
    mismatchResult.body.reason === 'FINANCE_VERIFICATION_MISMATCH',
  `应收流水核销应付账单未被明确拒绝：HTTP ${mismatchResult.response.status} ${mismatchResult.body.reason || mismatchResult.body.message || ''}`,
);
await request(`/api/v1/finance/cashflows/${confirmedWrongCashflow.id}/cancel`, {
  method: 'POST',
  body: JSON.stringify({
    id: confirmedWrongCashflow.id,
    expectedVersion: confirmedWrongCashflow.version,
    reason: '收付方向隔离验收完成',
  }),
});

const cashflowResponse = await request('/api/v1/finance/cashflows', {
  method: 'POST',
  body: JSON.stringify({
    direction: 'PAYABLE',
    settlementPartyId: supplier.id,
    currency: confirmedBill.currency,
    amount: '88.00000000',
    exchangeRate: '1',
    baseCurrency: confirmedBill.baseCurrency,
    transactionDate: today,
    ourAccount: '应付验收基本户',
    counterpartyAccount: '验收供应商账户',
    paymentMethod: '银行转账',
    bankReferenceNo: `ACC-AP-PAY-${stamp}`,
    note: '应付付款与核销端到端自动验收',
    idempotencyKey: `acc-ap-cashflow-${stamp}`,
  }),
});
const confirmedCashflowResponse = await request(
  `/api/v1/finance/cashflows/${cashflowResponse.data.id}/confirm`,
  {
    method: 'POST',
    body: JSON.stringify({
      id: cashflowResponse.data.id,
      expectedVersion: cashflowResponse.data.version,
    }),
  },
);
const confirmedCashflow = confirmedCashflowResponse.data;
assert(confirmedCashflow?.direction === 'PAYABLE', '付款流水未保存为应付方向');
const payableCashflows = await request(
  `/api/v1/finance/cashflows?page=1&pageSize=200&status=FINANCE_CASHFLOW_STATUS_CONFIRMED&direction=PAYABLE&settlementPartyId=${supplier.id}&currency=${confirmedBill.currency}`,
);
assert(
  payableCashflows.data?.some((item) => item.id === confirmedCashflow.id),
  '应付资金流水筛选未返回刚确认的付款',
);

const verificationResponse = await request('/api/v1/finance/verifications', {
  method: 'POST',
  body: JSON.stringify({
    allocations: [
      {
        cashflowId: confirmedCashflow.id,
        billId: confirmedBill.id,
        amount: '88.00000000',
      },
    ],
    verificationDate: today,
    note: '应付付款与账单自动核销',
    idempotencyKey: `acc-ap-verification-${stamp}`,
  }),
});
const verification = verificationResponse.data;
assert(verification?.direction === 'PAYABLE', '应付核销记录方向错误');
assert(verification.settlementPartyId === supplier.id, '应付核销结算单位错误');
assert(verification.amount === '88.00000000', '应付核销金额错误');

const [feesAfterVerification, billsAfterVerification, cashflowsAfterVerification] =
  await Promise.all([
    request(
      `/api/v1/finance/fees?page=1&pageSize=200&keyword=${encodeURIComponent(order.orderNo)}&direction=PAYABLE`,
    ),
    request(
      `/api/v1/finance/bills?page=1&pageSize=200&status=FINANCE_BILL_STATUS_CONFIRMED&direction=PAYABLE&settlementPartyId=${supplier.id}`,
    ),
    request(
      `/api/v1/finance/cashflows?page=1&pageSize=200&status=FINANCE_CASHFLOW_STATUS_CONFIRMED&direction=PAYABLE&settlementPartyId=${supplier.id}`,
    ),
  ]);
assert(
  feesAfterVerification.data?.find((item) => item.id === confirmedFee.id)
    ?.financialProgress === 4,
  '应付全额核销后费用状态不是 VERIFIED_UNINVOICED',
);
assert(
  billsAfterVerification.data?.find((item) => item.id === confirmedBill.id)
    ?.unverifiedAmount === '0.00000000',
  '应付全额核销后账单未核销余额不为零',
);
assert(
  cashflowsAfterVerification.data?.find(
    (item) => item.id === confirmedCashflow.id,
  )?.unverifiedAmount === '0.00000000',
  '应付全额核销后付款未核销余额不为零',
);

console.log('应付费用、应付账单、付款与核销真实 API 闭环验收通过。');
console.log(
  JSON.stringify(
    {
      supplierCode: supplier.code,
      supplierName: supplier.legalName,
      orderNo: order.orderNo,
      feeId: confirmedFee.id,
      feeDirection: 'PAYABLE',
      billNo: confirmedBill.billNo,
      billDirection: confirmedBill.direction,
      cashflowNo: confirmedCashflow.flowNo,
      cashflowDirection: confirmedCashflow.direction,
      verificationNo: verification.verificationNo,
      verificationDirection: verification.direction,
      directionMismatchRejected: true,
      financialProgress: 'VERIFIED_UNINVOICED',
    },
    null,
    2,
  ),
);
