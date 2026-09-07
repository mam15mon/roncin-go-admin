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
});
