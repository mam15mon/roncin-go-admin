import { readFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import path from 'node:path';
import { expect, test } from '@playwright/test';

const requireDependency = createRequire(__filename);
const less = requireDependency('less') as {
  render: (source: string) => Promise<{ css: string }>;
};

test('三档密度下测量行保持零高，嵌套表头与正常内容保持可见', async ({
  page,
}) => {
  const source = await readFile(
    path.resolve(__dirname, '../../src/global.less'),
    'utf8',
  );
  const { css } = await less.render(source);
  for (const size of ['small', 'middle', 'large']) {
    for (const bodyClass of ['ant-table-content', 'ant-table-body']) {
      await page.setContent(`
        <style>${css}</style>
        <div class="ant-table-wrapper">
          <div class="ant-table ant-table-${size}">
            <div class="${bodyClass}">
              <table style="width: 100%">
                <thead class="ant-table-thead"><tr><th>客户名称</th></tr></thead>
                <tbody class="ant-table-tbody">
                  <tr aria-hidden="true" class="ant-table-measure-row" style="height: 0">
                    <td style="padding: 0; border: 0; height: 0"><div style="height: 0; overflow: hidden">客户名称</div></td>
                  </tr>
                  <tr class="ant-table-row"><td>客户内容</td></tr>
                  <tr class="ant-table-expanded-row"><td>
                    <div class="ant-table-wrapper"><div class="ant-table ant-table-${size}">
                      <div class="ant-table-content"><table>
                        <thead class="ant-table-thead"><tr><th>嵌套表头</th></tr></thead>
                        <tbody class="ant-table-tbody"><tr class="ant-table-row"><td>嵌套内容</td></tr></tbody>
                      </table></div>
                    </div></div>
                  </td></tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      `);
      const dimensions = await page
        .locator('.ant-table-measure-row')
        .evaluate((row) => {
          const cell = row.firstElementChild;
          if (!cell) throw new Error('测量单元格缺失');
          return {
            rowHeight: row.getBoundingClientRect().height,
            padding: getComputedStyle(cell).padding,
          };
        });
      expect(dimensions, `${size} / ${bodyClass}`).toEqual({
        rowHeight: 0,
        padding: '0px',
      });
      await expect(page.getByText('客户内容', { exact: true })).toBeVisible();
      await expect(
        page.getByRole('columnheader', { name: '嵌套表头' }),
      ).toBeVisible();
      await expect(page.getByText('嵌套内容', { exact: true })).toBeVisible();
    }
  }
});
