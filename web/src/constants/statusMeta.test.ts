import { describe, expect, it } from 'vitest';
import { DingTalkInvitationStatus } from '@/enums.generated';
import {
  businessTypeMeta,
  dingTalkInvitationStatusMeta,
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
    expect(normalizeOrderFeeStatus(5)).toBe(5);
    expect(normalizeOrderFeeStatus('UNBILLED')).toBe(5);
    expect(normalizeOrderFeeStatus('ORDER_FEE_STATUS_UNBILLED')).toBe(5);
    expect(normalizeOrderFeeStatus('BILLED')).toBe(3);
    expect(normalizeOrderFeeStatus('CANCELLED')).toBe(4);
  });

  it('从同一份元数据生成表格枚举和展示文本', () => {
    expect(makeValueEnum(orderFeeStatusMeta)['5']).toEqual({
      text: '未建账',
    });
    expect(statusText(businessTypeMeta, 4)).toBe('空运进口');
  });

  it('钉钉邀请状态四态全覆盖且以生成常量为键', () => {
    expect(
      dingTalkInvitationStatusMeta[
        DingTalkInvitationStatus.DING_TALK_INVITATION_STATUS_PENDING
      ],
    ).toEqual({ text: '待使用', color: 'processing' });
    expect(
      dingTalkInvitationStatusMeta[
        DingTalkInvitationStatus.DING_TALK_INVITATION_STATUS_CONSUMED
      ],
    ).toEqual({ text: '已激活', color: 'success' });
    expect(
      dingTalkInvitationStatusMeta[
        DingTalkInvitationStatus.DING_TALK_INVITATION_STATUS_EXPIRED
      ],
    ).toEqual({ text: '已过期', color: 'default' });
    expect(
      dingTalkInvitationStatusMeta[
        DingTalkInvitationStatus.DING_TALK_INVITATION_STATUS_REVOKED
      ],
    ).toEqual({ text: '已撤销', color: 'default' });
    expect(Object.keys(dingTalkInvitationStatusMeta)).toHaveLength(4);
  });
});
