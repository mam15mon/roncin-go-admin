import { expect, test } from '@playwright/test';

function requiredEnvironment(name: string) {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`缺少环境变量 ${name}`);
  return value;
}

test('批量转账单、多开票抬头、订单费用入口与发票快照页面可完整访问', async ({
  page,
}) => {
  await page.goto('/user/login');
  await page
    .getByPlaceholder('用户名 / 邮箱')
    .fill(requiredEnvironment('BOOTSTRAP_ADMIN_USERNAME'));
  await page
    .getByPlaceholder('请输入密码')
    .fill(requiredEnvironment('BOOTSTRAP_ADMIN_PASSWORD'));
  await page.locator('button[type="submit"]').click();
  await expect(page).not.toHaveURL(/\/user\/login/, { timeout: 15_000 });

  await page.goto('/finance/bills');
  await expect(
    page.getByTestId('pro-table').getByText('账单管理', { exact: true }),
  ).toBeVisible();
  await page.getByRole('button', { name: '生成账单' }).click();
  await expect(page.getByText('费用批量转账单', { exact: true })).toBeVisible();
  await expect(page.getByText('选择费用', { exact: true })).toBeVisible();
  await expect(page.getByText('拆单策略', { exact: true })).toBeVisible();
  await expect(page.getByText('账单资料', { exact: true })).toBeVisible();
  await page
    .getByRole('dialog', { name: '费用批量转账单' })
    .getByRole('button', { name: /取\s*消/ })
    .click();

  const orders = await page.evaluate(async () => {
    const response = await fetch('/api/v1/orders?page=1&pageSize=100');
    return response.json();
  });
  const acceptanceOrder = orders.data?.find(
    (item: { customerReferenceNo?: string; customerId?: string }) =>
      item.customerReferenceNo?.startsWith('ACC-FIN-'),
  );
  expect(acceptanceOrder?.id).toBeTruthy();
  expect(acceptanceOrder?.customerId).toBeTruthy();
  await page.goto(`/orders/sea-export/${acceptanceOrder.id}/fees`);
  await expect(
    page.getByText('费用录入', { exact: true }).first(),
  ).toBeVisible();
  await expect(page.getByText('已进账单').first()).toBeVisible();
  await expect(
    page.getByRole('button', { name: /生成账单（0）/ }).first(),
  ).toBeDisabled();

  const customer = await page.evaluate(async (customerId) => {
    const response = await fetch(`/api/v1/partners/${customerId}`);
    return response.json();
  }, acceptanceOrder.customerId);
  await page.goto('/partners/customers');
  const customerRow = page
    .locator('.ant-table-tbody > tr.ant-table-row')
    .filter({ hasText: customer.data.legalName })
    .first();
  await expect(customerRow).toBeVisible();
  await customerRow.getByRole('button', { name: '账户/合同' }).click();
  const partnerDrawer = page.getByRole('dialog', {
    name: `往来商务档案：${customer.data.legalName}`,
  });
  await expect(
    partnerDrawer.getByText('开票抬头', { exact: true }),
  ).toBeVisible();
  await expect(
    partnerDrawer.locator('.ant-table-tbody > tr.ant-table-row').nth(1),
  ).toBeVisible();
  await expect(
    partnerDrawer.locator('.ant-tag').filter({ hasText: /^默认$/ }).first(),
  ).toBeVisible();

  await page.goto('/finance/invoices');
  await expect(
    page.getByTestId('pro-table').getByText('开票记录', { exact: true }),
  ).toBeVisible();
  const firstRow = page.locator('.ant-table-tbody > tr.ant-table-row').first();
  await expect(firstRow).toBeVisible();
  await firstRow.getByText('详情', { exact: true }).click();
  await expect(page.getByText(/开票详情/)).toBeVisible();
  await expect(page.getByText('发票抬头', { exact: true })).toBeVisible();
  await expect(page.getByText('纳税人识别号', { exact: true })).toBeVisible();
  await expect(page.getByText('开票项目', { exact: true })).toBeVisible();
  await expect(page.getByText('来源行数', { exact: true })).toBeVisible();
});
