import { expect, type Locator, type Page, test } from '@playwright/test';

function requiredEnvironment(name: string) {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`缺少环境变量 ${name}`);
  return value;
}

async function choose(page: Page, input: Locator, text: string) {
  await input.click();
  await input.fill(text);
  await page
    .locator('.ant-select-dropdown:visible .ant-select-item-option')
    .filter({ hasText: text })
    .first()
    .click();
}

test('空运页面建双票、共享主单修改与独立重量确认刷新重读', async ({
  page,
}, info) => {
  test.setTimeout(240_000);
  page.setDefaultTimeout(15_000);
  const pageErrors: string[] = [];
  page.on('pageerror', (error) => pageErrors.push(error.message));
  await page.goto('/user/login');
  await page.getByText('账号登录', { exact: true }).click();
  await page
    .getByPlaceholder('钉钉手机号 / 本地账号')
    .fill(requiredEnvironment('BOOTSTRAP_ADMIN_USERNAME'));
  await page
    .getByPlaceholder('请输入密码')
    .fill(requiredEnvironment('BOOTSTRAP_ADMIN_PASSWORD'));
  await page.locator('button[type="submit"]').click();
  await expect(
    page.getByRole('heading', { name: '选择进入的组织' }),
  ).toBeVisible();
  const me = (await (await page.request.get('/api/v1/auth/me')).json()).data;
  const company = me.organizations.find(
    (item: { name: string; kind: number }) =>
      item.kind === 2 && item.name.includes('北京'),
  );
  expect(company?.id).toBeTruthy();
  await page.getByRole('button', { name: `进入${company.name}` }).click();
  await expect(page).not.toHaveURL(/\/user\/login/);
  const read = async (path: string) => {
    const response = await page.request.get(path);
    const body = await response.json();
    expect(response.ok(), `${path}: ${body.message}`).toBeTruthy();
    return body;
  };
  const stamp = Date.now().toString();
  const serial = stamp.slice(-7);
  const masterNo = `781${serial}${Number(serial) % 7}`;

  const completeTicket = async (
    houseNo: string,
    weight: string,
    volume: string,
  ) => {
    await choose(
      page,
      page.getByLabel('委托单位', { exact: true }),
      '上海宏远',
    );
    await page
      .getByLabel('本票分运单号（HAWB）', { exact: true })
      .fill(houseNo);
    await page
      .getByLabel('委托品名 / 货物描述', { exact: true })
      .fill('空运浏览器验收货物');
    await page.getByLabel('本票毛重（KG）', { exact: true }).fill(weight);
    await page.getByLabel('本票体积（m³）', { exact: true }).fill(volume);
    const createdResponse = page.waitForResponse(
      (response) =>
        response.url().endsWith('/api/v1/orders') &&
        response.request().method() === 'POST',
    );
    await page.getByRole('button', { name: /创建订单$/ }).click();
    const response = await createdResponse;
    const body = await response.json();
    expect(response.ok(), body.message).toBeTruthy();
    expect(body.data.airExport.houseNo).toBe(houseNo);
    await expect(page).toHaveURL(
      new RegExp(`/orders/air-export/${body.data.id}`),
    );
    return body.data;
  };

  await page.goto('/orders/air-export/new');
  await page.getByLabel('主运单号（MAWB）', { exact: true }).fill(masterNo);
  await choose(page, page.getByLabel('航空公司', { exact: true }), 'MU');
  await choose(page, page.getByLabel('起运机场', { exact: true }), 'PVG');
  await choose(page, page.getByLabel('目的机场', { exact: true }), 'LAX');
  await page.getByLabel('航班号', { exact: true }).fill('MU583');
  for (const [label, days] of [
    ['计划起飞', 1],
    ['计划到达', 2],
  ] as const) {
    const date = new Date(Date.now() + days * 86_400_000);
    const value = new Intl.DateTimeFormat('sv-SE', {
      timeZone: 'Asia/Shanghai',
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    }).format(date);
    const input = page.getByLabel(label, { exact: true });
    await input.fill(value);
    await input.press('Enter');
  }
  const first = await completeTicket(`UI-AIR-A-${stamp}`, '100', '1.2');
  const masterId = first.airExport.master.id;
  await page.reload();
  await expect(
    page.getByLabel('本票分运单号（HAWB）', { exact: true }),
  ).toHaveValue(first.airExport.houseNo);
  await page
    .getByLabel('操作备注', { exact: true })
    .fill('仅本票修改，保留共享主单事实');
  const savedResponse = page.waitForResponse(
    (response) =>
      response.url().endsWith(`/api/v1/orders/${first.id}`) &&
      response.request().method() === 'PUT',
  );
  await page.getByRole('button', { name: /保存$/ }).click();
  expect((await savedResponse).ok()).toBeTruthy();
  expect((await read(`/api/v1/orders/${first.id}`)).data.operationNotes).toBe(
    '仅本票修改，保留共享主单事实',
  );
  await page.getByRole('tab', { name: '本票重量与确认', exact: true }).click();
  await page.getByText('订单委托货量', { exact: true }).first().click();
  await expect(page.getByText('200', { exact: true }).first()).toBeVisible();
  await page
    .getByRole('button', { name: '确认或调整计费重', exact: true })
    .click();
  const confirmation = page.getByRole('dialog', {
    name: '确认计费重量',
    exact: true,
  });
  await confirmation.getByLabel('最终确认重量（KG）').fill('130');
  await confirmation
    .getByLabel('确认依据 / 调整原因')
    .fill('隔离验收：客户最终确认 130KG');
  const confirmedResponse = page.waitForResponse(
    (response) =>
      response.url().endsWith('/api/v1/air-weights/confirm') &&
      response.request().method() === 'POST',
  );
  await confirmation
    .getByRole('button', { name: /确\s*定|提\s*交/ })
    .last()
    .click();
  expect((await confirmedResponse).ok()).toBeTruthy();
  await expect(confirmation).toBeHidden();
  await expect(page.getByText('130', { exact: true }).first()).toBeVisible();

  await page.getByRole('button', { name: '实际货物明细', exact: true }).click();
  const cargoDrawer = page.getByRole('dialog', { name: /订单货物明细/ });
  await cargoDrawer.getByRole('button', { name: /添加货物明细$/ }).click();
  const cargoForm = page.getByRole('dialog', {
    name: '添加货物明细',
    exact: true,
  });
  await cargoForm.getByLabel('货物名称', { exact: true }).fill('厘米尺寸验收');
  await cargoForm.getByLabel('件数', { exact: true }).fill('1');
  await cargoForm.getByLabel('毛重 (KG)', { exact: true }).fill('100');
  await cargoForm
    .getByRole('spinbutton', { name: '长度 cm', exact: true })
    .fill('120');
  await cargoForm
    .getByRole('spinbutton', { name: '宽度 cm', exact: true })
    .fill('100');
  await cargoForm
    .getByRole('spinbutton', { name: '高度 cm', exact: true })
    .fill('100');
  await cargoForm
    .getByRole('button', { name: '填入当前货物体积', exact: true })
    .click();
  await expect(cargoForm.getByLabel('体积 (CBM)', { exact: true })).toHaveValue(
    '1.2',
  );
  const cargoSaved = page.waitForResponse(
    (response) =>
      response.url().includes(`/api/v1/orders/${first.id}/cargo-items`) &&
      response.request().method() === 'POST',
  );
  await cargoForm.getByRole('button', { name: /确\s*定/ }).click();
  expect((await cargoSaved).ok()).toBeTruthy();
  await expect(cargoForm).toBeHidden();
  await cargoDrawer.getByRole('button', { name: '关闭', exact: true }).click();
  await expect(cargoDrawer).toBeHidden();
  await page.getByRole('radio', { name: '实际货物明细', exact: true }).click();
  await expect(page.getByText('200', { exact: true }).first()).toBeVisible();
  const actualWeight = await page.request.post('/api/v1/air-weights/preview', {
    data: {
      scope: 'ORDER',
      targetId: first.id,
      source: 'ACTUAL',
      volumetricDivisor: '6000',
      bubbleShareNumerator: 1,
      bubbleShareDenominator: 1,
    },
  });
  const actualWeightBody = await actualWeight.json();
  expect(actualWeight.ok()).toBeTruthy();
  expect(actualWeightBody.data.volumeCbm).toBe('1.2');
  expect(actualWeightBody.data.suggestedChargeableWeightKg).toBe('200');

  await page
    .getByRole('button', { name: '共享主单与成员', exact: true })
    .click();
  await expect(page).toHaveURL(new RegExp(`/masters/${masterId}`));
  await page.getByRole('button', { name: '修改共同资料', exact: true }).click();
  const shared = page.getByRole('dialog', {
    name: '修改共享主单共同资料',
    exact: true,
  });
  await shared.getByLabel('航班号', { exact: true }).fill('MU585');
  await shared
    .getByLabel('修改原因', { exact: true })
    .fill('隔离验收共同航班调整');
  await shared.getByRole('button', { name: '预览影响', exact: true }).click();
  await expect(shared.getByText(/影响 1 票订单/)).toBeVisible();
  await shared
    .getByRole('button', { name: '确认影响并提交', exact: true })
    .click();
  await expect(shared).toBeHidden();
  expect(
    (await read(`/api/v1/orders/${first.id}`)).data.airExport.master.content
      .flightNo,
  ).toBe('MU585');
  await page
    .getByRole('button', { name: '从本主单继续建票', exact: true })
    .click();
  await expect(
    page.getByText('关联后共同运输资料以主单为准', { exact: true }),
  ).toBeVisible();
  const second = await completeTicket(`UI-AIR-B-${stamp}`, '300', '0.6');
  expect(second.airExport.master.id).toBe(masterId);
  const memberDetail = await read(
    `/api/v1/air-export/master-waybills/${masterId}?page=1&pageSize=1`,
  );
  expect(memberDetail.members).toHaveLength(1);
  expect(memberDetail.total).toBe(2);
  expect(memberDetail.totals.grossWeightKg).toBe(400);
  expect(memberDetail.totals.volumeCbm).toBeCloseTo(1.8, 6);
  await page
    .getByRole('button', { name: '共享主单与成员', exact: true })
    .click();
  await page.getByText('订单委托货量', { exact: true }).first().click();
  await expect(page.getByText('400', { exact: true }).first()).toBeVisible();
  const firstAfter = (await read(`/api/v1/orders/${first.id}`)).data;
  expect(firstAfter.operationNotes).toBe('仅本票修改，保留共享主单事实');
  expect(firstAfter.airExport.master.content.flightNo).toBe('MU585');
  const firstWeight = await page.request.post('/api/v1/air-weights/preview', {
    data: {
      scope: 'ORDER',
      targetId: first.id,
      source: 'PLANNED',
      volumetricDivisor: '6000',
      bubbleShareNumerator: 1,
      bubbleShareDenominator: 1,
    },
  });
  const weight = (await firstWeight.json()).data;
  expect(firstWeight.ok()).toBeTruthy();
  expect(weight.currentConfirmation.confirmedWeightKg).toBe('130');
  expect(weight.confirmationStale ?? false).toBe(false);
  await expect(page.getByText('1.8', { exact: true }).first()).toBeVisible();
  await expect(
    page.getByText('1.7999999999999998', { exact: true }),
  ).toHaveCount(0);
  await page.evaluate(() => window.scrollTo(0, 0));
  await expect.poll(() => page.evaluate(() => window.scrollY)).toBe(0);
  await page.screenshot({
    path: info.outputPath('air-master-members.png'),
    fullPage: true,
  });

  await page.getByRole('button', { name: '修改共同资料', exact: true }).click();
  await shared.getByLabel('航班号', { exact: true }).fill('MU599');
  await shared.getByLabel('修改原因', { exact: true }).fill('隔离验收过期预览');
  await shared.getByRole('button', { name: '预览影响', exact: true }).click();
  await expect(shared.getByText(/影响 2 票订单/)).toBeVisible();
  const currentMaster = (
    await read(`/api/v1/air-export/master-waybills/${masterId}`)
  ).data;
  const concurrentInput = { ...currentMaster.content, flightNo: 'MU600' };
  const concurrentPreviewResponse = await page.request.post(
    `/api/v1/air-export/master-waybills/${masterId}/preview`,
    {
      data: {
        id: masterId,
        expectedVersion: currentMaster.version,
        input: concurrentInput,
      },
    },
  );
  const concurrentPreview = await concurrentPreviewResponse.json();
  expect(concurrentPreviewResponse.ok()).toBeTruthy();
  const concurrentWrite = await page.request.put(
    `/api/v1/air-export/master-waybills/${masterId}`,
    {
      data: {
        id: masterId,
        expectedVersion: currentMaster.version,
        input: concurrentInput,
        inputFingerprint: concurrentPreview.inputFingerprint,
        reason: '隔离验收模拟另一操作员更新',
      },
    },
  );
  expect(concurrentWrite.ok()).toBeTruthy();
  const conflictResponse = page.waitForResponse(
    (response) =>
      response
        .url()
        .endsWith(`/api/v1/air-export/master-waybills/${masterId}`) &&
      response.request().method() === 'PUT',
  );
  await shared
    .getByRole('button', { name: '确认影响并提交', exact: true })
    .click();
  expect((await conflictResponse).status()).toBe(409);
  await expect(shared).toBeVisible();
  expect(
    (await read(`/api/v1/orders/${first.id}`)).data.airExport.master.content
      .flightNo,
  ).toBe('MU600');
  await shared.getByRole('button', { name: /取\s*消/ }).click();
  await expect(shared).toBeHidden();
  const secondCurrent = (await read(`/api/v1/orders/${second.id}`)).data;
  const locked = await page.request.post(`/api/v1/orders/${second.id}/lock`, {
    data: {
      orderId: second.id,
      expectedOrderVersion: secondCurrent.version,
      idempotencyKey: `air-ui-lock-${stamp}`,
    },
  });
  expect(locked.ok()).toBeTruthy();
  await page.getByRole('button', { name: /刷\s*新/ }).click();
  await page.getByRole('button', { name: '修改共同资料', exact: true }).click();
  await shared.getByLabel('航班号', { exact: true }).fill('MU700');
  await shared
    .getByLabel('修改原因', { exact: true })
    .fill('隔离验收锁定成员阻断');
  await shared.getByRole('button', { name: '预览影响', exact: true }).click();
  await expect(
    shared.getByText('共享修改被阻止', { exact: true }),
  ).toBeVisible();
  expect(
    (await read(`/api/v1/orders/${first.id}`)).data.airExport.master.content
      .flightNo,
  ).toBe('MU600');
  await page.screenshot({
    path: info.outputPath('air-master-locked-member-blocked.png'),
    fullPage: true,
  });
  await shared.getByRole('button', { name: /取\s*消/ }).click();
  await expect(shared).toBeHidden();

  await page.goto(`/orders/air-export/${first.id}`);
  for (const [label, expectedStatus] of [
    ['已订舱', 2],
    ['已配舱', 3],
  ] as const) {
    await page
      .getByRole('button', { name: `流转到${label}`, exact: true })
      .click();
    const flow = page.getByRole('dialog', {
      name: '流转订单状态',
      exact: true,
    });
    await expect(flow.getByRole('textbox')).toHaveCount(0);
    const transitionedResponse = page.waitForResponse(
      (response) =>
        response.url().endsWith(`/api/v1/orders/${first.id}/status`) &&
        response.request().method() === 'POST',
    );
    await flow.getByRole('button', { name: /确\s*定/ }).click();
    const transitioned = await transitionedResponse;
    const body = await transitioned.json();
    expect(transitioned.ok(), body.message).toBeTruthy();
    expect(body.data.flowStatus).toBe(expectedStatus);
    await expect(flow).toBeHidden();
    expect((await read(`/api/v1/orders/${first.id}`)).data.flowStatus).toBe(
      expectedStatus,
    );
  }
  await page.evaluate(() => window.scrollTo(0, 0));
  await expect.poll(() => page.evaluate(() => window.scrollY)).toBe(0);
  await page.screenshot({
    path: info.outputPath('air-order-space-allocated.png'),
    fullPage: true,
  });
  expect(pageErrors).toEqual([]);
  await info.attach('air-export-evidence', {
    body: JSON.stringify({
      firstOrder: first.id,
      secondOrder: second.id,
      masterId,
      masterNo,
      totalWeight: memberDetail.totals.grossWeightKg,
      confirmedOrderWeight: weight.currentConfirmation.confirmedWeightKg,
      verifiedUiFlowStatus: 'SPACE_ALLOCATED',
    }),
    contentType: 'application/json',
  });
});
