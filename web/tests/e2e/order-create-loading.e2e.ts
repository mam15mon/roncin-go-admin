import { expect, test } from '@playwright/test';

function requiredEnvironment(name: string) {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`缺少环境变量 ${name}`);
  return value;
}

test('新建订单主数据加载按需过滤机场，且同一组织二次进入时复用缓存无重复请求', async ({
  page,
}) => {
  // 1. 登录
  await page.goto('/user/login');
  await page
    .getByPlaceholder('用户名 / 邮箱')
    .fill(requiredEnvironment('BOOTSTRAP_ADMIN_USERNAME'));
  await page
    .getByPlaceholder('请输入密码')
    .fill(requiredEnvironment('BOOTSTRAP_ADMIN_PASSWORD'));
  await page.locator('button[type="submit"]').click();
  await expect(page).not.toHaveURL(/\/user\/login/, { timeout: 15_000 });

  // 2. 监听请求
  const requestUrls: string[] = [];
  page.on('request', (req) => {
    requestUrls.push(req.url());
  });

  // 3. 首次进入海运新建订单页
  await page.goto('/orders/sea-export/new');
  await expect(page.getByRole('button', { name: '创建订单' })).toBeVisible({
    timeout: 15_000,
  });

  // 断言不发出 airports 请求
  const airportRequests = requestUrls.filter((url) =>
    url.includes('/api/v1/master-data/airports'),
  );
  expect(airportRequests).toHaveLength(0);

  // 记录首次请求数量
  const firstMasterOptions = requestUrls.filter((url) =>
    url.includes('/api/v1/master-data/options'),
  ).length;
  expect(firstMasterOptions).toBeGreaterThanOrEqual(1);

  // 4. 跳转至订单列表页后再二次进入新建订单页
  await page.goto('/orders/sea-export');
  await page.waitForLoadState('networkidle');

  const lengthBeforeSecondEnter = requestUrls.length;

  await page.goto('/orders/sea-export/new');
  await expect(page.getByRole('button', { name: '创建订单' })).toBeVisible({
    timeout: 15_000,
  });

  // 获取二次进入产生的请求
  const secondEnterRequests = requestUrls.slice(lengthBeforeSecondEnter);
  const secondMasterOptions = secondEnterRequests.filter((url) =>
    url.includes('/api/v1/master-data/options'),
  ).length;
  const secondPorts = secondEnterRequests.filter((url) =>
    url.includes('/api/v1/master-data/ports'),
  ).length;
  const secondAirports = secondEnterRequests.filter((url) =>
    url.includes('/api/v1/master-data/airports'),
  ).length;
  const secondPersonnel = secondEnterRequests.filter((url) =>
    url.includes('/api/v1/orders/personnel-options'),
  ).length;

  expect(secondAirports).toBe(0);
  expect(secondMasterOptions).toBe(0);
  expect(secondPorts).toBe(0);
  expect(secondPersonnel).toBe(0);
});
