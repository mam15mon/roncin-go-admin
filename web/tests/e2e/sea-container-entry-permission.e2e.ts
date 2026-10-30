import { expect, type Page, test } from '@playwright/test';

async function fixture(page: Page, shipmentType = 1) {
  let added = false;
  let denied = false;
  let candidatesFailed = false;
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  await page.route('**/api/**', async (route) => {
    const url = new URL(route.request().url());
    const path = url.pathname;
    const send = (data: unknown, extra = {}) =>
      route.fulfill({ json: { success: true, data, ...extra } });
    const org = { id: 'org', name: '浏览器验收公司', kind: 2 };
    const permissions = [
      'read',
      'create',
      'update',
      'container.read',
      'container.create',
      'container.update',
      'container.delete',
    ].map((op) => `business.order.se.${op}`);
    if (path === '/api/v1/auth/me')
      return send({
        id: 'test-user',
        displayName: '验收员',
        currentOrganization: org,
        permissionCapabilities: permissions.map((key) => ({
          key,
          dataScope: 'all',
        })),
      });
    if (path === '/api/v1/master-data/options')
      return send([
        { id: '40hq', name: '40HQ', kind: 7, code: '40HQ', enabled: true },
      ]);
    if (path === '/api/v1/reference/currencies')
      return send([{ code: 'USD', name: '美元' }]);
    if (path === '/api/v1/orders/order')
      return send({
        id: 'order',
        orderNo: 'SE-TEST',
        allowedActions: [1],
        businessType: 1,
        organizationId: 'org',
        version: '1',
        shipmentType,
        seaMasterBill: {
          masterNo: 'TESTMBL',
          shippingLineId: 'line',
          transportExecutionId: 'te',
          memberCount: 1,
        },
        seaDocumentSummary: { documentStructure: 1 },
      });
    if (path.endsWith('/lock-state'))
      return send({ isLocked: false, businessType: 1 });
    if (path.endsWith('/sea-documents'))
      return send({
        orderId: 'order',
        documentStructure: 1,
        linkVersion: '1',
        masterBill: {
          id: 'mbl',
          masterNo: 'TESTMBL',
          version: '1',
          content: {},
        },
        houseBill: {
          id: 'hbl',
          houseNo: 'HBLTEST',
          version: '1',
          status: 1,
          content: {},
        },
      });
    if (path.endsWith('/containers')) {
      if (route.request().method() === 'POST') {
        added = true;
        return send({ id: 'box' });
      }
      return send(
        added
          ? [
              {
                id: 'box',
                containerNo: 'TEST1234567',
                containerSpecId: '40hq',
                packageCount: 1,
                grossWeightKg: 2,
                volumeCbm: 3,
                version: '1',
              },
            ]
          : [],
      );
    }
    if (path.endsWith('/sea-shared-containers')) {
      if (denied)
        return route.fulfill({
          status: 403,
          json: { code: 403, message: '无权执行此操作' },
        });
      return send([
        {
          id: 'shared',
          containerNo: 'SHARED123',
          version: '1',
          status: 1,
          packageCount: 100,
          grossWeightKg: '100',
          volumeCbm: '10',
          allocations: [],
        },
      ]);
    }
    if (path.endsWith('/sea-shared-container-candidates')) {
      if (url.searchParams.get('page') === '2' && !candidatesFailed) {
        candidatesFailed = true;
        return route.fulfill({ status: 500, json: { message: '候选失败' } });
      }
      return send(
        [
          {
            orderId: 'order',
            orderNo: 'SE-TEST',
            allowedActions: [1],
            houseBillId: 'hbl',
            orderVersion: '1',
            houseBillVersion: '1',
            linkVersion: '1',
            cargoItems: [
              {
                id: 'cargo',
                cargoName: '测试货物',
                packageCount: 100,
                grossWeightKg: '100',
                volumeCbm: '10',
                version: '1',
              },
            ],
          },
        ],
        { total: 21 },
      );
    }
    if (path.includes('order-change-actions')) return send({});
    return send([]);
  });
  return {
    errors,
    deny: () => {
      denied = true;
    },
  };
}
for (const width of [1366, 1920]) {
  test(`实际箱保存不覆盖订单输入 ${width}`, async ({ page }) => {
    await page.setViewportSize({ width, height: width === 1366 ? 768 : 1080 });
    const state = await fixture(page);
    await page.goto('/orders/sea-export/order');
    await expect(
      page.getByRole('button', { name: '普通实际箱', exact: true }),
    ).toBeVisible();
    await page.locator('#goodsDescription').fill('尚未保存的订单备注');
    await page.getByRole('button', { name: '普通实际箱', exact: true }).click();
    await page.getByRole('button', { name: /添加集装箱/ }).click();
    await page.getByLabel('箱号', { exact: true }).fill('TEST1234567');
    await page.getByLabel('件数 (PCS)', { exact: true }).fill('1');
    await page.getByLabel('货物毛重 (KG)', { exact: true }).fill('2');
    await page.getByLabel('货物体积 (CBM)', { exact: true }).fill('3');
    await page.getByLabel('集装箱规格', { exact: true }).click();
    await page
      .locator('.ant-select-dropdown:visible')
      .getByText('40HQ (40HQ)', { exact: true })
      .first()
      .click();
    await page
      .getByRole('button', { name: /确\s*定/ })
      .last()
      .click();
    await expect(page.getByText('添加集装箱成功')).toBeVisible();
    await page
      .getByRole('dialog')
      .getByRole('button', { name: /关\s*闭/ })
      .first()
      .click();
    await expect(page.locator('#goodsDescription')).toHaveValue(
      '尚未保存的订单备注',
    );
    await expect(page.getByText(/独占箱 1 箱/)).toBeVisible();
    expect(state.errors).toEqual([]);
  });
}
test('共享箱 403 不显示正常零空态', async ({ page }) => {
  const state = await fixture(page);
  state.deny();
  await page.goto('/orders/sea-export/order');
  await page.getByRole('button', { name: '跨订单共享箱', exact: true }).click();
  await expect(
    page.getByText('无权读取共享箱，请联系管理员核对权限'),
  ).toBeVisible();
  await expect(page.getByText('当前航次暂无共享物理箱')).toHaveCount(0);
  await expect(page.getByText('立即创建共享箱')).toHaveCount(0);
  await expect(page.getByText(/本航次共享物理箱 \(0\)/)).toHaveCount(0);
  expect(state.errors).toEqual([]);
});
test('仅候选失败重试保留分配草稿', async ({ page }) => {
  await fixture(page);
  await page.goto('/orders/sea-export/order');
  await page.getByRole('button', { name: '跨订单共享箱', exact: true }).click();
  await expect(page.getByText('测试货物')).toBeVisible();
  await page.getByRole('dialog').getByRole('spinbutton').first().fill('37');
  await page.getByRole('dialog').getByTitle('2', { exact: true }).click();
  await page.getByRole('button', { name: '重试候选订单', exact: true }).click();
  await expect(page.getByText('测试货物')).toBeVisible();
  await expect(
    page.getByRole('dialog').getByRole('spinbutton').first(),
  ).toHaveValue('37');
});
test('新建保存后引导与拼箱散杂入口', async ({ page }) => {
  await fixture(page);
  await page.goto('/orders/sea-export/new');
  await expect(
    page.getByText(
      '保存订单后，可在此维护实际箱型、箱号、封号及件重尺。计划箱量与实际装箱分别维护。',
    ),
  ).toBeVisible();
  await expect(
    page.getByRole('button', { name: '普通实际箱', exact: true }),
  ).toHaveCount(0);
});
for (const shipmentType of [2, 3]) {
  test(`非整箱入口保持写入限制 ${shipmentType}`, async ({ page }) => {
    await fixture(page, shipmentType);
    await page.goto('/orders/sea-export/order');
    await expect(
      page.getByText(
        '普通独占箱仅支持整箱订单；拼箱请使用跨订单共享箱，散杂货不维护集装箱。',
      ),
    ).toBeVisible();
    if (shipmentType === 3)
      await expect(
        page.getByRole('button', { name: '跨订单共享箱', exact: true }),
      ).toBeDisabled();
    else
      await expect(
        page.getByRole('button', { name: '跨订单共享箱', exact: true }),
      ).toBeEnabled();
    await page.getByRole('button', { name: '普通实际箱', exact: true }).click();
    await expect(page.getByRole('button', { name: /添加集装箱/ })).toHaveCount(
      0,
    );
  });
}
