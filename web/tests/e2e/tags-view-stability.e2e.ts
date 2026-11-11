import { expect, type Page, test } from '@playwright/test';

async function measureSwitch(page: Page, title: string) {
  return page.evaluate(async (targetTitle) => {
    const tabs = [
      ...document.querySelectorAll<HTMLElement>('.roncin-chrome-tab'),
    ];
    const target = tabs.find((tab) => tab.textContent?.includes(targetTitle));
    if (!target) throw new Error('未找到目标页签');
    const snapshot = () =>
      tabs.map((tab) => {
        const rect = tab.getBoundingClientRect();
        return { x: rect.x, width: rect.width };
      });
    const samples = [snapshot()];
    target.click();
    const started = performance.now();
    await new Promise<void>((resolve) => {
      const frame = () => {
        samples.push(snapshot());
        if (performance.now() - started < 300) requestAnimationFrame(frame);
        else resolve();
      };
      requestAnimationFrame(frame);
    });
    return samples;
  }, title);
}

test('订单列表和新建页签切换时位置与宽度保持稳定', async ({ page }) => {
  const organization = { id: 'test-org', name: '验收公司', kind: 2 };
  await page.route('**/api/**', (route) => {
    const path = new URL(route.request().url()).pathname;
    return route.fulfill({
      json: {
        success: true,
        data:
          path === '/api/v1/auth/me'
            ? {
                id: 'test-user',
                username: 'test',
                currentOrganization: organization,
                organizations: [organization],
                permissionCapabilities: [
                  'business.order.se.read',
                  'business.order.se.create',
                ].map((key) => ({ key, dataScope: 'organization' })),
              }
            : [],
        total: 0,
      },
    });
  });
  await page.goto('/orders/sea-export');
  await page.getByText('新增海运出口订单').click();
  await expect(page.getByRole('tab', { name: /SE 新建订单/ })).toHaveAttribute(
    'aria-selected',
    'true',
  );
  for (const title of ['海运出口订单列表', 'SE 新建订单']) {
    const samples = await measureSwitch(page, title);
    for (const sample of samples) {
      sample.forEach((rect, index) => {
        expect(Math.abs(rect.x - samples[0][index].x)).toBeLessThan(0.5);
        expect(Math.abs(rect.width - samples[0][index].width)).toBeLessThan(
          0.5,
        );
      });
    }
  }
});
