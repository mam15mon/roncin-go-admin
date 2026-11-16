import { expect, type Page, test } from '@playwright/test';
import { installExtraFixture } from './order-pages-usability-extra.fixture';

async function assertNoOverflow(page: Page) {
  const widths = await page.evaluate(() => [
    document.documentElement.scrollWidth,
    document.documentElement.clientWidth,
  ]);
  expect(widths[0]).toBeLessThanOrEqual(widths[1] + 1);
}

async function assertImportPosition(page: Page) {
  const service = page.getByText('进口服务', { exact: true });
  await expect(service).toBeVisible();
  const beforeBusiness = await service.evaluate((node) => {
    const business = Array.from(document.querySelectorAll('*')).find(
      (item) => item.textContent === '业务与客户',
    );
    return business
      ? Boolean(
          node.compareDocumentPosition(business) &
            Node.DOCUMENT_POSITION_FOLLOWING,
        )
      : false;
  });
  expect(beforeBusiness).toBe(true);
}

for (const width of [390, 1440]) {
  for (const detail of [false, true]) {
    test(`海运进口${detail ? '详情' : '新建'} ${width}px`, async ({
      page,
    }, info) => {
      await page.setViewportSize({ width, height: 1000 });
      const fixture = await installExtraFixture(page, 'SI');
      await page.goto(`/orders/sea-import/${detail ? fixture.id : 'new'}`);
      await expect(page.getByText('业务与客户', { exact: true })).toBeVisible();
      await expect(page.getByText('承运与运输', { exact: true })).toBeVisible();
      await expect(page.getByText('货物与商业', { exact: true })).toBeVisible();
      if (detail) await assertImportPosition(page);
      else
        await expect(page.getByText('进口服务', { exact: true })).toHaveCount(
          0,
        );
      await assertNoOverflow(page);
      await page.screenshot({
        path: info.outputPath('sea-import-layout.png'),
        fullPage: true,
      });
      await fixture.report(info);
    });
  }
  for (const type of ['AE', 'AI'] as const) {
    test(`航空${type}直单与新建页签切换 ${width}px`, async ({ page }, info) => {
      await page.setViewportSize({ width, height: 1000 });
      const fixture = await installExtraFixture(page, type, true);
      const kind = type === 'AE' ? 'air-export' : 'air-import';
      await page.goto(`/orders/${kind}/${fixture.id}`);
      await expect(
        page.getByRole('tab', { name: '主运单 MAWB' }),
      ).toBeVisible();
      await expect(page.getByRole('tab', { name: '分运单 HAWB' })).toHaveCount(
        0,
      );
      await expect(page.getByText('直单发货人', { exact: true })).toBeVisible();
      if (type === 'AI') await assertImportPosition(page);
      await assertNoOverflow(page);
      await page.screenshot({
        path: info.outputPath('air-direct-layout.png'),
        fullPage: true,
      });
      await page.goto(`/orders/${kind}/new`);
      const house = page.getByRole('tab', { name: '分运单 HAWB' });
      await expect(house).toHaveAttribute('aria-selected', 'true');
      await page
        .getByRole('textbox', { name: '本票分运单号（HAWB）' })
        .fill('TEST-RETAIN-HAWB');
      if (width === 1440) {
        const panelWidth = await page
          .getByRole('tabpanel', { name: '分运单 HAWB' })
          .evaluate((node) => node.getBoundingClientRect().width);
        const fieldWidth = await page
          .getByRole('textbox', { name: '发货人名称与地址' })
          .evaluate((node) => node.getBoundingClientRect().width);
        const cardWidth = await page
          .locator('[data-section-key="airDocuments"]')
          .evaluate((node) => node.getBoundingClientRect().width);
        expect(panelWidth).toBeGreaterThan(cardWidth * 0.9);
        expect(fieldWidth).toBeGreaterThan(panelWidth * 0.4);
        await info.attach('单证字段实际宽度', {
          body: JSON.stringify({
            cardWidth,
            panelWidth,
            fieldWidth,
            ratio: fieldWidth / panelWidth,
          }),
          contentType: 'application/json',
        });
      }
      await page.getByRole('tab', { name: '主运单 MAWB' }).click();
      await expect(
        page.getByRole('tab', { name: '主运单 MAWB' }),
      ).toHaveAttribute('aria-selected', 'true');
      await house.click();
      await expect(
        page.getByRole('textbox', { name: '本票分运单号（HAWB）' }),
      ).toHaveValue('TEST-RETAIN-HAWB');
      await page
        .getByRole('radio', { name: '直单（DIRECT，无 HAWB）' })
        .check();
      await expect(house).toHaveCount(0);
      await expect(
        page.getByRole('tab', { name: '主运单 MAWB' }),
      ).toHaveAttribute('aria-selected', 'true');
      await page.getByRole('radio', { name: '主分单（HOUSE）' }).check();
      await house.click();
      await expect(
        page.getByRole('textbox', { name: '本票分运单号（HAWB）' }),
      ).toHaveValue('TEST-RETAIN-HAWB');
      await assertNoOverflow(page);
      await fixture.report(info);
    });
  }
}
