import { describe, expect, it } from 'vitest';
import { AdminUserStatus } from '@/enums.generated';
import {
  adminUserStatusMeta,
  businessTypeMeta,
  makeValueEnum,
  normalizeBusinessType,
  statusText,
} from './statusMeta';

describe('状态展示元数据', () => {
  it('将业务类型的数字、短码和枚举名规范为同一键', () => {
    expect(normalizeBusinessType(1)).toBe(1);
    expect(normalizeBusinessType('SE')).toBe(1);
    expect(normalizeBusinessType('BUSINESS_TYPE_SE')).toBe(1);
  });

  it('从同一份元数据生成表格枚举和展示文本', () => {
    expect(makeValueEnum(businessTypeMeta)['2']).toEqual({
      text: '海运进口',
    });
    expect(statusText(businessTypeMeta, 4)).toBe('空运进口');
  });

  it('待分配状态使用生成常量', () => {
    expect(
      adminUserStatusMeta[AdminUserStatus.ADMIN_USER_STATUS_PENDING_ASSIGNMENT],
    ).toEqual({ text: '待分配', color: 'warning' });
  });
});
