import { expect, test } from '@playwright/test';
import {
  installSeaStabilityFixture,
  stabilityOrderPath,
} from './sea-operation-stability.fixture';

// 使用真实页面与合成接口验证布局，不写开发业务库。
for (const width of [390, 1024, 1440, 1920]) {
  for (const [mode, path] of [
    ['新建', '/orders/sea-export/new'],
    ['详情', stabilityOrderPath],
  ] as const) {
    test(`海运${mode} ${width}px 字段、操作与页面宽度`, async ({
      page,
    }, testInfo) => {
      await page.setViewportSize({ width, height: 1000 });
      const state = await installSeaStabilityFixture(page);
      await page.goto(path);
      await expect(page.locator('#customerId')).toBeVisible();
      await expect(page.getByLabel('船名', { exact: true })).toBeVisible();
      await expect(page.getByLabel('航次', { exact: true })).toBeVisible();
      await expect(page.getByLabel('起运港', { exact: true })).toBeVisible();
      await expect(page.getByLabel('卸货港', { exact: true })).toBeVisible();
      await expect(
        page.getByText('最终目的港', { exact: true }).first(),
      ).toBeVisible();
      const overflow = await page.evaluate(() => ({
        viewport: document.documentElement.clientWidth,
        content: document.documentElement.scrollWidth,
      }));
      expect(overflow.content).toBeLessThanOrEqual(overflow.viewport + 1);
      await page.screenshot({
        path: testInfo.outputPath('layout.png'),
        fullPage: true,
      });
      expect(state.writes).toEqual([]);
      await state.report(testInfo);
    });
  }
}
