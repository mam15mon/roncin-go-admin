import { expect, type Page, type Route, test } from '@playwright/test';

// 所有业务接口均拦截，不依赖账户凭据，也不向开发库写入订单。
const staff = [
  { userId: 'operator-default', displayName: '默认操作员' },
  { userId: 'operator-manual', displayName: '手动操作员' },
  { userId: 'sales', displayName: '客户业务员' },
  { userId: 'service', displayName: '客户客服' },
];
const partner = {
  id: 'customer',
  code: 'C001',
  legalName: '录入体验测试客户',
  assignments: [
    { role: 2, userId: 'operator-default' },
    { role: 3, userId: 'sales' },
    { role: 4, userId: 'service' },
  ],
};
async function fixture(page: Page) {
  const writes: unknown[] = [];
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  const unexpected: string[] = [];
  let customerReply: (() => Promise<void>) | undefined;
  let holdCustomer = false;
  let holdCreate = false;
  let createReply: (() => Promise<void>) | undefined;
  let succeed = false;
  await page.route('**/api/**', async (route: Route) => {
    const path = new URL(route.request().url()).pathname;
    const send = (data: unknown) =>
      route.fulfill({ json: { success: true, data } });
    if (path === '/api/v1/auth/me') {
      const permissions = [
        'business.order.se.create',
        'business.order.se.read',
        'business.order.se.personnel.read',
        'business.partner.read',
        'system.master_data.option.read',
        'system.master_data.port.read',
      ];
      return send({
        id: 'entry-user',
        username: 'entry',
        displayName: '录入验收',
        currentOrganization: { id: 'entry-org', name: '验收公司', kind: 2 },
        permissions,
        permissionCapabilities: permissions.map((key) => ({
          key,
          dataScope: 'all',
        })),
      });
    }
    if (path === '/api/v1/master-data/options')
      return send([
        { id: 'dg', name: '危险品', code: 'DG', kind: 9, enabled: true },
        {
          id: 'general',
          name: '普货',
          code: 'GENERAL',
          kind: 9,
          enabled: true,
        },
      ]);
    if (path === '/api/v1/master-data/ports')
      return send([
        {
          id: 'sha',
          nameZh: '上海港',
          nameEn: 'SHANGHAI',
          unLocode: 'CNSHA',
          enabled: true,
        },
      ]);
    if (path === '/api/v1/reference/currencies')
      return send([{ code: 'USD', name: '美元', enabled: true }]);
    if (path === '/api/v1/order-personnel-options') return send(staff);
    if (path === '/api/v1/partners') return send([partner]);
    if (path === '/api/v1/partners/customer') {
      if (holdCustomer) {
        customerReply = () => send(partner);
        return;
      }
      return send(partner);
    }
    if (path === '/api/v1/master-data/shipping-lines')
      return send([
        { id: 'cosco', nameZh: '中远海运', nameEn: 'COSCO', scacCode: 'COSU' },
      ]);
    if (path.includes('sea-master-bill-candidate'))
      return route.fulfill({ json: { success: true, data: null } });
    if (path === '/api/v1/order-reference-check')
      return send({ duplicate: false });
    if (path === '/api/v1/orders' && route.request().method() === 'POST') {
      writes.push(route.request().postDataJSON());
      const reply = () =>
        succeed
          ? send({ id: 'created-entry' })
          : route.fulfill({ status: 503, json: { message: '模拟结果未确认' } });
      if (holdCreate) {
        createReply = reply;
        return;
      }
      return reply();
    }
    if (path === '/api/v1/orders/created-entry')
      return send({
        id: 'created-entry',
        orderNo: 'SE-ENTRY',
        organizationId: 'entry-org',
        businessType: 1,
        version: '1',
      });
    if (path.startsWith('/api/v1/orders/created-entry/')) return send([]);
    if (path === '/api/v1/master-data/items') return send([]);
    if (path === '/api/v1/master-data/options/search') return send([]);
    unexpected.push(path);
    return send([]);
  });
  return {
    writes,
    errors,
    unexpected,
    holdCreate: () => {
      holdCreate = true;
    },
    releaseCreate: async () => {
      await expect.poll(() => Boolean(createReply)).toBe(true);
      await createReply?.();
      holdCreate = false;
    },
    succeed: () => {
      succeed = true;
    },
    hold: () => {
      holdCustomer = true;
    },
    release: async () => {
      await expect.poll(() => Boolean(customerReply)).toBe(true);
      await customerReply?.();
      holdCustomer = false;
    },
  };
}
async function choose(page: Page, label: string, text: string) {
  const input = page
    .getByLabel(label, { exact: true })
    .filter({ visible: true })
    .first();
  await input.click();
  await input.fill(text);
  await expect(
    page
      .locator('.ant-select-dropdown:visible .ant-select-item-option')
      .filter({ hasText: text })
      .first(),
  ).toBeVisible();
  await input.press('ArrowDown');
  await input.press('Enter');
  await input.press('Tab');
}
for (const viewport of [
  { width: 1366, height: 768 },
  { width: 1920, height: 1080 },
]) {
  test(`海运新建 ${viewport.width} 键盘录入与日期冲突保留`, async ({
    page,
  }) => {
    await page.setViewportSize(viewport);
    const state = await fixture(page);
    await page.goto('/orders/sea-export/new');
    await expect(page.locator('#customerId')).toBeVisible();
    await choose(page, '委托单位', '录入体验');
    await expect(page.locator('#salesUserId').locator('..')).toContainText(
      '客户业务员',
    );
    await choose(page, '起运港', '上海');
    for (const [label, value] of [
      ['ETD', '2026-11-10'],
      ['ETA', '2026-11-12'],
    ] as const) {
      await page.getByLabel(label, { exact: true }).fill(value);
      await page.getByLabel(label, { exact: true }).press('Enter');
    }
    const description = page.getByLabel('委托品名 / 货物描述', { exact: true });
    await description.dispatchEvent('compositionstart');
    await description.fill('中文输入验收货物');
    await description.dispatchEvent('keydown', {
      key: 'Enter',
      code: 'Enter',
      keyCode: 229,
      isComposing: true,
    });
    await description.dispatchEvent('compositionend', {
      data: '中文输入验收货物',
    });
    expect(state.writes).toHaveLength(0);
    await page.getByLabel('ETD', { exact: true }).fill('2026-11-15');
    await page.getByLabel('ETD', { exact: true }).press('Enter');
    await expect(page.getByLabel('ETA', { exact: true })).toHaveValue(
      '2026-11-12',
    );
    await expect(
      page.getByText('ETA 不能早于 ETD', { exact: true }),
    ).toBeVisible();
    const footer = page.locator('.roncin-sticky-footer-bar');
    await page.getByLabel('操作人员', { exact: true }).scrollIntoViewIfNeeded();
    await expect(
      footer.getByRole('button', { name: '创建订单', exact: true }),
    ).toBeVisible();
    const footerBox = await footer.boundingBox();
    expect(footerBox?.y).toBeLessThan(viewport.height);
    await page
      .getByLabel('关联人员 2', { exact: true })
      .scrollIntoViewIfNeeded();
    const lastField = await page
      .getByLabel('关联人员 2', { exact: true })
      .boundingBox();
    const finalFooter = await footer.boundingBox();
    expect(
      lastField &&
        finalFooter &&
        lastField.y + lastField.height <= finalFooter.y,
    ).toBe(true);
    await page.screenshot({
      path: `/tmp/roncin-se-entry-artifacts/entry-${viewport.width}.png`,
      fullPage: true,
    });
    expect(state.unexpected).toEqual([]);
    expect(state.errors).toEqual([]);
  });
}

test('客户迟到默认人员不覆盖手动选择，刷新恢复完整人员草稿', async ({
  page,
}) => {
  const state = await fixture(page);
  state.hold();
  await page.goto('/orders/sea-export/new');
  await choose(page, '委托单位', '录入体验');
  await choose(page, '操作人员', '手动操作员');
  await state.release();
  await expect(page.locator('#operatorUserId').locator('..')).toContainText(
    '手动操作员',
  );
  await expect(page.locator('#salesUserId').locator('..')).toContainText(
    '客户业务员',
  );
  await page.reload();
  await expect(page.locator('#operatorUserId').locator('..')).toContainText(
    '手动操作员',
  );
  await expect(page.locator('#salesUserId').locator('..')).toContainText(
    '客户业务员',
  );
  expect(state.unexpected).toEqual([]);
  expect(state.errors).toEqual([]);
});

test('创建在途冻结输入，未知结果刷新后同载荷确认并进入原订单', async ({
  page,
}) => {
  const state = await fixture(page);
  state.holdCreate();
  await page.goto('/orders/sea-export/new');
  await choose(page, '委托单位', '录入体验');
  await expect(page.locator('#salesUserId').locator('..')).toContainText(
    '客户业务员',
  );
  await choose(page, '船公司', '中远');
  await page.getByLabel('MBL 主单号', { exact: true }).fill('TESTMBL123');
  await page.getByLabel('分单号 (HBL No.)', { exact: true }).fill('TESTHBL123');
  await page.getByText('危险品', { exact: true }).click();
  await page.getByLabel('UN NO.', { exact: true }).fill('1234');
  for (const [label, date] of [
    ['ETD', '2026-11-10'],
    ['ETA', '2026-11-12'],
  ] as const) {
    await page.getByLabel(label, { exact: true }).fill(date);
    await page.getByLabel(label, { exact: true }).press('Enter');
  }
  await page
    .getByLabel('委托品名 / 货物描述', { exact: true })
    .fill('冻结的原始内容');
  await page
    .getByRole('button', { name: /创建订单$/ })
    .first()
    .click();
  await expect.poll(() => state.writes.length).toBe(1);
  await expect(
    page.getByLabel('委托品名 / 货物描述', { exact: true }),
  ).toBeDisabled();
  const cargoInputs = page
    .locator('.roncin-currency-amount-input')
    .first()
    .locator('input');
  await expect(cargoInputs).toHaveCount(2);
  for (const input of await cargoInputs.all())
    await expect(input).toBeDisabled();
  for (const id of [
    'totalPackages',
    'totalPackageUnit',
    'unNumber',
    'hazardClass',
    'shippingLineId',
    'seaHouseBill_houseNo',
    'seaMasterBillContent_shipperText',
  ]) {
    await expect(page.locator(`#${id}`)).toBeDisabled();
  }
  await page.getByRole('tab', { name: /分单 \(HBL\)/ }).click();
  await expect(
    page.locator('#seaHouseBill_content_shipperText'),
  ).toBeDisabled();
  await state.releaseCreate();
  await expect(
    page.getByRole('button', { name: '确认创建结果', exact: true }).first(),
  ).toBeVisible();
  await page.reload();
  await expect(
    page.getByRole('button', { name: '确认创建结果', exact: true }).first(),
  ).toBeVisible();
  await expect(
    page.getByLabel('委托品名 / 货物描述', { exact: true }),
  ).toBeDisabled();
  state.succeed();
  await page
    .getByRole('button', { name: '确认创建结果', exact: true })
    .first()
    .click();
  await expect(page).toHaveURL(/\/orders\/sea-export\/created-entry$/);
  expect(state.writes).toHaveLength(2);
  expect(state.writes[1]).toEqual(state.writes[0]);
  await expect(page.getByRole('dialog')).toHaveCount(0);
  expect(state.errors).toEqual([]);
});

test('空表提交自动定位并聚焦首个必填字段', async ({ page }) => {
  const state = await fixture(page);
  await page.goto('/orders/sea-export/new');
  await expect(page.locator('#customerId')).toBeVisible();
  await page
    .getByRole('button', { name: /创建订单$/ })
    .first()
    .click();
  await expect(page.locator('#customerId')).toBeFocused();
  expect(state.writes).toHaveLength(0);
  expect(state.errors).toEqual([]);
});
