function requireEnvironment(name) {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`缺少必需环境变量 ${name}`);
  return value;
}

const baseURL = requireEnvironment('RONCIN_ACCEPTANCE_BASE_URL').replace(/\/+$/, '');

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
const { request, raw } = createClient(cookie);

// 1. 验证空环境前置条件
const [me, existingPartners, existingOrders, existingBills, existingCashflows, rules] =
  await Promise.all([
    request('/api/v1/auth/me'),
    request('/api/v1/partners?page=1&pageSize=10'),
    request('/api/v1/orders?page=1&pageSize=10'),
    request('/api/v1/finance/bills?page=1&pageSize=10'),
    request('/api/v1/finance/cashflows?page=1&pageSize=10'),
    request('/api/v1/master-data/number-rules'),
  ]);

const currentOrg = me.data?.currentOrganization;
assert(me.data?.id, '当前登录用户缺少用户编号');
assert(currentOrg?.id, '当前登录用户没有可用组织');
assert(
  currentOrg.baseCurrency === 'CNY',
  `外币全链路验收要求初始组织本位币为 CNY，实际为 ${currentOrg.baseCurrency}`,
);
assert(
  (existingPartners.data?.length ?? 0) === 0 &&
    (existingOrders.data?.length ?? 0) === 0 &&
    (existingBills.data?.length ?? 0) === 0 &&
    (existingCashflows.data?.length ?? 0) === 0,
  '外币全链路验收必须在无业务数据的全新环境中运行',
);

const requiredDocumentTypes = [
  [2, 'DOCUMENT_TYPE_BILL'],
  [4, 'DOCUMENT_TYPE_WRITE_OFF'],
  [5, 'DOCUMENT_TYPE_RECEIPT_PAYMENT'],
  [13, 'DOCUMENT_TYPE_COMMISSION'],
  [14, 'DOCUMENT_TYPE_BILL_BATCH'],
];
for (const [code, name] of requiredDocumentTypes) {
  assert(
    rules.data?.some((item) => item.enabled && item.documentType === code),
    `当前组织缺少启用的财务编号规则：${name} (type=${code})`,
  );
}

// 2. 将组织本位币变更为 USD（业务数据写入前）
const updateOrgResponse = await request(
  `/api/v1/admin/organizations/${currentOrg.id}`,
  {
    method: 'PUT',
    body: JSON.stringify({
      id: currentOrg.id,
      name: currentOrg.name,
      enabled: true,
      baseCurrency: 'USD',
    }),
  },
);
assert(
  updateOrgResponse.data?.baseCurrency === 'USD',
  `组织本位币设置失败，实际为 ${updateOrgResponse.data?.baseCurrency}`,
);

// 3. 配置五类时间标准
const timeStandardsResponse = await request(
  '/api/v1/finance/exchange-rate-time-standards',
  {
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
  },
);
assert(
  timeStandardsResponse.data?.length === 5,
  '汇率时间标准设置未返回 5 项配置',
);

// 4. 创建六条系统汇率（五条 EUR→USD，一条 CNY→USD）
const shanghaiDateFormatter = new Intl.DateTimeFormat('en-CA', {
  timeZone: 'Asia/Shanghai',
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
});
const today = shanghaiDateFormatter.format(new Date());
const stamp = `${today.replace(/-/g, '')}${Date.now().toString().slice(-6)}`;
const effectiveFrom = `${today}T00:00:00+08:00`;

const rateConfigs = [
  {
    key: 'BASE_CURRENCY',
    rateType: 'BASE_CURRENCY',
    fromCurrency: 'EUR',
    toCurrency: 'USD',
    rate: '1.10000000',
  },
  {
    key: 'BILL',
    rateType: 'BILL',
    fromCurrency: 'EUR',
    toCurrency: 'USD',
    rate: '1.20000000',
  },
  {
    key: 'INVOICE',
    rateType: 'INVOICE',
    fromCurrency: 'EUR',
    toCurrency: 'USD',
    rate: '1.22000000',
  },
  {
    key: 'SETTLEMENT',
    rateType: 'SETTLEMENT',
    fromCurrency: 'EUR',
    toCurrency: 'USD',
    rate: '1.25000000',
  },
  {
    key: 'WRITE_OFF',
    rateType: 'WRITE_OFF',
    fromCurrency: 'EUR',
    toCurrency: 'USD',
    rate: '1.30000000',
  },
  {
    key: 'CNY_WRITE_OFF',
    rateType: 'WRITE_OFF',
    fromCurrency: 'CNY',
    toCurrency: 'USD',
    rate: '0.14000000',
  },
];

const createdRateSettings = {};
for (const config of rateConfigs) {
  const response = await request('/api/v1/finance/exchange-rates', {
    method: 'POST',
    body: JSON.stringify({
      rateType: config.rateType,
      fromCurrency: config.fromCurrency,
      toCurrency: config.toCurrency,
      effectiveFrom,
      receivableRate: config.rate,
      payableRate: config.rate,
    }),
  });
  const setting = response.data;
  assert(
    setting?.id &&
      setting.rateType === config.rateType &&
      setting.fromCurrency === config.fromCurrency &&
      setting.toCurrency === config.toCurrency &&
      setting.receivableRate === config.rate,
    `汇率配置 ${config.key} 创建失败`,
  );
  createdRateSettings[config.key] = setting;
}

// 5. 创建客户
const customerCode = `ACC-FC-CUST-${stamp}`;
const customerLegalName = `外币验收客户 ${stamp}`;
const customerUSCC = `91310000EUR${stamp.slice(-7)}`;
const customerAddress = '外币自动验收地址';
const customerResponse = await request('/api/v1/partners', {
  method: 'POST',
  body: JSON.stringify({
    code: customerCode,
    legalName: customerLegalName,
    unifiedSocialCreditCode: customerUSCC,
    registeredAddress: customerAddress,
    roles: [{ type: 1, enabled: true }],
    contacts: [
      {
        name: '外币验收联系人',
        phone: '13800000000',
        isPrimary: true,
      },
    ],
    aliases: [{ aliasName: `FC CUSTOMER ${stamp}`, sortOrder: 1 }],
    profile: {
      nameEn: `Foreign Currency Acceptance Customer ${stamp}`,
      countryCode: 'CN',
      businessTypes: [1],
      remark: '外币财务全链路自动验收创建',
    },
    assignments: [
      {
        role: 3,
        userId: me.data.id,
        organizationId: me.data.currentOrganization.id,
      },
    ],
  }),
});
const customer = customerResponse.data;
assert(
  customer?.id &&
    customer?.code === customerCode &&
    customer?.legalName === customerLegalName &&
    customer?.unifiedSocialCreditCode === customerUSCC &&
    customer?.registeredAddress === customerAddress,
  '外币验收客户创建失败或字段不符',
);

// 6. 创建订单
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
    goodsDescription: '外币财务全链路验收货物',
    totalPackages: 1,
    totalGrossWeightKg: 100,
    totalVolumeCbm: 1,
    customerReferenceNo: `ACC-FC-${stamp}`,
    orderDate: new Date().toISOString(),
  }),
});
const order = orderResponse.data;
assert(order?.id && order?.orderNo, '外币验收订单创建失败');

const invoiceRule = rules.data?.find((item) => item.documentType === 11);
assert(invoiceRule?.id, '当前组织缺少发票编号规则');
if (!invoiceRule.enabled) {
  await request(`/api/v1/master-data/number-rules/${invoiceRule.id}`, {
    method: 'PUT',
    body: JSON.stringify({
      id: invoiceRule.id,
      prefix: 'INV',
      dateFormat: 1,
      sequenceLength: 5,
      resetPolicy: 1,
      enabled: true,
    }),
  });
}

// 7. 显式创建外币费用主数据与 EUR 应收费用（100 EUR @ 1.10 = 110.00000000 USD）
const createdUnit = await request('/api/v1/finance/billing-units', {
  method: 'POST',
  body: JSON.stringify({
    code: `UNIT_FC_${stamp.slice(-6)}`,
    name: '票',
    sortOrder: 1,
    isContainerUnit: false,
  }),
});
const billingUnit = createdUnit.data;
assert(billingUnit?.id, '外币计费单位创建失败');

const createdService = await request('/api/v1/finance/taxable-services', {
  method: 'POST',
  body: JSON.stringify({
    name: `外币物流服务_${stamp.slice(-6)}`,
    defaultTaxRate: '0.00',
  }),
});
const taxableService = createdService.data;
assert(taxableService?.id, '外币应税服务创建失败');

const createdFeeSetting = await request('/api/v1/finance/fee-settings', {
  method: 'POST',
  body: JSON.stringify({
    feeCode: `FEE_FC_${stamp.slice(-6)}`,
    nameZh: '外币海运运费',
    defaultCurrency: 'EUR',
    billingUnitId: billingUnit.id,
    taxableServiceId: taxableService.id,
    taxRate: '0.00',
    sortOrder: 1,
  }),
});
const feeSetting = createdFeeSetting.data;
assert(feeSetting?.id, '外币费用科目创建失败');

const options = await request(`/api/v1/orders/${order.id}/fee-options`);
assert(options.baseCurrency === 'USD', `费用选项本位币应为 USD，实际 ${options.baseCurrency}`);
const feeSettingOption = options.feeSettings?.find(
  (item) => item.id === feeSetting.id,
);
const settlementParty = options.settlementParties?.find(
  (item) => item.id === customer.id,
);
assert(feeSettingOption?.id, '费用选项中未返回新创建的费用科目');
assert(settlementParty?.id, '费用选项中未返回验收客户');

const feeResponse = await request(`/api/v1/orders/${order.id}/fees`, {
  method: 'POST',
  body: JSON.stringify({
    orderId: order.id,
    direction: 1,
    settlementPartyId: settlementParty.id,
    quantity: '1',
    unitPrice: '100.00',
    currency: 'EUR',
    expenseDate: today,
    feeSettingId: feeSetting.id,
    billingUnitId: billingUnit.id,
    idempotencyKey: `acc-fc-fee-${stamp}`,
    taxInclusive: true,
    note: '外币财务全链路自动验收',
  }),
});
const fee = feeResponse.data;
assert(fee?.id, 'EUR 应收费用创建失败');
assert(fee.currency === 'EUR', `费用币种应为 EUR，实际 ${fee.currency}`);
assert(fee.totalAmount === '100.00000000', `费用总额应为 100.00000000，实际 ${fee.totalAmount}`);
assert(fee.baseCurrency === 'USD', `费用本位币应为 USD，实际 ${fee.baseCurrency}`);
assert(fee.exchangeRate === '1.10000000', `费用系统汇率应为 1.10000000，实际 ${fee.exchangeRate}`);
assert(fee.exchangeRateSource === 'SYSTEM', `费用汇率来源应为 SYSTEM，实际 ${fee.exchangeRateSource}`);
assert(fee.exchangeRateDate === today, `费用汇率日期应为 ${today}，实际 ${fee.exchangeRateDate}`);
assert(
  fee.exchangeRateSettingId === createdRateSettings.BASE_CURRENCY.id,
  '费用汇率 setting ID 未匹配 BASE_CURRENCY 配置',
);
assert(
  fee.baseCurrencyAmount === '110.00000000',
  `费用折算本币应为 110.00000000 USD，实际 ${fee.baseCurrencyAmount}`,
);

// 确认费用
const confirmedFeeResponse = await request(
  `/api/v1/orders/${order.id}/fees/${fee.id}/confirm`,
  {
    method: 'POST',
    body: JSON.stringify({
      orderId: order.id,
      id: fee.id,
      expectedVersion: fee.version,
    }),
  },
);
assert(confirmedFeeResponse.data?.status === 2, '费用确认失败');

// 8. 账单预览与创建（100 EUR @ 1.20 = 120.00000000 USD）
const preview = await request('/api/v1/finance/bill-batches/preview', {
  method: 'POST',
  body: JSON.stringify({
    feeIds: [fee.id],
    groupingPolicy: { splitByOrder: true, splitByTaxRate: true },
  }),
});
assert(preview.previewToken?.length === 64, '账单预览未返回有效快照令牌');
assert(preview.data?.length === 1, '账单预览分组数不为 1');

const batchResponse = await request('/api/v1/finance/bill-batches', {
  method: 'POST',
  body: JSON.stringify({
    feeIds: [fee.id],
    groupingPolicy: { splitByOrder: true, splitByTaxRate: true },
    previewToken: preview.previewToken,
    idempotencyKey: `acc-fc-batch-${stamp}`,
    groups: [
      {
        groupKey: preview.data[0].groupKey,
        statementTitle: customer.legalName,
        billDate: today,
        paymentTermsDays: 30,
        note: '外币账单全链路自动验收',
      },
    ],
  }),
});
const batch = batchResponse.data;
assert(batch?.bills?.length === 1, '批量建单未生成 1 张账单');
const draftBill = batch.bills[0];
assert(draftBill.currency === 'EUR', `账单币种应为 EUR，实际 ${draftBill.currency}`);
assert(draftBill.totalAmount === '100.00000000', `账单金额应为 100.00000000，实际 ${draftBill.totalAmount}`);
assert(draftBill.baseCurrency === 'USD', `账单本位币应为 USD，实际 ${draftBill.baseCurrency}`);
assert(draftBill.exchangeRate === '1.20000000', `账单系统汇率应为 1.20000000，实际 ${draftBill.exchangeRate}`);
assert(draftBill.exchangeRateSource === 'SYSTEM', `账单汇率来源应为 SYSTEM，实际 ${draftBill.exchangeRateSource}`);
assert(draftBill.exchangeRateDate === today, `账单汇率日期应为 ${today}，实际 ${draftBill.exchangeRateDate}`);
assert(
  draftBill.exchangeRateSettingId === createdRateSettings.BILL.id,
  '账单汇率 setting ID 未匹配 BILL 配置',
);
assert(
  draftBill.baseCurrencyAmount === '120.00000000',
  `账单折算本币应为 120.00000000 USD，实际 ${draftBill.baseCurrencyAmount}`,
);

// 确认账单
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
assert(confirmedBill?.status === 2, '账单确认失败');

// 9. 创建开票资料、开票草稿并登记开具（100 EUR @ 1.22 = 122.00000000 USD）
const profileResponse = await request(
  `/api/v1/partners/${customer.id}/invoice-profiles`,
  {
    method: 'POST',
    body: JSON.stringify({
      partnerId: customer.id,
      invoiceTitle: customer.legalName,
      taxpayerIdentificationNo: customerUSCC,
      registeredAddress: customerAddress,
      registeredPhone: '',
      bankName: '',
      bankAccount: '',
      defaultInvoiceType: 'NORMAL',
      isDefault: true,
    }),
  },
);
const profile = profileResponse.data;
assert(profile?.id, '开票资料创建失败');

const invoiceResponse = await request('/api/v1/finance/invoices', {
  method: 'POST',
  body: JSON.stringify({
    billIds: [confirmedBill.id],
    invoiceProfileId: profile.id,
    invoiceType: 'NORMAL',
    note: '外币发票全链路自动验收',
    idempotencyKey: `acc-fc-invoice-${stamp}`,
  }),
});
const draftInvoice = invoiceResponse.data;
assert(draftInvoice?.id, '开票草稿创建失败');

const issuedInvoiceResponse = await request(
  `/api/v1/finance/invoices/${draftInvoice.id}/issue`,
  {
    method: 'POST',
    body: JSON.stringify({
      id: draftInvoice.id,
      expectedVersion: draftInvoice.version,
      taxInvoiceNo: `ACC-FC-TAX-${stamp}`,
      invoiceDate: today,
    }),
  },
);
const issuedInvoice = issuedInvoiceResponse.data;
assert(issuedInvoice?.status === 2, '发票登记开具失败');
assert(issuedInvoice.currency === 'EUR', `发票币种应为 EUR，实际 ${issuedInvoice.currency}`);
assert(issuedInvoice.totalAmount === '100.00000000', `发票金额应为 100.00000000，实际 ${issuedInvoice.totalAmount}`);
assert(issuedInvoice.baseCurrency === 'USD', `发票本位币应为 USD，实际 ${issuedInvoice.baseCurrency}`);
assert(issuedInvoice.exchangeRate === '1.22000000', `发票系统汇率应为 1.22000000，实际 ${issuedInvoice.exchangeRate}`);
assert(issuedInvoice.exchangeRateSource === 'SYSTEM', `发票汇率来源应为 SYSTEM，实际 ${issuedInvoice.exchangeRateSource}`);
assert(issuedInvoice.exchangeRateDate === today, `发票汇率日期应为 ${today}，实际 ${issuedInvoice.exchangeRateDate}`);
assert(
  issuedInvoice.exchangeRateSettingId === createdRateSettings.INVOICE.id,
  '发票汇率 setting ID 未匹配 INVOICE 配置',
);
assert(
  issuedInvoice.baseCurrencyAmount === '122.00000000',
  `发票折算本币应为 122.00000000 USD，实际 ${issuedInvoice.baseCurrencyAmount}`,
);

// 10. 创建 EUR 收款流水并确认（40 EUR @ 1.25 = 50.00000000 USD，禁止携带 exchangeRate）
const cashflowResponse = await request('/api/v1/finance/cashflows', {
  method: 'POST',
  body: JSON.stringify({
    direction: 'RECEIVABLE',
    settlementPartyId: customer.id,
    currency: 'EUR',
    amount: '40.00000000',
    baseCurrency: 'USD',
    transactionDate: today,
    ourAccount: '外币验收基本户',
    counterpartyAccount: '外币验收客户账户',
    paymentMethod: '银行转账',
    bankReferenceNo: `ACC-FC-BANK-${stamp}`,
    note: '外币资金收款全链路自动验收',
    idempotencyKey: `acc-fc-cashflow-${stamp}`,
  }),
});
const draftCashflow = cashflowResponse.data;
assert(draftCashflow?.id, '资金流水创建失败');
assert(draftCashflow.currency === 'EUR', `流水币种应为 EUR，实际 ${draftCashflow.currency}`);
assert(draftCashflow.amount === '40.00000000', `流水金额应为 40.00000000，实际 ${draftCashflow.amount}`);
assert(draftCashflow.baseCurrency === 'USD', `流水本位币应为 USD，实际 ${draftCashflow.baseCurrency}`);
assert(draftCashflow.exchangeRate === '1.25000000', `流水系统汇率应为 1.25000000，实际 ${draftCashflow.exchangeRate}`);
assert(draftCashflow.exchangeRateSource === 'SYSTEM', `流水汇率来源应为 SYSTEM，实际 ${draftCashflow.exchangeRateSource}`);
assert(draftCashflow.exchangeRateDate === today, `流水汇率日期应为 ${today}，实际 ${draftCashflow.exchangeRateDate}`);
assert(
  draftCashflow.exchangeRateSettingId === createdRateSettings.SETTLEMENT.id,
  '流水汇率 setting ID 未匹配 SETTLEMENT 配置',
);
assert(
  draftCashflow.baseAmount === '50.00000000',
  `流水折算本币应为 50.00000000 USD，实际 ${draftCashflow.baseAmount}`,
);

const confirmedCashflowResponse = await request(
  `/api/v1/finance/cashflows/${draftCashflow.id}/confirm`,
  {
    method: 'POST',
    body: JSON.stringify({
      id: draftCashflow.id,
      expectedVersion: draftCashflow.version,
    }),
  },
);
const confirmedCashflow = confirmedCashflowResponse.data;
assert(confirmedCashflow?.status === 2, '资金流水确认失败');

// 11. 执行 EUR 核销（40 EUR @ 1.30 = 52.00000000 USD）
const verificationResponse = await request('/api/v1/finance/verifications', {
  method: 'POST',
  body: JSON.stringify({
    allocations: [
      {
        cashflowId: confirmedCashflow.id,
        billId: confirmedBill.id,
        amount: '40.00000000',
      },
    ],
    verificationDate: today,
    note: '外币核销全链路自动验收',
    idempotencyKey: `acc-fc-verification-${stamp}`,
  }),
});
const verification = verificationResponse.data;
assert(verification?.id, '核销记录创建失败');
assert(verification.currency === 'EUR', `核销币种应为 EUR，实际 ${verification.currency}`);
assert(verification.amount === '40.00000000', `核销金额应为 40.00000000，实际 ${verification.amount}`);
assert(verification.baseCurrency === 'USD', `核销本位币应为 USD，实际 ${verification.baseCurrency}`);
assert(verification.exchangeRate === '1.30000000', `核销系统汇率应为 1.30000000，实际 ${verification.exchangeRate}`);
assert(verification.exchangeRateSource === 'SYSTEM', `核销汇率来源应为 SYSTEM，实际 ${verification.exchangeRateSource}`);
assert(verification.exchangeRateDate === today, `核销汇率日期应为 ${today}，实际 ${verification.exchangeRateDate}`);
assert(
  verification.exchangeRateSettingId === createdRateSettings.WRITE_OFF.id,
  '核销汇率 setting ID 未匹配 WRITE_OFF 配置',
);
assert(
  verification.baseAmount === '52.00000000',
  `核销本币金额应为 52.00000000 USD，实际 ${verification.baseAmount}`,
);
assert(
  verification.billBaseAmount === '48.00000000',
  `账单本币分摊应为 48.00000000 USD，实际 ${verification.billBaseAmount}`,
);
assert(
  verification.cashflowBaseAmount === '50.00000000',
  `资金流水本币分摊应为 50.00000000 USD，实际 ${verification.cashflowBaseAmount}`,
);
assert(
  verification.exchangeGainLoss === '2.00000000',
  `应收汇兑收益应为 2.00000000 USD，实际 ${verification.exchangeGainLoss}`,
);

// 12. 指派订单销售人员与创建 10% 毛利提成规则
await request(`/api/v1/orders/${order.id}/personnel`, {
  method: 'POST',
  body: JSON.stringify({
    orderId: order.id,
    userId: me.data.id,
    organizationId: currentOrg.id,
    role: 3, // SALES
  }),
});

const commissionRuleResponse = await request(
  '/api/v1/finance/commission-rules',
  {
    method: 'POST',
    body: JSON.stringify({
      rule: {
        name: `ACC-FC-COMM-${stamp}`,
        personnelRole: 'SALES',
        calculationBasis: 'REALIZED_PROFIT',
        ratePercent: '10',
        effectiveFrom: today,
        effectiveTo: today,
        enabled: true,
        note: '外币提成全链路自动验收',
      },
    }),
  },
);
const commissionRule = commissionRuleResponse.data;
assert(commissionRule?.id, '提成规则创建失败');

// 13. 提成预览与创建断言
const commissionPreviewResponse = await request(
  '/api/v1/finance/commissions/preview',
  {
    method: 'POST',
    body: JSON.stringify({
      verificationId: verification.id,
      employeeId: me.data.id,
      ruleId: commissionRule.id,
    }),
  },
);
const previewCommission = commissionPreviewResponse.data;
assert(previewCommission?.lines?.length === 1, '提成预览明细数不为 1');
assert(previewCommission.lines[0].orderId === order.id, '提成预览明细 orderId 不符');
assert(previewCommission.baseCurrency === 'USD', `提成预览本位币应为 USD，实际 ${previewCommission.baseCurrency}`);
assert(
  previewCommission.realizedRevenue === '44.00000000',
  `提成已实现收入应为 44.00000000 USD，实际 ${previewCommission.realizedRevenue}`,
);
assert(
  previewCommission.allocatedCost === '0.00000000',
  `提成分摊成本应为 0.00000000 USD，实际 ${previewCommission.allocatedCost}`,
);
assert(
  previewCommission.realizedProfit === '44.00000000',
  `提成已实现利润应为 44.00000000 USD，实际 ${previewCommission.realizedProfit}`,
);
assert(
  previewCommission.commissionAmount === '4.40000000',
  `USD 提成金额应为 4.40000000 USD，实际 ${previewCommission.commissionAmount}`,
);
assert(
  previewCommission.cnyExchangeRate === '7.14285714',
  `CNY 派生汇率应为 7.14285714，实际 ${previewCommission.cnyExchangeRate}`,
);
assert(
  previewCommission.cnyExchangeRateSource === 'DERIVED',
  `CNY 汇率来源应为 DERIVED，实际 ${previewCommission.cnyExchangeRateSource}`,
);
assert(
  previewCommission.cnyExchangeRateDate === today,
  `CNY 汇率日期应为 ${today}，实际 ${previewCommission.cnyExchangeRateDate}`,
);
assert(
  previewCommission.cnyExchangeRateSettingId ===
    createdRateSettings.CNY_WRITE_OFF.id,
  'CNY 汇率 setting ID 未匹配 CNY_WRITE_OFF 配置',
);
assert(
  previewCommission.cnyCommissionAmount === '31.42857142',
  `CNY 提成金额应为 31.42857142 CNY，实际 ${previewCommission.cnyCommissionAmount}`,
);

// 创建提成草稿
const commissionCreateResponse = await request(
  '/api/v1/finance/commissions',
  {
    method: 'POST',
    body: JSON.stringify({
      verificationId: verification.id,
      employeeId: me.data.id,
      ruleId: commissionRule.id,
      note: '外币提成全链路自动验收创建',
      idempotencyKey: `acc-fc-commission-${stamp}`,
    }),
  },
);
const createdCommission = commissionCreateResponse.data;
assert(createdCommission?.id && createdCommission?.commissionNo, '提成创建失败');
assert(createdCommission.baseCurrency === 'USD', `创建提成本位币应为 USD，实际 ${createdCommission.baseCurrency}`);
assert(
  createdCommission.realizedRevenue === '44.00000000',
  `创建提成已实现收入应为 44.00000000 USD，实际 ${createdCommission.realizedRevenue}`,
);
assert(
  createdCommission.commissionAmount === '4.40000000',
  `创建 USD 提成金额应为 4.40000000 USD，实际 ${createdCommission.commissionAmount}`,
);
assert(
  createdCommission.cnyExchangeRate === '7.14285714',
  `创建提成 CNY 派生汇率应为 7.14285714，实际 ${createdCommission.cnyExchangeRate}`,
);
assert(
  createdCommission.cnyExchangeRateSource === 'DERIVED',
  `创建提成 CNY 汇率来源应为 DERIVED，实际 ${createdCommission.cnyExchangeRateSource}`,
);
assert(
  createdCommission.cnyExchangeRateDate === today,
  `创建提成 CNY 汇率日期应为 ${today}，实际 ${createdCommission.cnyExchangeRateDate}`,
);
assert(
  createdCommission.cnyExchangeRateSettingId ===
    createdRateSettings.CNY_WRITE_OFF.id,
  '创建提成 CNY 汇率 setting ID 未匹配 CNY_WRITE_OFF 配置',
);
assert(
  createdCommission.cnyCommissionAmount === '31.42857142',
  `创建提成 CNY 金额应为 31.42857142 CNY，实际 ${createdCommission.cnyCommissionAmount}`,
);

// 14. 详情重读持久化快照
const commissionDetailResponse = await request(
  `/api/v1/finance/commissions/${createdCommission.id}`,
);
const persistedCommission = commissionDetailResponse.data;
assert(
  persistedCommission.commissionAmount === '4.40000000' &&
    persistedCommission.cnyExchangeRate === '7.14285714' &&
    persistedCommission.cnyExchangeRateSource === 'DERIVED' &&
    persistedCommission.cnyExchangeRateDate === today &&
    persistedCommission.cnyExchangeRateSettingId ===
      createdRateSettings.CNY_WRITE_OFF.id &&
    persistedCommission.cnyCommissionAmount === '31.42857142',
  '持久化提成详情快照重读不一致',
);

console.log('USD 本位币 EUR 业务币外币财务全链路 API 连续验收通过。');
console.log(
  JSON.stringify(
    {
      organizationBaseCurrency: 'USD',
      rateSettingIds: {
        baseCurrency: createdRateSettings.BASE_CURRENCY.id,
        bill: createdRateSettings.BILL.id,
        invoice: createdRateSettings.INVOICE.id,
        settlement: createdRateSettings.SETTLEMENT.id,
        writeOff: createdRateSettings.WRITE_OFF.id,
        cnyWriteOff: createdRateSettings.CNY_WRITE_OFF.id,
      },
      customerCode: customer.code,
      orderNo: order.orderNo,
      feeId: fee.id,
      feeAmountEUR: fee.totalAmount,
      feeExchangeRate: fee.exchangeRate,
      feeBaseAmountUSD: fee.baseCurrencyAmount,
      billNo: confirmedBill.billNo,
      billAmountEUR: confirmedBill.totalAmount,
      billExchangeRate: confirmedBill.exchangeRate,
      billBaseAmountUSD: confirmedBill.baseCurrencyAmount,
      invoiceRecordNo: issuedInvoice.recordNo,
      invoiceTaxNo: issuedInvoice.taxInvoiceNo,
      invoiceExchangeRate: issuedInvoice.exchangeRate,
      invoiceBaseAmountUSD: issuedInvoice.baseCurrencyAmount,
      cashflowNo: confirmedCashflow.flowNo,
      cashflowAmountEUR: confirmedCashflow.amount,
      cashflowExchangeRate: confirmedCashflow.exchangeRate,
      cashflowBaseAmountUSD: confirmedCashflow.baseAmount,
      verificationNo: verification.verificationNo,
      verificationAmountEUR: verification.amount,
      verificationExchangeRate: verification.exchangeRate,
      verificationBaseAmountUSD: verification.baseAmount,
      billBaseAmountUSD_Allocated: verification.billBaseAmount,
      cashflowBaseAmountUSD_Allocated: verification.cashflowBaseAmount,
      exchangeGainLossUSD: verification.exchangeGainLoss,
      commissionNo: createdCommission.commissionNo,
      commissionAmountUSD: createdCommission.commissionAmount,
      cnyDerivedExchangeRate: createdCommission.cnyExchangeRate,
      cnyExchangeRateSource: createdCommission.cnyExchangeRateSource,
      cnyCommissionAmount: createdCommission.cnyCommissionAmount,
      persistedSnapshotVerified: true,
    },
    null,
    2,
  ),
);
