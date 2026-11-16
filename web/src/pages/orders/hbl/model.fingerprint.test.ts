import { describe, expect, it, vi } from 'vitest';
import type { HblBundle } from './model';
import { hblBundleFingerprint } from './model';

const revision = vi.hoisted(() => ({ current: 'template-v1' }));
vi.mock('./pdf/types', async (original) => ({
  ...(await original<typeof import('./pdf/types')>()),
  get HBL_PDF_TEMPLATE_REVISION() {
    return revision.current;
  },
}));

const bundle: HblBundle = {
  order: { id: 'order', version: '1' },
  documents: {
    orderId: 'order',
    houseBill: {
      id: 'house',
      version: '3',
      content: {
        shipperText: '保存的发货人',
        goodsDescriptionText: '保存的货描',
      },
      printDetails: { issuePlaceText: '签发地', containersText: '箱号' },
    },
  },
};
describe('HBL输出内容与模板指纹', () => {
  it('保存正文或印字变化即使版本未变也使旧预览失效，摘要长度保持有界', () => {
    const original = hblBundleFingerprint(bundle);
    const next = structuredClone(bundle);
    const house = next.documents.houseBill;
    if (!house?.content || !house.printDetails)
      throw new Error('HBL测试资料缺失');
    house.content.goodsDescriptionText = '长货描'.repeat(3000);
    expect(hblBundleFingerprint(next)).not.toBe(original);
    expect(hblBundleFingerprint(next).length).toBe(original.length);
    house.content = bundle.documents.houseBill?.content;
    house.printDetails.containersText = '新箱号';
    expect(hblBundleFingerprint(next)).not.toBe(original);
  });
  it('字段顺序及空投影保持稳定，模板修订推进使旧输出失效', () => {
    const original = hblBundleFingerprint(bundle);
    const reordered = structuredClone(bundle);
    if (!reordered.documents.houseBill) throw new Error('HBL测试资料缺失');
    reordered.documents.houseBill.content = {
      goodsDescriptionText: '保存的货描',
      shipperText: '保存的发货人',
      notifyPartyText: '',
    };
    expect(hblBundleFingerprint(reordered)).toBe(original);
    revision.current = 'template-v2';
    expect(hblBundleFingerprint(bundle)).not.toBe(original);
    revision.current = 'template-v1';
  });
});
