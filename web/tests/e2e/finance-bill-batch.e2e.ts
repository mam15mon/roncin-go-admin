import { writeFile } from 'node:fs/promises';
import {
  expect as baseExpect,
  type Locator,
  type Page,
  test,
} from '@playwright/test';
import { FinanceBillStatus } from '../../src/enums.generated';

const expect = baseExpect.configure({ timeout: 20_000 });
const orderKind = process.env.RONCIN_ACCEPTANCE_ORDER_KIND || 'sea-export';
if (orderKind !== 'sea-export' && orderKind !== 'air-export') {
  throw new Error(
    'RONCIN_ACCEPTANCE_ORDER_KIND 只允许 sea-export 或 air-export',
  );
}
const isAirExport = orderKind === 'air-export';
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

async function inspectFinanceReports(
  page: Page,
  attach: (name: string, body: Buffer) => Promise<void>,
) {
  const reports = [
    ['profit-analysis', '完整筛选合计'],
    ['profit', '完整筛选合计'],
    ['cashflows', '完整筛选原币合计'],
    ['aging', '完整筛选原币账龄合计'],
  ] as const;
  for (const [route, summary] of reports) {
    await page.setViewportSize({ width: 1440, height: 960 });
    await page.goto(`/finance/reports/${route}`);
    await expect(page.getByText(summary, { exact: true })).toBeVisible();
    await expect(page.getByText(/^生成于 /)).toBeVisible();
    await expect(
      page.getByRole('button', { name: '导出全部筛选结果' }),
    ).toBeEnabled();
    if (route === 'profit') {
      const downloading = page.waitForEvent('download');
      await page.getByRole('button', { name: '导出全部筛选结果' }).click();
      const download = await downloading;
      expect(download.suggestedFilename()).toMatch(/\.csv$/);
      expect(await download.failure()).toBeNull();
    }
    await attach(
      `report-${route}-desktop`,
      await page.screenshot({ fullPage: true }),
    );
    await page.setViewportSize({ width: 390, height: 844 });
    await expect(page.getByText(summary, { exact: true })).toBeVisible();
    await attach(
      `report-${route}-narrow`,
      await page.screenshot({ fullPage: true }),
    );
  }
  await page.setViewportSize({ width: 1440, height: 960 });

  // Go 同域入口必须返回真实构建资源，不能由 SPA 吞掉 API 或缺失资源路径。
  const serverOrigin = requiredEnvironment('RONCIN_ACCEPTANCE_BASE_URL');
  for (const [route] of reports) {
    const response = await page.request.get(
      `${serverOrigin}/finance/reports/${route}`,
    );
    expect(response.ok()).toBeTruthy();
    expect(response.headers()['content-type']).toContain('text/html');
    const html = await response.text();
    const script = html.match(/<script[^>]+src="([^"]+\.js)"/);
    expect(script?.[1], '生产页面应包含构建后的脚本').toBeTruthy();
    const resource = await page.request.get(
      new URL(script?.[1] ?? '', serverOrigin).href,
    );
    expect(resource.ok()).toBeTruthy();
    expect(resource.headers()['content-type']).toMatch(/javascript/);
  }
  const identity = await page.request.get(`${serverOrigin}/api/v1/auth/me`);
  expect(identity.ok()).toBeTruthy();
  expect(identity.headers()['content-type']).toContain('application/json');
  const missing = await page.request.get(
    `${serverOrigin}/assets/finance-acceptance-missing.js`,
  );
  expect(missing.status()).toBe(404);
}

// 基础资料经 API 准备；费用、账单、收款、核销与提成均通过真实页面写入。
test(`${isAirExport ? '空运' : '海运'}订单完成录费、建账、收款核销与提成生成并刷新重读`, async ({
  page,
}, testInfo) => {
  test.setTimeout(360_000);
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
  await page.getByText('账号登录', { exact: true }).click();
  await page
    .getByPlaceholder('钉钉手机号 / 本地账号')
    .fill(requiredEnvironment('BOOTSTRAP_ADMIN_USERNAME'));
  await page.getByRole('button', { name: '下一步', exact: true }).click();
  await expect(page.getByPlaceholder('请输入密码')).toBeVisible();
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
      roles: isAirExport
        ? [
            { type: 1, enabled: true },
            { type: 2, enabled: true },
          ]
        : [{ type: 1, enabled: true }],
      contacts: [
        { name: '页面验收联系人', phone: '13800000000', isPrimary: true },
      ],
      // 客户档案保存岗位默认值；订单API仍须显式提交完整内部人员。
      assignments: [
        { role: 3, userId: employee.id },
        { role: 4, userId: me.id },
      ],
      profile: {
        nameEn: `UI Acceptance ${stamp}`,
        countryCode: 'CN',
        businessTypes: [isAirExport ? 3 : 1],
      },
    })
  ).data;
  const settlementAccount = (
    await api(`/api/v1/partners/${customer.id}/accounts`, {
      partnerId: customer.id,
      account: {
        name: '页面验收结算账户',
        accountHolder: customer.legalName,
        currency: 'CNY',
        bankName: '验收银行',
        accountNo: `UI${stamp}`,
        usage: isAirExport ? 3 : 1,
        isDefaultReceivable: true,
        isDefaultPayable: isAirExport,
        enabled: true,
      },
    })
  ).data;
  let transportInput: Record<string, unknown>;
  if (isAirExport) {
    const systemOrganization = me.organizations.find(
      (organization: { kind: number }) => organization.kind === 1,
    );
    expect(systemOrganization?.id).toBeTruthy();
    await api('/api/v1/auth/switch-organization', {
      organizationId: systemOrganization.id,
    });
    // 一次性验收库显式创建本场景需要的三条行业基础资料，不依赖开发库快照。
    const airline = (
      await api('/api/v1/master-data/airlines', {
        iataCode: 'MU',
        icaoCode: 'CES',
        awbPrefix: '781',
        nameZh: '验收航空',
        nameEn: 'Acceptance Airlines',
        countryCode: 'CN',
        source: 'MANUAL',
      })
    ).data;
    const origin = (
      await api('/api/v1/master-data/airports', {
        iataCode: 'PVG',
        icaoCode: 'ZSPD',
        nameZh: '验收上海机场',
        nameEn: 'Acceptance Shanghai',
        cityNameZh: '上海',
        cityNameEn: 'Shanghai',
        countryCode: 'CN',
      })
    ).data;
    const destination = (
      await api('/api/v1/master-data/airports', {
        iataCode: 'LAX',
        icaoCode: 'KLAX',
        nameZh: '验收洛杉矶机场',
        nameEn: 'Acceptance Los Angeles',
        cityNameZh: '洛杉矶',
        cityNameEn: 'Los Angeles',
        countryCode: 'US',
      })
    ).data;
    await api('/api/v1/auth/switch-organization', { organizationId });
    expect(airline?.awbPrefix).toMatch(/^\d{3}$/);
    expect(origin?.id).toBeTruthy();
    expect(destination?.id).toBeTruthy();
    const serial = stamp.slice(-7);
    transportInput = {
      airExport: {
        documentMode: 'HOUSE',
        master: {
          masterNo: `${airline.awbPrefix}${serial}${Number(serial) % 7}`,
          airlineId: airline.id,
          originAirportId: origin.id,
          destinationAirportId: destination.id,
          flightNo: 'MU583',
          etd: new Date(Date.now() + 86_400_000).toISOString(),
          eta: new Date(Date.now() + 172_800_000).toISOString(),
        },
        house: { houseNo: `UI-AE-${stamp}` },
      },
    };
  } else {
    const carrier = (
      await api('/api/v1/master-data/shipping-lines?page=1&pageSize=10')
    ).data[0];
    transportInput = {
      shippingLineId: carrier.id,
      seaMasterBill: { masterNo: `UIMB${stamp}` },
      seaDocument: { documentStructure: 2 },
      shipmentType: 2,
      etd: new Date(Date.now() + 86_400_000).toISOString(),
      eta: new Date(Date.now() + 172_800_000).toISOString(),
    };
  }
  const order = (
    await api('/api/v1/orders', {
      customerId: customer.id,
      ...transportInput,
      businessType: isAirExport ? 3 : 1,
      tradeDirection: 1,
      tradeTerm: 3,
      paymentTerm: 1,
      goodsDescription: '同订单页面闭环验收',
      totalPackages: 1,
      totalGrossWeightKg: 100,
      totalVolumeCbm: 1,
      customerReferenceNo: `ACC-UI-${stamp}`,
      orderDate: new Date().toISOString(),
      // API验收显式提交操作、销售与客服，默认带入由真实页面表单负责。
      personnelAssignments: [
        { userId: me.id, role: 2 },
        { userId: employee.id, role: 3 },
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
    (item: {
      defaultBillingUnitId?: string;
      taxRate?: string;
      defaultCurrency?: string;
    }) =>
      item.defaultBillingUnitId &&
      item.taxRate !== undefined &&
      Number(item.taxRate) === 0 &&
      item.defaultCurrency === 'CNY',
  );
  expect(feeSetting).toBeTruthy();
  await page.goto(`/orders/${orderKind}/${order.id}/fees`);
  await page.getByRole('button', { name: '新增应收费用' }).click();
  const editRow = page.locator('.ant-table-row[data-row-key^="new_"]').first();
  await selectOption(
    page,
    editRow.locator('td').nth(2).locator('.ant-select'),
    feeSetting.nameZh,
  );
  // 等待费用项目选择写入行模型，再录金额，避免紧接着输入触发行预览时读取旧选择。
  await expect(editRow.locator('td').nth(1)).toHaveText(feeSetting.feeCode);
  await expect(editRow.locator('td').nth(2)).toContainText(feeSetting.nameZh);
  await editRow.getByPlaceholder('0.00').fill('125');
  const feeResponse = page.waitForResponse(
    (r) =>
      r.url().endsWith(`/orders/${order.id}/fees:save`) &&
      r.request().method() === 'POST',
  );
  await page.getByRole('button', { name: '保存应收费用' }).click();
  const fee = (await (await feeResponse).json()).data[0];
  expect(fee.totalAmount).toBe('125.00000000');
  expect(fee.hasActiveBill).not.toBe(true);
  await expect(page.getByText('125.00', { exact: true }).first()).toBeVisible();
  await page.screenshot({
    path: testInfo.outputPath('01-fee.png'),
    fullPage: true,
  });
  // 审核锁统一在订单详情办理，费用页只展示锁定状态。
  await page.getByRole('button', { name: '返回订单详情' }).click();
  await page
    .getByRole('button', { name: /锁定订单$/ })
    .first()
    .click();
  const lockDialog = page.getByRole('dialog', { name: /锁定.*订单/ });
  await expect(
    lockDialog.getByRole('columnheader', { name: /^原币金额/ }),
  ).toHaveCount(2);
  await expect(
    lockDialog.getByRole('columnheader', { name: /^不含税金额/ }),
  ).toHaveCount(2);
  await expect(
    lockDialog.getByText(feeSetting.nameZh, { exact: true }),
  ).toBeVisible();
  await lockDialog.getByRole('button', { name: '确认锁定' }).click();
  await expect(lockDialog).not.toBeVisible();
  await page.goto(`/orders/${orderKind}/${order.id}/fees`);
  await expect(
    page.getByRole('button', { name: '新增应收费用' }),
  ).toBeDisabled();
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
  await billDrawer.getByRole('button', { name: '刷新账单' }).click();
  const billPreview = await (await billPreviewResponse).json();
  expect(billPreview.data[0].groupToken).toBeTruthy();
  expect(billPreview.data).toHaveLength(1);
  expect(billPreview.data[0].configurationComplete).toBe(true);
  const billResponse = page.waitForResponse(
    (r) =>
      r.url().endsWith('/finance/bill-batches') &&
      r.request().method() === 'POST',
  );
  await billDrawer.getByRole('button', { name: /生成 1 张账单/ }).click();
  const billResult = await (await billResponse).json();
  expect(
    billResult.results.every(
      (group: { success?: boolean; bill?: { id?: string } }) =>
        group.success && group.bill?.id,
    ),
  ).toBe(true);
  const batch = billResult.data;
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
  // 账单免人工确认：生成即为正常状态，结果页不再提供确认入口。
  expect(bill.status).toBe(FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED);
  await expect(billDrawer.getByText(/账单无需确认/)).toBeVisible();
  await expect(
    billDrawer.getByRole('button', { name: '确认本批全部账单' }),
  ).toHaveCount(0);
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
  await expect(cashDialog.getByLabel('所属公司')).toHaveCount(0);
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
  // 抬头属于基础资料；已有发票的明确分配和税额由真实页面提交。
  await api(`/api/v1/partners/${customer.id}/invoice-profiles`, {
    partnerId: customer.id,
    invoiceTitle: customer.legalName,
    taxpayerIdentificationNo: customer.unifiedSocialCreditCode,
    registeredAddress: customer.registeredAddress,
    defaultInvoiceType: 'DIGITAL_NORMAL',
    isDefault: true,
  });
  await page.goto('/finance/bills');
  const registeredBillRow = page.locator(
    `.ant-table-row[data-row-key="${bill.id}"]`,
  );
  await registeredBillRow.getByText(bill.billNo, { exact: true }).click();
  await page.getByRole('button', { name: '继续登记发票', exact: true }).click();
  const invoiceDialog = page.getByRole('dialog', {
    name: '登记已有发票',
    exact: true,
  });
  // 唯一抬头、余额及逐费分配在打开时带入；这里只填写实际票面资料。
  await expect(
    invoiceDialog.getByLabel('开票抬头', { exact: true }),
  ).toHaveCount(1);
  await expect(
    invoiceDialog.getByText(customer.legalName, { exact: false }),
  ).toBeVisible();
  await expect(
    invoiceDialog.getByLabel('发票含税总额', { exact: true }),
  ).toHaveValue('125');
  await expect(
    invoiceDialog.getByLabel('本次含税金额', { exact: true }),
  ).toHaveValue('125');
  await selectOption(
    page,
    invoiceDialog.getByLabel('发票类型', { exact: true }),
    '纸质普票',
  );
  await invoiceDialog
    .getByLabel('已有发票号码', { exact: true })
    .fill(`UI-TAX-${stamp}`);
  await invoiceDialog.getByLabel('本次登记税额', { exact: true }).fill('7.89');
  const invoiceSaved = page.waitForResponse(
    (response) =>
      response.url().endsWith('/api/v1/finance/invoices/register') &&
      response.request().method() === 'POST',
  );
  await invoiceDialog
    .getByRole('button', { name: '保存登记', exact: true })
    .click();
  const invoiceSaveResponse = await invoiceSaved;
  expect(invoiceSaveResponse.ok()).toBeTruthy();
  const issuedInvoice = (await invoiceSaveResponse.json()).data[0];
  expect(issuedInvoice.invoiceType).toBe('PAPER_NORMAL');
  expect(issuedInvoice.taxInvoiceNo).toBe(`UI-TAX-${stamp}`);
  expect(issuedInvoice.taxAmount).toBe('7.89000000');
  expect(issuedInvoice.netAmount).toBe('117.11000000');
  await expect(invoiceDialog).toBeHidden();
  await expect(page.locator('.ant-notification-notice-error')).toHaveCount(0);
  expect(
    (await api(`/api/v1/finance/bills/${bill.id}`)).data.completed ?? false,
  ).toBe(false);
  const sourceAfterInvoice = (
    await api(`/api/v1/orders/${order.id}/fees`)
  ).data.find((item: { id: string }) => item.id === fee.id);
  expect(sourceAfterInvoice.netAmount).toBe(fee.netAmount);
  expect(sourceAfterInvoice.taxAmount).toBe(fee.taxAmount);
  await page.screenshot({
    path: testInfo.outputPath('04-invoice-registration.png'),
    fullPage: true,
  });

  await page.goto('/finance/verifications');
  await page.getByRole('button', { name: '新建核销' }).click();
  const verificationDialog = page.getByRole('dialog', {
    name: '资金与账单核销工作台',
  });
  await expect(verificationDialog.getByLabel('所属公司')).toHaveCount(0);
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
  await expect(commissionDialog.getByLabel('所属公司')).toHaveCount(0);
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
  expect(persistedBill.completed).toBe(true);
  expect(persistedCashflow.verifiedAmount).toBe('125.00000000');
  expect(persistedCashflow.unverifiedAmount).toBe('0.00000000');
  expect(persisted.commissionAmount).toBe('12.50000000');
  if (isAirExport) {
    // 空运使用同一页面登记结果，再覆盖应付完整链路。
    const master = order.airExport.master;
    const payableSerial = String(
      (Number(stamp.slice(-7)) + 1) % 10_000_000,
    ).padStart(7, '0');
    const payableOrder = (
      await api('/api/v1/orders', {
        customerId: customer.id,
        businessType: 3,
        tradeDirection: 1,
        tradeTerm: 3,
        paymentTerm: 1,
        orderDate: new Date().toISOString(),
        goodsDescription: '空运应付财务闭环验收',
        totalPackages: 1,
        totalGrossWeightKg: 100,
        totalVolumeCbm: 1,
        // API验收显式提交完整内部人员，不依赖服务端猜测客户默认值。
        personnelAssignments: [
          { userId: me.id, role: 2 },
          { userId: employee.id, role: 3 },
          { userId: me.id, role: 4 },
        ],
        airExport: {
          documentMode: 'HOUSE',
          master: {
            ...master.content,
            masterNo: `${master.content.masterNo.slice(0, 3)}${payableSerial}${Number(payableSerial) % 7}`,
          },
          house: { houseNo: `UI-AE-PAY-${stamp}` },
        },
      })
    ).data;
    const payableFee = (
      await api(`/api/v1/orders/${payableOrder.id}/fees`, {
        orderId: payableOrder.id,
        direction: 2,
        settlementPartyId: customer.id,
        quantity: '1',
        unitPrice: '25',
        currency: 'CNY',
        feeSettingId: feeSetting.id,
        billingUnitId: feeSetting.defaultBillingUnitId,
        taxInclusive: true,
        idempotencyKey: `ui-ae-pay-fee-${stamp}`,
      })
    ).data;
    const lockState = (
      await api(`/api/v1/orders/${payableOrder.id}/lock-state`)
    ).data;
    const locked = (
      await api(`/api/v1/orders/${payableOrder.id}/lock`, {
        orderId: payableOrder.id,
        expectedOrderVersion: lockState.orderVersion,
        expectedFeeContentVersion: lockState.feeContentVersion,
        feeContentFingerprint: lockState.feeContentFingerprint,
        comment: '隔离验收仅录应付以验证亏损核对及空运应付链路',
        idempotencyKey: `ui-ae-pay-lock-${stamp}`,
      })
    ).data;
    expect(locked.state.isLocked).toBe(true);
    const batchInput = {
      feeIds: [payableFee.id],
      organizationId,
      groupingPolicy: { mode: 1, splitByOrder: true, splitByTaxRate: true },
    };
    const initial = await api(
      '/api/v1/finance/bill-batches/preview',
      batchInput,
    );
    expect(initial.data).toHaveLength(1);
    const group = {
      groupKey: initial.data[0].groupKey,
      settlementAccountId: settlementAccount.id,
    };
    const preview = await api('/api/v1/finance/bill-batches/preview', {
      ...batchInput,
      groupConfigs: [group],
    });
    const payableBatch = (
      await api('/api/v1/finance/bill-batches', {
        organizationId,
        groupingPolicy: batchInput.groupingPolicy,
        idempotencyKey: `ui-ae-pay-batch-${stamp}`,
        groups: [
          {
            ...group,
            groupToken: preview.data[0].groupToken,
            feeIds: preview.data[0].fees.map((item: { id: string }) => item.id),
            statementTitle: customer.legalName,
            paymentTermsDays: 0,
          },
        ],
      })
    ).data;
    const payableBill = payableBatch.bills[0];
    expect(payableBill.status).toBe(
      FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED,
    );
    expect(payableBill.direction).toBe('PAYABLE');
    // 本段验证应付核销，显式免票后再进入资金分配。
    await api(`/api/v1/finance/bills/${payableBill.id}/invoice-exemption`, {
      id: payableBill.id,
      expectedVersion: payableBill.version,
      reason: '页面验收应付核销免票',
    });
    const payment = (
      await api('/api/v1/finance/cashflows', {
        organizationId,
        direction: 'PAYABLE',
        settlementPartyId: customer.id,
        currency: 'CNY',
        amount: '25',
        exchangeRate: '1',
        baseCurrency: 'CNY',
        transactionDate: today,
        ourAccount: '隔离验收付款户',
        counterpartyAccount: settlementAccount.accountNo,
        paymentMethod: '银行转账',
        idempotencyKey: `ui-ae-pay-cashflow-${stamp}`,
      })
    ).data;
    const confirmedPayment = (
      await api(`/api/v1/finance/cashflows/${payment.id}/confirm`, {
        id: payment.id,
        expectedVersion: payment.version,
      })
    ).data;
    const payableVerification = (
      await api('/api/v1/finance/verifications', {
        allocations: [
          {
            cashflowId: confirmedPayment.id,
            billId: payableBill.id,
            amount: '25',
          },
        ],
        verificationDate: today,
        idempotencyKey: `ui-ae-pay-verification-${stamp}`,
      })
    ).data;
    expect(payableVerification.direction).toBe('PAYABLE');
    const settled = (await api(`/api/v1/finance/bills/${payableBill.id}`)).data;
    expect(settled.verifiedAmount).toBe('25.00000000');
    expect(settled.unverifiedAmount).toBe('0.00000000');
    await testInfo.attach('air-payable-invoice-evidence', {
      body: JSON.stringify({
        orderId: payableOrder.id,
        feeId: payableFee.id,
        billId: payableBill.id,
        verificationId: payableVerification.id,
        invoiceId: issuedInvoice.id,
      }),
      contentType: 'application/json',
    });
  }
  await inspectFinanceReports(page, async (name, body) => {
    const path = testInfo.outputPath(`${name}.png`);
    await writeFile(path, body);
    await testInfo.attach(name, { path, contentType: 'image/png' });
  });
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
    invoiceId: issuedInvoice.id,
    invoiceTaxAmount: issuedInvoice.taxAmount,
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
