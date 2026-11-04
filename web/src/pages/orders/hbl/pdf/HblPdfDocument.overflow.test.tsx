// @vitest-environment node
import path from 'node:path';
import { Font, renderToBuffer } from '@react-pdf/renderer';
import { cloneElement } from 'react';
import { expect, it } from 'vitest';
import { HBL_PDF_FONT_FAMILY } from './font';
import { HblPdfDocument } from './HblPdfDocument';

type LayoutNode = {
  type: string;
  value?: string;
  box?: { top: number; left: number; width: number; height: number };
  props?: { fixed?: boolean };
  children?: LayoutNode[];
};

const publicRoot = path.resolve(import.meta.dirname, '../../../../../public');

it('长度上限的连续中文按短行跨页，正文全部保留且行框不会互相压缩', async () => {
  Font.register({
    family: HBL_PDF_FONT_FAMILY,
    src: path.join(publicRoot, 'fonts/roncin-hbl/RoncinHBLSansSC-Regular.ttf'),
  });
  let layout: LayoutNode | undefined;
  const document = HblPdfDocument({
    model: {
      houseNo: '⸻'.repeat(128),
      content: {
        shipperText: '无空白地址'.repeat(1000),
        goodsDescriptionText: '连续货描'.repeat(2500),
        marksText: '连续唛头'.repeat(1250),
        packageCount: 10,
        packageUnit: '包装单位'.repeat(25),
        grossWeightKg: 123456789012345,
        volumeCbm: 123456789012345,
      },
      printDetails: {
        bookingNoText: '宽⸻'.repeat(250),
        exportReferencesText: '出口参考'.repeat(750),
        containersText: '箱号明细'.repeat(3000),
      },
    },
    variant: 'copy',
    logoSrc: path.join(publicRoot, 'images/roncin-hbl-logo.png'),
  });
  const bytes = await renderToBuffer(
    cloneElement(document, {
      onRender: (data: unknown) => {
        // 渲染器内部布局只用于回归探针，生产代码不依赖其私有结构。
        layout = (data as { _INTERNAL__LAYOUT__DATA_: LayoutNode })
          ._INTERNAL__LAYOUT__DATA_;
      },
    }),
  );
  expect(bytes.subarray(0, 8).toString()).toBe('%PDF-1.3');
  const pages = layout?.children ?? [];
  expect(pages.length).toBeGreaterThan(10);
  expect(pages.length).toBeLessThan(25);

  let text = '';
  function collect(node: LayoutNode) {
    if (node.type === 'TEXT_INSTANCE') text += node.value ?? '';
    for (const child of node.children ?? []) collect(child);
  }
  for (const page of pages) {
    let bottom = 98;
    for (const node of page.children ?? []) {
      collect(node);
      if (node.props?.fixed || !node.box) continue;
      expect(node.box.top).toBeGreaterThanOrEqual(bottom - 0.05);
      expect(node.box.top + node.box.height).toBeLessThan(782);
      bottom = node.box.top + node.box.height;
      if (node.type !== 'VIEW') continue;
      // 多列短行的文字高度必须落在行框内，防止高列越过下方行。
      for (const column of node.children ?? []) {
        if (!column.box) continue;
        expect(column.box.height).toBeLessThanOrEqual(node.box.height + 0.05);
        for (const cellText of column.children ?? []) {
          if (!cellText.box) continue;
          expect(cellText.box.top + cellText.box.height).toBeLessThanOrEqual(
            column.box.height + 0.05,
          );
        }
      }
    }
  }
  for (const [char, count] of [
    ['无', 1000],
    ['空', 1000],
    ['白', 1000],
    ['址', 1000],
    ['描', 2500],
    ['唛', 1250],
    ['头', 1250],
    ['出', 750],
    ['参', 750],
    ['箱', 3000],
    ['号', 3000],
  ] as const) {
    expect([...text].filter((value) => value === char)).toHaveLength(count);
  }
}, 30_000);
