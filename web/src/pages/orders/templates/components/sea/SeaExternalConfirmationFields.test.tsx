import dayjs from 'dayjs';
import { describe, expect, it } from 'vitest';
import { buildSeaExternalConfirmation } from './SeaExternalConfirmationFields';

describe('buildSeaExternalConfirmation', () => {
  it('清理确认文本并保留当前订单附件引用', () => {
    expect(
      buildSeaExternalConfirmation({
        confirmedByParty: '  XX 船代  ',
        confirmedAt: dayjs('2026-09-07T08:00:00.000Z'),
        confirmationNote: '  已邮件确认可改  ',
        confirmationAttachmentId: 'attachment-1',
      }),
    ).toEqual({
      confirmedByParty: 'XX 船代',
      confirmedAt: '2026-09-07T08:00:00.000Z',
      confirmationNote: '已邮件确认可改',
      confirmationAttachmentId: 'attachment-1',
    });
  });

  it('拒绝缺少必填事实，不能静默把确认时间补成当前时间', () => {
    expect(() =>
      buildSeaExternalConfirmation({
        confirmedByParty: '测试船代',
        confirmationNote: '已确认',
      }),
    ).toThrow('外部确认方、确认时间和确认说明不能为空');
  });
});
