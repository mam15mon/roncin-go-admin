import { expect, test } from '@playwright/test';

function requiredEnvironment(name: string) {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`缺少环境变量 ${name}`);
  return value;
}

async function login(page: import('@playwright/test').Page) {
  await page.goto('/user/login');
  await page
    .getByPlaceholder('用户名 / 邮箱')
    .fill(requiredEnvironment('BOOTSTRAP_ADMIN_USERNAME'));
  await page
    .getByPlaceholder('请输入密码')
    .fill(requiredEnvironment('BOOTSTRAP_ADMIN_PASSWORD'));
  await page.locator('button[type="submit"]').click();
  await expect(page).not.toHaveURL(/\/user\/login/, { timeout: 15_000 });
}

async function currencyDropdownVisible(
  page: import('@playwright/test').Page,
  index: number,
) {
  return page.evaluate((i) => {
    const selectRoot = document.querySelectorAll(
      '.ant-input-suffix .ant-select',
    )[i];
    return Boolean(
      selectRoot?.className.includes('ant-select-open') &&
        Array.from(document.querySelectorAll('.ant-select-dropdown')).some(
          (el) =>
            !el.className.includes('ant-select-dropdown-hidden') &&
            (el as HTMLElement).offsetParent !== null,
        ),
    );
  }, index);
}

test('货值/保费 suffix 币种下拉在真人点击节奏（按住停顿）下保持打开', async ({
  page,
}) => {
  await login(page);
  await page.goto('/orders/sea-export/new');
  await expect(page.getByRole('button', { name: '创建订单' })).toBeVisible({
    timeout: 15_000,
  });

  for (const [label, index] of [
    ['货值', 0],
    ['保费', 1],
  ] as const) {
    const select = page.locator('.ant-input-suffix .ant-select').nth(index);
    await select.scrollIntoViewIfNeeded();
    const box = await select.boundingBox();
    expect(box, `${label} 币种下拉应可见`).toBeTruthy();
    const { x, y, width, height } = box as { x: number; y: number; width: number; height: number };

    // 模拟真人点击：mousedown 后按住约 200ms 再松开。
    // 修复前：mouseup 的 click 冒泡到外层金额输入框触发 triggerFocus，
    // select 内部输入框失焦，rc-select 延迟关闭刚打开的下拉。
    await page.mouse.move(x + width / 2, y + height / 2);
    await page.mouse.down();
    await page.waitForTimeout(200);
    await page.mouse.up();
    await page.waitForTimeout(600);

    expect(
      await currencyDropdownVisible(page, index),
      `${label} 币种下拉应在按住停顿点击后保持打开`,
    ).toBe(true);

    // 点击页面空白处收起，进入下一个用例
    await page.mouse.click(10, 300);
    await page.waitForTimeout(300);
  }
});
