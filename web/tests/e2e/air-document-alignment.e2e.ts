import { writeFile } from 'node:fs/promises';
import { expect, type Locator, type Page, test } from '@playwright/test';
import { OrderAllowedAction } from '../../src/enums.generated';
import {
  airOrderIds,
  installAirFixture,
} from './order-pages-usability-air.fixture';
import { installSeaStabilityFixture } from './sea-operation-stability.fixture';

async function box(locator: Locator) {
  await expect(locator).toBeVisible();
  const bounds = await locator.boundingBox();
  if (!bounds) throw new Error('控件缺少可测量边界');
  return bounds;
}
async function checkPair(left: Locator, right: Locator, width: number) {
  const a = await box(left);
  const b = await box(right);
  if (width === 1440) {
    expect(Math.abs(a.y - b.y)).toBeLessThanOrEqual(2);
    expect(b.x).toBeGreaterThan(a.x + a.width);
    expect(Math.abs(a.width - b.width)).toBeLessThanOrEqual(2);
  } else {
    expect(Math.abs(a.x - b.x)).toBeLessThanOrEqual(2);
    expect(b.y).toBeGreaterThan(a.y + a.height);
  }
  return { left: a, right: b };
}
async function airFields(page: Page) {
  const tab = page
    .getByRole('tabpanel')
    .filter({
      has: page.getByRole('textbox', { name: '发货人名称与地址', exact: true }),
    })
    .filter({ visible: true });
  return {
    tab,
    shipper: tab.getByRole('textbox', {
      name: '发货人名称与地址',
      exact: true,
    }),
    consignee: tab.getByRole('textbox', {
      name: '收货人名称与地址',
      exact: true,
    }),
    notify: tab.getByRole('textbox', { name: '通知人', exact: true }),
    agent: tab.getByRole('textbox', { name: '代理名称与地址', exact: true }),
    marks: tab.getByRole('textbox', { name: '唛头', exact: true }),
    goods: tab.getByRole('textbox', { name: '运单品名', exact: true }),
  };
}
for (const width of [1440, 390]) {
  test(`海空单证内部边界对照 ${width}px`, async ({ page }, testInfo) => {
    await page.setViewportSize({ width, height: 1000 });
    const sea = await installSeaStabilityFixture(page);
    await page.goto('/orders/sea-export/new');
    const seaShipper = page
      .getByPlaceholder('请输入发货人英文名称与详细地址')
      .filter({ visible: true });
    const seaConsignee = page
      .getByPlaceholder('请输入收货人名称与地址 (TO ORDER 或具体收货人)')
      .filter({ visible: true });
    const seaNotify = page
      .getByPlaceholder('请输入通知人名称与详细地址 (例如：SAME AS CONSIGNEE)')
      .filter({ visible: true });
    const seaAgent = page
      .locator('textarea[id$="agentText"],textarea[id$="foreignAgentText"]')
      .filter({ visible: true });
    const seaMarks = page
      .locator('textarea[id$="marksText"]')
      .filter({ visible: true });
    const seaGoods = page
      .locator('textarea[id$="goodsDescriptionText"]')
      .filter({ visible: true });
    const seaGeometry = {
      parties: await checkPair(seaShipper, seaConsignee, width),
      notify: await checkPair(seaNotify, seaAgent, width),
      cargo: await checkPair(seaMarks, seaGoods, width),
    };
    const seaCard = seaShipper.locator(
      'xpath=ancestor::*[contains(@class,"ant-card")][1]',
    );
    await seaCard.screenshot({ path: testInfo.outputPath('sea-document.png') });
    await sea.report(testInfo);
    await page.unrouteAll();
    const air = await installAirFixture(page, 'AE');
    await page.goto(`/orders/air-export/${airOrderIds.ae}`);
    // 复用已截获的合成详情，开启其编辑能力；此fetch仍由同一白名单夹具处理。
    const response = await page.evaluate(async (id) => {
      const result = await fetch(`/api/v1/orders/${id}`);
      return result.json() as Promise<{ success: boolean; data: API.Order }>;
    }, airOrderIds.ae);
    response.data.allowedActions = [
      OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT,
    ];
    await page.route(`**/api/v1/orders/${airOrderIds.ae}`, async (route) => {
      if (route.request().method() !== 'GET') return route.fallback();
      return route.fulfill({ json: response });
    });
    await page.reload();
    const fields = await airFields(page);
    await expect(fields.shipper).toBeEditable();
    const airGeometry = {
      parties: await checkPair(fields.shipper, fields.consignee, width),
      notify: await checkPair(fields.notify, fields.agent, width),
      cargo: await checkPair(fields.marks, fields.goods, width),
    };
    await fields.tab.screenshot({
      path: testInfo.outputPath('air-house-document.png'),
    });
    expect(await fields.shipper.getAttribute('rows')).toBe('4');
    expect(await fields.goods.getAttribute('rows')).toBe('5');
    expect(
      Math.abs(
        airGeometry.parties.left.height - seaGeometry.parties.left.height,
      ),
    ).toBeLessThanOrEqual(3);
    expect(
      Math.abs(airGeometry.cargo.left.height - seaGeometry.cargo.left.height),
    ).toBeLessThanOrEqual(3);
    await page.getByRole('tab', { name: '主运单 MAWB', exact: true }).click();
    const read = await airFields(page);
    await expect(read.shipper).toHaveAttribute('readonly', '');
    const readGeometry = await checkPair(read.shipper, read.consignee, width);
    expect(
      Math.abs(readGeometry.left.height - airGeometry.parties.left.height),
    ).toBeLessThanOrEqual(1);
    expect(
      Math.abs(readGeometry.left.width - airGeometry.parties.left.width),
    ).toBeLessThanOrEqual(1);
    await read.tab.screenshot({
      path: testInfo.outputPath('air-master-document.png'),
    });
    await testInfo.attach('海空单证实测边界', {
      body: JSON.stringify({ seaGeometry, airGeometry, readGeometry }, null, 2),
      contentType: 'application/json',
    });
    await writeFile(
      testInfo.outputPath('geometry.json'),
      JSON.stringify({ seaGeometry, airGeometry, readGeometry }, null, 2),
    );
    await air.report(testInfo);
  });
}

test('航空通知人切换、低频展开及显式带入触发离开保护', async ({
  page,
}, testInfo) => {
  const state = await installAirFixture(page, 'AE');
  // 详情开启真实编辑能力；仅覆写该单读取，不把合成字段写入数据库。
  await page.route(`**/api/v1/orders/${airOrderIds.ae}`, async (route) => {
    if (route.request().method() !== 'GET') return route.fallback();
    return route.fulfill({
      json: {
        success: true,
        data: {
          id: airOrderIds.ae,
          orderNo: 'AE-ALIGN',
          organizationId: 'air-acceptance-org',
          businessType: 3,
          version: '1',
          allowedActions: [OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT],
          customerId: 'air-acceptance-customer',
          goodsDescription: '订单货描带入测试',
          airExport: {
            documentMode: 'HOUSE',
            linkId: 'link-AE',
            linkVersion: '1',
            active: true,
            master: {
              id: 'master-ae',
              organizationId: 'air-acceptance-org',
              businessType: 'AE',
              version: '1',
              content: {
                masterNo: '78112345675',
                flightNo: 'CA123',
                airlineId: 'airline-1',
                originAirportId: 'airport-pvg',
                destinationAirportId: 'airport-nrt',
                waybillContent: { shipperText: '只读主单发货人' },
              },
            },
            house: {
              id: 'house-AE',
              houseNo: 'AEHAWB001',
              version: '1',
              status: 'DRAFT',
              content: {
                notifyPartyText: '通知人初值',
                secondNotifyPartyText: '第二通知人初值',
              },
            },
          },
        },
      },
    });
  });
  await page.route('**/api/v1/partners/air-acceptance-customer', (route) =>
    route.fulfill({
      json: {
        success: true,
        data: {
          id: 'air-acceptance-customer',
          organizationId: 'air-acceptance-org',
          legalName: '合成客户',
          profile: { nameEn: 'AIR CUSTOMER LTD', addressEn: '100 TEST ROAD' },
        },
      },
    }),
  );
  await page.goto(`/orders/air-export/${airOrderIds.ae}`);
  const fields = await airFields(page);
  await fields.tab.getByRole('button', { name: /从委托客户带入/ }).click();
  await expect(fields.shipper).toHaveValue('AIR CUSTOMER LTD\n100 TEST ROAD');
  await page.getByRole('button', { name: '返回列表', exact: true }).click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible();
  await expect(page).toHaveURL(new RegExp(airOrderIds.ae));
  await page.screenshot({
    path: testInfo.outputPath('import-leave-guard.png'),
  });
  await dialog.getByRole('button', { name: '继续编辑', exact: true }).click();
  await expect(fields.notify).toHaveValue('通知人初值');
  await fields.tab.getByText('第二通知人 · 已填写', { exact: true }).click();
  const second = fields.tab.getByRole('textbox', {
    name: '第二通知人',
    exact: true,
  });
  await expect(second).toHaveValue('第二通知人初值');
  await second.fill('第二通知人编辑后');
  await fields.tab.getByText('通知人', { exact: true }).click();
  await expect(fields.notify).toHaveValue('通知人初值');
  await fields.tab.getByText('第二通知人 · 已填写', { exact: true }).click();
  await expect(second).toHaveValue('第二通知人编辑后');
  await fields.tab
    .getByRole('button', { name: '更多运单资料', exact: true })
    .click();
  await expect(
    fields.tab.getByRole('button', { name: '路线与航班印字', exact: true }),
  ).toBeVisible();
  await state.report(testInfo);
});
