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
  const memberships = await withAcceptanceSystemWorkspace(request, () =>
    request(`/api/v1/admin/users/${me.data.id}/memberships`),
  );
  const membership = memberships.data?.find(
    (item) => item.organizationId === company.id,
  );
  if (!membership) {
    if (!allowCreate)
      throw new Error(
        '前置检查缺少验收公司成员资格；显式 --apply 才可创建夹具',
      );
    await withAcceptanceSystemWorkspace(request, async () => {
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

// 独立未核销样本验证建账占用与取消释放，不扰动主闭环。
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
    organizationId, groupingPolicy,
    idempotencyKey: key,
    groups: p.data.map((group) => ({
      groupKey: group.groupKey,
      groupToken: group.groupToken,
      feeIds: (group.fees || []).map((item) => item.id),
      billDate,
      settlementAccountId,
      statementTitle: '取消重建验收',
      paymentTermsDays: 30,
    })),
  });
  const batch = requireSuccessfulBillBatch(await post('/api/v1/finance/bill-batches', createBody(firstPreview, `aux-batch-${stamp}`)), firstPreview.data.length);
  const bill = batch.bills[0];
  const occupiedFee = (
    await request(`/api/v1/orders/${order.id}/fees`)
  ).data.find((item) => item.id === fee.id);
  assert(occupiedFee.hasActiveBill === true, '账单创建必须立即占用费用');
  const duplicate = await raw('/api/v1/finance/bill-batches', {
    method: 'POST',
    body: JSON.stringify(createBody(firstPreview, `aux-competing-${stamp}`)),
  });
  assert(duplicate.response.status === 200 && duplicate.body.results?.every((group) => group.success !== true && group.error), '已建账占用的费用必须逐组返回失败');
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
      update.body.reason === 'ORDER_FEE_BILL_OCCUPIED',
    '默认规则必须拒绝修改已建账费用',
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
  const rebuilt = requireSuccessfulBillBatch(await post('/api/v1/finance/bill-batches', createBody(await preview(), `aux-rebuilt-${stamp}`)), 1);
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

// 批量HTTP 200只表示逐组处理完成；验收必须检查每组成功与账单身份。
export function requireSuccessfulBillBatch(response, expectedGroups) {
  if (!response?.data?.id || response.success === false || !Array.isArray(response.results) || response.results.length !== expectedGroups || response.results.some((group) => group.success !== true || !group.bill?.id || group.error)) {
    throw new Error(`批量建账存在失败组：${JSON.stringify(response?.results?.map((group) => ({ success: group.success, reason: group.error?.reason, message: group.error?.message })))}`);
  }
  if (response.data.bills?.some((bill) => bill.status !== 2)) throw new Error('创建账单必须直接为CONFIRMED');
  return response.data;
}

// 正式发票登记一次提交明确账单与逐费用金额/税额，不再生成开票草稿后单独issue。
export function invoiceRegistrationForBills(bills, { invoiceProfileId, taxInvoiceNo, invoiceDate, invoiceType = 'DIGITAL_NORMAL', note }) {
  const toUnits = (text) => {
    const [whole, fraction = ''] = String(text).split('.');
    return BigInt(whole) * 100000000n + BigInt(fraction.padEnd(8, '0'));
  };
  const total = bills.reduce((sum, bill) => sum + toUnits(bill.totalAmount), 0n);
  const totalAmount = `${total / 100000000n}.${String(total % 100000000n).padStart(8, '0')}`;
  return { invoiceProfileId, taxInvoiceNo, invoiceDate, invoiceType, totalAmount, note,
    allocations: bills.map((bill) => ({ billId: bill.id, expectedVersion: bill.version, amount: bill.totalAmount,
      fees: (bill.lines || []).filter((line) => line.active === true).map((line) => ({ orderFeeId: line.orderFeeId, amount: line.totalAmount, taxAmount: line.taxAmount || '0' })),
    })),
  };
}

// 同一隔离运行中串起新费用、两张已有发票、两笔差额核销与三个报表API。
export async function verifyCompositeFinanceWorkflow(request, { organizationId, orderId, orderNo, settlementPartyId, feeSettingId, billingUnitId, currency, settlementAccountId, invoiceProfileId, billDate, stamp }) {
  const assert = (condition, message) => { if (!condition) throw new Error(message); };
  const post = (path, payload) => request(path, { method: 'POST', body: JSON.stringify(payload) });
  const fee = (await post(`/api/v1/orders/${orderId}/fees`, { orderId, direction: 1, settlementPartyId, quantity: '1', unitPrice: '100', currency, feeSettingId, billingUnitId, taxInclusive: true, idempotencyKey: `combo-fee-${stamp}` })).data;
  const common = { organizationId, feeIds: [fee.id], groupingPolicy: { mode: 1 } };
  const initial = await post('/api/v1/finance/bill-batches/preview', common);
  const preview = await post('/api/v1/finance/bill-batches/preview', { ...common, groupConfigs: initial.data.map((group) => ({ groupKey: group.groupKey, billDate, settlementAccountId })) });
  const group = preview.data[0];
  const invoice = (amount, index) => ({ invoiceProfileId, invoiceType: 'DIGITAL_NORMAL', taxInvoiceNo: `COMBO-TAX-${stamp}-${index}`, invoiceDate: billDate, totalAmount: String(amount), fees: [{ orderFeeId: fee.id, amount: String(amount), taxAmount: index === 1 ? '1.23' : '4.56' }] });
  const verification = (amount) => ({ verificationDate: billDate, allocations: [{ cashflowId: '', amount: String(amount) }], shortfallCashflow: { amount: String(amount), transactionDate: billDate, ourAccount: `组合验收账户-${stamp}`, paymentMethod: 'BANK_TRANSFER' } });
  const baseGroup = { groupKey: group.groupKey, groupToken: group.groupToken, feeIds: [fee.id], billDate, settlementAccountId, statementTitle: '组合登记核销验收', invoices: [invoice(40, 1), invoice(60, 2)] };
  const failed = await post('/api/v1/finance/bill-batches', { organizationId, groupingPolicy: common.groupingPolicy, idempotencyKey: `combo-fail-${stamp}`, groups: [{ ...baseGroup, verifications: [verification(40), verification(61)] }] });
  assert(failed.results?.[0].success !== true && failed.results?.[0].error?.items?.[0].step === 'VERIFICATION', '组合超额应定位核销失败');
  const afterFailure = (await request(`/api/v1/orders/${orderId}/fees`)).data.find((item) => item.id === fee.id);
  assert(afterFailure.hasActiveBill !== true, '组合失败不能遗留账单费用占用');
  const command = { organizationId, groupingPolicy: common.groupingPolicy, idempotencyKey: `combo-success-${stamp}`, groups: [{ ...baseGroup, verifications: [verification(40), verification(60)] }] };
  const saved = await post('/api/v1/finance/bill-batches', command);
  const batch = requireSuccessfulBillBatch(saved, 1);
  const bill = (await request(`/api/v1/finance/bills/${batch.bills[0].id}`)).data;
  const activeLines = (bill.lines ?? []).filter((line) => line.active === true);
  assert(bill.completed === true && bill.verifiedAmount === '100.00000000' && bill.invoiceProgress === 4 && bill.settlementProgress === 3 && activeLines.length === 1 && activeLines[0].invoicedAmount === '100.00000000' && activeLines[0].invoiceableAmount === '0.00000000', `组合成功的账单与逐费开票/核销投影应完整结清：${JSON.stringify({ completed: bill.completed, verifiedAmount: bill.verifiedAmount, invoiceProgress: bill.invoiceProgress, settlementProgress: bill.settlementProgress, lines: activeLines.map((line) => ({ invoicedAmount: line.invoicedAmount, invoiceableAmount: line.invoiceableAmount })) })}`);
  const replay = requireSuccessfulBillBatch(await post('/api/v1/finance/bill-batches', command), 1);
  assert(replay.bills[0].id === bill.id, '组合重试不得重复生成账单');
  const registrations = await request(`/api/v1/finance/invoices?page=1&pageSize=200&billId=${bill.id}&organizationId=${organizationId}`);
  assert(registrations.data?.length === 2, '组合重试后仍须恰有两张发票');
  assert(registrations.data.some((item) => item.taxAmount === '1.23000000' && item.netAmount === '38.77000000') && registrations.data.some((item) => item.taxAmount === '4.56000000' && item.netAmount === '55.44000000'), '多票登记必须保留财务明确填写的税额及相减未税额');
  const persistedFee = (await request(`/api/v1/orders/${orderId}/fees`)).data.find((item) => item.id === fee.id);
  assert(persistedFee.taxAmount === fee.taxAmount && persistedFee.netAmount === fee.netAmount && persistedFee.taxRate === fee.taxRate, '发票手填税额不得反改来源费用税基');
  const query = `organizationId=${organizationId}&page=1&pageSize=200`;
  const profits = await request(`/api/v1/finance/reports/profits?${query}&keyword=${encodeURIComponent(orderNo)}`);
  const profit = profits.data?.find((row) => row.orderId === orderId);
  assert(profit && profit.orderNo === orderNo && /^-?\d+\.\d{8}$/.test(profit.profitAmount) && /^-?\d+\.\d{8}$/.test(profit.netProfitAmount), '利润报表应包含组合来源订单及含税/未税金额');
  const cashflows = await request(`/api/v1/finance/reports/cashflows?${query}&ourAccount=${encodeURIComponent(`组合验收账户-${stamp}`)}`);
  assert(cashflows.data?.length === 2, '实际收付款报表应恰有两笔组合差额流水');
  assert(cashflows.data.every((row) => row.transactionDate === billDate && row.direction === 'RECEIVABLE' && row.currency === currency), '实际收付款报表的方向、币种及交易日期须保留来源快照');
  assert(cashflows.nativeTotals?.find((row) => row.currency === currency)?.receivableAmount === '100.00000000', '实际收付款报表原币收款合计须为100');
  const cashflowFacts = await Promise.all(cashflows.data.map(async (row) => {
    const source = await request(`/api/v1/finance/cashflows?${query}&keyword=${encodeURIComponent(row.flowNo)}`);
    const matches = source.data?.filter((item) => item.id === row.id) ?? [];
    assert(matches.length === 1, '按流水号读取的正式资金列表须包含唯一来源事实');
    const fact = matches[0];
    assert(fact.status === 2, '实际收付报表只允许采用已确认资金流水');
    assert(fact.amount === row.amount && fact.baseAmount === row.baseAmount && fact.transactionDate === row.transactionDate, '实际收付报表应直接采用流水经济快照');
    return fact;
  }));
  // 账单汇率与实际收付汇率相互独立，完整合计只能与资金事实核对。
  const baseSum = cashflowFacts.reduce((sum, row) => sum + BigInt(row.baseAmount.replace('.', '')), 0n);
  const totalBase = cashflows.baseTotals?.find((row) => row.currency === bill.baseCurrency)?.receivableAmount;
  assert(totalBase && BigInt(totalBase.replace('.', '')) === baseSum, '实际收付款报表本币合计须等于真实流水快照合计');
  for (const direction of ['RECEIVABLE', 'PAYABLE']) {
    const aging = await request(`/api/v1/finance/reports/aging?${query}&direction=${direction}&keyword=${encodeURIComponent(bill.billNo)}`);
    assert(Number(aging.total ?? 0) === 0, `${direction}账龄表不得计入已结清组合账单`);
  }
  console.log('建账＋多票登记＋差额核销原子回滚、同键重放与利润/收付/账龄报表链路通过。');
  return { billNo: bill.billNo, invoiceCount: registrations.data.length, cashflowCount: cashflows.data.length, completed: bill.completed };
}
