// @vitest-environment node
import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { Font, renderToBuffer } from '@react-pdf/renderer';
import { act, cloneElement } from 'react';
import { beforeAll, expect, it } from 'vitest';
import { ORDER_PDF_FONT_FAMILY } from '../../documents/pdf/font';
import { AwbPdfDocument } from './AwbPdfDocument';
import { awbPdfSample } from './AwbPdfDocument.fixtures';

beforeAll(() =>
  Font.register({
    family: ORDER_PDF_FONT_FAMILY,
    src: path.resolve(
      import.meta.dirname,
      '../../../../../public/fonts/roncin-hbl/RoncinHBLSansSC-Regular.ttf',
    ),
  }),
);

it('已保存航空分单使用嵌入字体生成纯英文真实 A4，主/分和归档版共用完整分区', async () => {
  for (const target of ['HAWB', 'MAWB'] as const) {
    const model = {
      ...awbPdfSample,
      target,
      outputKind:
        target === 'MAWB'
          ? ('EXTERNAL_ARCHIVE' as const)
          : ('SELF_ISSUED' as const),
    };
    type LayoutNode = { type: string; value?: string; children?: LayoutNode[] };
    let layout: LayoutNode | undefined;
    const bytes = await act(async () =>
      renderToBuffer(
        cloneElement(AwbPdfDocument({ model }), {
          onRender: (data: unknown) => {
            layout = (data as { _INTERNAL__LAYOUT__DATA_: LayoutNode })
              ._INTERNAL__LAYOUT__DATA_;
          },
        }),
      ),
    );
    function textOf(node: LayoutNode): string {
      return node.type === 'TEXT_INSTANCE'
        ? (node.value ?? '')
        : (node.children ?? []).map(textOf).join('');
    }
    expect(layout).toBeDefined();
    expect(textOf(layout as LayoutNode)).not.toMatch(/\p{Script=Han}/u);
    expect(bytes.subarray(0, 8).toString()).toBe('%PDF-1.3');
    expect(bytes.toString('latin1')).toContain('/FontFile2');
    const pages = bytes.toString('latin1').match(/\/Type \/Page\b/g)?.length;
    expect(pages).toBeGreaterThanOrEqual(1);
    expect(pages).toBeLessThan(5);
    if (process.env.AWB_PDF_VALIDATION_DIR) {
      await mkdir(process.env.AWB_PDF_VALIDATION_DIR, { recursive: true });
      await writeFile(
        path.join(process.env.AWB_PDF_VALIDATION_DIR, `normal-${target}.pdf`),
        bytes,
      );
      await writeFile(
        path.join(process.env.AWB_PDF_VALIDATION_DIR, `normal-${target}.json`),
        JSON.stringify(model),
      );
    }
  }
});
