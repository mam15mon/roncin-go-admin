import { randomUUID } from 'node:crypto';
import {
  type APIRequestContext,
  expect,
  type Locator,
  type Page,
  test,
} from '@playwright/test';

function requiredEnvironment(name: string) {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`缺少环境变量 ${name}`);
  return value;
}

async function wire<T>(
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
  return body as T;
}

async function api<T>(
  request: APIRequestContext,
  path: string,
  method = 'GET',
  data?: unknown,
): Promise<T> {
  const body = await wire<{ data: T }>(request, path, method, data);
  return body.data;
}

async function choose(page: Page, input: Locator, keyword: string) {
  await input.click();
  await input.fill(keyword);
  await page
    .locator('.ant-select-dropdown:visible .ant-select-item-option')
    .filter({ hasText: keyword })
    .first()
    .click();
  const select = input.locator(
    'xpath=ancestor::div[contains(concat(" ", normalize-space(@class), " "), " ant-select ")][1]',
  );
  await expect(select).toContainText(keyword);
}

async function orderAction(page: Page, label: string) {
  await page.getByRole('button', { name: /更多操作/ }).click();
  await page.getByRole('menuitem', { name: label, exact: true }).click();
}

async function dateTime(input: Locator, value: string) {
  await input.fill(value);
  await input.press('Enter');
}

function masterNumber(serial: number) {
  const body = String(serial).slice(-7).padStart(7, '0');
  return `990${body}${Number(body) % 7}`;
}

async function saveOrder(page: Page, orderId: string) {
  const response = page.waitForResponse(
    (item) =>
      item.url().endsWith(`/api/v1/orders/${orderId}`) &&
      item.request().method() === 'PUT',
  );
  await page.getByRole('button', { name: /保存$/ }).click();
  const saved = await response;
  expect(saved.ok(), (await saved.json()).message).toBeTruthy();
}

async function confirmWeight(page: Page, weight: string, reason: string) {
  await page
    .getByRole('button', { name: '确认或调整计费重', exact: true })
    .click();
  const modal = page.getByRole('dialog', { name: '确认计费重量', exact: true });
  await modal.getByLabel('最终确认重量（KG）', { exact: true }).fill(weight);
  await modal.getByLabel('确认依据 / 调整原因', { exact: true }).fill(reason);
  const response = page.waitForResponse(
    (item) =>
      item.url().endsWith('/api/v1/air-weights/confirm') &&
      item.request().method() === 'POST',
  );
  await modal
    .getByRole('button', { name: /确\s*定|提\s*交/ })
    .last()
    .click();
  const saved = await response;
  expect(saved.ok(), (await saved.json()).message).toBeTruthy();
  await expect(modal).toBeHidden();
}

// 本地隔离库的伪造夹具。API不拦截响应；建票、重量和服务操作均使用真实控件。
test('仅AI权限的直单UI建票、主分单共享、重量来源与进口服务输入保护', async ({
  page,
  request,
}, info) => {
  test.setTimeout(420_000);
  page.setDefaultTimeout(15_000);
  const pageErrors: string[] = [];
  page.on('pageerror', (error) => pageErrors.push(error.message));
  const stamp = Date.now();
  const fixture =
    await test.step('隔离公司、公共字典和仅AI权限身份', async () => {
      await api(request, '/api/v1/auth/login', 'POST', {
        username: requiredEnvironment('BOOTSTRAP_ADMIN_USERNAME'),
        password: requiredEnvironment('BOOTSTRAP_ADMIN_PASSWORD'),
      });
      const admin = await api<API.CurrentUser>(request, '/api/v1/auth/me');
      const systemId = admin.currentOrganization?.id;
      expect(systemId).toBeTruthy();
      const company = await api<API.AdminOrganization>(
        request,
        '/api/v1/admin/organizations',
        'POST',
        {
          code: `AI_BROWSER_${stamp}`,
          name: `空运进口隔离验收公司${stamp}`,
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
          nameZh: '空运进口隔离航司',
          nameEn: 'Isolated AI Airline',
          countryCode: 'CN',
        },
      );
      const airports: API.Airport[] = [];
      for (const [iataCode, nameZh] of [
        ['XAA', '隔离起运机场'],
        ['XAB', '隔离目的机场'],
      ])
        airports.push(
          await api<API.Airport>(
            request,
            '/api/v1/master-data/airports',
            'POST',
            {
              iataCode,
              nameZh,
              nameEn: `Isolated ${iataCode}`,
              cityNameZh: '隔离验收城市',
              countryCode: 'CN',
            },
          ),
        );
      const username = `ai_operator_${stamp}`;
      const password = `Ai-${randomUUID()}`;
      const operator = await api<API.AdminUser>(
        request,
        '/api/v1/admin/users',
        'POST',
        {
          username,
          displayName: '仅AI业务权限验收操作员',
          password,
          roleIds: [],
        },
      );
      await api(request, '/api/v1/auth/switch-organization', 'POST', {
        organizationId: company.id,
      });
      const permissions = await api<API.AdminPermission[]>(
        request,
        '/api/v1/admin/permissions',
      );
      const publicReads = new Set([
        'business.partner.read',
        'system.master_data.option.read',
        'system.master_data.item.read',
        'system.master_data.currency.read',
        'system.master_data.airport.read',
        'system.master_data.airline.read',
      ]);
      const permissionKeys = permissions.flatMap((permission) =>
        permission.key &&
        (permission.key.startsWith('business.order.ai.') ||
          publicReads.has(permission.key))
          ? [permission.key]
          : [],
      );
      expect(permissionKeys).toContain('business.order.ai.create');
      expect(
        permissionKeys.some((key) => key.startsWith('business.order.ae.')),
      ).toBe(false);
      const role = await api<API.AdminRole>(
        request,
        '/api/v1/admin/roles',
        'POST',
        {
          code: `ai_browser_${stamp}`,
          name: '仅AI业务与必要公共读取',
          permissionKeys,
        },
      );
      await api(request, '/api/v1/auth/switch-organization', 'POST', {
        organizationId: systemId,
      });
      await api(
        request,
        `/api/v1/admin/users/${operator.id}/memberships:batchCreate`,
        'POST',
        {
          userId: operator.id,
          items: [{ organizationId: company.id, roleIds: [role.id] }],
        },
      );
      await api(request, '/api/v1/auth/switch-organization', 'POST', {
        organizationId: company.id,
      });
      const customer = await api<API.Partner>(
        request,
        '/api/v1/partners',
        'POST',
        {
          legalName: `空运进口隔离客户${stamp}`,
          roles: [
            { type: 1, enabled: true },
            { type: 2, enabled: true },
          ],
          profile: {
            nameEn: 'ISOLATED AI CUSTOMER',
            countryCode: 'CN',
            businessTypes: [4],
          },
          assignments: [3, 4].map((assignmentRole) => ({
            role: assignmentRole,
            userId: operator.id,
            organizationId: company.id,
          })),
        },
      );
      return {
        company,
        operator,
        username,
        password,
        customer,
        airline,
        origin: airports[0],
        destination: airports[1],
      };
    });

  await test.step('真实登录选择AI工作区且不具备AE业务权限', async () => {
    await page.goto('/user/login');
    await page.getByText('账号登录', { exact: true }).click();
    await page.getByPlaceholder('钉钉手机号 / 本地账号').fill(fixture.username);
    await page.getByRole('button', { name: '下一步', exact: true }).click();
    await page.getByPlaceholder('请输入密码').fill(fixture.password);
    await page.locator('button[type="submit"]').click();
    await expect(
      page.getByRole('heading', { name: '选择进入的组织' }),
    ).toBeVisible();
    await page
      .getByRole('button', { name: `进入${fixture.company.name}` })
      .click();
    await expect(page).not.toHaveURL(/\/user\/login/);
    const me = await api<API.CurrentUser>(page.request, '/api/v1/auth/me');
    expect(me.currentOrganization?.id).toBe(fixture.company.id);
    expect(
      me.permissionCapabilities?.some(
        (item) => item.key === 'business.order.ai.create',
      ),
    ).toBe(true);
    expect(
      me.permissionCapabilities?.some((item) =>
        item.key?.startsWith('business.order.ae.'),
      ),
    ).toBe(false);
    const forbidden = await page.request.get(
      '/api/v1/air-export/master-waybills?businessType=AE&page=1&pageSize=20',
    );
    expect(forbidden.status()).toBe(403);
  });

  const direct =
    await test.step('DIRECT从真实新建表单提交并刷新编辑', async () => {
      await page.goto('/orders/air-import/new');
      await expect(
        page.getByRole('combobox', { name: /委托单位/ }),
      ).toBeVisible({ timeout: 60_000 });
      await page
        .getByRole('radio', { name: '直单（DIRECT，无 HAWB）', exact: true })
        .check();
      await expect(
        page.getByLabel('本票分运单号（HAWB）', { exact: true }),
      ).toHaveCount(0);
      await choose(
        page,
        page.getByRole('combobox', { name: /委托单位/ }),
        fixture.customer.legalName || '',
      );
      await page
        .getByLabel('主运单号（MAWB）', { exact: true })
        .fill(masterNumber(stamp));
      await choose(page, page.getByLabel('航空公司', { exact: true }), 'QX');
      await choose(page, page.getByLabel('起运机场', { exact: true }), 'XAA');
      await choose(page, page.getByLabel('目的机场', { exact: true }), 'XAB');
      await page.getByLabel('航班号', { exact: true }).fill('QX101');
      await dateTime(
        page.getByLabel('计划起飞', { exact: true }),
        '2026-11-17 08:00:00',
      );
      await dateTime(
        page.getByLabel('计划到达', { exact: true }),
        '2026-11-17 18:00:00',
      );
      await page
        .getByLabel('委托品名 / 货物描述', { exact: true })
        .fill('空运进口直单UI验收货物');
      await page.getByLabel('本票毛重（KG）', { exact: true }).fill('100');
      await page.getByLabel('本票体积（m³）', { exact: true }).fill('1.2');
      await choose(
        page,
        page.getByRole('combobox', { name: /操作人员/ }),
        '仅AI业务权限验收操作员',
      );
      const response = page.waitForResponse(
        (item) =>
          item.url().endsWith('/api/v1/orders') &&
          item.request().method() === 'POST',
      );
      await page.getByRole('button', { name: /创建订单$/ }).click();
      const created = await response;
      const body = await created.json();
      expect(created.ok(), body.message).toBeTruthy();
      const order = body.data as API.Order;
      expect(order.businessType).toBe(4);
      expect(order.tradeDirection).toBe(2);
      expect(order.airExport?.documentMode).toBe('DIRECT');
      expect(order.airExport?.house).toBeUndefined();
      expect(order.airExport?.master?.businessType).toBe('AI');
      await expect(page).toHaveURL(
        new RegExp(`/orders/air-import/${order.id}$`),
      );
      await page.reload();
      await expect(
        page.getByText('直单：本票使用主运单，无 HAWB', { exact: true }),
      ).toBeVisible();
      await page
        .getByTestId('sea-notes-operationNotes')
        .getByRole('button', { name: '展开', exact: true })
        .click();
      await page
        .getByPlaceholder('请输入操作备注', { exact: true })
        .fill('AI直单保存刷新后仍无HAWB');
      await saveOrder(page, order.id || '');
      await page.reload();
      await expect(
        page.getByPlaceholder('请输入操作备注', { exact: true }),
      ).toHaveValue('AI直单保存刷新后仍无HAWB');
      expect(
        (await api<API.Order>(page.request, `/api/v1/orders/${order.id}`))
          .airExport?.house,
      ).toBeUndefined();
      return order;
    });

  await test.step('ORDER的PLANNED与ACTUAL确认互相独立', async () => {
    await page
      .getByRole('tab', { name: '本票重量与确认', exact: true })
      .click();
    await page
      .getByRole('radio', { name: '订单委托货量', exact: true })
      .check();
    await confirmWeight(page, '150', 'AI隔离验收：本票委托来源人工确认150KG');
    await orderAction(page, '实际货物明细');
    const cargo = page.getByRole('dialog', { name: /订单货物明细/ });
    await cargo.getByRole('button', { name: /添加货物明细$/ }).click();
    const form = page.getByRole('dialog', {
      name: '添加货物明细',
      exact: true,
    });
    await form.getByLabel('货物名称', { exact: true }).fill('AI实际货物');
    await form.getByLabel('件数', { exact: true }).fill('1');
    await form.getByLabel('毛重 (KG)', { exact: true }).fill('100');
    for (const [name, value] of [
      ['长度 cm', '120'],
      ['宽度 cm', '100'],
      ['高度 cm', '100'],
    ])
      await form.getByRole('spinbutton', { name, exact: true }).fill(value);
    await form
      .getByRole('button', { name: '填入当前货物体积', exact: true })
      .click();
    const cargoResponse = page.waitForResponse(
      (item) =>
        item.url().endsWith(`/api/v1/orders/${direct.id}/cargo-items`) &&
        item.request().method() === 'POST',
    );
    await form.getByRole('button', { name: /确\s*定/ }).click();
    expect((await cargoResponse).ok()).toBeTruthy();
    await expect(form).toBeHidden();
    await cargo.getByRole('button', { name: '关闭', exact: true }).click();
    await page
      .getByRole('radio', { name: '实际货物明细', exact: true })
      .check();
    await confirmWeight(page, '180', 'AI隔离验收：本票实际来源人工确认180KG');
    for (const [source, weight] of [
      ['PLANNED', '150'],
      ['ACTUAL', '180'],
    ]) {
      const preview = await api<API.AirWeightPreview>(
        page.request,
        '/api/v1/air-weights/preview',
        'POST',
        {
          scope: 'ORDER',
          targetId: direct.id,
          source,
          volumetricDivisor: '6000',
          bubbleShareNumerator: 1,
          bubbleShareDenominator: 1,
        },
      );
      expect(preview.currentConfirmation?.confirmedWeightKg).toBe(weight);
      expect(preview.confirmationStale ?? false).toBe(false);
    }
  });

  await test.step('MASTER确认不覆盖本票，费用来源绑定本票确认', async () => {
    await orderAction(page, '共享主单与成员');
    await expect(page).toHaveURL(
      new RegExp(`/orders/air-import/masters/${direct.airExport?.master?.id}$`),
    );
    await expect(
      page.getByRole('button', { name: '修改共同资料', exact: true }),
    ).toBeEnabled();
    await expect(
      page.getByRole('button', { name: '从本主单继续建票', exact: true }),
    ).toBeDisabled();
    await page
      .getByRole('radio', { name: '订单委托货量', exact: true })
      .check();
    await confirmWeight(page, '210', 'AI隔离验收：主单独立确认210KG');
    const preview = await api<API.AirWeightPreview>(
      page.request,
      '/api/v1/air-weights/preview',
      'POST',
      {
        scope: 'ORDER',
        targetId: direct.id,
        source: 'PLANNED',
        volumetricDivisor: '6000',
        bubbleShareNumerator: 1,
        bubbleShareDenominator: 1,
      },
    );
    expect(preview.currentConfirmation?.confirmedWeightKg).toBe('150');
    const options = await wire<API.ListFeeOptionsResponse>(
      page.request,
      `/api/v1/orders/${direct.id}/fee-options`,
    );
    const kg = options.billingUnits?.find((item) => item.code === 'KG');
    expect(kg?.id).toBeTruthy();
    for (const [source, weight] of [
      ['PLANNED', '150'],
      ['ACTUAL', '180'],
    ]) {
      const quantity = await api<API.FeeQuantityPreview>(
        page.request,
        `/api/v1/orders/${direct.id}/fee-quantity-preview?billingUnitId=${kg?.id}&source=${source}&quantityBasis=CONFIRMED_CHARGEABLE_WEIGHT`,
      );
      expect(quantity.available).toBe(true);
      expect(Number(quantity.quantity)).toBe(Number(weight));
      expect(quantity.quantitySource?.confirmationId).toBeTruthy();
      expect(quantity.quantitySource?.source).toBe(source);
    }
  });

  const house =
    await test.step('HOUSE真实共享主单双成员与AI候选隔离', async () => {
      const input = {
        customerId: fixture.customer.id,
        businessType: 4,
        tradeDirection: 2,
        paymentTerm: 1,
        goodsDescription: 'AI主分单隔离验收货物',
        totalGrossWeightKg: 100,
        totalVolumeCbm: 0.6,
        personnelAssignments: [2, 3, 4].map((role) => ({
          role,
          userId: fixture.operator.id,
        })),
      };
      const first = await api<API.Order>(request, '/api/v1/orders', 'POST', {
        ...input,
        airExport: {
          documentMode: 'HOUSE',
          master: {
            masterNo: masterNumber(stamp + 1),
            airlineId: fixture.airline.id,
            originAirportId: fixture.origin.id,
            destinationAirportId: fixture.destination.id,
            flightNo: 'QX201',
            etd: '2026-11-17T00:00:00Z',
            eta: '2026-11-17T10:00:00Z',
          },
          house: { houseNo: `AIHOUSE-A-${stamp}`, status: 'DRAFT' },
        },
      });
      const master = first.airExport?.master;
      expect(master?.businessType).toBe('AI');
      const second = await api<API.Order>(request, '/api/v1/orders', 'POST', {
        ...input,
        airExport: {
          documentMode: 'HOUSE',
          masterWaybillId: master?.id,
          expectedMasterVersion: master?.version,
          house: { houseNo: `AIHOUSE-B-${stamp}`, status: 'DRAFT' },
        },
      });
      expect(second.airExport?.master?.id).toBe(master?.id);
      expect(first.airExport?.house?.id).toBeTruthy();
      expect(second.airExport?.house?.id).not.toBe(first.airExport?.house?.id);
      const candidates = await api<API.AirMasterWaybill[]>(
        page.request,
        `/api/v1/air-export/master-waybills?businessType=AI&keyword=${master?.content?.masterNo}&page=1&pageSize=20`,
      );
      expect(candidates.some((item) => item.id === master?.id)).toBe(true);
      expect(candidates.every((item) => item.businessType === 'AI')).toBe(true);
      await page.goto(`/orders/air-import/masters/${master?.id}`);
      await page
        .getByRole('button', { name: '修改共同资料', exact: true })
        .click();
      const shared = page.getByRole('dialog', {
        name: '修改共享主单共同资料',
        exact: true,
      });
      await shared.getByLabel('航班号', { exact: true }).fill('QX299');
      await dateTime(
        shared.getByLabel('计划到达', { exact: true }),
        '2026-11-18 00:30:00',
      );
      await shared
        .getByLabel('修改原因', { exact: true })
        .fill('AI共享双票ETA跨中国午夜验收');
      await shared
        .getByRole('button', { name: '预览影响', exact: true })
        .click();
      await expect(shared.getByText(/影响 2 票订单/)).toBeVisible();
      const updated = page.waitForResponse(
        (item) =>
          item
            .url()
            .endsWith(`/api/v1/air-export/master-waybills/${master?.id}`) &&
          item.request().method() === 'PUT',
      );
      await shared
        .getByRole('button', { name: '确认影响并提交', exact: true })
        .click();
      expect((await updated).ok()).toBeTruthy();
      await expect(shared).toBeHidden();
      for (const orderId of [first.id, second.id]) {
        const current = await api<API.Order>(
          page.request,
          `/api/v1/orders/${orderId}`,
        );
        expect(current.airExport?.master?.content?.flightNo).toBe('QX299');
        expect(new Date(current.eta || '').toISOString()).toBe(
          '2026-11-17T16:30:00.000Z',
        );
      }
      const aggregate = await wire<API.GetAirMasterWaybillResponse>(
        page.request,
        `/api/v1/air-export/master-waybills/${master?.id}?page=1&pageSize=1`,
      );
      expect(aggregate.total).toBe(2);
      expect(aggregate.members).toHaveLength(1);
      expect(aggregate.totals?.grossWeightKg).toBe(200);
      await page
        .getByRole('button', { name: '从本主单继续建票', exact: true })
        .click();
      await expect(page).toHaveURL(
        new RegExp(`/orders/air-import/new\\?masterId=${master?.id}`),
      );
      await expect(
        page.getByText('关联后共同运输资料以主单为准', { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByLabel('本票分运单号（HAWB）', { exact: true }),
      ).toBeVisible();
      return first;
    });

  await test.step('AI服务子集保存、主表草稿与外部版本输入保护', async () => {
    await page.goto(`/orders/air-import/${house.id}`);
    const note = page.getByTestId('sea-notes-operationNotes');
    await note.getByRole('button', { name: '展开', exact: true }).click();
    const operation = page.getByPlaceholder('请输入操作备注', { exact: true });
    await operation.fill('AI服务保存后保留主表草稿并继续提交');
    await page
      .getByRole('button', { name: '维护承接与完成情况', exact: true })
      .click();
    let modal = page.getByRole('dialog', {
      name: '维护本票进口服务',
      exact: true,
    });
    await modal
      .getByRole('checkbox', { name: '承接换单', exact: true })
      .check();
    await modal
      .getByRole('checkbox', { name: '承接派送', exact: true })
      .check();
    await dateTime(
      modal.getByLabel('换单实际完成时间', { exact: true }),
      '2026-11-18 09:00:00',
    );
    await dateTime(
      modal.getByLabel('派送实际完成时间', { exact: true }),
      '2026-11-18 10:00:00',
    );
    const response = page.waitForResponse(
      (item) =>
        item.url().endsWith(`/api/v1/orders/${house.id}/import-handling`) &&
        item.request().method() === 'PUT',
    );
    await modal
      .getByRole('button', { name: '保存服务资料', exact: true })
      .click();
    expect((await response).ok()).toBeTruthy();
    await expect(modal).toBeHidden();
    await expect(operation).toHaveValue('AI服务保存后保留主表草稿并继续提交');
    await saveOrder(page, house.id || '');
    await page.reload();
    await expect(operation).toHaveValue('AI服务保存后保留主表草稿并继续提交');
    const handling = await api<API.OrderImportHandling>(
      page.request,
      `/api/v1/orders/${house.id}/import-handling`,
    );
    expect(handling.steps?.filter((item) => item.undertaken)).toHaveLength(2);
    expect(handling.steps?.filter((item) => item.occurredAt)).toHaveLength(2);
    for (const detection of ['polling', 'focus'] as const) {
      if (detection === 'focus') await page.reload();
      const baseline = await api<API.OrderImportHandling>(
        page.request,
        `/api/v1/orders/${house.id}/import-handling`,
      );
      await page
        .getByRole('button', { name: '维护承接与完成情况', exact: true })
        .click();
      modal = page.getByRole('dialog', {
        name: '维护本票进口服务',
        exact: true,
      });
      const preserved = `AI外部变动后必须保留的输入-${detection}`;
      await modal.getByLabel('变更原因', { exact: true }).fill(preserved);
      await api(request, `/api/v1/orders/${house.id}/import-handling`, 'PUT', {
        expectedVersion: baseline.orderVersion,
        reason: 'AI另一会话更正已完成派送参考号',
        steps: baseline.steps?.map((item) => ({
          ...item,
          referenceNo:
            item.kind === 5 ? `AI-CONCURRENT-${detection}` : item.referenceNo,
        })),
      });
      if (detection === 'focus')
        await page.evaluate(() => {
          window.dispatchEvent(new Event('blur'));
          window.dispatchEvent(new Event('focus'));
        });
      await expect(
        modal.getByText('这个订单已经发生变动，请重新刷新。', { exact: true }),
      ).toBeVisible({ timeout: detection === 'polling' ? 25_000 : 10_000 });
      await expect(modal.getByLabel('变更原因', { exact: true })).toHaveValue(
        preserved,
      );
      await expect(
        modal.getByRole('button', { name: '保存服务资料', exact: true }),
      ).toBeDisabled();
      await page.screenshot({
        path: info.outputPath(`ai-service-${detection}-input-preserved.png`),
        fullPage: true,
      });
      await modal.getByRole('button', { name: /取\s*消/ }).click();
      await expect(modal).toBeHidden();
    }
  });

  expect(pageErrors).toEqual([]);
  await info.attach('air-import-evidence', {
    body: JSON.stringify({
      companyId: fixture.company.id,
      directOrderId: direct.id,
      houseOrderId: house.id,
      directMasterId: direct.airExport?.master?.id,
      houseMasterId: house.airExport?.master?.id,
      aiOnlyIdentity: fixture.operator.id,
      plannedOrderWeight: '150',
      actualOrderWeight: '180',
      plannedMasterWeight: '210',
    }),
    contentType: 'application/json',
  });
});
