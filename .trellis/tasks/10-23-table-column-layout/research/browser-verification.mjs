import assert from 'node:assert/strict';
import { writeFile } from 'node:fs/promises';
import { openFixture } from './browser-fixture.mjs';

const fixture = await openFixture();
const { page } = fixture;
const results = [];
const record = (name, detail = {}) => { const result = { name, pass: true, ...detail }; results.push(result); console.log(JSON.stringify(result)); };
const row = key => page.locator(`[data-column-key="${key}"]`);
const zone = side => page.locator(`[data-freeze-zone="${side}"]`);
const dialog = page.getByRole('dialog', { name: '列设置', exact: true });
const open = async () => {
  await page.getByRole('button', { name: '列设置', exact: true }).click();
  await dialog.waitFor();
};
const close = async () => {
  await dialog.getByRole('button', { name: /关闭|Close/ }).click();
  await dialog.waitFor({ state: 'hidden' });
};
const mount = async config => {
  await page.evaluate(config => window.__mount(config), config);
  await page.getByRole('button', { name: '列设置', exact: true }).waitFor();
  await open();
};
const selectFreeze = async (key, label) => {
  await row(key).getByRole('combobox').click();
  await page.locator('.ant-select-dropdown:visible .ant-select-item-option').filter({ hasText: label }).click();
};
const assertOrder = async expected => {
  try {
    await page.waitForFunction(expected => JSON.stringify(Array.from(document.querySelectorAll('[data-column-key]')).map(node => node.getAttribute('data-column-key'))) === JSON.stringify(expected), expected);
  } catch (error) {
    console.log(JSON.stringify(await page.evaluate(() => ({ actual: Array.from(document.querySelectorAll('[data-column-key]')).map(node => node.getAttribute('data-column-key')), effective: window.__effectiveColumns.map(column => column.key), values: Array.from(document.querySelectorAll('[data-column-key] input')).map(input => ({ value: input.value, label: input.getAttribute('aria-label') })) }))));
    throw error;
  }
  const actual = await page.evaluate(() => window.__effectiveColumns.filter(column => column.key !== 'option').map(column => column.key));
  assert.deepEqual(actual, expected);
};

try {
  for (const [width, height] of [[1366, 768], [1440, 900], [1920, 1080]]) {
    await page.setViewportSize({ width, height });
    for (const fees of [false, true]) {
      await mount({ count: 40, fees });
      const measure = await dialog.evaluate(node => {
        const fields = Array.from(node.querySelectorAll('[data-column-key]'));
        const rects = fields.map(field => { const rect = field.getBoundingClientRect(); return { key: field.getAttribute('data-column-key'), x: rect.x, y: rect.y, right: rect.right, bottom: rect.bottom, width: rect.width, height: rect.height }; });
        const scrolling = Array.from(node.querySelectorAll('*')).filter(element => ['auto', 'scroll'].includes(getComputedStyle(element).overflowY) && element.scrollHeight > element.clientHeight + 1).map(element => element.className);
        return { count: fields.length, maxBottom: Math.max(...rects.map(rect => rect.bottom)), maxRight: Math.max(...rects.map(rect => rect.right)), minX: Math.min(...rects.map(rect => rect.x)), rows: new Set(rects.map(rect => rect.y)).size, columns: new Set(rects.map(rect => rect.x)).size, scrolling, outside: rects.filter(rect => rect.x < 0 || rect.y < 0 || rect.right > innerWidth || rect.bottom > innerHeight) };
      });
      assert.equal(measure.count, fees ? 19 : 40);
      assert.deepEqual(measure.outside, [], `${width}×${height} 所有字段必须一次可见`);
      assert.deepEqual(measure.scrolling, [], '不允许设置列表内部滚动');
      record(`全量可见 ${fees ? '费用19字段' : '40字段'} ${width}×${height}`, measure);
      if (width === 1440 && !fees) await page.screenshot({ path: '/tmp/roncin-column-layout-1440.png' });
      await close();
    }
  }
  await page.setViewportSize({ width: 1366, height: 768 });
  await page.evaluate(() => {
    const order = Array.from({ length: 40 }, (_, i) => `field${i + 1}`);
    localStorage.setItem('roncin:column-settings:v1:browser-distributed:browser-test:browser-org', JSON.stringify({ order, hidden: [], freeze: { left: order.slice(0, 14), right: [...order.slice(27), 'option'] } }));
  });
  await mount({ count: 40, tableKey: 'browser-distributed' });
  const distribution = await dialog.evaluate(node => ({ fields: Array.from(node.querySelectorAll('[data-column-key]')).map(field => ({ key: field.getAttribute('data-column-key'), bottom: field.getBoundingClientRect().bottom })), dialogBottom: node.getBoundingClientRect().bottom }));
  console.log(JSON.stringify({ distribution }));
  assert(distribution.fields.every(field => field.bottom <= 768), '40字段分布到三个区域仍应全部可见');
  assert(distribution.dialogBottom <= 768, '结构列及底栏也应可直接操作，无需滚页面');
  record('40字段14左13普通13右在1366×768全部可见', { maxBottom: Math.max(...distribution.fields.map(field => field.bottom)), dialogBottom: distribution.dialogBottom });
  await close();
  await page.setViewportSize({ width: 1440, height: 900 });
  await mount({ count: 12, tableKey: 'browser-interaction' });
  const keys = Array.from({ length: 12 }, (_, i) => `field${i + 1}`);
  const firstBox = await row('field1').boundingBox();
  await row('field12').getByLabel('拖动业务字段12').dragTo(row('field1'), { targetPosition: { x: 3, y: firstBox.height / 2 } });
  await assertOrder(['field12', ...keys.slice(0, -1)]);
  record('真实跨行拖动末列到首位');

  await selectFreeze('field1', '左冻结');
  await selectFreeze('field2', '左冻结');
  await selectFreeze('field11', '右冻结');
  assert.equal(await zone('left').locator('[data-column-key]').count(), 2);
  assert.equal(await zone('right').locator('[data-column-key]').count(), 1);
  await assertOrder(['field1', 'field2', 'field12', ...keys.slice(2, 10), 'field11']);
  record('左右多列冻结与面板表格顺序一致');

  await row('field12').getByLabel('拖动业务字段12').dragTo(row('field2'), { targetPosition: { x: 3, y: 10 } });
  await assertOrder(['field1', 'field12', 'field2', ...keys.slice(2, 10), 'field11']);
  assert.equal(await zone('left').locator('[data-column-key]').count(), 3);
  record('真实跨区域拖动并更新冻结侧');

  const position = row('field10').getByRole('spinbutton');
  await position.fill('1');
  await position.press('Enter');
  await assertOrder(['field10', 'field1', 'field12', 'field2', ...keys.slice(2, 9), 'field11']);
  record('目标序号精确移位并同步冻结区域');

  await row('field1').getByRole('checkbox').uncheck();
  assert(!await page.evaluate(() => window.__effectiveColumns.some(column => column.key === 'field1')));
  await row('field1').getByRole('checkbox').check();
  assert.equal(await row('field1').getByRole('combobox').getAttribute('aria-expanded'), 'false');
  assert(await row('field1').innerText().then(text => text.includes('左冻结')));
  record('隐藏再显示保留冻结选择');

  await dialog.getByPlaceholder('搜索列名').fill('业务字段12');
  assert.equal(await dialog.locator('[data-column-key]').count(), 1);
  assert.equal(await row('field12').getAttribute('data-column-position'), '3');
  assert.equal(await row('field12').getByRole('spinbutton').isDisabled(), true);
  await dialog.getByPlaceholder('搜索列名').fill('');
  record('搜索保留完整编号并禁止排序');
  await close();
  await mount({ count: 12, tableKey: 'browser-interaction' });
  assert.equal(await zone('left').locator('[data-column-key]').count(), 4);
  await dialog.getByRole('button', { name: '恢复默认', exact: true }).click();
  await assertOrder(keys);
  record('重挂载读取偏好及恢复默认');
  await row('field12').getByLabel('拖动业务字段12').dragTo(zone('left').locator('.roncin-column-settings-empty'));
  assert.equal(await zone('left').locator('[data-column-key]').count(), 1);
  await assertOrder(['field12', ...keys.slice(0, -1)]);
  record('真实拖到空冻结区域');
  const beforeCancel = await page.evaluate(() => window.__effectiveColumns.map(column => column.key));
  const handleBox = await row('field1').getByLabel('拖动业务字段01').boundingBox();
  await page.mouse.move(handleBox.x + handleBox.width / 2, handleBox.y + handleBox.height / 2);
  await page.mouse.down();
  await page.mouse.move(handleBox.x + 25, handleBox.y + 10, { steps: 8 });
  await page.keyboard.press('Escape');
  await page.mouse.up();
  assert.deepEqual(await page.evaluate(() => window.__effectiveColumns.map(column => column.key)), beforeCancel);
  record('取消真实拖动不改变顺序');
  await close();

  for (const kind of ['plain', 'pro', 'editable']) {
    await mount({ count: 12, kind });
    await selectFreeze('field1', '左冻结');
    await selectFreeze('field2', '左冻结');
    await selectFreeze('field12', '右冻结');
    await close();
    await page.locator('#table-host .ant-table-tbody tr[data-row-key]').first().waitFor();
    if (kind === 'editable') {
      const input = page.locator('#table-host .ant-table-tbody input').first();
      await input.fill('尚未提交的输入');
      await input.evaluate(node => { window.__editingNode = node; });
      await page.locator('#table-host th [data-column-resize-handle]').first().dblclick();
      assert(await input.evaluate(node => node === window.__editingNode && node.value === '尚未提交的输入'));
      record('可编辑表调宽保持输入节点和未提交值');
    }
    const initial = await page.locator('#table-host').evaluate(node => {
      const header = Array.from(node.querySelectorAll('th'));
      const labels = ['业务字段01', '业务字段02', '业务字段12'];
      return labels.map(label => { const cell = header.find(cell => cell.textContent.includes(label)); const rect = cell.getBoundingClientRect(); return { label, x: rect.x, right: rect.right, position: getComputedStyle(cell).position }; });
    });
    const requestBefore = await page.evaluate(() => window.__requests);
    await page.locator('#table-host').evaluate(node => {
      for (const element of node.querySelectorAll('*')) if (element.scrollWidth > element.clientWidth && ['auto', 'scroll'].includes(getComputedStyle(element).overflowX)) element.scrollLeft = 500;
    });
    await page.waitForFunction(() => Array.from(document.querySelectorAll('#table-host th')).filter(cell => ['业务字段01', '业务字段02', '业务字段12'].some(label => cell.textContent.includes(label))).every(cell => getComputedStyle(cell).position === 'sticky'));
    const final = await page.locator('#table-host').evaluate(node => ['业务字段01', '业务字段02', '业务字段12'].map(label => { const cell = Array.from(node.querySelectorAll('th')).find(cell => cell.textContent.includes(label)); const rect = cell.getBoundingClientRect(); return { label, x: rect.x, right: rect.right, position: getComputedStyle(cell).position }; }));
    for (let i = 0; i < initial.length; i++) assert(Math.abs(initial[i].x - final[i].x) < 2, `${kind} ${initial[i].label}横向滚动应保持定位`);
    assert.equal(await page.evaluate(() => window.__requests), requestBefore);
    record(`${kind} 左右多列实际sticky几何`, { initial, final });
  }
  await mount({ count: 40 });
  await page.setViewportSize({ width: 390, height: 844 });
  const narrow = await dialog.evaluate(node => ({ left: node.getBoundingClientRect().left, right: node.getBoundingClientRect().right, fields: node.querySelectorAll('[data-column-key]').length, documentWidth: document.documentElement.scrollWidth }));
  assert(narrow.left >= 0 && narrow.right <= 390);
  assert.equal(narrow.fields, 40);
  record('窄屏全部字段可访问且面板无横向溢出', narrow);
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.evaluate(() => window.__setFixtureDisabled(true));
  await dialog.waitFor({ state: 'hidden' });
  assert(await page.getByRole('button', { name: '列设置', exact: true }).isDisabled());
  record('编辑禁用时关闭面板并禁用入口');
  assert.deepEqual(fixture.errors, []);
} finally {
  await writeFile(new URL('./browser-verification.json', import.meta.url), JSON.stringify({ results, errors: fixture.errors }, null, 2));
  await fixture.browser.close();
}
