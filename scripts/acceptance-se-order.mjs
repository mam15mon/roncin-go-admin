const baseURL = (
  process.env.RONCIN_ACCEPTANCE_BASE_URL || 'http://127.0.0.1:8000'
).replace(/\/$/, '');
const apply = process.argv.includes('--apply');

function requireEnvironment(name) {
  const value = process.env[name]?.trim();
  if (!value) {
    throw new Error(`缺少环境变量 ${name}`);
  }
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
  if (!cookie) {
    throw new Error('管理员登录成功但响应未设置会话 Cookie');
  }
  return cookie;
}

function createClient(cookie) {
  return async function request(path, options = {}) {
    const response = await fetch(`${baseURL}${path}`, {
      ...options,
      headers: {
        cookie,
        ...(options.body ? { 'content-type': 'application/json' } : {}),
        ...options.headers,
      },
    });
    const body = await readJSON(response);
    if (!response.ok) {
      throw new Error(
        `${options.method || 'GET'} ${path} 失败（HTTP ${response.status}）：${body.message || '未知错误'}`,
      );
    }
    return body;
  };
}

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

function compactPrerequisite(item) {
  return item
    ? { id: item.id, code: item.code, name: item.name || item.legalName }
    : null;
}

const cookie = await login();
const request = createClient(cookie);
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
const orderRule = rules.data?.find(
  (item) =>
    item.documentType === 1 || item.documentType === 'DOCUMENT_TYPE_ORDER',
);

assert(me.data?.currentOrganization?.id, '当前登录用户没有可用组织');
assert(customer?.id, '当前组织没有启用的客户，请先在往来单位中创建客户');
assert(statusTemplate?.id, '当前组织没有已发布且默认的海运出口状态模板');
assert(
  statusTemplate.items?.some((item) => item.code === 'DRAFT' && item.enabled),
  '海运出口状态模板缺少启用的 DRAFT 状态',
);
assert(
  statusTemplate.items?.some((item) => item.code === 'BOOKED' && item.enabled),
  '海运出口状态模板缺少启用的 BOOKED 状态',
);
assert(orderRule?.enabled, '当前组织没有启用的订单编号规则');

const prerequisiteSummary = {
  organization: compactPrerequisite(me.data.currentOrganization),
  customer: compactPrerequisite(customer),
  statusTemplate: compactPrerequisite(statusTemplate),
  orderRule: {
    prefix: orderRule.prefix || '',
    dateFormat: orderRule.dateFormat,
    sequenceLength: orderRule.sequenceLength,
    resetPolicy: orderRule.resetPolicy,
  },
};

if (!apply) {
  console.log('海运出口订单闭环前置条件检查通过。');
  console.log(JSON.stringify(prerequisiteSummary, null, 2));
  console.log('如需创建验收订单并执行闭环检查，请追加 --apply。');
  process.exit(0);
}

const stamp = new Date()
  .toISOString()
  .replace(/[-:.TZ]/g, '')
  .slice(0, 14);
const customerReferenceNo = `ACC-SE-${stamp}`;
const masterNo = `ACC-MBL-${stamp}`;
const houseNo = `ACC-HBL-${stamp}`;
const declarationCutoffAt = new Date(
  Date.now() + 24 * 60 * 60 * 1000,
).toISOString();

const createdResponse = await request('/api/v1/orders', {
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
    declarationCutoffAt,
    goodsDescription: '海运出口闭环验收货物',
    totalPackages: 10,
    totalGrossWeightKg: 1500,
    totalVolumeCbm: 12,
    totalPackageUnit: 'CTNS',
    customerReferenceNo,
    orderDate: new Date().toISOString(),
    operationNotes: '自动化验收创建',
    shippingDocuments: [
      {
        masterNo,
        houseNo,
        releaseType: 'ORIGINAL',
        masterDocumentType: 'ORIGINAL_BL',
        masterReleaseMethod: 'ORIGINAL',
        note: '海运出口闭环验收提单',
      },
    ],
  }),
});

const created = createdResponse.data;
assert(created?.id, '创建订单响应缺少订单 ID');
assert(created.orderNo, '创建订单响应缺少订单编号');
assert(
  created.orderNo.startsWith(`${orderRule.prefix || ''}SE`),
  '订单编号未使用订单规则和 SE 业务代码',
);
assert(
  created.status === 'DRAFT',
  `新建订单状态应为 DRAFT，实际为 ${created.status}`,
);

const detailResponse = await request(`/api/v1/orders/${created.id}`);
const detail = detailResponse.data;
assert(
  detail?.customerReferenceNo === customerReferenceNo,
  '订单详情未正确回显客户业务编号',
);
assert(detail?.loadingTerms === 'CFS-CFS', '订单详情未正确回显运输条款');
assert(
  detail?.declarationCutoffAt === declarationCutoffAt,
  '订单详情未正确回显独立截申报时间',
);
assert(detail?.shippingDocuments?.length === 1, '订单详情未正确回显一条提单');
assert(
  detail.shippingDocuments[0].masterNo === masterNo,
  '订单详情主单号与创建输入不一致',
);
assert(
  detail.shippingDocuments[0].houseNo === houseNo,
  '订单详情分单号与创建输入不一致',
);

const updatedResponse = await request(`/api/v1/orders/${created.id}`, {
  method: 'PUT',
  body: JSON.stringify({
    id: created.id,
    expectedStatus: 'DRAFT',
    notes: '海运出口闭环验收草稿已更新',
    shippingDocuments: detail.shippingDocuments.map((document) => ({
      id: document.id,
      masterNo: document.masterNo,
      houseNo: document.houseNo,
      releaseType: document.releaseType,
      note: document.note,
      masterDocumentType: document.masterDocumentType,
      masterReleaseMethod: document.masterReleaseMethod,
    })),
    containerRequests: (detail.containerRequests || []).map((item) => ({
      id: item.id,
      containerSpecId: item.containerSpecId,
      quantity: item.quantity,
    })),
  }),
});
assert(
  updatedResponse.data?.notes === '海运出口闭环验收草稿已更新',
  '订单草稿更新未生效',
);
assert(
  updatedResponse.data?.shippingDocuments?.length === 1,
  '更新草稿时提单数据被意外清空',
);

const cargoResponse = await request(
  `/api/v1/orders/${created.id}/cargo-items`,
  {
    method: 'POST',
    body: JSON.stringify({
      orderId: created.id,
      cargoName: '验收货物',
      packageCount: 10,
      grossWeightKg: 1250.5,
      volumeCbm: 12.6,
      netWeightKg: 1180,
      note: '海运出口闭环验收货物明细',
    }),
  },
);
assert(cargoResponse.data?.id, '新增货物明细响应缺少 ID');

const cargoList = await request(`/api/v1/orders/${created.id}/cargo-items`);
assert(
  cargoList.data?.some((item) => item.id === cargoResponse.data.id),
  '货物明细列表未返回新增记录',
);

const siblingReferenceNo = `${customerReferenceNo}-02`;
const siblingHouseNo = `${houseNo}-02`;
const siblingResponse = await request('/api/v1/orders', {
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
    goodsDescription: '海运出口自拼汇总验收货物',
    totalPackages: 5,
    totalGrossWeightKg: 800,
    totalVolumeCbm: 6.2,
    customerReferenceNo: siblingReferenceNo,
    orderDate: new Date().toISOString(),
    shippingDocuments: [
      {
        masterNo,
        houseNo: siblingHouseNo,
        releaseType: 'ORIGINAL',
        masterDocumentType: 'ORIGINAL_BL',
        masterReleaseMethod: 'ORIGINAL',
      },
    ],
  }),
});
const sibling = siblingResponse.data;
assert(sibling?.id, '创建自拼成员订单响应缺少 ID');

await request(`/api/v1/orders/${sibling.id}/cargo-items`, {
  method: 'POST',
  body: JSON.stringify({
    orderId: sibling.id,
    cargoName: '自拼成员验收货物',
    packageCount: 5,
    grossWeightKg: 760,
    volumeCbm: 6,
  }),
});

const consolidationResponse = await request(
  `/api/v1/orders/${created.id}/consolidations`,
);
const consolidation = consolidationResponse.data?.find(
  (item) => item.masterNo === masterNo,
);
assert(consolidation, '自拼汇总未返回当前主单批次');
assert(consolidation.memberCount === 2, '自拼汇总成员票数应为 2');
assert(
  consolidation.entrusted?.packages === 15 &&
    consolidation.entrusted?.grossWeightKg === 2300 &&
    consolidation.entrusted?.volumeCbm === 18.2,
  '自拼汇总委托件重尺不正确',
);
assert(
  consolidation.actual?.packages === 15 &&
    consolidation.actual?.grossWeightKg === 2010.5 &&
    consolidation.actual?.volumeCbm === 18.6,
  '自拼汇总实际件重尺不正确',
);
assert(
  consolidation.members?.some(
    (member) =>
      member.orderId === sibling.id &&
      member.houseNos?.includes(siblingHouseNo),
  ),
  '自拼汇总未返回成员订单及分单号',
);

const transitionedResponse = await request(
  `/api/v1/orders/${created.id}/status`,
  {
    method: 'POST',
    body: JSON.stringify({
      id: created.id,
      expectedStatus: 'DRAFT',
      targetStatus: 'BOOKED',
      reason: '海运出口闭环验收：完成订舱',
    }),
  },
);
assert(
  transitionedResponse.data?.status === 'BOOKED',
  '订单未成功流转到 BOOKED',
);

const finalDetail = await request(`/api/v1/orders/${created.id}`);
assert(
  finalDetail.data?.status === 'BOOKED',
  '状态流转后订单详情未回显 BOOKED',
);
assert(
  finalDetail.data?.shippingDocuments?.length === 1,
  '状态流转后提单数据不完整',
);
assert(
  finalDetail.data?.loadingTerms === 'CFS-CFS' &&
    finalDetail.data?.declarationCutoffAt === declarationCutoffAt,
  '状态流转后运输条款或截申报时间被意外清空',
);

const orderList = await request(
  `/api/v1/orders?page=1&pageSize=20&keyword=${encodeURIComponent(created.orderNo)}`,
);
assert(
  orderList.data?.some((item) => item.id === created.id),
  '订单列表无法按订单编号检索验收订单',
);

console.log('海运出口订单闭环验收通过。');
console.log(
  JSON.stringify(
    {
      ...prerequisiteSummary,
      order: {
        id: finalDetail.data.id,
        orderNo: finalDetail.data.orderNo,
        status: finalDetail.data.status,
        customerReferenceNo: finalDetail.data.customerReferenceNo,
        shippingDocumentCount: finalDetail.data.shippingDocuments?.length || 0,
        cargoItemCount: cargoList.data?.length || 0,
        consolidationMemberCount: consolidation.memberCount,
      },
    },
    null,
    2,
  ),
);
