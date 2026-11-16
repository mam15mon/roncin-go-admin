import {
  type APIRequestContext,
  expect,
  type Locator,
  type Page,
  test,
} from '@playwright/test';

function requiredEnvironment(name: string) {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`缺少环境变量 ${name}`);
  return value;
}

async function api<T>(
  request: APIRequestContext,
  path: string,
  method = 'GET',
  data?: unknown,
): Promise<T> {
  const response = await request.fetch(path, { method, data });
  const text = await response.text();
  const body = text ? JSON.parse(text) : {};
  expect(
    response.ok(),
    `${method} ${path} HTTP ${response.status()}: ${body.message}`,
  ).toBeTruthy();
  expect(
    body.success,
    `${method} ${path} HTTP ${response.status()}: ${body.message}`,
  ).not.toBe(false);
  return body.data as T;
}

async function dateTime(input: Locator, value: string) {
  await input.fill(value);
  await input.press('Enter');
}

async function savedService(page: Page, orderId: string) {
  const response = page.waitForResponse(
    (item) =>
      item.url().endsWith(`/api/v1/orders/${orderId}/import-handling`) &&
      item.request().method() === 'PUT',
  );
  await page
    .getByRole('dialog', { name: '维护本票进口服务', exact: true })
    .getByRole('button', { name: '保存服务资料', exact: true })
    .click();
  const result = await response;
  expect(result.ok(), (await result.json()).message).toBeTruthy();
  await expect(
    page.getByRole('dialog', { name: '维护本票进口服务', exact: true }),
  ).toBeHidden();
}

// 仅使用本地隔离库的伪造夹具；真实请求、真实组织权限和真实页面，不拦截业务响应。
test('海运进口直单与主分单、服务并行撤回及外部HBL归档真实闭环', async ({
  page,
  request,
}, info) => {
  test.setTimeout(300_000);
  page.setDefaultTimeout(15_000);
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  const stamp = Date.now().toString();
  await api(request, '/api/v1/auth/login', 'POST', {
    username: requiredEnvironment('BOOTSTRAP_ADMIN_USERNAME'),
    password: requiredEnvironment('BOOTSTRAP_ADMIN_PASSWORD'),
  });
  const me = await api<API.CurrentUser>(request, '/api/v1/auth/me');
  const company = await api<API.AdminOrganization>(
    request,
    '/api/v1/admin/organizations',
    'POST',
    {
      code: `SI_BROWSER_${stamp}`,
      name: `海运进口隔离验收公司${stamp}`,
      kind: 2,
      baseCurrency: 'CNY',
    },
  );
  const roles = await api<API.AdminRole[]>(
    request,
    `/api/v1/admin/organizations/${company.id}/roles`,
  );
  const administrator = roles.find((role) => role.code === 'administrator');
  expect(administrator?.id).toBeTruthy();
  const memberships = await api<API.AdminUserMembership[]>(
    request,
    `/api/v1/admin/users/${me.id}/memberships`,
  );
  const companyMembership = memberships.find(
    (item) => item.organizationId === company.id,
  );
  expect(companyMembership?.enabled).toBe(true);
  expect(companyMembership?.roleIds).toContain(administrator?.id);
  const carrier = await api<API.ShippingLine>(
    request,
    '/api/v1/master-data/shipping-lines',
    'POST',
    {
      scacCode: `S${Array.from({ length: 3 }, () => String.fromCharCode(65 + Math.floor(Math.random() * 26))).join('')}`,
      nameZh: `隔离验收船司${stamp}`,
      nameEn: 'Isolated SI Carrier',
      countryCode: 'CN',
    },
  );
  await api(request, '/api/v1/auth/switch-organization', 'POST', {
    organizationId: company.id,
  });
  const partner = await api<API.Partner>(request, '/api/v1/partners', 'POST', {
    legalName: `外部真实签发主体验收${stamp}`,
    roles: [
      { type: 1, enabled: true },
      { type: 2, enabled: true },
    ],
    profile: {
      nameEn: 'EXTERNAL ISSUER ACCEPTANCE',
      countryCode: 'CN',
      businessTypes: [2],
    },
    assignments: [3, 4].map((role) => ({
      role,
      userId: me.id,
      organizationId: company.id,
    })),
  });
  const input = {
    customerId: partner.id,
    shippingLineId: carrier.id,
    businessType: 2,
    tradeDirection: 2,
    tradeTerm: 3,
    paymentTerm: 1,
    shipmentType: 2,
    shipmentMode: 1,
    goodsDescription: '海运进口隔离验收货物',
    totalPackages: 2,
    totalGrossWeightKg: 200,
    totalVolumeCbm: 2,
    orderDate: new Date().toISOString(),
    etd: new Date().toISOString(),
    eta: new Date(Date.now() + 86_400_000).toISOString(),
    personnelAssignments: [2, 3, 4].map((role) => ({ role, userId: me.id })),
  };
  const direct = await api<API.Order>(request, '/api/v1/orders', 'POST', {
    ...input,
    seaMasterBill: { masterNo: `SIDIRECT${stamp}` },
    seaDocument: { documentStructure: 2 },
  });
  const houseNo = `SIHOUSE${stamp}`;
  const house = await api<API.Order>(request, '/api/v1/orders', 'POST', {
    ...input,
    seaMasterBill: { masterNo: `SIMASTER${stamp}` },
    seaDocument: {
      documentStructure: 3,
      houseBill: { houseNo, issuerSource: 3, issuerPartnerId: partner.id },
    },
  });
  const directDocuments = await api<API.SeaOrderDocuments>(
    request,
    `/api/v1/orders/${direct.id}/sea-documents`,
  );
  expect(directDocuments.documentStructure).toBe(2);
  expect(directDocuments.houseBill).toBeUndefined();
  const houseDocuments = await api<API.SeaOrderDocuments>(
    request,
    `/api/v1/orders/${house.id}/sea-documents`,
  );
  expect(houseDocuments.houseBill?.issuerPartnerId).toBe(partner.id);
  expect(houseDocuments.houseBill?.printTemplate).toBe('EXTERNAL_ARCHIVE');

  await page.goto('/user/login');
  await page.getByText('账号登录', { exact: true }).click();
  await page
    .getByPlaceholder('钉钉手机号 / 本地账号')
    .fill(requiredEnvironment('BOOTSTRAP_ADMIN_USERNAME'));
  await page.getByRole('button', { name: '下一步', exact: true }).click();
  await page
    .getByPlaceholder('请输入密码')
    .fill(requiredEnvironment('BOOTSTRAP_ADMIN_PASSWORD'));
  await page.locator('button[type="submit"]').click();
  await expect(
    page.getByRole('heading', { name: '选择进入的组织' }),
  ).toBeVisible();
  await page.getByRole('button', { name: `进入${company.name}` }).click();
  await expect(page).not.toHaveURL(/\/user\/login/);
  await page.goto(`/orders/sea-import/${direct.id}`);
  // 首次进入巨型海运详情需冷编译；限定只放宽这个页面入口。
  const directNotes = page.getByTestId('sea-notes-operationNotes');
  await expect(directNotes).toBeVisible({ timeout: 60_000 });
  await expect(
    page.getByRole('button', { name: '维护共同航程', exact: true }),
  ).toBeEnabled();
  await directNotes.getByRole('button', { name: '展开', exact: true }).click();
  await page
    .getByPlaceholder('请输入操作备注', { exact: true })
    .fill('直单刷新编辑验收');
  const directSave = page.waitForResponse(
    (item) =>
      item.url().endsWith(`/api/v1/orders/${direct.id}`) &&
      item.request().method() === 'PUT',
  );
  await page.getByRole('button', { name: /保存$/ }).click();
  expect((await directSave).ok()).toBeTruthy();
  await page.reload();
  await expect(
    page.getByPlaceholder('请输入操作备注', { exact: true }),
  ).toHaveValue('直单刷新编辑验收');

  await page.goto(`/orders/sea-import/${house.id}`);
  await page
    .getByTestId('sea-notes-operationNotes')
    .getByRole('button', { name: '展开', exact: true })
    .click();
  const operation = page.getByPlaceholder('请输入操作备注', { exact: true });
  await operation.fill('服务保存必须保留本票未保存输入');
  await page
    .getByRole('button', { name: '维护承接与完成情况', exact: true })
    .click();
  let dialog = page.getByRole('dialog', {
    name: '维护本票进口服务',
    exact: true,
  });
  await dialog.getByRole('checkbox', { name: '承接换单', exact: true }).check();
  await dialog.getByRole('checkbox', { name: '承接派送', exact: true }).check();
  // 无到货/清关/提货承接，仍允许并行完成换单与派送。
  await dateTime(
    dialog.getByLabel('换单实际完成时间', { exact: true }),
    '2026-11-16 10:00:00',
  );
  await dateTime(
    dialog.getByLabel('派送实际完成时间', { exact: true }),
    '2026-11-16 11:00:00',
  );
  await savedService(page, house.id || '');
  await expect(operation).toHaveValue('服务保存必须保留本票未保存输入');
  let handling = await api<API.OrderImportHandling>(
    request,
    `/api/v1/orders/${house.id}/import-handling`,
  );
  expect(handling.steps?.filter((step) => step.undertaken)).toHaveLength(2);
  expect(handling.steps?.filter((step) => step.occurredAt)).toHaveLength(2);
  const mainSave = page.waitForResponse(
    (item) =>
      item.url().endsWith(`/api/v1/orders/${house.id}`) &&
      item.request().method() === 'PUT',
  );
  await page.getByRole('button', { name: /保存$/ }).click();
  expect((await mainSave).ok()).toBeTruthy();
  await page.reload();
  await expect(operation).toHaveValue('服务保存必须保留本票未保存输入');

  await page
    .getByRole('button', { name: '维护承接与完成情况', exact: true })
    .click();
  dialog = page.getByRole('dialog', { name: '维护本票进口服务', exact: true });
  const deliveryDate = dialog.getByLabel('派送实际完成时间', { exact: true });
  const deliveryPicker = deliveryDate.locator(
    'xpath=ancestor::div[contains(concat(" ", normalize-space(@class), " "), " ant-picker ")][1]',
  );
  await deliveryPicker.hover();
  await deliveryPicker
    .getByRole('button', { name: '清除', exact: true })
    .click();
  await expect(deliveryDate).toHaveValue('');
  await dialog
    .getByRole('button', { name: '保存服务资料', exact: true })
    .click();
  await expect(
    dialog.getByText('撤回、取消或纠正已完成服务必须填写原因', { exact: true }),
  ).toBeVisible();
  const reason = '隔离浏览器验收：派送事实登记错误，撤回后重新办理';
  await dialog.getByLabel('变更原因', { exact: true }).fill(reason);
  await savedService(page, house.id || '');
  handling = await api<API.OrderImportHandling>(
    request,
    `/api/v1/orders/${house.id}/import-handling`,
  );
  expect(
    handling.steps?.find((step) => step.kind === 5)?.occurredAt,
  ).toBeUndefined();
  expect(handling.steps?.find((step) => step.kind === 5)?.undertaken).toBe(
    true,
  );
  const events = await api<API.ListImportHandlingEventsData>(
    request,
    `/api/v1/orders/${house.id}/import-handling/events?page=1&pageSize=20`,
  );
  const withdrawal = events.items?.find((event) => event.reason === reason);
  expect(
    withdrawal?.before?.steps?.find((step) => step.kind === 5)?.occurredAt,
  ).toBeTruthy();
  expect(
    withdrawal?.after?.steps?.find((step) => step.kind === 5)?.occurredAt,
  ).toBeUndefined();
  await page.getByText('服务变更历史', { exact: true }).click();
  await expect(page.getByText(reason, { exact: true })).toBeVisible();

  await page
    .getByRole('button', { name: '维护承接与完成情况', exact: true })
    .click();
  dialog = page.getByRole('dialog', { name: '维护本票进口服务', exact: true });
  await dialog
    .getByLabel('派送备注', { exact: true })
    .fill('并发变更后必须保留的输入');
  handling = await api<API.OrderImportHandling>(
    request,
    `/api/v1/orders/${house.id}/import-handling`,
  );
  await api(request, `/api/v1/orders/${house.id}/import-handling`, 'PUT', {
    expectedVersion: handling.orderVersion,
    steps: handling.steps?.map((step) => ({
      ...step,
      notes: step.kind === 5 ? '另一会话修改' : step.notes,
    })),
  });
  await page.evaluate(() => {
    window.dispatchEvent(new Event('blur'));
    window.dispatchEvent(new Event('focus'));
  });
  await expect(
    dialog.getByText('这个订单已经发生变动，请重新刷新。', { exact: true }),
  ).toBeVisible({ timeout: 25_000 });
  await expect(dialog.getByLabel('派送备注', { exact: true })).toHaveValue(
    '并发变更后必须保留的输入',
  );
  await expect(
    dialog.getByRole('button', { name: '保存服务资料', exact: true }),
  ).toBeDisabled();
  await dialog.getByRole('button', { name: /取\s*消/ }).click();
  await expect(dialog).toBeHidden();

  await page.goto(`/orders/sea-import/${house.id}/hbl`);
  await expect(
    page.getByText(new RegExp(`${partner.legalName}.*外部提单登记资料`)),
  ).toBeVisible();
  await expect(page.getByLabel('HBL 输出版本')).toHaveCount(0);
  const longText = `${Array.from(
    { length: 85 },
    (_, index) =>
      `归档长文第${String(index + 1).padStart(3, '0')}行：进口原单货物资料完整保留，逐行核对中文及英文 ACCEPTANCE ${index + 1}`,
  ).join('\n')}\nSI_ARCHIVE_LAST_LINE_085`;
  const shipperText = `EXTERNAL SHIPPER\n${Array.from({ length: 20 }, (_, index) => `境外原单发货人详细地址行 ${index + 1}`).join('\n')}\nSI_SHIPPER_LAST_LINE`;
  const containersText = `${Array.from({ length: 60 }, (_, index) => `CONTAINER ${String(index + 1).padStart(3, '0')} / SEAL ${index + 1} / 40HQ`).join('\n')}\nSI_CONTAINER_LAST_060`;
  await page.getByLabel('发货人 (Shipper)', { exact: true }).fill(shipperText);
  await page
    .getByLabel('收货人 (Consignee)', { exact: true })
    .fill('IMPORT CONSIGNEE\n进口收货人');
  await page
    .getByLabel('货物描述 (Description of Goods)', { exact: true })
    .fill(longText);
  await page
    .getByLabel('箱号／封号／箱型', { exact: true })
    .fill(containersText);
  const hblSave = page.waitForResponse(
    (item) =>
      item.url().includes('/sea-documents/house-bills/') &&
      item.request().method() === 'PUT',
  );
  await page
    .getByRole('button', { name: /保存制作资料/ })
    .first()
    .click();
  expect((await hblSave).ok()).toBeTruthy();
  await page.reload();
  await expect(
    page.getByLabel('货物描述 (Description of Goods)', { exact: true }),
  ).toHaveValue(longText);
  await page
    .getByRole('button', { name: /预览／打印／下载/ })
    .first()
    .click();
  const preview = page.getByRole('dialog', {
    name: new RegExp(`HBL 预览.*${houseNo}`),
  });
  await expect(preview.locator('iframe')).toBeVisible({ timeout: 60_000 });
  const downloadLink = preview.getByRole('link', {
    name: /下载 PDF/,
  });
  await expect(downloadLink).toBeEnabled();
  expect(await downloadLink.getAttribute('href')).toBe(
    await preview.locator('iframe').getAttribute('src'),
  );
  expect(
    await preview.getByRole('link', { name: /打开打印/ }).getAttribute('href'),
  ).toBe(await downloadLink.getAttribute('href'));
  const downloadEvent = page.waitForEvent('download');
  await downloadLink.click();
  const download = await downloadEvent;
  expect(download.suggestedFilename()).toContain('-archive-');
  await download.saveAs(info.outputPath('si-external-archive.pdf'));
  await page.screenshot({
    path: info.outputPath('si-archive-preview.png'),
    fullPage: true,
  });
  await preview.getByRole('button', { name: '关闭', exact: true }).click();
  const savedDocuments = await api<API.SeaOrderDocuments>(
    request,
    `/api/v1/orders/${house.id}/sea-documents`,
  );
  expect(savedDocuments.houseBill?.content?.goodsDescriptionText).toBe(
    longText,
  );
  expect(savedDocuments.houseBill?.printDetails?.containersText).toBe(
    containersText,
  );
  expect(savedDocuments.houseBill?.content?.shipperText).toBe(shipperText);
  expect(savedDocuments.houseBill?.issuerPartnerName).toBe(partner.legalName);
  await info.attach('si-validation-identities', {
    body: JSON.stringify({
      directOrderId: direct.id,
      houseOrderId: house.id,
      issuerName: partner.legalName,
      houseNo,
      savedHblVersion: savedDocuments.houseBill?.version,
    }),
    contentType: 'application/json',
  });

  const productionBase = requiredEnvironment('RONCIN_ACCEPTANCE_BASE_URL');
  const productionHbl = `${productionBase}/orders/sea-import/${house.id}/hbl`;
  const htmlResponse = await page.request.get(productionHbl);
  expect(htmlResponse.status()).toBe(200);
  expect(htmlResponse.headers()['content-type']).toContain('text/html');
  const html = await htmlResponse.text();
  const asset = html.match(/src="([^"]*assets\/[^"]+\.js)"/);
  expect(asset?.[1]).toBeTruthy();
  const assetResponse = await page.request.get(
    new URL(asset?.[1] || '', productionBase).href,
  );
  expect(assetResponse.status()).toBe(200);
  const font = await page.request.get(
    `${productionBase}/fonts/roncin-hbl/RoncinHBLSansSC-Regular.ttf`,
  );
  expect(font.status()).toBe(200);
  expect(font.headers()['content-type']).toMatch(/font|octet-stream/);
  expect(
    (
      await page.request.get(
        `${productionBase}/assets/si-acceptance-missing.js`,
      )
    ).status(),
  ).toBe(404);
  expect(
    (await page.request.get(`${productionBase}/health/ready`)).status(),
  ).toBe(200);
  await page.goto(productionHbl);
  await expect(
    page.getByLabel('货物描述 (Description of Goods)', { exact: true }),
  ).toHaveValue(longText);
  await page
    .getByRole('button', { name: /预览／打印／下载/ })
    .first()
    .click();
  const productionPreview = page.getByRole('dialog', {
    name: new RegExp(`HBL 预览.*${houseNo}`),
  });
  await expect(productionPreview.locator('iframe')).toBeVisible({
    timeout: 60_000,
  });
  await page.screenshot({
    path: info.outputPath('si-production-archive-preview.png'),
    fullPage: true,
  });
  await productionPreview
    .getByRole('button', { name: '关闭', exact: true })
    .click();

  for (const kind of ['sea-export', 'air-export']) {
    await page.goto(`/orders/${kind}/new`);
    await expect(
      page.getByRole('combobox', { name: /委托单位/ }),
    ).toBeVisible();
    await expect(page.getByRole('button', { name: /创建订单$/ })).toBeVisible();
  }
  expect(errors).toEqual([]);
});
