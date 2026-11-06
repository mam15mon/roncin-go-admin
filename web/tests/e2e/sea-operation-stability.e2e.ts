import { expect, type Page, test } from '@playwright/test';
import {
  installSeaStabilityFixture,
  stabilityOrderPath,
} from './sea-operation-stability.fixture';

// 本组覆盖完整企业桌面详情与导航，采用与人工验收一致的桌面工作区尺寸。
test.use({ viewport: { width: 1560, height: 1000 } });

async function expectFullDetail(page: Page, goodsDescription: string) {
  await expect(page.locator('textarea#goodsDescription')).toHaveValue(
    goodsDescription,
  );
  await expect(page.locator('#customerId')).toBeVisible();
  await expect(page.locator('#shippingLineId')).toBeVisible();
  await expect(page.getByRole('tab', { name: /主单 \(MBL\)/ })).toBeVisible();
  await expect(page.getByRole('tab', { name: /分单 \(HBL\)/ })).toBeVisible();
  await expect(page.getByText('页面暂时无法打开', { exact: true })).toHaveCount(
    0,
  );
}

async function enterSplit(page: Page) {
  await page.getByText('更多操作', { exact: true }).click();
  const split = page.getByRole('menuitem', { name: /scissor 拆票/ });
  await expect(split).not.toHaveAttribute('aria-disabled', 'true');
  await expect(split).not.toHaveClass(/ant-dropdown-menu-item-disabled/);
  await split.click();
  await expect(page).toHaveURL(`${stabilityOrderPath}/split`);
  await expect(
    page.getByText('原始订单基线汇总', { exact: true }),
  ).toBeVisible();
}

async function saveOrder(page: Page, description: string) {
  await page.locator('textarea#goodsDescription').fill(description);
  await page
    .getByRole('button', { name: /^(?:save )?保存$/ })
    .first()
    .click();
}

// 保留真实路由、OrderFormTemplate 和 SeaDocumentSection，仅在 API 边界使用合成响应。
test('拆票往返及浏览器前进后退在缓存与迟到单证响应下保持完整详情', async ({
  page,
}, testInfo) => {
  const state = await installSeaStabilityFixture(page);
  try {
    await page.goto(stabilityOrderPath);
    await expectFullDetail(page, '初始合成货物描述');
    for (let turn = 0; turn < 3; turn += 1) {
      await enterSplit(page);
      state.holdNext(turn % 2 === 0 ? 'documents' : 'order');
      await page
        .getByRole('button', { name: '返回订单详情', exact: true })
        .click();
      await expect(page).toHaveURL(stabilityOrderPath);
      await state.waitHeld();
      state.release();
      await expectFullDetail(page, '初始合成货物描述');
    }
    await page.goBack();
    await expect(page).toHaveURL(`${stabilityOrderPath}/split`);
    await expect(
      page.getByText('原始订单基线汇总', { exact: true }),
    ).toBeVisible();
    await page.goForward();
    await expect(page).toHaveURL(stabilityOrderPath);
    await expectFullDetail(page, '初始合成货物描述');
    expect(state.writes).toEqual([]);
  } finally {
    state.release();
    await state.report(testInfo);
  }
});

test('保存后分批重读及再次进入详情保留已保存字段并使用新版本', async ({
  page,
}, testInfo) => {
  const state = await installSeaStabilityFixture(page);
  try {
    await page.goto(stabilityOrderPath);
    await expectFullDetail(page, '初始合成货物描述');
    for (const [turn, kind] of ['documents', 'order'].entries()) {
      const description = `保存后真实详情回归-${turn + 1}`;
      state.holdNext(kind as 'documents' | 'order');
      await saveOrder(page, description);
      await state.waitHeld();
      // 已确认写入、聚合重读尚未完成时，原表单实例与输入仍应存在。
      await expect(page.locator('textarea#goodsDescription')).toHaveValue(
        description,
      );
      await expect(
        page.getByText('页面暂时无法打开', { exact: true }),
      ).toHaveCount(0);
      state.release();
      await expect(
        page.getByText('保存订单成功', { exact: true }),
      ).toBeVisible();
      await expectFullDetail(page, description);
      await enterSplit(page);
      await page
        .getByRole('button', { name: '返回订单详情', exact: true })
        .click();
      await expectFullDetail(page, description);
    }
    expect(state.writes.map((body) => body.expectedVersion)).toEqual([
      '1',
      '2',
    ]);
    await page.reload();
    await expectFullDetail(page, '保存后真实详情回归-2');
    expect(state.getOrder().version).toBe('3');
  } finally {
    state.release();
    await state.report(testInfo);
  }
});

test('写入成功但单证重读失败时保留页面，重试只读取且不重复保存', async ({
  page,
}, testInfo) => {
  const state = await installSeaStabilityFixture(page);
  try {
    await page.goto(stabilityOrderPath);
    await expectFullDetail(page, '初始合成货物描述');
    state.failNextSavedDocuments();
    await saveOrder(page, '写入成功后读取失败仍保留');
    await expect(
      page.getByText('订单已保存，最新数据读取失败', { exact: true }),
    ).toBeVisible();
    await expectFullDetail(page, '写入成功后读取失败仍保留');
    expect(state.writes).toHaveLength(1);
    await page.getByRole('button', { name: '重试读取', exact: true }).click();
    await expect(
      page.getByText('订单数据已同步', { exact: true }),
    ).toBeVisible();
    await expectFullDetail(page, '写入成功后读取失败仍保留');
    expect(state.writes).toHaveLength(1);
    await page.reload();
    await expectFullDetail(page, '写入成功后读取失败仍保留');
  } finally {
    state.release();
    await state.report(testInfo);
  }
});
