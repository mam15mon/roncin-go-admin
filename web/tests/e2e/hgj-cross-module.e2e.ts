import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { expect, type Page, test } from '@playwright/test';

test.use({ timezoneId: 'Asia/Shanghai' });

function shanghaiMinute(input: string) {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23',
  }).formatToParts(new Date(input));
  const value = (type: string) =>
    parts.find((part) => part.type === type)?.value;
  return `${value('year')}-${value('month')}-${value('day')} ${value('hour')}:${value('minute')}`;
}

function requiredEnvironment(name: string) {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`缺少环境变量 ${name}`);
  return value;
}

function parseCsv(input: string) {
  const rows: string[][] = [];
  let row: string[] = [],
    cell = '',
    quoted = false;
  for (let index = 1; index < input.length; index++) {
    const char = input[index];
    if (char === '"') {
      if (quoted && input[index + 1] === '"') {
        cell += '"';
        index++;
      } else quoted = !quoted;
    } else if (char === ',' && !quoted) {
      row.push(cell);
      cell = '';
    } else if (char === '\r' && input[index + 1] === '\n' && !quoted) {
      row.push(cell);
      rows.push(row);
      row = [];
      cell = '';
      index++;
    } else cell += char;
  }
  if (cell || row.length) {
    row.push(cell);
    rows.push(row);
  }
  return rows;
}

async function login(
  page: Page,
  username: string,
  password: string,
  companyName: string,
) {
  await page.goto('/user/login');
  await page.getByText('账号登录', { exact: true }).click();
  await page.getByPlaceholder('钉钉手机号 / 本地账号').fill(username);
  await page.getByRole('button', { name: '下一步', exact: true }).click();
  await page.getByPlaceholder('请输入密码').fill(password);
  await page.locator('button[type="submit"]').click();
  await expect(
    page.getByRole('heading', { name: '选择进入的组织' }),
  ).toBeVisible();
  await page.getByRole('button', { name: `进入${companyName}` }).click();
  await expect(page).not.toHaveURL(/\/user\/login/);
}

async function filterByCustomerReference(page: Page, keyword: string) {
  const filterToggle = page.getByRole('button', {
    name: /展开筛选|收起筛选/,
  });
  await expect(filterToggle).toBeVisible();
  if ((await filterToggle.textContent())?.includes('展开筛选')) {
    await filterToggle.click();
  }
  const composite = page
    .locator('.ant-form-item')
    .filter({ has: page.getByText('复合单号', { exact: true }) });
  await composite.getByRole('combobox').click();
  await page
    .locator('.ant-select-dropdown:visible')
    .getByText('客户业务号', { exact: true })
    .click();
  await page.getByPlaceholder('输入单号', { exact: true }).fill(keyword);
  await page.getByRole('button', { name: /查\s*询/ }).click();
}

test('四类完整筛选CSV真实205行、采纳排序与授权裁列', async ({
  page,
  browser,
}, info) => {
  test.setTimeout(300_000);
  page.setDefaultTimeout(15_000);
  const artifacts = requiredEnvironment('RONCIN_HGJ_FACTS_DIR');
  const facts = JSON.parse(
    await readFile(path.join(artifacts, 'hgj-cross-module-facts.json'), 'utf8'),
  );
  const identities = JSON.parse(
    await readFile(path.join(artifacts, 'hgj-private-identities.json'), 'utf8'),
  );
  await login(
    page,
    requiredEnvironment('BOOTSTRAP_ADMIN_USERNAME'),
    requiredEnvironment('BOOTSTRAP_ADMIN_PASSWORD'),
    facts.companyA.name,
  );
  for (const fixture of facts.csvFixtures) {
    await test.step(`${fixture.code}完整CSV205行与真实筛选排序`, async () => {
      await page.goto(`/orders/${fixture.path}`);
      await filterByCustomerReference(page, fixture.keyword);
      const header = page.getByRole('columnheader', { name: /订单编号/ });
      await header.click();
      const button = page.getByRole('button', { name: /导出完整筛选结果/ });
      await expect(button).toBeEnabled();
      const pending = page.waitForRequest((request) =>
        request.url().includes('/api/v1/orders/export'),
      );
      const downloaded = page.waitForEvent('download');
      await button.click();
      const exported = await pending;
      const params = new URL(exported.url()).searchParams;
      expect(params.get('page')).toBeNull();
      expect(params.get('pageSize')).toBeNull();
      expect(params.get('numberKeyword')).toBe(fixture.keyword);
      expect(params.get('sortBy')).toBe('orderNo');
      expect(params.get('sortOrder')).toBe('ascend');
      const download = await downloaded;
      const file = info.outputPath(
        `${fixture.code.toLowerCase()}-complete-205.csv`,
      );
      await download.saveAs(file);
      const bytes = await readFile(file);
      expect([...bytes.subarray(0, 3)]).toEqual([239, 187, 191]);
      const text = bytes.toString('utf8');
      expect(text).toContain('\r\n');
      const [headers, ...rows] = parseCsv(text);
      expect(rows).toHaveLength(205);
      const column = (name: string) => {
        const index = headers.indexOf(name);
        expect(index).toBeGreaterThanOrEqual(0);
        return index;
      };
      expect(rows.map((row) => row[column('订单编号')])).toEqual(
        fixture.orderedNumbers,
      );
      expect(
        rows.every((row) =>
          row[column('客户业务号')].startsWith(fixture.keyword),
        ),
      ).toBe(true);
      expect(rows[0][column('企业内部编号')]).toBe("'=SUM(1,2)");
      expect(rows[0][column('品名')]).toBe(
        `四类完整导出中文,"引号"\n第二行 ${fixture.code}`,
      );
      expect(rows[0][column('货值')]).toBe('15362.50');
      expect(rows[0][column('毛重(KG)')]).toBe('12.345');
      expect(rows[0][column('体积(m³)')]).toBe('1.234');
      expect(rows[0][column(fixture.direction === 1 ? 'ETD' : 'ETA')]).toBe(
        `${facts.businessDate} 00:30`,
      );
      for (const [header, wire] of [
        ['创建时间', fixture.firstTimes.createdAt],
        ['ETD', fixture.firstTimes.etd],
        ['ETA', fixture.firstTimes.eta],
      ]) {
        expect(rows[0][column(header)]).toBe(shanghaiMinute(wire));
        expect(rows[0][column(header)]).not.toBe(
          new Date(wire).toISOString().slice(0, 16).replace('T', ' '),
        );
      }
      for (const wire of [fixture.firstTimes.etd, fixture.firstTimes.eta]) {
        expect(shanghaiMinute(wire).slice(0, 10)).not.toBe(
          new Date(wire).toISOString().slice(0, 10),
        );
      }
      expect(shanghaiMinute(fixture.firstTimes.orderDate).slice(0, 10)).toBe(
        facts.businessDate,
      );
      expect(rows[0][column('业务日期')]).toBe(facts.businessDate);
      expect(headers.some((name) => /内部ID|版本号|UUID/.test(name))).toBe(
        false,
      );
      await page.screenshot({
        path: info.outputPath(
          `${fixture.code.toLowerCase()}-complete-filter.png`,
        ),
        fullPage: true,
      });
    });
  }
  await test.step('新构建生产同域SPA、JS MIME与真实205行完整CSV', async () => {
    const origin = new URL(requiredEnvironment('RONCIN_ACCEPTANCE_BASE_URL'))
      .origin;
    const fixture = facts.csvFixtures.find(
      (item: { code: string }) => item.code === 'SI',
    );
    const response = await page.goto(`${origin}/orders/sea-import`);
    expect(response?.status()).toBe(200);
    expect(response?.headers()['content-type']).toContain('text/html');
    const html = await response?.text();
    // go run必须嵌入主会话本轮build，不能以旧构建或开发入口代替生产同域验收。
    expect(html).toBe(
      await readFile('../server/internal/webassets/dist/index.html', 'utf8'),
    );
    expect(html).not.toContain('/@vite/client');
    const script = await page
      .locator('script[type="module"][src]')
      .first()
      .getAttribute('src');
    expect(script).toMatch(/^\/assets\/.+\.js$/);
    if (!script) throw new Error('生产SPA缺少模块脚本入口');
    const javascript = await page.request.get(new URL(script, origin).href);
    expect(javascript.status()).toBe(200);
    expect(javascript.headers()['content-type']).toMatch(
      /(?:text|application)\/javascript/,
    );
    expect((await javascript.text()).trimStart()).not.toMatch(
      /^(?:<!doctype|<html)/i,
    );
    const missing = await page.request.get(
      `${origin}/assets/hgj-acceptance-intentionally-missing.js`,
    );
    expect(missing.status()).toBe(404);
    await filterByCustomerReference(page, fixture.keyword);
    const button = page.getByRole('button', { name: /导出完整筛选结果/ });
    await expect(button).toBeEnabled();
    const pendingResponse = page.waitForResponse((result) =>
      result.url().includes('/api/v1/orders/export'),
    );
    const pendingDownload = page.waitForEvent('download');
    await button.click();
    const exported = await pendingResponse;
    expect(exported.status()).toBe(200);
    expect(new URL(exported.url()).origin).toBe(origin);
    const params = new URL(exported.url()).searchParams;
    expect(params.get('businessType')).toBe('2');
    expect(params.get('numberKeyword')).toBe(fixture.keyword);
    expect(params.get('page')).toBeNull();
    expect(params.get('pageSize')).toBeNull();
    const payload = await exported.json();
    expect(payload.success).toBe(true);
    expect(payload.total).toBe(205);
    expect(payload.data).toHaveLength(205);
    const file = info.outputPath('si-production-same-origin-complete-205.csv');
    await (await pendingDownload).saveAs(file);
    const bytes = await readFile(file);
    expect([...bytes.subarray(0, 3)]).toEqual([239, 187, 191]);
    const [headers, ...rows] = parseCsv(bytes.toString('utf8'));
    expect(rows).toHaveLength(payload.total);
    const numberIndex = headers.indexOf('订单编号');
    expect(numberIndex).toBeGreaterThanOrEqual(0);
    expect(rows.map((row) => row[numberIndex])).toEqual(
      payload.data.map((item: { orderNo: string }) => item.orderNo),
    );
    const referenceIndex = headers.indexOf('客户业务号');
    expect(referenceIndex).toBeGreaterThanOrEqual(0);
    expect(
      rows.every((row) => row[referenceIndex].startsWith(fixture.keyword)),
    ).toBe(true);
    await page.screenshot({
      path: info.outputPath('si-production-same-origin-complete-filter.png'),
      fullPage: true,
    });
  });
  const identity = identities.find(
    (item: { code: string }) => item.code === 'SI',
  );
  const context = await browser.newContext({
    timezoneId: 'Asia/Shanghai',
    baseURL: requiredEnvironment('RONCIN_WEB_BASE_URL'),
  });
  try {
    const restrictedPage = await context.newPage();
    await login(
      restrictedPage,
      identity.username,
      identity.password,
      facts.companyA.name,
    );
    await restrictedPage.goto('/orders/sea-import');
    const button = restrictedPage.getByRole('button', {
      name: /导出完整筛选结果/,
    });
    await expect(button).toBeEnabled();
    const pending = restrictedPage.waitForEvent('download');
    await button.click();
    const download = await pending;
    const file = info.outputPath('si-read-only-authorized-columns.csv');
    await download.saveAs(file);
    const [headers] = parseCsv(await readFile(file, 'utf8'));
    expect(headers).not.toContain('附件数');
    expect(headers).not.toContain('实际箱号');
  } finally {
    await context.close();
  }
});
