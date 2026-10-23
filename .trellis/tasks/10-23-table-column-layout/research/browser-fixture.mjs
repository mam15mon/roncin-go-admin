// 使用 Vite 提供的真实公共组件，隔离业务接口与真实用户偏好进行浏览器几何验收。
import { createRequire } from 'node:module';
import { fileURLToPath } from 'node:url';
import { writeFile } from 'node:fs/promises';

const require = createRequire(`${process.cwd()}/web/package.json`);
const { chromium } = require('@playwright/test');
export const baseURL = 'http://127.0.0.1:8001';

export async function openFixture() {
  const browser = await chromium.launch({ headless: true, args: ['--no-sandbox'] });
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 }, timezoneId: 'Asia/Shanghai' });
  const page = await context.newPage();
  page.setDefaultTimeout(15000);
  const errors = [];
  page.on('pageerror', (error) => errors.push(error.message));
  await page.route('**/__column-layout-browser', (route) => route.fulfill({
    contentType: 'text/html',
    body: '<!doctype html><html lang="zh"><head><meta charset="UTF-8"><style>body{margin:0;padding:12px;box-sizing:border-box;font-family:Arial,sans-serif}*{box-sizing:border-box}</style></head><body><div id="root"></div></body></html>',
  }));
  await page.goto(`${baseURL}/__column-layout-browser`);
  await page.evaluate(async () => {
    const refresh = await import('/@react-refresh');
    refresh.default.injectIntoGlobalHook(window);
    window.$RefreshReg$ = () => {};
    window.$RefreshSig$ = () => (type) => type;
    window.__vite_plugin_react_preamble_installed__ = true;
    const React = (await import('/node_modules/.vite/deps/react.js')).default;
    const domModule = await import('/node_modules/.vite/deps/react-dom_client.js');
    const ReactDOM = domModule.default ?? domModule;
    const { App, ConfigProvider, Input } = await import('/node_modules/.vite/deps/antd.js');
    const { useColumnSettings } = await import('/src/components/ui/column-settings/index.ts');
    const tables = await import('/src/components/ui/responsive-table/index.ts');
    const { DEFAULT_FEE_COLUMN_DEFS } = await import('/src/pages/orders/components/fees/feeColumnPreference.ts');
    const h = React.createElement;
    const root = ReactDOM.createRoot(document.getElementById('root'));
    let instance = 0;
    window.__requests = 0;
    window.__setFixtureDisabled = () => {};
    window.__mount = (config = {}) => {
      const count = config.count ?? 40;
      const kind = config.kind ?? 'plain';
      const fields = config.fees
        ? DEFAULT_FEE_COLUMN_DEFS.filter((field) => field.key !== 'option').map((field) => ({ key: field.key, title: field.title }))
        : Array.from({ length: count }, (_, index) => ({ key: `field${index + 1}`, title: `业务字段${String(index + 1).padStart(2, '0')}` }));
      const columns = fields.map((field) => ({ ...field, dataIndex: field.key, width: 150, ...(kind === 'editable' ? { renderFormItem: () => h(Input) } : {}) }));
      columns.push({ key: 'option', title: '操作', valueType: 'option', width: 100, fixed: 'right', render: () => '查看' });
      const data = [{ id: 'browser-row', ...Object.fromEntries(fields.map((field) => [field.key, `${field.title}内容`])) }];
      const tableKey = config.tableKey ?? `browser-layout-${++instance}`;
      function Fixture() {
        const [disabled, setDisabled] = React.useState(false);
        window.__setFixtureDisabled = setDisabled;
        const settings = useColumnSettings({ tableKey, columns, disabled, disabledReason: '验收编辑中', scope: { userId: 'browser-test', organizationId: 'browser-org' } });
        window.__effectiveColumns = settings.columns;
        const Table = kind === 'pro' ? tables.ResponsiveProTable : kind === 'editable' ? tables.ResponsiveEditableProTable : tables.ResponsiveTable;
        const request = React.useCallback(async () => {
          window.__requests += 1;
          return { data, total: data.length, success: true };
        }, []);
        const props = {
          columns: settings.columns,
          rowKey: 'id',
          pagination: false,
          sticky: false,
          search: false,
          options: { setting: false, density: false, fullScreen: false, reload: false },
          ...(kind === 'plain' ? { dataSource: data } : kind === 'pro' ? { request } : {
            value: data, recordCreatorProps: false,
            editable: { type: 'multiple', editableKeys: ['browser-row'], actionRender: () => [] },
          }),
        };
        return h('div', null, h('div', { id: 'settings-entry', style: { display: 'flex', justifyContent: 'flex-end', marginBottom: 8 } }, settings.entry), h('div', { id: 'table-host' }, h(Table, props)));
      }
      root.render(h(ConfigProvider, { theme: { token: { motion: false } } }, h(App, null, h(Fixture, { key: instance }))));
    };
  });
  return { browser, context, page, errors };
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const fixture = await openFixture();
  try {
    await fixture.page.evaluate(() => window.__mount());
    await fixture.page.getByRole('button', { name: '列设置', exact: true }).click();
    console.log(await fixture.page.getByRole('dialog').innerText());
    await writeFile('/tmp/roncin-column-browser-bootstrap.json', JSON.stringify({ errors: fixture.errors }));
  } finally {
    await fixture.browser.close();
  }
}
