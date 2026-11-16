import { writeFile } from 'node:fs/promises';
import {
  type APIRequestContext,
  expect,
  type Page,
  type TestInfo,
  test,
} from '@playwright/test';

type AirTarget = 'MAWB' | 'HAWB';
type Ticket = {
  order: API.Order;
  kind: 'air-export' | 'air-import';
  mode: 'DIRECT' | 'HOUSE';
  code: 'AE' | 'AI';
  issuer: string;
};
type PdfEvidence = {
  file: string;
  target: AirTarget;
  issuer: string;
  outputKind: string;
  version: string;
  markers: string[];
  multiPage: boolean;
  tokens: string[];
  selfCompany: string;
};

function requiredEnvironment(name: string) {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`缺少环境变量 ${name}`);
  return value;
}

async function api<T>(
  request: APIRequestContext,
  path: string,
  method = 'GET',
  data?: unknown,
): Promise<T> {
  const response = await request.fetch(path, { method, data });
  const text = await response.text();
  const body = text ? JSON.parse(text) : {};
  expect(
    response.ok(),
    `${method} ${path} HTTP ${response.status()}: ${body.message}`,
  ).toBeTruthy();
  expect(body.success, `${method} ${path}: ${body.message}`).not.toBe(false);
  return body.data as T;
}

function masterNumber(value: number) {
  const serial = String(value).slice(-7).padStart(7, '0');
  return `990${serial}${Number(serial) % 7}`;
}

function content(
  marker: string,
  issuer: string,
  long: boolean,
  arranged: boolean,
): API.AirWaybillContent {
  const lines = (prefix: string, count: number) =>
    `${Array.from({ length: count }, (_, index) => `${prefix} ${String(index + 1).padStart(3, '0')} 中文地址及货物完整保留 ACCEPTANCE`).join('\n')}\n${prefix}_LAST`;
  return {
    issuedBy: issuer,
    shipperText: long
      ? lines(`${marker}_SHIPPER`, 25)
      : `${marker}_SHIPPER 普通发货人`,
    consigneeText: long
      ? lines(`${marker}_CONSIGNEE`, 20)
      : `${marker}_CONSIGNEE 普通收货人`,
    agentText: `${marker}_AGENT`,
    shipperAccount: 'ACCOUNT-0',
    consigneeAccount: '',
    departureText: 'XAA / 隔离起运机场',
    destinationText: 'XAB / 隔离目的机场',
    flightText: 'QX101',
    flightDate: '2026-11-18',
    currency: 'USD',
    weightPayment: 'PPD',
    otherPayment: 'COLL',
    carriageValue:
      marker === 'AE_DIRECT_M'
        ? { kind: 'AMOUNT', amount: '0' }
        : { kind: 'NVD' },
    customsValue:
      marker === 'AI_HOUSE_H'
        ? { kind: 'AMOUNT', amount: '123.45' }
        : { kind: 'NCV' },
    insuranceValue: { kind: 'XXX' },
    accountingInformation: `${marker}_ACCOUNTING`,
    handlingInformation: `${marker}_HANDLING`,
    marks: long ? lines(`${marker}_MARKS`, 20) : `${marker}_MARKS`,
    goodsDescription: long
      ? lines(`${marker}_GOODS`, 45)
      : `${marker}_GOODS 普通运单品名`,
    asArranged: arranged,
    rateLines: Array.from({ length: long ? 8 : 1 }, (_, index) => ({
      pieces: index === 0 ? '0' : '2',
      grossWeightKg: '12.345',
      rateClass: 'Q',
      commodityItemNo: `${marker}_RATE_${index + 1}`,
      chargeableWeightKg: '15.5',
      rate: '6.7183',
      amount: index === 0 ? '0' : '123.45',
      goodsDescription: `${marker}_RATE_${index + 1}_LAST 中文运价行`,
    })),
    otherCharges: Array.from({ length: long ? 8 : 1 }, (_, index) => ({
      name: `${marker}_OTHER_${index + 1}_LAST`,
      billingUnit: 'KG',
      quantity: '1.25',
      currency: 'USD',
      unitPrice: '2.5',
      amount: index === 0 ? '0' : '3.125',
      dueTo: index % 2 ? 'CARRIER' : 'AGENT',
      payment: index % 2 ? 'COLL' : 'PPD',
    })),
    prepaid: { valuationCharge: '0', tax: '0' },
    collect: { valuationCharge: '1.25', tax: '0' },
    destinationCurrency: 'USD',
    currencyConversionRate: '6.7183',
    chargesAtDestination: '0',
    issueDate: '2026-11-18',
    issuePlace: 'SHANGHAI',
    remarks: `${marker}_SAVED_REMARKS`,
  };
}

async function openSection(page: Page, label: string) {
  const trigger = page.getByRole('button', { name: label, exact: true });
  if ((await trigger.getAttribute('aria-expanded')) !== 'true')
    await trigger.click();
}

async function saveBody(page: Page, target: AirTarget) {
  if (target === 'MAWB')
    await page
      .getByLabel('修改原因', { exact: true })
      .fill('隔离浏览器验收：保存主运单正文');
  await page
    .getByRole('button', { name: /保存正文/ })
    .first()
    .click();
  if (target === 'MAWB')
    await page
      .getByRole('dialog', { name: '确认主单正文共享修改', exact: true })
      .getByRole('button', { name: '确认影响并保存', exact: true })
      .click();
}

async function expectUnsavedOutputBlocked(page: Page) {
  const button = page
    .getByRole('button', { name: /预览\s*\/\s*下载\s*\/\s*打印/ })
    .first();
  if (await button.isEnabled()) await button.click();
  else await expect(button).toBeDisabled();
  await expect(page.locator('iframe[title$="PDF 预览"]')).toHaveCount(0);
}

async function readTicket(request: APIRequestContext, ticket: Ticket) {
  return api<API.Order>(request, `/api/v1/orders/${ticket.order.id}`);
}

function bodyOf(order: API.Order, target: AirTarget) {
  return target === 'MAWB'
    ? order.airExport?.master?.content?.waybillContent
    : order.airExport?.house?.content;
}

function versionOf(order: API.Order, target: AirTarget) {
  return target === 'MAWB'
    ? order.airExport?.master?.version
    : order.airExport?.house?.version;
}

async function output(page: Page, info: TestInfo, file: string) {
  await page
    .getByRole('button', { name: /预览\s*\/\s*下载\s*\/\s*打印/ })
    .first()
    .click();
  const preview = page
    .getByRole('dialog')
    .filter({ has: page.locator('iframe[title$="PDF 预览"]') });
  await expect(preview.locator('iframe')).toBeVisible({ timeout: 60_000 });
  const downloadLink = preview.getByRole('link', { name: /下载 PDF/ });
  const url = await preview.locator('iframe').getAttribute('src');
  expect(url).toMatch(/^blob:/);
  expect(await downloadLink.getAttribute('href')).toBe(url);
  expect(
    await preview.getByRole('link', { name: /打开打印/ }).getAttribute('href'),
  ).toBe(url);
  const pending = page.waitForEvent('download');
  await downloadLink.click();
  const download = await pending;
  await download.saveAs(info.outputPath(file));
  await page.screenshot({
    path: info.outputPath(file.replace('.pdf', '-preview.png')),
    fullPage: true,
  });
  await preview.getByRole('button', { name: '关闭', exact: true }).click();
}

// 真实数据库和请求，不伪造业务响应；拦截仅用于延迟本次真实保存，形成可重复的并发时序。
test('AE与AI适用主分运单制作、保存快照、真实PDF与生产按需加载', async ({
  page,
  request,
}, info) => {
  test.setTimeout(720_000);
  page.setDefaultTimeout(15_000);
  const pageErrors: string[] = [];
  page.on('pageerror', (error) => pageErrors.push(error.message));
  const stamp = Date.now();
  const companyNameEn = 'ISOLATED REAL AIR COMPANY';
  const evidence: PdfEvidence[] = [];
  await api(request, '/api/v1/auth/login', 'POST', {
    username: requiredEnvironment('BOOTSTRAP_ADMIN_USERNAME'),
    password: requiredEnvironment('BOOTSTRAP_ADMIN_PASSWORD'),
  });
  const admin = await api<API.CurrentUser>(request, '/api/v1/auth/me');
  const company = await api<API.AdminOrganization>(
    request,
    '/api/v1/admin/organizations',
    'POST',
    {
      code: `AWB_BROWSER_${stamp}`,
      name: `航空单证隔离验收公司${stamp}`,
      nameEn: companyNameEn,
      kind: 2,
      baseCurrency: 'CNY',
    },
  );
  const airline = await api<API.Airline>(
    request,
    '/api/v1/master-data/airlines',
    'POST',
    {
      iataCode: 'QX',
      awbPrefix: '990',
      nameZh: '隔离航空单证航司',
      nameEn: 'ISOLATED CARRIER',
      countryCode: 'CN',
    },
  );
  const airports: API.Airport[] = [];
  for (const [iataCode, nameZh] of [
    ['XAA', '隔离起运机场'],
    ['XAB', '隔离目的机场'],
  ])
    airports.push(
      await api<API.Airport>(request, '/api/v1/master-data/airports', 'POST', {
        iataCode,
        nameZh,
        nameEn: `ISOLATED ${iataCode}`,
        cityNameZh: '隔离城市',
        countryCode: 'CN',
      }),
    );
  const shippingLine = await api<API.ShippingLine>(
    request,
    '/api/v1/master-data/shipping-lines',
    'POST',
    {
      scacCode: 'AWBX',
      nameZh: '隔离HBL相邻船司',
      nameEn: 'ISOLATED HBL CARRIER',
      countryCode: 'CN',
    },
  );
  await api(request, '/api/v1/auth/switch-organization', 'POST', {
    organizationId: company.id,
  });
  const customer = await api<API.Partner>(request, '/api/v1/partners', 'POST', {
    legalName: `单证隔离验收客户${stamp}`,
    roles: [{ type: 1, enabled: true }],
    profile: {
      nameEn: 'ISOLATED DOCUMENT CUSTOMER',
      countryCode: 'CN',
      businessTypes: [1, 2, 3, 4],
    },
    assignments: [3, 4].map((role) => ({
      role,
      userId: admin.id,
      organizationId: company.id,
    })),
  });
  const tickets: Ticket[] = [];
  for (const [code, type, kind] of [
    ['AE', 3, 'air-export'],
    ['AI', 4, 'air-import'],
  ] as const)
    for (const mode of ['DIRECT', 'HOUSE'] as const) {
      const marker = `${code}_${mode}`;
      const issuer = `EXTERNAL ISSUER ${marker}`;
      const order = await api<API.Order>(request, '/api/v1/orders', 'POST', {
        customerId: customer.id,
        businessType: type,
        tradeDirection: code === 'AE' ? 1 : 2,
        paymentTerm: 1,
        goodsDescription: `${marker} 单证验收货物`,
        totalPackages: 1,
        totalGrossWeightKg: 100,
        totalVolumeCbm: 1.2,
        personnelAssignments: [2, 3, 4].map((role) => ({
          role,
          userId: admin.id,
        })),
        airExport: {
          documentMode: mode,
          master: {
            masterNo: masterNumber(stamp + tickets.length),
            airlineId: airline.id,
            originAirportId: airports[0].id,
            destinationAirportId: airports[1].id,
            flightNo: 'QX101',
            etd: '2026-11-18T00:00:00Z',
            eta: '2026-11-18T10:00:00Z',
            waybillContent: content(
              `${marker}_M`,
              issuer,
              false,
              code === 'AI',
            ),
          },
          ...(mode === 'HOUSE'
            ? {
                house: {
                  houseNo: `${marker}-${stamp}`,
                  status: 'DRAFT',
                  content: content(
                    `${marker}_H`,
                    code === 'AI' ? issuer : 'SAVED AE HOUSE ISSUER TEXT',
                    code === 'AE',
                    code === 'AI',
                  ),
                },
              }
            : {}),
        },
      });
      tickets.push({ order, kind, mode, code, issuer });
    }
  const hbl = await api<API.Order>(request, '/api/v1/orders', 'POST', {
    customerId: customer.id,
    businessType: 1,
    tradeDirection: 1,
    paymentTerm: 1,
    shipmentType: 2,
    goodsDescription: 'HBL相邻输出',
    shippingLineId: shippingLine.id,
    seaMasterBill: { masterNo: `AWBHBL${stamp}` },
    seaDocument: {
      documentStructure: 3,
      houseBill: {
        houseNo: `AWBHBL-${stamp}`,
        issuerSource: 1,
        content: {
          shipperText: 'HBL SHIPPER',
          consigneeText: 'HBL CONSIGNEE',
          goodsDescriptionText: 'HBL原输出相邻检查',
        },
      },
    },
    etd: '2026-11-18T00:00:00Z',
    eta: '2026-11-19T00:00:00Z',
    personnelAssignments: [2, 3, 4].map((role) => ({ role, userId: admin.id })),
  });

  await page.goto('/user/login');
  await page.getByText('账号登录', { exact: true }).click();
  await page
    .getByPlaceholder('钉钉手机号 / 本地账号')
    .fill(requiredEnvironment('BOOTSTRAP_ADMIN_USERNAME'));
  await page.getByRole('button', { name: '下一步', exact: true }).click();
  await page
    .getByPlaceholder('请输入密码')
    .fill(requiredEnvironment('BOOTSTRAP_ADMIN_PASSWORD'));
  await page.locator('button[type="submit"]').click();
  await expect(
    page.getByRole('heading', { name: '选择进入的组织' }),
  ).toBeVisible();
  await page.getByRole('button', { name: `进入${company.name}` }).click();
  await expect(page).not.toHaveURL(/\/user\/login/);

  for (const ticket of tickets) {
    await test.step(`${ticket.code} ${ticket.mode}适用主分保存及真实输出`, async () => {
      const original = await readTicket(request, ticket);
      if (ticket.mode === 'DIRECT') {
        await page.goto(`/orders/${ticket.kind}/${ticket.order.id}`);
        await expect(
          page.getByText('直单：本票使用主运单，无 HAWB', { exact: true }),
        ).toBeVisible({ timeout: 60_000 });
        expect(original.airExport?.house).toBeUndefined();
        await page.getByRole('button', { name: /更多操作/ }).click();
        await expect(
          page.getByRole('menuitem', { name: /HAWB.*制作/ }),
        ).toHaveCount(0);
        await page.keyboard.press('Escape');
      }
      for (const target of ticket.mode === 'HOUSE'
        ? (['MAWB', 'HAWB'] as const)
        : (['MAWB'] as const)) {
        await page.goto(
          `/orders/${ticket.kind}/${ticket.order.id}/awb/${target}`,
        );
        await openSection(page, '签发与备注');
        const remark = `${ticket.code}_${ticket.mode}_${target}_UI_SAVED`;
        await page.getByLabel('单证备注', { exact: true }).fill(remark);
        const save = page.waitForResponse(
          (item) =>
            item.request().method() === 'PUT' &&
            (target === 'MAWB'
              ? item
                  .url()
                  .endsWith(
                    `/api/v1/air-export/master-waybills/${ticket.order.airExport?.master?.id}`,
                  )
              : item.url().endsWith(`/api/v1/orders/${ticket.order.id}`)),
        );
        await saveBody(page, target);
        const saved = await save;
        expect(saved.ok(), (await saved.json()).message).toBeTruthy();
        const current = await readTicket(request, ticket);
        expect(bodyOf(current, target)?.remarks).toBe(remark);
        expect(bodyOf(current, target)?.rateLines?.[0]?.amount).toBe('0');
        expect(bodyOf(current, target)?.consigneeAccount ?? '').toBe('');
        expect(bodyOf(current, target)?.rateLines?.[0]?.rate).toBe('6.7183');
        expect(BigInt(versionOf(current, target) || '0')).toBeGreaterThan(
          BigInt(versionOf(original, target) || '0'),
        );
        const other = target === 'MAWB' ? 'HAWB' : 'MAWB';
        if (ticket.mode === 'HOUSE')
          expect(bodyOf(current, other)?.goodsDescription).toBe(
            bodyOf(original, other)?.goodsDescription,
          );
        await page.reload();
        await openSection(page, '签发与备注');
        await expect(page.getByLabel('单证备注', { exact: true })).toHaveValue(
          remark,
        );
        const file = `${ticket.code.toLowerCase()}-${ticket.mode.toLowerCase()}-${target.toLowerCase()}.pdf`;
        await output(page, info, file);
        const long =
          ticket.code === 'AE' && ticket.mode === 'HOUSE' && target === 'HAWB';
        const marker = `${ticket.code}_${ticket.mode}_${target === 'MAWB' ? 'M' : 'H'}`;
        evidence.push({
          file,
          target,
          issuer:
            target === 'HAWB' && ticket.code === 'AE'
              ? companyNameEn
              : ticket.issuer,
          outputKind:
            target === 'HAWB' && ticket.code === 'AE'
              ? 'SELF_ISSUED'
              : 'EXTERNAL_ARCHIVE',
          version: versionOf(current, target) || '',
          selfCompany: companyNameEn,
          multiPage: long,
          tokens: [
            bodyOf(current, target)?.carriageValue?.kind,
            bodyOf(current, target)?.customsValue?.kind,
            bodyOf(current, target)?.insuranceValue?.kind,
          ].filter((value): value is string =>
            Boolean(value && value !== 'AMOUNT'),
          ),
          markers: [
            remark,
            `${marker}_RATE_${long ? 8 : 1}_LAST`,
            `${marker}_OTHER_${long ? 8 : 1}_LAST`,
            ...(long
              ? [
                  `${marker}_SHIPPER_LAST`,
                  `${marker}_CONSIGNEE_LAST`,
                  `${marker}_MARKS_LAST`,
                  `${marker}_GOODS_LAST`,
                ]
              : [`${marker}_GOODS`]),
          ],
        });
        const afterPrint = await readTicket(request, ticket);
        expect(afterPrint.flowStatus).toBe(current.flowStatus);
        expect(afterPrint.version).toBe(current.version);
        expect(afterPrint.airExport?.house?.status).toBe(
          current.airExport?.house?.status,
        );
        expect(afterPrint.airExport?.master?.content?.status).toBe(
          current.airExport?.master?.content?.status,
        );
        expect(
          (await api<API.OrderFee[]>(
            request,
            `/api/v1/orders/${ticket.order.id}/fees`,
          )) ?? [],
        ).toHaveLength(0);
      }
    });
  }

  const representative = tickets.find(
    (item) => item.code === 'AE' && item.mode === 'HOUSE',
  );
  expect(representative).toBeTruthy();
  if (!representative) throw new Error('缺少AE主分单验收票');
  await test.step('保存等待期间的新输入保留且不输出假新稿', async () => {
    await page.goto(
      `/orders/${representative.kind}/${representative.order.id}/awb/HAWB`,
    );
    await openSection(page, '签发与备注');
    const remarks = page.getByLabel('单证备注', { exact: true });
    await remarks.fill('AWB_INFLIGHT_SUBMITTED');
    let release: () => void = () => undefined;
    let intercepted: () => void = () => undefined;
    const gate = new Promise<void>((resolve) => {
      release = resolve;
    });
    const seen = new Promise<void>((resolve) => {
      intercepted = resolve;
    });
    const pattern = `**/api/v1/orders/${representative.order.id}`;
    await page.route(pattern, async (route) => {
      if (route.request().method() === 'PUT') {
        intercepted();
        await gate;
      }
      await route.continue();
    });
    const response = page.waitForResponse(
      (item) =>
        item.url().endsWith(`/api/v1/orders/${representative.order.id}`) &&
        item.request().method() === 'PUT',
    );
    await saveBody(page, 'HAWB');
    await seen;
    await remarks.fill('AWB_INFLIGHT_LATER_INPUT_PRESERVED');
    release();
    expect((await response).ok()).toBeTruthy();
    await page.unroute(pattern);
    await expect(
      page.getByText(
        '保存期间输入已继续修改，当前草稿已保留，请刷新核对后再保存。',
        { exact: true },
      ),
    ).toBeVisible();
    await expect(remarks).toHaveValue('AWB_INFLIGHT_LATER_INPUT_PRESERVED');
    expect(
      bodyOf(await readTicket(request, representative), 'HAWB')?.remarks,
    ).toBe('AWB_INFLIGHT_SUBMITTED');
    await expectUnsavedOutputBlocked(page);
    await page.screenshot({
      path: info.outputPath('awb-inflight-new-input-preserved.png'),
      fullPage: true,
    });
  });

  await test.step('真实409保存保留输入且未保存稿不能输出', async () => {
    page.once('dialog', (dialog) => void dialog.accept());
    await page.goto(
      `/orders/${representative.kind}/${representative.order.id}/awb/HAWB`,
    );
    await openSection(page, '签发与备注');
    const remarks = page.getByLabel('单证备注', { exact: true });
    await remarks.fill('AWB_CONFLICT_INPUT_PRESERVED');
    let release: () => void = () => undefined;
    let intercepted: () => void = () => undefined;
    const gate = new Promise<void>((resolve) => {
      release = resolve;
    });
    const seen = new Promise<void>((resolve) => {
      intercepted = resolve;
    });
    const pattern = `**/api/v1/orders/${representative.order.id}`;
    await page.route(pattern, async (route) => {
      if (route.request().method() === 'PUT') {
        intercepted();
        await gate;
      }
      await route.continue();
    });
    const response = page.waitForResponse(
      (item) =>
        item.url().endsWith(`/api/v1/orders/${representative.order.id}`) &&
        item.request().method() === 'PUT',
    );
    await page
      .getByRole('button', { name: /保存正文/ })
      .first()
      .click();
    await seen;
    const before = await readTicket(request, representative);
    await api(request, `/api/v1/orders/${before.id}`, 'PUT', {
      id: before.id,
      expectedVersion: before.version,
      customerReferenceNo: 'AWB_CONCURRENT_EXTERNAL',
    });
    release();
    expect((await response).status()).toBe(409);
    await page.unroute(pattern);
    await expect(remarks).toHaveValue('AWB_CONFLICT_INPUT_PRESERVED');
    expect(
      bodyOf(await readTicket(request, representative), 'HAWB')?.remarks,
    ).not.toBe('AWB_CONFLICT_INPUT_PRESERVED');
    await expectUnsavedOutputBlocked(page);
    await page.screenshot({
      path: info.outputPath('awb-conflict-input-preserved.png'),
      fullPage: true,
    });
  });

  await test.step('前台15秒与回焦检查保留输入', async () => {
    page.once('dialog', (dialog) => void dialog.accept());
    await page.reload();
    await openSection(page, '签发与备注');
    const remarks = page.getByLabel('单证备注', { exact: true });
    for (const detection of ['polling', 'focus'] as const) {
      if (detection === 'focus') {
        page.once('dialog', (dialog) => void dialog.accept());
        await page.reload();
        await openSection(page, '签发与备注');
      }
      await remarks.fill(`AWB_EXTERNAL_INPUT_${detection}`);
      const current = await readTicket(request, representative);
      await api(request, `/api/v1/orders/${current.id}`, 'PUT', {
        id: current.id,
        expectedVersion: current.version,
        customerReferenceNo: `AWB_EXTERNAL_${detection}`,
      });
      if (detection === 'focus')
        await page.evaluate(() => {
          window.dispatchEvent(new Event('blur'));
          window.dispatchEvent(new Event('focus'));
        });
      await expect(
        page.getByText('这个航空运单已经发生变动，请重新刷新。', {
          exact: true,
        }),
      ).toBeVisible({ timeout: detection === 'polling' ? 25_000 : 10_000 });
      await expect(remarks).toHaveValue(`AWB_EXTERNAL_INPUT_${detection}`);
    }
  });

  await test.step('生产同域静态入口、按需资源与实际AWB/HBL预览', async () => {
    const base = requiredEnvironment('RONCIN_ACCEPTANCE_BASE_URL');
    page.once('dialog', (dialog) => void dialog.accept());
    const resources: string[] = [];
    page.on('request', (item) => resources.push(item.url()));
    await page.goto(
      `${base}/orders/${representative.kind}/${representative.order.id}`,
    );
    await expect(
      page.getByPlaceholder('请输入操作备注', { exact: true }),
    ).toBeAttached({ timeout: 60_000 });
    expect(
      resources.filter((url) =>
        /\.ttf(?:\?|$)|react-pdf|renderer|AwbPdf|HblPdf|pdf-vendor/.test(url),
      ),
    ).toEqual([]);
    const deep = `${base}/orders/${representative.kind}/${representative.order.id}/awb/HAWB`;
    const htmlResponse = await page.request.get(deep);
    expect(htmlResponse.status()).toBe(200);
    expect(htmlResponse.headers()['content-type']).toContain('text/html');
    const asset = (await htmlResponse.text()).match(
      /src="([^"]*assets\/[^"]+\.js)"/,
    );
    expect(asset?.[1]).toBeTruthy();
    const javascript = await page.request.get(
      new URL(asset?.[1] || '', base).href,
    );
    expect(javascript.status()).toBe(200);
    expect(javascript.headers()['content-type']).toMatch(/javascript/);
    const font = await page.request.get(
      `${base}/fonts/roncin-hbl/RoncinHBLSansSC-Regular.ttf`,
    );
    expect(font.status()).toBe(200);
    expect(font.headers()['content-type']).toMatch(/font|octet-stream/);
    expect(
      (
        await page.request.get(`${base}/assets/awb-acceptance-missing.js`)
      ).status(),
    ).toBe(404);
    expect((await page.request.get(`${base}/health/ready`)).status()).toBe(200);
    await page.goto(deep);
    await output(page, info, 'production-ae-house-hawb.pdf');
    await page.goto(`${base}/orders/sea-export/${hbl.id}/hbl`);
    await expect(page.getByLabel('HBL 输出版本')).toBeVisible();
    await page
      .getByRole('button', { name: /预览／打印／下载/ })
      .first()
      .click();
    const preview = page.getByRole('dialog', { name: /HBL 预览/ });
    await expect(preview.locator('iframe')).toBeVisible({ timeout: 60_000 });
    await page.screenshot({
      path: info.outputPath('hbl-production-neighbor-preview.png'),
      fullPage: true,
    });
    await preview.getByRole('button', { name: '关闭', exact: true }).click();
  });

  await writeFile(
    info.outputPath('awb-pdf-manifest.json'),
    JSON.stringify(evidence, null, 2),
    { mode: 0o600 },
  );
  expect(pageErrors).toEqual([]);
});
