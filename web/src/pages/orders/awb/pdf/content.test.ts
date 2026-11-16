import { describe, expect, it } from 'vitest';
import { awbPdfSample } from './AwbPdfDocument.fixtures';
import {
  awbAmount,
  awbDecimal,
  awbDeclared,
  awbNumberSummary,
  awbPdfFileName,
  awbPdfFingerprint,
  validateAwbPdf,
} from './content';

describe('航空 PDF 输出事实', () => {
  it('零值、未知与声明令牌分别输出，数量/单价/汇率保留有效精度', () => {
    expect(awbDeclared({ kind: 'AMOUNT', amount: '0' })).toBe('0.00');
    expect(awbDeclared()).toBe('-');
    for (const kind of ['NVD', 'NCV', 'XXX'])
      expect(awbDeclared({ kind })).toBe(kind);
    expect(awbAmount('15362.50000000')).toBe('15,362.50');
    expect(awbAmount('')).toBe('-');
    expect(awbAmount('0')).toBe('0.00');
    expect(awbDecimal('1234567890123456.12345678')).toBe(
      '1234567890123456.12345678',
    );
    expect(awbDecimal('6.71830000')).toBe('6.7183');
  });
  it('递归拦截明细末行缺字，不借主体或错误模板生成文件', () => {
    expect(validateAwbPdf(awbPdfSample)).toBeNull();
    expect(
      validateAwbPdf({ ...awbPdfSample, issuerDisplayName: '' }),
    ).toContain('真实签发主体');
    expect(validateAwbPdf({ ...awbPdfSample, target: 'MAWB' })).toContain(
      '不适用',
    );
    expect(
      validateAwbPdf({
        ...awbPdfSample,
        content: { otherCharges: [{ name: '中文😀' }] },
      }),
    ).toContain('其他费用第 1 行 · 费用名称');
  });
  it('签发主体、号码与嵌套明细均拒绝中文，保留字段路径', () => {
    for (const delta of [
      { houseNo: '中文号' },
      { issuerDisplayName: '中文签发方' },
      { content: { shipperText: '中文发货人' } },
      { content: { destinationText: '新加坡' } },
      { content: { rateLines: [{ goodsDescription: '中文品名' }] } },
      { content: { otherCharges: [{ name: '中文费用' }] } },
    ])
      expect(validateAwbPdf({ ...awbPdfSample, ...delta })).toContain(
        '须使用英文',
      );
  });
  it('主体、正文与版本均影响输出指纹，号码摘要有界且下载名去除非法字符', () => {
    const fingerprint = awbPdfFingerprint(awbPdfSample);
    for (const delta of [
      { issuerDisplayName: 'OTHER COMPANY' },
      { version: '8' },
      { content: { remarks: '新备注' } },
      { contentHash: 'newhash' },
    ])
      expect(awbPdfFingerprint({ ...awbPdfSample, ...delta })).not.toBe(
        fingerprint,
      );
    expect([
      ...awbNumberSummary({ ...awbPdfSample, houseNo: '号\n'.repeat(200) }),
    ]).toHaveLength(33);
    expect(
      awbPdfFileName(
        { ...awbPdfSample, houseNo: 'HAWB/001' },
        new Date(2026, 10, 16, 1, 2, 3),
      ),
    ).toBe('HAWB-HAWB_001-waybill-2026-11-16 01-02-03.pdf');
  });
});
