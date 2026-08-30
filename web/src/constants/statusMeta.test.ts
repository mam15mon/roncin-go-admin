import { describe, expect, it } from 'vitest';
import {
  businessTypeMeta,
  makeValueEnum,
  normalizeBusinessType,
  normalizeOrderFeeStatus,
  orderFeeStatusMeta,
  statusText,
} from './statusMeta';

describe('状态展示元数据', () => {
  it('将业务类型的数字、短码和枚举名规范为同一键', () => {
    expect(normalizeBusinessType(1)).toBe(1);
    expect(normalizeBusinessType('SE')).toBe(1);
    expect(normalizeBusinessType('BUSINESS_TYPE_SE')).toBe(1);
  });

  it('将费用状态的数字、短码和枚举名规范为同一键', () => {
    expect(normalizeOrderFeeStatus(3)).toBe(3);
    expect(normalizeOrderFeeStatus('BILLED')).toBe(3);
    expect(normalizeOrderFeeStatus('ORDER_FEE_STATUS_BILLED')).toBe(3);
  });

  it('从同一份元数据生成表格枚举和展示文本', () => {
    expect(makeValueEnum(orderFeeStatusMeta)['2']).toEqual({ text: '已确认' });
    expect(statusText(businessTypeMeta, 4)).toBe('空运进口');
  });
});
