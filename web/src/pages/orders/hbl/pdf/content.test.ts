import { describe, expect, it } from 'vitest';
import {
  hblPdfFileName,
  pdfDate,
  pdfQuantity,
  pdfText,
  validateHblPdfCharacters,
} from './content';

describe('HBL PDF 显示边界', () => {
  it('中文、英文和换行可打印，缺字说明定位到实际字段', () => {
    expect(
      validateHblPdfCharacters({
        houseNo: 'HBL-2026-001',
        issuerNameEn: 'SAMPLE EAST LOGISTICS CO., LTD',
        content: { shipperText: '上海测试企业\nSAMPLE EXPORTER' },
        printDetails: { containersText: 'ABCD1234567 / 20GP' },
      }),
    ).toBeNull();
    expect(
      validateHblPdfCharacters({
        houseNo: 'HBL-001',
        issuerNameEn: 'SAMPLE EAST LOGISTICS CO., LTD',
        printDetails: { deliveryContactText: '联系人😀' },
      }),
    ).toBe('交货联系信息含暂不支持打印的字符「😀」，请调整后重试。');
  });

  it('缺少签发英文名或公司名含缺字时明确阻止输出', () => {
    expect(
      validateHblPdfCharacters({ houseNo: 'HBL-001', issuerNameEn: '   ' }),
    ).toBe('签发公司尚未维护英文名称，请在公司档案中补齐后输出。');
    expect(
      validateHblPdfCharacters({
        houseNo: 'HBL-001',
        issuerNameEn: 'SAMPLE COMPANY 😀',
      }),
    ).toBe('签发公司英文名称含暂不支持打印的字符「😀」，请调整后重试。');
  });

  it('数量保留有效精度，日期按纯日期显示，空值不伪造零或当前日期', () => {
    expect(pdfQuantity(6.7183, 'CBM')).toBe('6.7183 CBM');
    expect(pdfQuantity(0, 'KGS')).toBe('0 KGS');
    expect(pdfQuantity(undefined, 'KGS')).toBe('');
    expect(pdfDate('2026-11-04')).toBe('2026-11-04');
    expect(pdfDate()).toBe('');
    const longNo = '海'.repeat(128);
    const wrapped = pdfText(longNo, 16);
    expect(wrapped.split('\n')).toHaveLength(8);
    expect(wrapped.replaceAll('\n', '')).toBe(longNo);
    const wideText = '⸻'.repeat(16);
    const wideWrapped = pdfText(wideText, 16);
    expect(wideWrapped.split('\n')).toHaveLength(3);
    expect(wideWrapped.replaceAll('\n', '')).toBe(wideText);
  });

  it('下载名保留真实号码和版本，移除文件名非法字符并使用可读本地时间', () => {
    const fileName = hblPdfFileName(
      'HBL/01:02',
      'surrendered',
      new Date(2026, 10, 4, 9, 8, 7),
    );
    expect(fileName).toBe('HBL-HBL_01_02-surrendered-2026-11-04 09-08-07.pdf');
  });
});
