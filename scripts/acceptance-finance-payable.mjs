import { requireSuccessfulBillBatch, invoiceRegistrationForBills, withAcceptanceSystemWorkspace, prepareAcceptanceCompany } from './acceptance-finance-fixtures.mjs';

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
    const rotatedCookie = response.headers.getSetCookie().map((value) => value.split(';', 1)[0]).join('; ');
    if (rotatedCookie) cookie = rotatedCookie;
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
await prepareAcceptanceCompany(request, { allowCreate: apply });
const me = await request('/api/v1/auth/me');
assert(me.data?.currentOrganization?.id, '当前登录用户没有可用组织');
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
const today = new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date());
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

// 海运出口订单必须选择船公司并提供主单信息（047b3e79 起为强约束）。
const shippingLineResponse = await withAcceptanceSystemWorkspace(request, () => request('/api/v1/master-data/shipping-lines', {
  method: 'POST',
  body: JSON.stringify({
    scacCode: `A${Array.from({ length: 3 }, () => String.fromCharCode(65 + Math.floor(Math.random() * 26))).join('')}`,
    nameZh: `验收船公司${stamp}`,
    nameEn: `Acceptance Carrier ${stamp}`,
    countryCode: 'CN',
  }),
}));
const shippingLine = shippingLineResponse.data;
assert(shippingLine?.id, '验收船公司创建失败');
const orderResponse = await request('/api/v1/orders', {
  method: 'POST',
  body: JSON.stringify({
    personnelAssignments: [2, 3, 4].map((role) => ({ userId: me.data.id, role })),
    customerId: customer.id,
    shippingLineId: shippingLine.id,
    seaMasterBill: { masterNo: `ACCMB${stamp}` },
    seaDocument: { documentStructure: 2 }, // SEA_DOCUMENT_STRUCTURE_DIRECT 直单
    businessType: 1,
    tradeDirection: 1,
    tradeTerm: 3,
    paymentTerm: 1,
    shipmentType: 2,
    goodsDescription: '应付费用自动验收货物',
    totalPackages: 1,
    totalGrossWeightKg: 100,
    totalVolumeCbm: 1,
    customerReferenceNo: `ACC-AP-${stamp}`,
    orderDate: new Date().toISOString(),
    // 显式提供完整航期；出口费用汇率按 ETD 解析，不使用录入日回退。
    etd: new Date().toISOString(),
    eta: new Date(Date.now() + 86_400_000).toISOString(),
  }),
});
const order = orderResponse.data;
assert(order?.id && order?.orderNo, '应付验收订单创建失败');

let createdFeeSetting = null;
if (apply) {
  const createdUnit = await withAcceptanceSystemWorkspace(request, () => request('/api/v1/finance/billing-units', {
    method: 'POST',
    body: JSON.stringify({
      code: `UNIT_AP_${stamp.slice(-6)}`,
      name: '票',
      sortOrder: 1,
      isContainerUnit: false,
    }),
  }));
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
    feeSettingId: feeSetting.id,
    billingUnitId: feeSetting.defaultBillingUnitId,
    idempotencyKey: `acc-ap-fee-${stamp}`,
    taxInclusive: true,
    note: '应付费用端到端自动验收',
  }),
});
const createdFee = feeResponse.data;
assert(isEnumValue(createdFee?.direction, 2, 'PAYABLE'), '新建费用未保存为应付方向');
assert(createdFee.settlementPartyId === supplier.id, '应付费用未关联供应商');
// 费用无独立状态与确认动作：是否建账由有效账单关联（hasActiveBill）表达。
assert(createdFee.hasActiveBill !== true, '新建应付费用不应携带有效账单关联');
const confirmedFee = createdFee;
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

// 账单必须固化结算账户快照：为供应商创建本位币应付结算账户（币种与费用本位币一致）。
const accountResponse = await request(`/api/v1/partners/${supplier.id}/accounts`, {
  method: 'POST',
  body: JSON.stringify({
    partnerId: supplier.id,
    account: {
      name: `验收应付账户${stamp}`,
      accountHolder: supplierName,
      currency: options.baseCurrency,
      bankName: '验收银行',
      accountNo: `ACP${stamp}`,
      usage: 2, // PARTNER_ACCOUNT_USAGE_PAYABLE
      isDefaultPayable: true,
      enabled: true,
    },
  }),
});
const settlementAccount = accountResponse.data;
assert(settlementAccount?.id, '验收应付结算账户创建失败');

const initialPreview = await request('/api/v1/finance/bill-batches/preview', {
  method: 'POST',
  body: JSON.stringify({
    feeIds: [confirmedFee.id],
    groupingPolicy: { mode: 1, splitByOrder: true, splitByTaxRate: true },
    organizationId: me.data.currentOrganization.id,
  }),
});
assert(initialPreview.data?.length === 1, '单笔应付费用应只生成一个预览分组');
assert(
  initialPreview.data[0].configurationComplete !== true,
  '初次预览未配置账户时应为待补齐状态',
);

const preview = await request('/api/v1/finance/bill-batches/preview', {
  method: 'POST',
  body: JSON.stringify({
    feeIds: [confirmedFee.id],
    groupingPolicy: { mode: 1, splitByOrder: true, splitByTaxRate: true },
    organizationId: me.data.currentOrganization.id,
    groupConfigs: [
      {
        groupKey: initialPreview.data[0].groupKey,
        settlementAccountId: settlementAccount.id,
      },
    ],
  }),
});
assert(preview.data?.every((group) => group.groupToken), '应付账单配置后预览未签发创建令牌');
assert(preview.data?.length === 1, '配置后预览分组数不为 1');
const previewGroup = preview.data[0];
assert(previewGroup.direction === 'PAYABLE', '应付账单预览方向错误');
assert(previewGroup.settlementPartyId === supplier.id, '应付账单预览结算单位错误');

const batchResponse = await request('/api/v1/finance/bill-batches', {
  method: 'POST',
  body: JSON.stringify({
    groupingPolicy: { mode: 1, splitByOrder: true, splitByTaxRate: true },
    organizationId: me.data.currentOrganization.id,
      idempotencyKey: `acc-ap-batch-${stamp}`,
    groups: [
      {
        groupKey: previewGroup.groupKey,
        groupToken: previewGroup.groupToken,
        feeIds: previewGroup.fees.map((item) => item.id),
        statementTitle: supplierName,
        paymentTermsDays: 30,
        settlementAccountId: settlementAccount.id,
        note: '应付账单端到端自动验收',
      },
    ],
  }),
});
const batch = requireSuccessfulBillBatch(batchResponse, preview.data.length);
assert(batch?.id && batch?.bills?.length === 1, '应付账单批次创建失败');
const draftBill = batch.bills[0];
assert(draftBill.direction === 'PAYABLE', '生成的账单不是应付方向');
assert(draftBill.settlementPartyId === supplier.id, '应付账单未关联供应商');
assert(draftBill.totalAmount === '88.00000000', '应付账单金额快照错误');

const confirmedBatchResponse = { data: batch };
const confirmedBill = confirmedBatchResponse.data?.bills?.[0];
assert(confirmedBill?.status === 2, '应付账单确认失败');
assert(confirmedBill.direction === 'PAYABLE', '确认账单时应付方向丢失');

// 正向核销用例先建立真实发票事实，不使用已退役的无票核销前置。
const invoiceProfileResponse = await request(`/api/v1/partners/${supplier.id}/invoice-profiles`, {
  method: 'POST',
  body: JSON.stringify({ partnerId: supplier.id, invoiceTitle: supplier.legalName,
    taxpayerIdentificationNo: supplier.unifiedSocialCreditCode, defaultInvoiceType: 'DIGITAL_NORMAL', isDefault: true }),
});
const invoiceRule = rules.data?.find((item) => isEnumValue(item.documentType, 11, 'DOCUMENT_TYPE_INVOICE'));
assert(invoiceRule?.id, '当前组织缺少发票编号规则');
if (!invoiceRule.enabled) {
  await request(`/api/v1/master-data/number-rules/${invoiceRule.id}`, {
    method: 'PUT',
    body: JSON.stringify({ id: invoiceRule.id, prefix: invoiceRule.prefix,
      dateFormat: invoiceRule.dateFormat, sequenceLength: invoiceRule.sequenceLength,
      resetPolicy: invoiceRule.resetPolicy, enabled: true }),
  });
}
const invoiceBill = (await request(`/api/v1/finance/bills/${confirmedBill.id}`)).data;
await request('/api/v1/finance/invoices/register', {
  method: 'POST',
  body: JSON.stringify({ idempotencyKey: `acc-ap-invoice-${stamp}`, records: [invoiceRegistrationForBills([invoiceBill], {
    invoiceProfileId: invoiceProfileResponse.data.id, taxInvoiceNo: `ACC-AP-TAX-${stamp}`, invoiceDate: today,
  })] }),
});

const wrongCashflowResponse = await request('/api/v1/finance/cashflows', {
  method: 'POST',
  body: JSON.stringify({
    organizationId: me.data.currentOrganization.id,
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
    organizationId: me.data.currentOrganization.id,
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
    ?.financialProgress === 7,
  '应付票齐且全额核销后费用财务进度不是 COMPLETED',
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
      financialProgress: 'COMPLETED',
    },
    null,
    2,
  ),
);
