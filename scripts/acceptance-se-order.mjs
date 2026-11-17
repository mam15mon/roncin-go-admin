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

function isEnumValue(value, code, name) {
  return value === code || value === name;
}

function compactPrerequisite(item) {
  return item
    ? { id: item.id, code: item.code, name: item.name || item.legalName }
    : null;
}

const cookie = await login();
const request = createClient(cookie);
const [me, customers, rules, shippingLines] = await Promise.all([
  request('/api/v1/auth/me'),
  request('/api/v1/partners?page=1&pageSize=100&role=1&enabled=true'),
  request('/api/v1/master-data/number-rules'),
  request('/api/v1/master-data/shipping-lines?page=1&pageSize=1&enabled=true'),
]);

const customer = customers.data?.[0];
const shippingLine = shippingLines.data?.[0];
const orderRule = rules.data?.find(
  (item) =>
    item.documentType === 1 || item.documentType === 'DOCUMENT_TYPE_ORDER',
);

assert(me.data?.currentOrganization?.id, '当前登录用户没有可用组织');
assert(customer?.id, '当前组织没有启用的客户，请先在往来单位中创建客户');
assert(shippingLine?.id, '缺少启用船公司，请先同步行业主数据');
assert(orderRule?.enabled, '当前组织没有启用的订单编号规则');

const prerequisiteSummary = {
  organization: compactPrerequisite(me.data.currentOrganization),
  customer: compactPrerequisite(customer),
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
const masterNo = `ACCMBL${stamp}`;
const houseNo = `ACC-HBL-${stamp}`;
const etd = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString();
const eta = new Date(Date.now() + 21 * 24 * 60 * 60 * 1000).toISOString();
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
    shipmentType: 2,
    shippingLineId: shippingLine.id,
    personnelAssignments: [2, 3, 4].map((role) => ({ role, userId: me.data.id })),
    etd,
    eta,
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
    seaMasterBill: { masterNo },
    seaDocument: {
      documentStructure: 3,
      houseBill: { houseNo, issuerSource: 1 },
    },
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
  isEnumValue(created.flowStatus, 1, 'ORDER_FLOW_STATUS_DRAFT'),
  `新建订单主流程应为 DRAFT，实际为 ${created.flowStatus}`,
);
assert(created.version === '1' || created.version === 1, '新建订单版本号应为 1');

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
assert(detail?.seaDocumentSummary?.houseNo === houseNo, '订单详情未正确回显分单');
assert(
  detail.seaMasterBill?.masterNo === masterNo,
  '订单详情主单号与创建输入不一致',
);
assert(
  detail.seaDocumentSummary?.houseNo === houseNo,
  '订单详情分单号与创建输入不一致',
);

const updatedResponse = await request(`/api/v1/orders/${created.id}`, {
  method: 'PUT',
  body: JSON.stringify({
    id: created.id,
    expectedVersion: detail.version,
    notes: '海运出口闭环验收草稿已更新',
    seaMasterBill: { masterNo },
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
  updatedResponse.data?.seaDocumentSummary?.houseNo === houseNo,
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

const sharedMaster = (await request(`/api/v1/orders/${created.id}`)).data?.seaMasterBill;
assert(sharedMaster?.masterBillId && sharedMaster.transportExecutionId, '缺少当前共享主单与航程');
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
    shipmentType: 2,
    shippingLineId: shippingLine.id,
    personnelAssignments: [2, 3, 4].map((role) => ({ role, userId: me.data.id })),
    etd,
    eta,
    goodsDescription: '海运出口共享主单验收货物',
    totalPackages: 5,
    totalGrossWeightKg: 800,
    totalVolumeCbm: 6.2,
    customerReferenceNo: siblingReferenceNo,
    orderDate: new Date().toISOString(),
    seaMasterBill: {
      masterNo,
      candidateId: sharedMaster.masterBillId,
      expectedCandidateVersion: sharedMaster.version,
      candidateTeId: sharedMaster.transportExecutionId,
      expectedCandidateTeVersion: sharedMaster.transportExecutionVersion,
    },
    seaDocument: {
      documentStructure: 3,
      houseBill: { houseNo: siblingHouseNo, issuerSource: 1 },
    },
  }),
});
const sibling = siblingResponse.data;
assert(sibling?.id, '创建共享主单成员订单响应缺少 ID');

await request(`/api/v1/orders/${sibling.id}/cargo-items`, {
  method: 'POST',
  body: JSON.stringify({
    orderId: sibling.id,
    cargoName: '共享主单成员验收货物',
    packageCount: 5,
    grossWeightKg: 760,
    volumeCbm: 6,
  }),
});

const [currentShared, siblingDetail, sameBatch, siblingCargo] = await Promise.all([
  request(`/api/v1/orders/${created.id}`),
  request(`/api/v1/orders/${sibling.id}`),
  request(`/api/v1/orders/${created.id}/same-batch`),
  request(`/api/v1/orders/${sibling.id}/cargo-items`),
]);
assert(
  currentShared.data?.seaMasterBill?.masterBillId === siblingDetail.data?.seaMasterBill?.masterBillId &&
    currentShared.data?.seaMasterBill?.transportExecutionId === siblingDetail.data?.seaMasterBill?.transportExecutionId,
  '两票拼箱订单未共享同一主单与航程',
);
assert(currentShared.data?.seaMasterBill?.memberCount === 2, '共享主单成员票数应为 2');
assert(
  sameBatch.data?.some((item) => item.orderId === sibling.id && item.houseNo === siblingHouseNo),
  '同批订单未返回共享主单的成员订单及分单号',
);
const memberOrders = [currentShared.data, siblingDetail.data];
assert(
  memberOrders.reduce((sum, item) => sum + (item.totalPackages || 0), 0) === 15 &&
    memberOrders.reduce((sum, item) => sum + (item.totalGrossWeightKg || 0), 0) === 2300 &&
    memberOrders.reduce((sum, item) => sum + (item.totalVolumeCbm || 0), 0) === 18.2,
  '共享主单成员委托件重尺不正确',
);
const memberCargo = [...(cargoList.data || []), ...(siblingCargo.data || [])];
assert(
  memberCargo.reduce((sum, item) => sum + (item.packageCount || 0), 0) === 15 &&
    memberCargo.reduce((sum, item) => sum + (item.grossWeightKg || 0), 0) === 2010.5 &&
    memberCargo.reduce((sum, item) => sum + (item.volumeCbm || 0), 0) === 18.6,
  '共享主单成员实际件重尺不正确',
);

const transitionedResponse = await request(
  `/api/v1/orders/${created.id}/status`,
  {
    method: 'POST',
    body: JSON.stringify({
      id: created.id,
      expectedVersion: updatedResponse.data.version,
      targetFlowStatus: 2,
      reason: '海运出口闭环验收：完成订舱',
    }),
  },
);
assert(
  isEnumValue(
    transitionedResponse.data?.flowStatus,
    2,
    'ORDER_FLOW_STATUS_BOOKED',
  ),
  '订单未成功流转到 BOOKED',
);

const finalDetail = await request(`/api/v1/orders/${created.id}`);
assert(
  isEnumValue(
    finalDetail.data?.flowStatus,
    2,
    'ORDER_FLOW_STATUS_BOOKED',
  ),
  '状态流转后订单详情未回显 BOOKED',
);
assert(
  finalDetail.data?.seaDocumentSummary?.houseNo === houseNo,
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

const terminatingResponse = await request(
  `/api/v1/orders/${created.id}/termination`,
  {
    method: 'POST',
    body: JSON.stringify({
      id: created.id,
      expectedVersion: finalDetail.data.version,
      targetStatus: 2,
      terminationType: 3,
      reason: '海运出口闭环验收：发起退关',
    }),
  },
);
assert(
  isEnumValue(
    terminatingResponse.data?.terminationStatus,
    2,
    'ORDER_TERMINATION_STATUS_TERMINATING',
  ),
  '订单未成功进入退关中状态',
);

const terminatedResponse = await request(
  `/api/v1/orders/${created.id}/termination`,
  {
    method: 'POST',
    body: JSON.stringify({
      id: created.id,
      expectedVersion: terminatingResponse.data.version,
      targetStatus: 3,
      terminationType: 3,
      reason: '海运出口闭环验收：退关完成',
    }),
  },
);
assert(
  isEnumValue(
    terminatedResponse.data?.terminationStatus,
    3,
    'ORDER_TERMINATION_STATUS_TERMINATED',
  ),
  '订单未成功进入已退关状态',
);

const closedResponse = await request(`/api/v1/orders/${created.id}/closure`, {
  method: 'POST',
  body: JSON.stringify({
    id: created.id,
    expectedVersion: terminatedResponse.data.version,
    targetStatus: 2,
    reason: '海运出口闭环验收：费用已处理并结案',
  }),
});
assert(
  isEnumValue(
    closedResponse.data?.closureStatus,
    2,
    'ORDER_CLOSURE_STATUS_CLOSED',
  ),
  '终止订单未成功完结',
);

console.log('海运出口订单闭环验收通过。');
console.log(
  JSON.stringify(
    {
      ...prerequisiteSummary,
      order: {
        id: finalDetail.data.id,
        orderNo: finalDetail.data.orderNo,
        flowStatus: finalDetail.data.flowStatus,
        terminationStatus: closedResponse.data.terminationStatus,
        closureStatus: closedResponse.data.closureStatus,
        version: closedResponse.data.version,
        customerReferenceNo: finalDetail.data.customerReferenceNo,
        houseBillCount: finalDetail.data.seaDocumentSummary?.houseNo ? 1 : 0,
        cargoItemCount: cargoList.data?.length || 0,
        sharedMasterMemberCount: currentShared.data.seaMasterBill.memberCount,
      },
    },
    null,
    2,
  ),
);
