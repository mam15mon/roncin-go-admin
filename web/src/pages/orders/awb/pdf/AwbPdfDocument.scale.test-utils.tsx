import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { Font, renderToBuffer } from '@react-pdf/renderer';
import { act, cloneElement } from 'react';
import { expect } from 'vitest';
import { ORDER_PDF_FONT_FAMILY } from '../../documents/pdf/font';
import { AwbPdfDocument } from './AwbPdfDocument';
import type { AwbPdfModel } from './types';

type Node = {
  type: string;
  value?: string;
  children?: Node[];
  box?: { top: number; left: number; height: number };
  props?: { fixed?: boolean };
};

export async function verifyScale(
  model: AwbPdfModel,
  name: string,
  minimumPages: number,
  markers: readonly string[],
) {
  Font.register({
    family: ORDER_PDF_FONT_FAMILY,
    src: path.resolve(
      import.meta.dirname,
      '../../../../../public/fonts/roncin-hbl/RoncinHBLSansSC-Regular.ttf',
    ),
  });
  let layout: Node | undefined;
  let bytes: Buffer = Buffer.alloc(0);
  await act(async () => {
    bytes = await renderToBuffer(
      cloneElement(AwbPdfDocument({ model }), {
        onRender: (data: unknown) => {
          layout = (data as { _INTERNAL__LAYOUT__DATA_: Node })
            ._INTERNAL__LAYOUT__DATA_;
        },
      }),
    );
  });
  // 先留独立验证样张，断言失败仍能检查最终实际字形；不进入 Git。
  if (process.env.AWB_PDF_VALIDATION_DIR) {
    await mkdir(process.env.AWB_PDF_VALIDATION_DIR, { recursive: true });
    await writeFile(
      path.join(process.env.AWB_PDF_VALIDATION_DIR, `${name}.pdf`),
      bytes,
    );
    await writeFile(
      path.join(process.env.AWB_PDF_VALIDATION_DIR, `${name}.json`),
      JSON.stringify(model),
    );
  }
  const pages = layout?.children ?? [];
  expect(pages.length).toBeGreaterThanOrEqual(minimumPages);
  expect(pages.length).toBeLessThan(60);
  const columns = new Map<number, string>();
  function collect(node: Node, left = 0) {
    const x = left + (node.box?.left ?? 0);
    if (node.type === 'TEXT_INSTANCE')
      columns.set(x, (columns.get(x) ?? '') + (node.value ?? ''));
    for (const child of node.children ?? []) collect(child, x);
  }
  for (const page of pages) {
    collect(page);
    for (const node of page.children ?? []) {
      if (!node.box || node.props?.fixed) continue;
      expect(node.box.top).toBeGreaterThanOrEqual(70);
      expect(node.box.top + node.box.height).toBeLessThan(795);
    }
  }
  // 双列的末尾可能在不同短行，按相同水平坐标收集，避免穿插另一列误判。
  const text = [...columns.values()]
    .map((value) => value.replaceAll(/\s/gu, ''))
    .join('\n');
  expect(text).not.toMatch(/\p{Script=Han}/u);
  for (const marker of markers)
    expect(
      text.includes(marker.replaceAll(/\s/gu, '')),
      `缺少标记 ${marker}`,
    ).toBe(true);
  // 布局树只作回归探针；真实字形另以独立提字和逐页图片验收。
}
