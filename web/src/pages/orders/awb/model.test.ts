import { describe, expect, it } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import {
  acceptsAwbHouseSave,
  acceptsAwbMasterSave,
  awbFingerprint,
  awbHouseSavePayload,
  awbOutputBlockedReason,
  buildAwbPdfModel,
  requireAwbBundle,
} from './model';
import { aeIdentity, awbFixture, savedHouseOrder } from './model.fixtures';

function required<T>(value: T | undefined): T {
  if (value === undefined) throw new Error('测试资料缺失');
  return value;
}

describe('航空制作保存快照与主体', () => {
  it('真实 wire 空正文、optional未设及非optional零值均保留；身份缺失拒绝', () => {
    const bundle = awbFixture();
    required(bundle.master.content).waybillContent = {};
    required(required(required(bundle.order).airExport).house).content = {
      asArranged: false,
      customsValue: { kind: 'AMOUNT', amount: '0' },
    };
    expect(requireAwbBundle(bundle, aeIdentity)).toBe(bundle);
    expect(
      awbHouseSavePayload(bundle, {
        content: {
          carriageValue: {},
          prepaid: {},
          asArranged: false,
          remarks: '',
        },
      }).airExport?.house?.content,
    ).toEqual({ asArranged: false, remarks: '' });
    delete required(bundle.order).organizationId;
    expect(() => requireAwbBundle(bundle, aeIdentity)).toThrow('公司');
  });
  it('跨公司、错类型及缺合法版本不能为旧mock放行', () => {
    for (const patch of [
      { organizationId: 'other' },
      { businessType: 'AI' },
      { version: undefined },
      { id: undefined },
    ]) {
      const bundle = awbFixture();
      Object.assign(bundle.master, patch);
      expect(() => requireAwbBundle(bundle, aeIdentity)).toThrow();
    }
  });
  it('DIRECT没有分单输出或保存；明确缺正文不是空消息', () => {
    const bundle = awbFixture();
    bundle.master.documentMode = 'DIRECT';
    required(required(bundle.order).airExport).documentMode = 'DIRECT';
    required(required(required(bundle.order).airExport).master).documentMode =
      'DIRECT';
    delete required(required(bundle.order).airExport).house;
    expect(() => requireAwbBundle(bundle, aeIdentity)).toThrow('HAWB');
    expect(() => awbHouseSavePayload(bundle, { content: {} })).toThrow();
    expect(requireAwbBundle(bundle, { ...aeIdentity, target: 'MAWB' })).toBe(
      bundle,
    );
    delete required(bundle.master.content).waybillContent;
    expect(() =>
      requireAwbBundle(bundle, { ...aeIdentity, target: 'MAWB' }),
    ).toThrow('正文');
  });
  it('主单模式缺失、非法或与本票关联/内嵌主单不一致均拒绝', () => {
    for (const mode of [undefined, 'UNKNOWN']) {
      const bundle = awbFixture();
      bundle.master.documentMode = mode;
      expect(() => requireAwbBundle(bundle, aeIdentity)).toThrow('模式');
    }
    const wrongLink = awbFixture();
    required(required(wrongLink.order).airExport).documentMode = 'DIRECT';
    expect(() => requireAwbBundle(wrongLink, aeIdentity)).toThrow('关联');
    const wrongEmbedded = awbFixture();
    required(
      required(required(wrongEmbedded.order).airExport).master,
    ).documentMode = 'DIRECT';
    expect(() => requireAwbBundle(wrongEmbedded, aeIdentity)).toThrow('关联');
    expect(requireAwbBundle(awbFixture(), aeIdentity)).toBeDefined();
  });
  it('本公司HAWB真实抬头与已保存issuedBy独立，进口主分只取真实已保存印发人', () => {
    const bundle = awbFixture();
    required(
      required(required(required(bundle.order).airExport).house).content,
    ).issuedBy = 'SAVED PRINT TEXT';
    expect(buildAwbPdfModel(bundle, 'HAWB')).toMatchObject({
      outputKind: 'SELF_ISSUED',
      issuerDisplayName: 'ACTUAL COMPANY LTD',
      content: { issuedBy: 'SAVED PRINT TEXT' },
    });
    expect(buildAwbPdfModel(bundle, 'MAWB')).toMatchObject({
      outputKind: 'EXTERNAL_ARCHIVE',
      issuerDisplayName: 'REAL AIRLINE',
    });
    bundle.master.businessType = 'AI';
    required(bundle.order).businessType = OrderBusinessType.BUSINESS_TYPE_AI;
    required(required(bundle.order).airExport).master = structuredClone(
      bundle.master,
    );
    expect(buildAwbPdfModel(bundle, 'HAWB')).toMatchObject({
      outputKind: 'EXTERNAL_ARCHIVE',
      issuerDisplayName: 'SAVED PRINT TEXT',
    });
    delete required(
      required(required(required(bundle.order).airExport).house).content,
    ).issuedBy;
    expect(awbOutputBlockedReason(bundle, 'HAWB')).toContain('印发人');
  });
  it('缺英文名只限制输出且抬头变化无需主单版本改变便使指纹失效', () => {
    const bundle = awbFixture();
    const before = awbFingerprint(bundle);
    bundle.master.organizationNameEn = '';
    expect(awbOutputBlockedReason(bundle, 'HAWB')).toContain('英文名称');
    expect(awbFingerprint(bundle)).not.toBe(before);
    expect(
      awbHouseSavePayload(bundle, { content: {} }).airExport?.house?.content,
    ).toEqual({});
  });
  it('分单请求只有原身份基线和明确正文，不复制主单或经营字段', () => {
    const bundle = awbFixture();
    expect(
      awbHouseSavePayload(bundle, {
        content: { remarks: '新正文' },
        reason: '更正',
      }),
    ).toEqual({
      id: 'order-1',
      expectedVersion: '4',
      airExport: {
        documentMode: 'HOUSE',
        masterWaybillId: 'master-1',
        expectedMasterVersion: '2',
        expectedLinkVersion: '3',
        house: {
          houseNo: 'H001',
          status: 'DRAFT',
          expectedVersion: '5',
          content: { remarks: '新正文' },
        },
        changeReason: '更正',
      },
    });
  });
  it('仅采纳分单本次确切保存，后续版本/身份/抬头变化全部拒绝', () => {
    const bundle = awbFixture();
    const values = { content: { remarks: '新正文' } };
    const payload = awbHouseSavePayload(bundle, values);
    const saved = savedHouseOrder(bundle, values.content);
    expect(acceptsAwbHouseSave(bundle, payload, saved)).toBe(true);
    for (const change of [
      (order: API.Order) => {
        order.version = '6';
      },
      (order: API.Order) => {
        order.businessType = OrderBusinessType.BUSINESS_TYPE_AI;
      },
      (order: API.Order) => {
        required(required(order.airExport).house).version = '7';
      },
      (order: API.Order) => {
        required(required(order.airExport).master).organizationNameEn =
          'NEW COMPANY';
      },
    ]) {
      const wrong = structuredClone(saved);
      change(wrong);
      expect(acceptsAwbHouseSave(bundle, payload, wrong)).toBe(false);
    }
  });
  it('主单仅采纳恰增一的独立保存版本，后续正文/版本/主体拒绝', () => {
    const bundle = awbFixture();
    const values = { content: { remarks: '新主单' } };
    const saved = structuredClone(bundle.master);
    saved.version = '3';
    required(saved.content).waybillContent = values.content;
    expect(acceptsAwbMasterSave(bundle, values, saved)).toBe(true);
    saved.version = '4';
    expect(acceptsAwbMasterSave(bundle, values, saved)).toBe(false);
    saved.version = '3';
    saved.organizationNameEn = 'NEW';
    expect(acceptsAwbMasterSave(bundle, values, saved)).toBe(false);
  });
});
