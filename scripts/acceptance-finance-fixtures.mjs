// 验收账号按正式成员资格进入公司工作台；不以 system 组织开展经营。
export async function prepareAcceptanceCompany(
  request,
  { allowCreate = false, baseCurrency = 'CNY' } = {},
) {
  const me = await request('/api/v1/auth/me');
  const companies = await request('/api/v1/admin/organizations');
  let company = companies.data
    ?.filter(
      (item) =>
        item.enabled &&
        (item.kind === 2 || item.kind === 'ORGANIZATION_KIND_COMPANY') &&
        item.baseCurrency === baseCurrency,
    )
    .sort((a, b) => a.code.localeCompare(b.code))[0];
  if (!company && allowCreate) {
    company = (
      await withAcceptanceSystemWorkspace(request, () =>
        request('/api/v1/admin/organizations', {
          method: 'POST',
          body: JSON.stringify({
            code: `ACC_${baseCurrency}_${Date.now()}`,
            name: `${baseCurrency}隔离验收公司`,
            kind: 2,
            baseCurrency,
          }),
        }),
      )
    ).data;
  }
  if (!company) throw new Error('验收环境缺少符合本币的启用公司');
  const memberships = await request(
    `/api/v1/admin/users/${me.data.id}/memberships`,
  );
  const membership = memberships.data?.find(
    (item) => item.organizationId === company.id,
  );
  if (!membership) {
    if (!allowCreate)
      throw new Error(
        '前置检查缺少验收公司成员资格；显式 --apply 才可创建夹具',
      );
    const roles = await request(
      `/api/v1/admin/organizations/${company.id}/roles`,
    );
    const administrator = roles.data?.find(
      (item) => item.code === 'administrator' && item.enabled,
    );
    if (!administrator) throw new Error('验收公司缺少启用管理员角色');
    await request(`/api/v1/admin/users/${me.data.id}/memberships`, {
      method: 'POST',
      body: JSON.stringify({
        userId: me.data.id,
        organizationId: company.id,
        roleIds: [administrator.id],
        primary: true,
      }),
    });
  } else if (!membership.enabled) {
    throw new Error('验收公司成员资格已停用');
  }
  if (me.data.currentOrganization.id !== company.id) {
    await request('/api/v1/auth/switch-organization', {
      method: 'POST',
      body: JSON.stringify({ organizationId: company.id }),
    });
  }
  const current = await request('/api/v1/auth/me');
  if (current.data.currentOrganization.id !== company.id)
    throw new Error('验收会话未进入目标公司');
  return current.data;
}

// 独立未核销样本验证草稿占用与取消释放，不扰动主闭环。
export async function verifyDraftFeeBillLifecycle(
  request,
  raw,
  {
    organizationId,
    orderInput,
    feeInput,
    settlementAccountId,
    billDate,
    stamp,
  },
) {
  const assert = (condition, message) => {
    if (!condition) throw new Error(message);
  };
  const post = (path, body) =>
    request(path, { method: 'POST', body: JSON.stringify(body) });
  const order = (
    await post('/api/v1/orders', {
      ...orderInput,
      seaMasterBill: { masterNo: `AUXMB${stamp}` },
      customerReferenceNo: `ACC-AUX-${stamp}`,
    })
  ).data;
  const fee = (
    await post(`/api/v1/orders/${order.id}/fees`, {
      ...feeInput,
      orderId: order.id,
      idempotencyKey: `aux-fee-${stamp}`,
    })
  ).data;
  const groupingPolicy = { mode: 1, splitByOrder: true, splitByTaxRate: true };
  const common = { organizationId, feeIds: [fee.id], groupingPolicy };
  async function preview() {
    const initial = await post('/api/v1/finance/bill-batches/preview', common);
    return post('/api/v1/finance/bill-batches/preview', {
      ...common,
      groupConfigs: initial.data.map((group) => ({
        groupKey: group.groupKey,
        billDate,
        settlementAccountId,
      })),
    });
  }
  const firstPreview = await preview();
  const createBody = (p, key) => ({
    ...common,
    previewToken: p.previewToken,
    idempotencyKey: key,
    groups: p.data.map((group) => ({
      groupKey: group.groupKey,
      billDate,
      settlementAccountId,
      statementTitle: '取消重建验收',
      paymentTermsDays: 30,
    })),
  });
  const batch = (
    await post(
      '/api/v1/finance/bill-batches',
      createBody(firstPreview, `aux-batch-${stamp}`),
    )
  ).data;
  const bill = batch.bills[0];
  const occupiedFee = (
    await request(`/api/v1/orders/${order.id}/fees`)
  ).data.find((item) => item.id === fee.id);
  assert(occupiedFee.hasActiveBill === true, '草稿账单必须立即占用费用');
  const duplicate = await raw('/api/v1/finance/bill-batches', {
    method: 'POST',
    body: JSON.stringify(createBody(firstPreview, `aux-competing-${stamp}`)),
  });
  assert(duplicate.response.status === 409, '草稿占用的费用必须拒绝重复建账');
  const update = await raw(`/api/v1/orders/${order.id}/fees/${fee.id}`, {
    method: 'PUT',
    body: JSON.stringify({
      ...feeInput,
      orderId: order.id,
      id: fee.id,
      expectedVersion: occupiedFee.version,
      unitPrice: '999',
    }),
  });
  assert(
    update.response.status === 409 &&
      update.body.reason === 'BILLED_FEE_EDIT_DISABLED',
    '默认规则必须拒绝修改草稿已建账费用',
  );
  const removal = await raw(
    `/api/v1/orders/${order.id}/fees/${fee.id}?expectedVersion=${occupiedFee.version}`,
    { method: 'DELETE' },
  );
  assert(
    removal.response.status === 409 &&
      removal.body.reason === 'ORDER_FEE_BILL_OCCUPIED',
    '已建账费用普通删除必须返回占用冲突',
  );
  await post(`/api/v1/finance/bills/${bill.id}/cancel`, {
    id: bill.id,
    expectedVersion: bill.version,
    reason: '独立样本取消释放验证',
  });
  const releasedFee = (
    await request(`/api/v1/orders/${order.id}/fees`)
  ).data.find((item) => item.id === fee.id);
  assert(releasedFee.hasActiveBill !== true, '取消账单必须释放费用占用');
  assert(
    releasedFee.totalAmount === fee.totalAmount,
    '拒绝修改后费用金额必须保持原值',
  );
  const rebuilt = (
    await post(
      '/api/v1/finance/bill-batches',
      createBody(await preview(), `aux-rebuilt-${stamp}`),
    )
  ).data;
  assert(
    rebuilt.bills[0].id !== bill.id &&
      rebuilt.bills[0].lines[0].orderFeeId === fee.id,
    '释放后的同一费用必须可进入新账单',
  );
  return {
    orderId: order.id,
    feeId: fee.id,
    cancelledBillId: bill.id,
    rebuiltBillId: rebuilt.bills[0].id,
    amount: fee.totalAmount,
    draftOccupied: true,
    duplicateRejected: true,
    updateRejected: true,
    deleteRejected: true,
    cancellationReleased: true,
    rebuildSucceeded: true,
  };
}

export async function acceptanceChargeCategoryId(request) {
  const options = await request('/api/v1/master-data/options');
  const category = options.data?.find(
    (item) =>
      (item.kind === 8 || item.kind === 'MASTER_DATA_KIND_CHARGE_CATEGORY') &&
      item.enabled !== false,
  );
  if (!category?.id) throw new Error('验收环境缺少启用费用大类');
  return category.id;
}

// 全局船公司与计费单位按正式系统工作台权限写入，完成后恢复经营公司。
export async function withAcceptanceSystemWorkspace(request, action) {
  const me = (await request('/api/v1/auth/me')).data;
  const originalId = me.currentOrganization.id;
  const system = me.organizations.find(
    (item) => item.kind === 1 || item.kind === 'ORGANIZATION_KIND_SYSTEM',
  );
  if (!system) throw new Error('验收账号缺少合法系统工作台成员资格');
  if (system.id !== originalId)
    await request('/api/v1/auth/switch-organization', {
      method: 'POST',
      body: JSON.stringify({ organizationId: system.id }),
    });
  try {
    return await action();
  } finally {
    if (system.id !== originalId)
      await request('/api/v1/auth/switch-organization', {
        method: 'POST',
        body: JSON.stringify({ organizationId: originalId }),
      });
  }
}
