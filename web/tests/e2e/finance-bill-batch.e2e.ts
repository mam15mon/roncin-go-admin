import {
  expect as baseExpect,
  type Locator,
  type Page,
  test,
} from '@playwright/test';

const expect = baseExpect.configure({ timeout: 20_000 });
function requiredEnvironment(name: string) {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`缺少环境变量 ${name}`);
  return value;
}
async function selectOption(
  page: Page,
  input: Locator,
  text: string,
  { search = false } = {},
) {
  await input.click();
  if (search) await input.fill(text);
  await page
    .locator('.ant-select-dropdown:visible .ant-select-item-option')
    .filter({ hasText: text })
    .first()
    .click();
}

// 基础资料经 API 准备；费用、账单、收款、核销与提成均通过真实页面写入。
test('同一订单完成录费、建账、收款核销与提成生成并刷新重读', async ({
  page,
}, testInfo) => {
  test.setTimeout(240_000);
  page.setDefaultTimeout(15_000);
  const errors: string[] = [];
  const abortedRequests: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  page.on('requestfailed', (request) => {
    if (!request.url().includes('/api/')) return;
    const detail = `${request.method()} ${new URL(request.url()).pathname}: ${request.failure()?.errorText}`;
    if (
      request.failure()?.errorText === 'net::ERR_ABORTED' &&
      request.method() === 'GET'
    )
      abortedRequests.push(detail);
    else errors.push(detail);
  });
  page.on('response', (response) => {
    if (response.url().includes('/api/') && response.status() >= 400)
      errors.push(
        `${response.request().method()} ${new URL(response.url()).pathname}: HTTP ${response.status()}`,
      );
  });
  await page.goto('/user/login');
  await page
    .getByPlaceholder('用户名 / 邮箱')
    .fill(requiredEnvironment('BOOTSTRAP_ADMIN_USERNAME'));
  await page
    .getByPlaceholder('请输入密码')
    .fill(requiredEnvironment('BOOTSTRAP_ADMIN_PASSWORD'));
  await page.locator('button[type="submit"]').click();
  await expect(
    page.getByRole('heading', { name: '选择进入的组织' }),
  ).toBeVisible();
  const loginMeResponse = await page.request.get('/api/v1/auth/me');
  expect(loginMeResponse.ok()).toBeTruthy();
  const loginMe = (await loginMeResponse.json()).data;
  const acceptanceCompany = loginMe.organizations.find(
    (organization: { kind: number; baseCurrency: string }) =>
      organization.kind === 2 && organization.baseCurrency === 'CNY',
  );
  expect(acceptanceCompany?.name).toBeTruthy();
  await page
    .getByRole('button', { name: `进入${acceptanceCompany.name}` })
    .click();
  await expect(page).not.toHaveURL(/\/user\/login/);
  const api = async (path: string, data?: unknown) => {
    const response = data
      ? await page.request.post(path, { data })
      : await page.request.get(path);
    const body = await response.json();
    expect(response.ok(), `${path}: ${body.message}`).toBeTruthy();
    return body;
  };
  const stamp = Date.now().toString();
  const today = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(new Date());
  const me = (await api('/api/v1/auth/me')).data;
  expect(me.currentOrganization.kind).toBe(2);
  const organizationId = me.currentOrganization.id;
  const companyName = me.currentOrganization.name;
  const employee = (
    await api('/api/v1/admin/users', {
      username: `acc_ui_${stamp}`,
      displayName: `验收销售${stamp}`,
      password: `Ui${stamp}!aZ9`,
    })
  ).data;
  const employeeMemberships = await api(
    `/api/v1/admin/users/${employee.id}/memberships`,
  );
  expect(
    employeeMemberships.data.some(
      (membership: { organizationId: string; enabled: boolean }) =>
        membership.organizationId === organizationId && membership.enabled,
    ),
  ).toBe(true);
  const customer = (
    await api('/api/v1/partners', {
      code: `ACC-UI-${stamp}`,
      legalName: `页面验收客户${stamp}`,
      unifiedSocialCreditCode: `91310000ACC${stamp.slice(-7)}`,
      registeredAddress: '隔离验收地址',
      roles: [{ type: 1, enabled: true }],
      contacts: [
        { name: '页面验收联系人', phone: '13800000000', isPrimary: true },
      ],
      profile: {
        nameEn: `UI Acceptance ${stamp}`,
        countryCode: 'CN',
        businessTypes: [1],
      },
    })
  ).data;
  await api(`/api/v1/partners/${customer.id}/accounts`, {
    partnerId: customer.id,
    account: {
      name: '页面验收结算账户',
      accountHolder: customer.legalName,
      currency: 'CNY',
      bankName: '验收银行',
      accountNo: `UI${stamp}`,
      usage: 1,
      isDefaultReceivable: true,
      enabled: true,
    },
  });
  const carrier = (
    await api('/api/v1/master-data/shipping-lines?page=1&pageSize=10')
  ).data[0];
  const chargeCategories = (await api('/api/v1/master-data/options')).data;
  const chargeCategory = chargeCategories.find(
    (item: { kind: number; enabled: boolean }) =>
      item.kind === 8 && item.enabled,
  );
  expect(chargeCategory?.id).toBeTruthy();
  const order = (
    await api('/api/v1/orders', {
      serviceTypeIds: [chargeCategory.id],
      customerId: customer.id,
      shippingLineId: carrier.id,
      seaMasterBill: { masterNo: `UIMB${stamp}` },
      seaDocument: { documentStructure: 2 },
      businessType: 1,
      tradeDirection: 1,
      tradeTerm: 3,
      paymentTerm: 1,
      shipmentType: 2,
      shipmentMode: 1,
      goodsDescription: '同订单页面闭环验收',
      totalPackages: 1,
      totalGrossWeightKg: 100,
      totalVolumeCbm: 1,
      customerReferenceNo: `ACC-UI-${stamp}`,
      orderDate: new Date().toISOString(),
      personnelAssignments: [
        { userId: employee.id, role: 3 },
        { userId: me.id, role: 2 },
        { userId: me.id, role: 4 },
      ],
    })
  ).data;
  await api('/api/v1/finance/commission-rules', {
    organizationId,
    rule: {
      name: `UI提成方案${stamp}`,
      personnelRole: 'SALES',
      calculationBasis: 'REALIZED_PROFIT',
      ratePercent: '10',
      effectiveFrom: today,
      effectiveTo: today,
      enabled: true,
      employeeIds: [employee.id],
    },
  });
  const options = await api(`/api/v1/orders/${order.id}/fee-options`);
  const feeSetting = options.feeSettings.find(
    (item: { defaultBillingUnitId?: string; taxRate?: string }) =>
      item.defaultBillingUnitId &&
      item.taxRate !== undefined &&
      Number(item.taxRate) === 0,
  );
  expect(feeSetting).toBeTruthy();
  await page.goto(`/orders/sea-export/${order.id}/fees`);
  await page.getByRole('button', { name: '新增应收费用' }).click();
  const editRow = page.locator('.ant-table-row[data-row-key^="new_"]').first();
  await selectOption(
    page,
    editRow.locator('td').nth(2).locator('.ant-select'),
    feeSetting.nameZh,
  );
  await editRow.getByPlaceholder('0.00').fill('125');
  const feeResponse = page.waitForResponse(
    (r) =>
      r.url().endsWith(`/orders/${order.id}/fees`) &&
      r.request().method() === 'POST',
  );
  await editRow.getByText('保存', { exact: true }).click();
  const fee = (await (await feeResponse).json()).data;
  expect(fee.totalAmount).toBe('125.00000000');
  expect(fee.hasActiveBill).not.toBe(true);
  await expect(page.getByText('125.00', { exact: true }).first()).toBeVisible();
  await page.screenshot({
    path: testInfo.outputPath('01-fee.png'),
    fullPage: true,
  });
  await page.locator('.ant-table-row input[type="checkbox"]').first().check();
  await page
    .getByRole('button', { name: /生成账单（1）/ })
    .first()
    .click();
  const billDrawer = page.getByRole('dialog', { name: '费用批量转账单' });
  await expect(
    billDrawer.getByText('页面验收结算账户', { exact: false }),
  ).toBeVisible();
  const billPreviewResponse = page.waitForResponse(
    (r) =>
      r.url().endsWith('/finance/bill-batches/preview') &&
      r.request().method() === 'POST',
  );
  await billDrawer.getByRole('button', { name: '刷新快照' }).click();
  const billPreview = await (await billPreviewResponse).json();
  expect(billPreview.previewToken).toBeTruthy();
  expect(billPreview.data).toHaveLength(1);
  expect(billPreview.data[0].configurationComplete).toBe(true);
  const billResponse = page.waitForResponse(
    (r) =>
      r.url().endsWith('/finance/bill-batches') &&
      r.request().method() === 'POST',
  );
  await billDrawer.getByRole('button', { name: /原子生成 1 张账单/ }).click();
  const batch = (await (await billResponse).json()).data;
  const bill = batch.bills[0];
  expect(bill.totalAmount).toBe('125.00000000');
  expect(bill.lines).toHaveLength(1);
  expect(bill.lines[0].orderId).toBe(order.id);
  expect(bill.lines[0].orderFeeId).toBe(fee.id);
  const feesInDraft = (await api(`/api/v1/orders/${order.id}/fees`)).data;
  expect(
    feesInDraft.find((item: { id: string }) => item.id === fee.id)
      .hasActiveBill,
  ).toBe(true);
  await billDrawer.getByRole('button', { name: '确认本批全部账单' }).click();
  await expect(billDrawer.getByText(/当前已全部确认/)).toBeVisible();
  await page.screenshot({
    path: testInfo.outputPath('02-bill.png'),
    fullPage: true,
  });
  await billDrawer.locator('.ant-drawer-close').click();
  await page.reload();
  const billedRow = page.locator(`.ant-table-row[data-row-key="${fee.id}"]`);
  await expect(
    billedRow.getByText('125.00', { exact: true }).first(),
  ).toBeVisible();
  const persistedFeeAfterReload = (
    await api(`/api/v1/orders/${order.id}/fees`)
  ).data.find((item: { id: string }) => item.id === fee.id);
  expect(persistedFeeAfterReload.hasActiveBill).toBe(true);
  await expect(billedRow.getByText('删除', { exact: true })).toHaveCount(0);
  await billedRow.locator('input[type="checkbox"]').check();
  await page
    .getByRole('button', { name: /生成账单（1）/ })
    .first()
    .click();
  await expect(page.getByText(/已建账，不能.*账单/)).toBeVisible();
  await expect(
    page.getByRole('dialog', { name: '费用批量转账单' }),
  ).toHaveCount(0);
  await page.screenshot({
    path: testInfo.outputPath('02-billed-guard.png'),
    fullPage: true,
  });
  await page.goto('/finance/cashflows');
  await page.getByRole('button', { name: '登记流水' }).click();
  const cashDialog = page.getByRole('dialog', { name: '登记资金流水' });
  await selectOption(page, cashDialog.getByLabel('所属公司'), companyName);
  await selectOption(
    page,
    cashDialog.getByLabel('往来结算单位'),
    customer.legalName,
    { search: true },
  );
  await cashDialog.getByLabel('发生金额').fill('125');
  await cashDialog.getByLabel('我方账户').fill('页面验收收款账户');
  const cashResponse = page.waitForResponse(
    (r) =>
      r.url().endsWith('/finance/cashflows') && r.request().method() === 'POST',
  );
  await cashDialog.getByRole('button', { name: /确\s*定/ }).click();
  const cashflow = (await (await cashResponse).json()).data;
  await expect(cashDialog).toBeHidden();
  const cashRow = page
    .locator('.ant-table-tbody > tr.ant-table-row')
    .filter({ hasText: cashflow.flowNo });
  await cashRow.getByText('确认', { exact: true }).click();
  await page
    .getByRole('tooltip')
    .getByRole('button', { name: /确\s*定/ })
    .click();
  await expect(cashRow.getByText('已确认', { exact: true })).toBeVisible();
  await page.goto('/finance/verifications');
  await page.getByRole('button', { name: '新建核销' }).click();
  const verificationDialog = page.getByRole('dialog', {
    name: '资金与账单核销工作台',
  });
  await selectOption(
    page,
    verificationDialog.getByLabel('所属公司'),
    companyName,
  );
  await selectOption(
    page,
    verificationDialog.getByLabel('结算单位'),
    customer.legalName,
    { search: true },
  );
  await verificationDialog
    .locator('.ant-card')
    .filter({ hasText: '待核销资金' })
    .first()
    .locator('.ant-table-row')
    .filter({ hasText: cashflow.flowNo })
    .locator('input[type="checkbox"]')
    .check();
  await verificationDialog
    .locator('.ant-card')
    .filter({ hasText: '待核销账单' })
    .first()
    .locator('.ant-table-row')
    .filter({ hasText: bill.billNo })
    .locator('input[type="checkbox"]')
    .check();
  await verificationDialog
    .getByRole('button', { name: '按余额自动分配' })
    .click();
  await expect(verificationDialog.getByLabel('第 1 行核销金额')).toHaveValue(
    '125.00000000',
  );
  const verificationResponse = page.waitForResponse(
    (r) =>
      r.url().endsWith('/finance/verifications') &&
      r.request().method() === 'POST',
  );
  await verificationDialog.getByRole('button', { name: /提交核销/ }).click();
  const verification = (await (await verificationResponse).json()).data;
  expect(verification.amount).toBe('125.00000000');
  expect(verification.allocations).toHaveLength(1);
  expect(verification.allocations[0].billId).toBe(bill.id);
  expect(verification.allocations[0].cashflowId).toBe(cashflow.id);
  await expect(verificationDialog).toBeHidden();
  await page.screenshot({
    path: testInfo.outputPath('03-verification.png'),
    fullPage: true,
  });
  await page.goto('/finance/commissions');
  await page.getByRole('button', { name: '生成提成' }).click();
  const commissionDialog = page.getByRole('dialog', { name: '生成提成' });
  await selectOption(
    page,
    commissionDialog.locator('.ant-select').first(),
    companyName,
  );
  await selectOption(
    page,
    commissionDialog.getByLabel('有效应收核销'),
    verification.verificationNo,
    { search: true },
  );
  await selectOption(
    page,
    commissionDialog.getByLabel('计提候选（员工 / 身份 / 已解析方案）'),
    employee.displayName,
    { search: true },
  );
  await commissionDialog
    .getByRole('button', { name: '计算并核对预览' })
    .click();
  await expect(
    commissionDialog.getByText('12.50 CNY', { exact: true }).first(),
  ).toBeVisible();
  await expect(
    commissionDialog.getByText(order.orderNo, { exact: true }),
  ).toBeVisible();
  const commissionResponse = page.waitForResponse(
    (r) =>
      r.url().endsWith('/finance/commissions') &&
      r.request().method() === 'POST',
  );
  await commissionDialog.getByRole('button', { name: '生成草稿' }).click();
  const commission = (await (await commissionResponse).json()).data;
  expect(commission.commissionAmount).toBe('12.50000000');
  expect(commission.verificationId).toBe(verification.id);
  expect(commission.lines).toHaveLength(1);
  expect(commission.lines[0].orderId).toBe(order.id);
  expect(commission.lines[0].fees).toHaveLength(1);
  expect(commission.lines[0].fees[0].feeId).toBe(fee.id);
  expect(commission.realizedRevenue).toBe('125.00000000');
  expect(commission.allocatedCost).toBe('0.00000000');
  expect(commission.realizedProfit).toBe('125.00000000');
  await expect(commissionDialog).toBeHidden();
  await page.reload();
  const commissionRow = page
    .locator('.ant-table-tbody > tr.ant-table-row')
    .filter({ hasText: commission.commissionNo });
  await commissionRow.getByText('明细', { exact: true }).click();
  const detail = page.getByRole('dialog', {
    name: new RegExp(`提成明细.*${commission.commissionNo}`),
  });
  await expect(detail.getByText(order.orderNo, { exact: true })).toBeVisible();
  await expect(
    detail.getByText('12.50 CNY', { exact: true }).first(),
  ).toBeVisible();
  await page.screenshot({
    path: testInfo.outputPath('04-commission.png'),
    fullPage: true,
  });
  const persisted = (await api(`/api/v1/finance/commissions/${commission.id}`))
    .data;
  expect(persisted.lines).toHaveLength(1);
  expect(persisted.lines[0].orderId).toBe(order.id);
  expect(persisted.lines[0].fees[0].feeId).toBe(fee.id);
  expect(persisted.verificationId).toBe(verification.id);
  await expect(
    detail.getByText(verification.verificationNo, { exact: true }),
  ).toBeVisible();
  expect(await detail.innerText()).not.toMatch(/\d{4}-\d{2}-\d{2}T\d{2}:\d{2}/);
  const persistedBill = (await api(`/api/v1/finance/bills/${bill.id}`)).data;
  const persistedCashflow = (
    await api(
      `/api/v1/finance/cashflows?page=1&pageSize=200&organizationId=${organizationId}`,
    )
  ).data.find((item: { id: string }) => item.id === cashflow.id);
  expect(persistedCashflow).toBeTruthy();
  expect(persistedBill.verifiedAmount).toBe('125.00000000');
  expect(persistedCashflow.verifiedAmount).toBe('125.00000000');
  expect(persistedCashflow.unverifiedAmount).toBe('0.00000000');
  expect(persisted.commissionAmount).toBe('12.50000000');
  expect(errors).toEqual([]);
  const evidence = {
    orderId: order.id,
    orderNo: order.orderNo,
    feeId: fee.id,
    billId: bill.id,
    billNo: bill.billNo,
    cashflowId: cashflow.id,
    cashflowNo: cashflow.flowNo,
    verificationId: verification.id,
    verificationNo: verification.verificationNo,
    commissionId: commission.id,
    commissionNo: commission.commissionNo,
    expectedFeeAndBillAndVerification: '125.00000000',
    actualFee: fee.totalAmount,
    actualBill: bill.totalAmount,
    actualVerification: verification.amount,
    expectedCommission: '12.50000000',
    actualCommission: persisted.commissionAmount,
  };
  console.info(`[finance-ui-evidence] ${JSON.stringify(evidence)}`);
  await testInfo.attach('same-order-evidence', {
    body: JSON.stringify({
      ...evidence,
      abortedRequests,
    }),
    contentType: 'application/json',
  });
});
