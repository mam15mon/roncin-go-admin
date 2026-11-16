import { expect, test } from '@playwright/test';
import {
  airOrderIds,
  installAirFixture,
} from './order-pages-usability-air.fixture';

for (const width of [390, 1440]) {
  for (const type of ['AE', 'AI'] as const) {
    for (const [mode, path] of [
      ['新建', `/orders/${type === 'AE' ? 'air-export' : 'air-import'}/new`],
      [
        '详情',
        `/orders/${type === 'AE' ? 'air-export' : 'air-import'}/${type === 'AE' ? airOrderIds.ae : airOrderIds.ai}`,
      ],
    ] as const) {
      test(`空运${type}${mode} ${width}px`, async ({ page }, testInfo) => {
        await page.setViewportSize({ width, height: 1000 });
        const state = await installAirFixture(page, type);
        await page.goto(path);
        await expect(
          page.getByText('业务与客户', { exact: true }),
        ).toBeVisible();
        await expect(
          page.getByText(type === 'AI' ? '承运与运输' : '订舱与运输', {
            exact: true,
          }),
        ).toBeVisible();
        await expect(
          page.getByText('货物与商业', { exact: true }),
        ).toBeVisible();
        await expect(page.getByText('单证信息', { exact: true })).toBeVisible();
        if (mode === '详情') {
          await expect(
            page.getByText('主运单 MAWB', { exact: true }),
          ).toBeVisible();
          await expect(
            page.getByText('分运单 HAWB', { exact: true }),
          ).toBeVisible();
          await expect(
            page.getByText('实际货物与计费重量', { exact: true }),
          ).toBeVisible();
          await expect(
            page.getByText('已确认计费重 KG', { exact: true }),
          ).toBeVisible();
          if (type === 'AI')
            await expect(
              page.getByText('进口服务', { exact: true }),
            ).toBeVisible();
          if (type === 'AE') {
            await page
              .getByRole('button', { name: '修改共同运输资料' })
              .click();
            await expect(
              page.getByText('本次仅保存共同运输资料', { exact: true }),
            ).toBeVisible();
            await page.keyboard.press('Escape');
          }
        }
        const overflow = await page.evaluate(() => ({
          viewport: document.documentElement.clientWidth,
          content: document.documentElement.scrollWidth,
        }));
        expect(overflow.content).toBeLessThanOrEqual(overflow.viewport + 1);
        await page.screenshot({
          path: testInfo.outputPath('layout.png'),
          fullPage: true,
        });
        await state.report(testInfo);
      });
    }
  }
}
