import { expect, type Page, test } from '@playwright/test';

test.use({ viewport: { width: 1440, height: 900 } });

test.beforeEach(async ({ page }) => {
  const organization = {
    id: 'sidebar-company',
    name: '侧栏验收公司',
    kind: 2,
    baseCurrency: 'CNY',
  };
  // 使用真实布局和动画，隔离身份与业务数据，不读写本地数据库。
  await page.route('**/api/**', (route) => {
    const pathname = new URL(route.request().url()).pathname;
    return route.fulfill({
      json:
        pathname === '/api/v1/auth/me'
          ? {
              data: {
                id: 'sidebar-user',
                username: 'sidebar',
                currentOrganization: organization,
                organizations: [organization],
                permissionCapabilities: [
                  'system.platform.access',
                  'business.order.se.read',
                  'business.order.ae.read',
                  'system.finance.bill.read',
                ].map((key) => ({ key, dataScope: 'organization' })),
              },
            }
          : { data: pathname.endsWith('/overview') ? {} : [], total: 0 },
    });
  });
});

async function sampleToggle(page: Page, reverseDuringMotion = false) {
  return page.evaluate(async (reverse) => {
    const sider = document.querySelector<HTMLElement>(
      '[data-testid="pro-layout-sider"]',
    );
    const header = document.querySelector('.ant-pro-layout-header');
    const placeholder = sider?.previousElementSibling;
    const trigger = sider?.querySelector<HTMLElement>(
      '.ant-pro-sider-collapsed-button',
    );
    if (!sider || !header || !placeholder || !trigger)
      throw new Error('侧栏或正文占位未就绪');

    const snapshot = () => ({
      width: sider.getBoundingClientRect().width,
      headerLeft: header.getBoundingClientRect().left,
      placeholderWidth: placeholder.getBoundingClientRect().width,
      icons: [...sider.querySelectorAll('.roncin-sider-menu-icon svg')].map(
        (icon) => {
          const rect = icon.getBoundingClientRect();
          return { x: rect.x + rect.width / 2, y: rect.y + rect.height / 2 };
        },
      ),
      logo: sider.querySelector('img')?.getBoundingClientRect().x,
      itemHeight: sider.querySelector('.ant-menu-item')?.getBoundingClientRect()
        .height,
      popupCount: document.querySelectorAll('.ant-menu-submenu-popup').length,
    });
    const samples = [snapshot()];
    const started = performance.now();
    let reversed = false;
    trigger.click();
    await new Promise<void>((resolve) => {
      const frame = () => {
        const elapsed = performance.now() - started;
        if (reverse && !reversed && elapsed >= 70) {
          trigger.click();
          reversed = true;
        }
        samples.push(snapshot());
        // 逐帧等待真实动画结束；保留中间帧，不能只检查最终展开/收起态。
        if (elapsed < 400) requestAnimationFrame(frame);
        else resolve();
      };
      requestAnimationFrame(frame);
    });
    return samples;
  }, reverseDuringMotion);
}

function expectStableMotion(samples: Awaited<ReturnType<typeof sampleToggle>>) {
  const initial = samples[0];
  expect(initial.icons.length).toBeGreaterThan(1);
  expect(
    samples.some((sample) => sample.width > 49 && sample.width < 207),
  ).toBe(true);
  for (const sample of samples) {
    expect(Math.abs(sample.width - sample.headerLeft)).toBeLessThan(1);
    expect(Math.abs(sample.width - sample.placeholderWidth)).toBeLessThan(1);
    expect(sample.itemHeight).toBe(36);
    expect(sample.logo).toBe(initial.logo);
    expect(sample.icons).toHaveLength(initial.icons.length);
    sample.icons.forEach((icon, index) => {
      expect(Math.abs(icon.x - initial.icons[index].x)).toBeLessThan(0.5);
      expect(Math.abs(icon.y - initial.icons[index].y)).toBeLessThan(0.5);
    });
  }
}

test('展开、收起全过程保持侧栏边界同步与图标位置稳定', async ({ page }) => {
  await page.goto('/welcome');
  const sider = page.getByTestId('pro-layout-sider');
  await expect(sider).toHaveCSS('width', '208px');
  await expect(sider.locator('.roncin-sider-menu-icon').first()).toBeVisible();

  const collapsing = await sampleToggle(page);
  expectStableMotion(collapsing);
  expect(collapsing.at(-1)?.width).toBe(48);
  await expect(sider.locator('.roncin-sider-menu-text').first()).toHaveCSS(
    'opacity',
    '0',
  );
  expect(collapsing.at(-1)?.icons[0].x).toBe(24);

  const expanding = await sampleToggle(page);
  expectStableMotion(expanding);
  expect(expanding.at(-1)?.width).toBe(208);
  await expect(sider.locator('.roncin-sider-menu-text').first()).toHaveCSS(
    'opacity',
    '1',
  );

  const reversed = await sampleToggle(page, true);
  expectStableMotion(reversed);
  expect(reversed.at(-1)?.width).toBe(208);
});

test('折叠态悬停仍展示纯白子菜单并可跳转', async ({ page }) => {
  await page.goto('/welcome');
  await expect(page.getByTestId('pro-layout-sider')).toBeVisible();
  await sampleToggle(page);
  await page
    .locator('.ant-pro-sider .ant-menu-submenu-title')
    .filter({ hasText: '订单管理' })
    .hover();
  const popup = page.locator('.ant-menu-submenu-popup:visible');
  await expect(popup).toBeVisible();
  await expect(popup.getByRole('menu')).toHaveCSS(
    'background-color',
    'rgb(255, 255, 255)',
  );
  await popup.getByRole('link', { name: '海运出口' }).click();
  await expect(page).toHaveURL(/\/orders\/sea-export$/);
});

test('当前订单子菜单已展开时切换侧栏，不自动弹出或残留折叠态浮层', async ({
  page,
}) => {
  await page.goto('/orders/sea-export');
  const sider = page.getByTestId('pro-layout-sider');
  const orders = sider
    .locator('.ant-menu-submenu-title')
    .filter({ hasText: '订单管理' });
  await expect(orders).toHaveAttribute('aria-expanded', 'true');
  await expect(sider.getByRole('link', { name: '海运出口' })).toBeVisible();

  const collapsing = await sampleToggle(page);
  expect(collapsing.every((sample) => sample.popupCount === 0)).toBe(true);
  await orders.hover();
  await expect(page.locator('.ant-menu-submenu-popup:visible')).toBeVisible();
  // 指针离开后退场尚未结束就展开，覆盖截图中的两套子菜单重叠。
  await page.mouse.move(24, 800);
  const expanding = await sampleToggle(page);
  expect(expanding.slice(1).every((sample) => sample.popupCount === 0)).toBe(
    true,
  );
  await expect(sider).toHaveCSS('width', '208px');
  await expect(sider.getByRole('link', { name: '海运出口' })).toBeVisible();
  await expect(sider.locator('.ant-menu-item-selected a')).toHaveAttribute(
    'href',
    '/orders/sea-export',
  );
  await orders.click();
  await expect(sider.getByRole('link', { name: '海运出口' })).not.toBeVisible();
  await orders.click();
  await expect(sider.getByRole('link', { name: '海运出口' })).toBeVisible();
});

test('移动端菜单展开与点击关闭保持正常，顶栏铺满视口', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/welcome');
  const header = page.locator('.ant-pro-layout-header');
  await expect(header).toHaveCSS('left', '0px');
  await expect(header).toHaveCSS('width', '390px');
  await page.locator('.ant-pro-global-header-collapsed-button').click();
  const drawer = page.getByRole('dialog');
  await expect(drawer).toBeVisible();
  await drawer.getByRole('menuitem', { name: '订单管理' }).click();
  await drawer.getByRole('link', { name: '海运出口' }).click();
  await expect(page).toHaveURL(/\/orders\/sea-export$/);
  await expect(drawer).not.toBeVisible();
});
