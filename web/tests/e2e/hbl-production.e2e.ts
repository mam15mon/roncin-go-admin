import { expect, type Page, type Route, test } from '@playwright/test';

const orderId = 'hbl-browser-order';
const houseId = 'hbl-browser-house';
const path = `/orders/sea-export/${orderId}/hbl`;

/** 业务 API 用独立夹具，浏览器、字体和 PDF 渲染器使用真实生产实现。 */
async function installFixture(
  page: Page,
  options: { locked?: boolean; initialPrintDetails?: boolean } = {},
) {
  const writes: Record<string, unknown>[] = [];
  const pageErrors: string[] = [];
  const unexpected: string[] = [];
  let houseVersion = 1;
  let linkVersion = 1;
  let content: API.SeaBillContent = {
    shipperText: '制单验收发货人 / HBL TEST SHIPPER',
    consigneeText: 'TO ORDER',
    notifyPartyText: '制单验收通知人 / HBL TEST NOTIFY',
    goodsDescriptionText: '测试阀门 / TEST VALVES\nGOODS-END-MARKER',
    marksText: 'HBL-TEST-MARKS',
    packageCount: 11,
    packageUnit: 'CASES',
    grossWeightKg: 10918,
    volumeCbm: 21.56,
    freightTerms: '预付',
    transportTerms: 'CY-CY',
  };
  let printDetails: API.SeaHouseBillPrintDetails | undefined =
    options.initialPrintDetails === false
      ? undefined
      : {
          bookingNoText: 'BOOKING-BROWSER-001',
          vesselVoyageText: 'TEST VESSEL / V001',
          portOfLoadingText: 'SHANGHAI, CHINA',
          portOfDischargeText: 'SINGAPORE',
          placeOfDeliveryText: 'SINGAPORE',
          originalCount: 3,
          shippedOnBoardDate: '2026-11-04',
          issuePlaceText: 'BEIJING',
          issueDate: '2026-11-04',
          containersText: 'ABCD1234567 / SEAL TEST001 / 20GP',
        };
  const house = (): API.SeaHouseBill => ({
    id: houseId,
    orderId,
    organizationId: 'hbl-browser-org',
    houseNo: 'HBL-BROWSER-20261104',
    issuerSource: 1,
    issuerOrganizationId: 'hbl-browser-org',
    issuerOrganizationName: '融迅北京制单验收',
    status: 1,
    version: String(houseVersion),
    printTemplate: 'RONCIN_BEIJING',
    content,
    printDetails,
  });
  page.on('pageerror', (error) => pageErrors.push(error.message));
  await page.route('**/api/**', async (route: Route) => {
    const url = new URL(route.request().url());
    const send = (data: unknown) =>
      route.fulfill({ json: { success: true, code: 0, data } });
    if (url.pathname === '/api/v1/auth/me') {
      const permissions = [
        'business.order.se.read',
        'business.order.se.update',
      ];
      return send({
        id: 'hbl-browser-user',
        username: 'hbl-browser',
        displayName: 'HBL 制单验收',
        currentOrganization: {
          id: 'hbl-browser-org',
          code: 'BJ',
          name: '融迅北京制单验收',
          kind: 2,
        },
        permissions,
        permissionCapabilities: permissions.map((key) => ({
          key,
          dataScope: 'all',
        })),
      });
    }
    if (url.pathname === `/api/v1/orders/${orderId}`)
      return send({
        id: orderId,
        organizationId: 'hbl-browser-org',
        organizationName: '融迅北京制单验收',
        orderNo: 'SE-HBL-BROWSER',
        businessType: 1,
        version: '1',
        terminationStatus: 1,
        bookingNo: 'ORDER-BOOKING-001',
        vesselVoyage: 'SAVED VESSEL / V001',
        originLocationName: 'SHANGHAI',
        dischargeLocationName: 'SINGAPORE',
        destinationLocationName: 'SINGAPORE',
      });
    if (url.pathname === `/api/v1/orders/${orderId}/sea-documents`)
      return send({
        orderId,
        documentStructure: 3,
        linkVersion: String(linkVersion),
        allowedActions: [4],
        masterBill: { id: 'hbl-browser-master', version: '1' },
        houseBill: house(),
      });
    if (url.pathname === `/api/v1/orders/${orderId}/lock-state`)
      return send({
        orderId,
        isLocked: options.locked ?? false,
        businessType: 1,
      });
    if (
      url.pathname ===
        `/api/v1/orders/${orderId}/sea-documents/house-bills/${houseId}` &&
      route.request().method() === 'PUT'
    ) {
      const body = route.request().postDataJSON();
      writes.push(body);
      expect(body.expectedVersion).toBe(String(houseVersion));
      expect(body.expectedLinkVersion).toBe(String(linkVersion));
      expect(body.houseBill.id).toBe(houseId);
      content = body.houseBill.content;
      printDetails = body.houseBill.printDetails;
      houseVersion += 1;
      linkVersion += 1;
      return send(house());
    }
    unexpected.push(url.pathname);
    return route.fulfill({
      status: 404,
      json: { success: false, message: '验收未声明的请求' },
    });
  });
  return { writes, pageErrors, unexpected };
}

async function openPreview(page: Page) {
  await page
    .getByRole('button', { name: /预览／打印／下载/ })
    .first()
    .click();
  const dialog = page.getByRole('dialog', { name: /HBL 预览/ });
  await expect(dialog).toBeVisible();
  const download = dialog.getByRole('link', { name: /下载 PDF/ });
  await expect(download).toHaveAttribute('href', /^blob:/, { timeout: 30_000 });
  const blobUrl = await download.getAttribute('href');
  expect(blobUrl).toBeTruthy();
  await expect(dialog.locator('iframe')).toHaveAttribute(
    'src',
    blobUrl as string,
  );
  await expect(dialog.getByRole('link', { name: /打开打印/ })).toHaveAttribute(
    'href',
    blobUrl as string,
  );
  return { dialog, download };
}

test('同一 HBL 保存后可生成四种真实 PDF，预览、下载和打印共用文件', async ({
  page,
}, testInfo) => {
  test.setTimeout(120_000);
  const state = await installFixture(page);
  const fontRequests: string[] = [];
  page.on('request', (request) => {
    if (new URL(request.url()).pathname.includes('/fonts/'))
      fontRequests.push(request.url());
  });
  await page.goto(path);
  await expect(
    page.getByText('HBL 制作', { exact: true }).first(),
  ).toBeVisible();
  const footer = page.locator('.roncin-sticky-footer-bar');
  await expect(footer).toBeVisible();
  const footerWidth = (await footer.boundingBox())?.width ?? 0;
  const sectionWidth =
    (await page.locator('.roncin-section-card').first().boundingBox())?.width ??
    0;
  expect(footerWidth).toBeGreaterThan(0);
  expect(Math.abs(footerWidth - sectionWidth)).toBeLessThanOrEqual(2);
  const description = page.getByLabel('货物描述 (Description of Goods)', {
    exact: true,
  });
  await description.fill('已保存货描 / SAVED GOODS\nSAVED-GOODS-END');
  await page
    .getByRole('button', { name: /保存制作资料/ })
    .first()
    .click();
  await expect(page.getByText('HBL 制作资料已保存')).toBeVisible();
  expect(state.writes).toHaveLength(1);
  expect(state.writes[0].houseBill).toMatchObject({
    id: houseId,
    content: {
      goodsDescriptionText: '已保存货描 / SAVED GOODS\nSAVED-GOODS-END',
    },
    printDetails: { originalCount: 3, issueDate: '2026-11-04' },
  });
  for (const variant of ['草稿', '正本', '电放', '副本']) {
    await page.getByText(variant, { exact: true }).click();
    const { dialog, download } = await openPreview(page);
    const pending = page.waitForEvent('download');
    await download.click();
    const file = await pending;
    expect(file.suggestedFilename()).toContain('HBL-BROWSER-20261104');
    expect(file.suggestedFilename()).toMatch(/\.pdf$/);
    await file.saveAs(testInfo.outputPath(`${variant}.pdf`));
    await dialog.getByRole('button', { name: '关闭', exact: true }).click();
    await expect(dialog).not.toBeVisible();
  }
  expect(fontRequests.length).toBeGreaterThan(0);
  expect(
    fontRequests.every(
      (url) => new URL(url).origin === new URL(page.url()).origin,
    ),
  ).toBe(true);
  expect(state.pageErrors).toEqual([]);
  expect(state.unexpected).toEqual([]);
});

test('锁定且首次制单的 HBL 不必保存即可输出正本，编辑仍受锁约束', async ({
  page,
}, testInfo) => {
  const state = await installFixture(page, {
    locked: true,
    initialPrintDetails: false,
  });
  await page.goto(path);
  await expect(
    page.getByText('海运出口订单已审核，如需修改请先撤销审核'),
  ).toBeVisible();
  await expect(page.getByText(/测试阀门 \/ TEST VALVES/)).toBeVisible();
  await expect(
    page.getByLabel('货物描述 (Description of Goods)', { exact: true }),
  ).toHaveCount(0);
  await expect(
    page.getByRole('button', { name: /保存制作资料/ }).first(),
  ).toBeDisabled();
  await page.getByText('正本', { exact: true }).click();
  const { dialog, download } = await openPreview(page);
  await expect(dialog).toContainText('正本');
  const pending = page.waitForEvent('download');
  await download.click();
  const file = await pending;
  await file.saveAs(testInfo.outputPath('锁定首次正本.pdf'));
  expect(state.writes).toHaveLength(0);
  expect(state.pageErrors).toEqual([]);
  expect(state.unexpected).toEqual([]);
});
