import { expect, type Page, test } from '@playwright/test';

async function fixture(page: Page) {
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  await page.route('**/api/**', async (route) => {
    const url = new URL(route.request().url());
    const send = (data: unknown, extra = {}) =>
      route.fulfill({ json: { success: true, data, ...extra } });
    if (url.pathname === '/api/v1/auth/me') {
      const permissions = [
        'system.master_data.port.read',
        'system.finance.fee.read',
      ];
      return send({
        id: 'patrol-user',
        username: 'patrol',
        displayName: '巡检验收',
        currentOrganization: { id: 'patrol-org', name: '巡检公司', kind: 2 },
        permissions,
        permissionCapabilities: permissions.map((key) => ({
          key,
          dataScope: 'all',
        })),
      });
    }
    if (url.pathname === '/api/v1/master-data/ports') {
      if (url.searchParams.has('enabled')) {
        return send([], {
          total: url.searchParams.get('enabled') === 'true' ? 17526 : 2,
        });
      }
      const searched = url.searchParams.get('keyword') === 'CNSHA';
      const second = url.searchParams.get('page') === '2';
      return send(
        [
          {
            id: searched ? 'shanghai' : second ? 'rail-port' : 'sea-port',
            unLocode: searched ? 'CNSHA' : second ? 'CNRAI' : 'CNSEA',
            nameZh: searched ? '上海港' : second ? '联运港' : '测试海港',
            nameEn: 'PORT',
            countryCode: 'CN',
            source: 'manual',
            enabled: true,
            transportModes: second && !searched ? ['SEA', 'RAIL'] : ['SEA'],
          },
        ],
        { total: searched ? 1 : 17528 },
      );
    }
    if (url.pathname === '/api/v1/finance/fees') {
      return send(
        [
          {
            id: 'partial',
            orderId: 'order',
            orderNo: 'SE2026110800001',
            organizationId: 'patrol-org',
            direction: 'RECEIVABLE',
            currency: 'CNY',
            feeName: '部分核销费用',
            totalAmount: '300.00000000',
            verifiedAmount: '125.00000000',
            unverifiedAmount: '150.00000000',
            nettedAmount: '25.00000000',
            financialProgress: 6,
          },
          {
            id: 'full',
            orderId: 'order',
            orderNo: 'SE2026110800001',
            organizationId: 'patrol-org',
            direction: 'RECEIVABLE',
            currency: 'CNY',
            feeName: '已核销费用',
            totalAmount: '300.00000000',
            verifiedAmount: '300.00000000',
            unverifiedAmount: '0.00000000',
            nettedAmount: '0.00000000',
            financialProgress: 4,
          },
        ],
        { total: 2, summary: {}, matchedSummary: { count: 2 } },
      );
    }
    return send([]);
  });
  return errors;
}

test('港口搜索与翻页明确统计范围，上海港按标准运输类型展示', async ({
  page,
}) => {
  const errors = await fixture(page);
  await page.goto('/master-data');
  const stat = (label: string) =>
    page
      .locator('.ant-statistic')
      .filter({ has: page.getByText(label, { exact: true }) })
      .locator('.ant-statistic-content-value');
  await expect(stat('筛选结果港口')).toHaveText('17,528');
  await expect(stat('全目录启用中')).toHaveText('17,526');
  await expect(stat('本页海港枢纽')).toHaveText('1');
  await expect(stat('本页铁路联运枢纽')).toHaveText('0');
  await page.locator('.ant-pagination-item-2').click();
  await expect(page.getByText('CNRAI', { exact: true })).toBeVisible();
  await expect(stat('筛选结果港口')).toHaveText('17,528');
  await expect(stat('本页铁路联运枢纽')).toHaveText('1');
  await page
    .getByPlaceholder('搜索五字码(如 CNSHG) / 港口中英文名...')
    .fill('CNSHA');
  await expect(page.getByText('CNSHA', { exact: true })).toBeVisible();
  await expect(stat('筛选结果港口')).toHaveText('1');
  await expect(stat('全目录启用中')).toHaveText('17,526');
  await expect(stat('本页海港枢纽')).toHaveText('1');
  const row = page.getByRole('row').filter({ hasText: 'CNSHA' });
  await expect(row.getByText('海港', { exact: true })).toBeVisible();
  await expect(row).not.toContainText('maritime');
  await page.screenshot({ path: '/tmp/roncin-patrol/ports-search.png' });
  expect(errors).toEqual([]);
});

test('费用台账显示实际核销、对冲与余额，已结清余额显示零', async ({ page }) => {
  const errors = await fixture(page);
  await page.goto('/finance/fees');
  const check = async (fee: string, title: string, amount: string) => {
    const headers = await page.locator('thead th').allTextContents();
    const index = headers.findIndex((header) => header.includes(title));
    expect(index).toBeGreaterThanOrEqual(0);
    await expect(
      page.getByRole('row').filter({ hasText: fee }).locator('td').nth(index),
    ).toHaveText(amount);
  };
  await expect(page.getByText('部分核销费用', { exact: true })).toBeVisible();
  await check('部分核销费用', '已核销金额', '125.00');
  await check('部分核销费用', '未核销金额', '150.00');
  await check('部分核销费用', '已对冲金额', '25.00');
  await check('已核销费用', '已核销金额', '300.00');
  await check('已核销费用', '未核销金额', '0.00');
  await check('已核销费用', '已对冲金额', '0.00');
  expect(errors).toEqual([]);
});
