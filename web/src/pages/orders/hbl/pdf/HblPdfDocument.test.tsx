// @vitest-environment node
import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { Font, renderToBuffer } from '@react-pdf/renderer';
import { beforeAll, describe, expect, it } from 'vitest';
import { HBL_PDF_FONT_FAMILY } from './font';
import { HblPdfDocument } from './HblPdfDocument';
import type { HblPdfModel } from './types';

const publicRoot = path.resolve(import.meta.dirname, '../../../../../public');
const model: HblPdfModel = {
  houseNo: 'HBL-TEST-001',
  content: {
    shipperText: '测试发货企业 SAMPLE EXPORTER\nSHANGHAI, CHINA',
    consigneeText: 'TO ORDER',
    notifyPartyText: '测试通知人 SAMPLE NOTIFY PARTY',
    marksText: 'TEST CARGO 1-11',
    goodsDescriptionText: 'VALVES AND SPARE PARTS\n阀门及配件',
    packageCount: 11,
    packageUnit: 'CASES',
    grossWeightKg: 10918,
    volumeCbm: 21.56,
    freightTerms: '预付',
    transportTerms: 'CY-CY',
  },
  printDetails: {
    bookingNoText: 'BOOKING-001',
    vesselVoyageText: 'SAMPLE VESSEL 2601E',
    portOfLoadingText: 'SHANGHAI, CHINA',
    portOfDischargeText: 'SINGAPORE',
    containersText: 'TEST1234567 / SEAL-001 / 20GP',
    originalCount: 3,
    issuePlaceText: 'BEIJING',
    issueDate: '2026-11-04',
    shippedOnBoardDate: '2026-11-03',
  },
};

beforeAll(() => {
  Font.register({
    family: HBL_PDF_FONT_FAMILY,
    src: path.join(publicRoot, 'fonts/roncin-hbl/RoncinHBLSansSC-Regular.ttf'),
  });
});

function countPages(bytes: Buffer) {
  return bytes.toString('latin1').match(/\/Type \/Page\b/g)?.length ?? 0;
}

describe('HBL PDF 实际渲染', () => {
  it('使用仓库 React 和嵌入中文字体生成单页 A4 提单', async () => {
    const bytes = await renderToBuffer(
      <HblPdfDocument
        model={model}
        variant="original"
        logoSrc={path.join(publicRoot, 'images/roncin-hbl-logo.png')}
      />,
    );
    expect(bytes.subarray(0, 8).toString()).toBe('%PDF-1.3');
    expect(countPages(bytes)).toBe(1);
    const dimensions = bytes
      .toString('latin1')
      .match(/\/MediaBox \[0 0 ([\d.]+) ([\d.]+)\]/);
    expect(Number(dimensions?.[1])).toBeCloseTo(595.28, 2);
    expect(Number(dimensions?.[2])).toBeCloseTo(841.89, 2);
    expect(bytes.toString('latin1')).toContain('/FontFile2');
    const source = Font.getFont({ fontFamily: HBL_PDF_FONT_FAMILY });
    expect(
      source.data?.hasGlyphForCodePoint('阀'.codePointAt(0) as number),
    ).toBe(true);
  });

  it('长联系方式、地址、货描和箱号正常续页，不产生整页不可分页块', async () => {
    const longModel: HblPdfModel = {
      ...model,
      content: {
        ...model.content,
        shipperText: Array.from(
          { length: 55 },
          (_, i) => `发货地址第${i + 1}行 SAMPLE ADDRESS`,
        ).join('\n'),
        goodsDescriptionText: Array.from(
          { length: 110 },
          (_, i) => `货物明细第${i + 1}行 VALVE PARTS`,
        ).join('\n'),
      },
      printDetails: {
        ...model.printDetails,
        headerContactText: Array.from(
          { length: 50 },
          (_, i) => `联系人第${i + 1}行 CONTACT`,
        ).join('\n'),
        containersText: Array.from(
          { length: 55 },
          (_, i) => `TEST${String(i).padStart(7, '0')} / SEAL / 20GP`,
        ).join('\n'),
        footerReferenceText: '参考信息'.repeat(60),
      },
    };
    const bytes = await renderToBuffer(
      <HblPdfDocument
        model={longModel}
        variant="draft"
        logoSrc={path.join(publicRoot, 'images/roncin-hbl-logo.png')}
      />,
    );
    expect(countPages(bytes)).toBeGreaterThan(3);
    expect(countPages(bytes)).toBeLessThan(10);
    expect(bytes.length).toBeGreaterThan(10_000);
  });

  it('字体资产为静态字重，并附有再分发授权', async () => {
    const font = await readFile(
      path.join(publicRoot, 'fonts/roncin-hbl/RoncinHBLSansSC-Regular.ttf'),
    );
    const license = await readFile(
      path.join(publicRoot, 'fonts/roncin-hbl/OFL.txt'),
      'utf8',
    );
    expect(font.subarray(0, 4)).toEqual(Buffer.from([0, 1, 0, 0]));
    expect(license).toContain('SIL OPEN FONT LICENSE Version 1.1');
  });

  it('含多次换行的号码在正文正常续页，固定页脚不会撑满页面', async () => {
    const bytes = await renderToBuffer(
      <HblPdfDocument
        model={{ ...model, houseNo: '号\n'.repeat(64) }}
        variant="copy"
        logoSrc={path.join(publicRoot, 'images/roncin-hbl-logo.png')}
      />,
    );
    expect(countPages(bytes)).toBeGreaterThan(1);
    expect(countPages(bytes)).toBeLessThan(5);
  });
});
