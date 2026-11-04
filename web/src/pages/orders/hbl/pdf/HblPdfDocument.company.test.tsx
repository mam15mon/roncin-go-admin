// @vitest-environment node
import path from 'node:path';
import { Font, renderToBuffer } from '@react-pdf/renderer';
import { cloneElement } from 'react';
import { beforeAll, describe, expect, it } from 'vitest';
import { HBL_PDF_FONT_FAMILY } from './font';
import { HblPdfDocument } from './HblPdfDocument';
import {
  HBL_PDF_VARIANT_MARKS,
  type HblPdfModel,
  type HblPdfVariant,
} from './types';

type LayoutNode = {
  type: string;
  value?: string;
  box?: { top: number; left: number; width: number; height: number };
  props?: { fixed?: boolean };
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

async function renderCompany(model: HblPdfModel, variant: HblPdfVariant) {
  let layout: LayoutNode | undefined;
  const document = HblPdfDocument({
    model,
    variant,
    logoSrc: path.join(publicRoot, 'images/roncin-hbl-logo.png'),
  });
  const bytes = await renderToBuffer(
    cloneElement(document, {
      onRender: (data: unknown) => {
        // 实际布局探针仅用于检查渲染结果，不进入产品代码。
        layout = (data as { _INTERNAL__LAYOUT__DATA_: LayoutNode })
          ._INTERNAL__LAYOUT__DATA_;
      },
    }),
  );
  if (!layout) throw new Error('未取得实际 PDF 布局');
  return { bytes, layout };
}

describe('HBL PDF 签发公司抬头', () => {
  it('四版本使用传入公司的抬头、签发条款及文件作者，不残留北京抬头', async () => {
    const variants = Object.keys(HBL_PDF_VARIANT_MARKS) as HblPdfVariant[];
    for (const [index, variant] of variants.entries()) {
      const issuerNameEn =
        index % 2 === 0
          ? 'SAMPLE EAST LOGISTICS CO., LTD'
          : 'SAMPLE WEST FORWARDING CO., LTD';
      const { bytes, layout } = await renderCompany(
        {
          houseNo: 'HBL-COMPANY-001',
          issuerNameEn,
          content: { shipperText: '真实发货人 / SAMPLE SHIPPER' },
        },
        variant,
      );
      expect(layout.children).toHaveLength(1);
      const text = textOf(layout).replaceAll('\n', '');
      expect(text.split(issuerNameEn)).toHaveLength(4);
      expect(text).not.toContain('RONCIN (BEIJING)');
      expect(text).toContain(HBL_PDF_VARIANT_MARKS[variant]);
      const pdf = bytes.toString('latin1');
      const authorObject = pdf.match(/\/Author (\d+) 0 R/)?.[1];
      expect(authorObject).toBeDefined();
      expect(pdf).toContain(`${authorObject} 0 obj\n(${issuerNameEn})\nendobj`);
    }
  });

  it('200 字符无空白名称完整换行，并为各页抬头预留空间', async () => {
    const issuerNameEn = 'Z'.repeat(200);
    const { layout } = await renderCompany(
      {
        houseNo: 'HBL-LONG-COMPANY-001',
        issuerNameEn,
        content: {
          shipperText: '多页抬头验收发货人',
          goodsDescriptionText: '货描续页验证'.repeat(600),
        },
      },
      'original',
    );
    const pages = layout.children ?? [];
    expect(pages.length).toBeGreaterThan(1);
    expect(pages.length).toBeLessThan(8);
    expect([...textOf(layout)].filter((char) => char === 'Z')).toHaveLength(
      200 * (pages.length + 2),
    );
    for (const page of pages) {
      const header = page.children?.find((node) => node.props?.fixed);
      expect(header?.box).toBeDefined();
      const company = header?.children?.[0];
      const subtitle = header?.children?.[1];
      const logo = header?.children?.[2];
      expect(textOf(company as LayoutNode)).toBe(issuerNameEn);
      expect(company?.box).toBeDefined();
      expect(subtitle?.box).toBeDefined();
      expect(logo?.box).toBeDefined();
      expect(
        (company?.box?.left ?? 0) + (company?.box?.width ?? 0),
      ).toBeLessThan(logo?.box?.left ?? 0);
      expect(
        (company?.box?.top ?? 0) + (company?.box?.height ?? 0),
      ).toBeLessThanOrEqual(subtitle?.box?.top ?? 0);
      const headerBottom = (header?.box?.top ?? 0) + (header?.box?.height ?? 0);
      const body = page.children?.filter((node) => !node.props?.fixed) ?? [];
      for (const node of body) {
        if (!node.box) continue;
        expect(node.box.top).toBeGreaterThan(headerBottom);
        expect(node.box.top + node.box.height).toBeLessThan(797);
      }
    }
    expect([...textOf(layout)].filter((char) => char === '描')).toHaveLength(
      600,
    );
  });

  it('直接渲染也拒绝空抬头，不生成缺少签发主体的文件', () => {
    expect(() =>
      HblPdfDocument({
        model: { houseNo: 'HBL-COMPANY-001', issuerNameEn: '' },
        variant: 'draft',
      }),
    ).toThrow('签发公司尚未维护英文名称，请在公司档案中补齐后输出。');
  });
});
