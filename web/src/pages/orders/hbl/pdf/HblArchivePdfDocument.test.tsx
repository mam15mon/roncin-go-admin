// @vitest-environment node
import { writeFile } from 'node:fs/promises';
import path from 'node:path';
import { Font, renderToBuffer } from '@react-pdf/renderer';
import { act, cloneElement } from 'react';
import { beforeAll, describe, expect, it } from 'vitest';
import { HBL_PDF_FONT_FAMILY } from './font';
import { HblArchivePdfDocument } from './HblArchivePdfDocument';
import type { HblPdfModel } from './types';

type LayoutNode = {
  type: string;
  value?: string;
  box?: { top: number; height: number };
  props?: { fixed?: boolean };
  lines?: { box: { height: number } }[];
  children?: LayoutNode[];
};
const publicRoot = path.resolve(import.meta.dirname, '../../../../../public');
beforeAll(() => {
  Font.register({
    family: HBL_PDF_FONT_FAMILY,
    src: path.join(publicRoot, 'fonts/roncin-hbl/RoncinHBLSansSC-Regular.ttf'),
  });
});
function textOf(node: LayoutNode): string {
  return node.type === 'TEXT_INSTANCE'
    ? (node.value ?? '')
    : (node.children ?? []).map(textOf).join('');
}
const model: HblPdfModel = {
  outputKind: 'external-archive',
  houseNo: 'IMPORT-HBL-001',
  issuerNameEn: '',
  issuerDisplayName: 'EXTERNAL ISSUER',
  savedVersion: '7',
  content: {
    shipperText: 'SAMPLE SHIPPER',
    consigneeText: 'SAMPLE CONSIGNEE',
    packageCount: 0,
    grossWeightKg: 123.456,
    volumeCbm: 2.5,
    goodsDescriptionText: 'VALVES',
  },
  printDetails: {
    issueDate: '2026-11-16',
    originalCount: 0,
    containersText: 'TEST1234567 / TEST-SEAL / 20GP',
  },
};
async function renderArchive(input: HblPdfModel) {
  let layout: LayoutNode | undefined;
  const document = HblArchivePdfDocument({ model: input });
  const bytes = await act(async () =>
    renderToBuffer(
      cloneElement(document, {
        onRender: (data: unknown) => {
          layout = (data as { _INTERNAL__LAYOUT__DATA_: LayoutNode })
            ._INTERNAL__LAYOUT__DATA_;
        },
      }),
    ),
  );
  if (!layout) throw new Error('没有取得PDF实际渲染布局');
  function checkTextBox(node: LayoutNode) {
    if (node.type === 'TEXT' && node.lines?.length && node.box) {
      const actualLineHeight = node.lines.reduce(
        (total, line) => total + line.box.height,
        0,
      );
      // 正文字符存在不等于可见：实际行框必须容纳字形且落入文字容器。
      expect(actualLineHeight).toBeGreaterThanOrEqual(7);
      expect(actualLineHeight).toBeLessThanOrEqual(node.box.height + 0.1);
    }
    for (const child of node.children ?? []) checkTextBox(child);
  }
  checkTextBox(layout);
  return { bytes, layout };
}
describe('外部 HBL 归档 PDF 实际渲染', () => {
  it('A4归档资料保存真实主体/版本/显式零，未套用公司签发条款或正本电放标识', async () => {
    const { bytes, layout } = await renderArchive(model);
    const text = textOf(layout).replaceAll('\n', '');
    expect(bytes.subarray(0, 8).toString()).toBe('%PDF-1.3');
    expect(text).toContain('EXTERNAL BILL OF LADING RECORD (ARCHIVE)');
    expect(text).toContain('EXTERNAL ISSUER');
    expect(text).toContain('IMPORT-HBL-001');
    expect(text).not.toMatch(/\p{Script=Han}/u);
    expect(text).toContain('2026-11-16');
    expect(text).toContain('123.456');
    expect(text).not.toContain('COMBINED TRANSPORT BILL OF LADING');
    expect(text).not.toContain('SURRENDERED');
    expect(text).not.toContain('RONCIN');
    // 保存合成样张供独立提字与实图验收，不进入仓库。
    await writeFile('/tmp/roncin-si-hbl-archive.pdf', bytes);
  });

  it('长多行号码正文完整，固定页脚仅展示有界摘要并保留版本', async () => {
    const houseNo = `${'NO'.repeat(40)}\nFULL-NUMBER-END`;
    const { bytes, layout } = await renderArchive({ ...model, houseNo });
    const text = textOf(layout);
    expect(text.replaceAll('\n', '')).toContain(houseNo.replaceAll('\n', ''));
    const page = layout.children?.[0];
    const footer = page?.children?.find(
      (node) => node.props?.fixed && textOf(node).includes('Saved version'),
    );
    if (!footer) throw new Error('缺少固定归档页脚');
    expect(textOf(footer)).toContain(`HBL ${'NO'.repeat(16)}…`);
    expect(textOf(footer)).not.toContain('FULL-NUMBER-END');
    await writeFile('/tmp/roncin-si-hbl-archive-long-number.pdf', bytes);
  });

  it('连续英文和多行续页完整，实际页面正文边界内无裁切', async () => {
    const input = {
      ...model,
      content: {
        ...model.content,
        shipperText: 'qqq'.repeat(1000),
        goodsDescriptionText: `${'zz'.repeat(3000)}\nARCHIVE-END-MARKER`,
      },
      printDetails: {
        ...model.printDetails,
        containersText: Array.from(
          { length: 110 },
          (_, index) =>
            `TEST${String(index).padStart(7, '0')} / SEAL-${index} / 20GP`,
        ).join('\n'),
      },
    };
    const { bytes, layout } = await renderArchive(input);
    const text = textOf(layout).replaceAll('\n', '');
    expect(layout.children?.length).toBeGreaterThan(3);
    expect(text.match(/q/g)).toHaveLength(3000);
    expect(text.match(/z/g)).toHaveLength(6000);
    expect(text).toContain('ARCHIVE-END-MARKER');
    expect(text).toContain('TEST0000109 / SEAL-109 / 20GP');
    for (const page of layout.children ?? []) {
      const pageText = textOf(page);
      expect(pageText).toContain('HBL IMPORT-HBL-001');
      expect(pageText).toContain('Saved version 7');
      for (const node of page.children ?? []) {
        if (node.props?.fixed || !node.box) continue;
        expect(node.box.top + node.box.height).toBeLessThan(802);
      }
    }
    await writeFile('/tmp/roncin-si-hbl-archive-long.pdf', bytes);
  });
});
