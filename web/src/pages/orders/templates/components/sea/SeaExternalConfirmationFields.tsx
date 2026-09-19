import { DatePicker, Form, Input, Select } from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import { useEffect, useState } from 'react';
import { orderAttachmentServiceListAttachments } from '@/services/roncin/orderAttachmentService';

export type SeaExternalConfirmationFormValues = {
  confirmedByParty?: string;
  confirmedAt?: string | Dayjs;
  confirmationNote?: string;
  confirmationAttachmentId?: string;
};

export function buildSeaExternalConfirmation(
  values: SeaExternalConfirmationFormValues,
): API.SeaExternalConfirmationInput {
  const confirmedByParty = values.confirmedByParty?.trim();
  const confirmationNote = values.confirmationNote?.trim();
  const confirmedAt = values.confirmedAt ? dayjs(values.confirmedAt) : null;
  if (!confirmedByParty || !confirmationNote || !confirmedAt?.isValid()) {
    throw new Error('外部确认方、确认时间和确认说明不能为空');
  }
  return {
    confirmedByParty,
    confirmedAt: confirmedAt.toISOString(),
    confirmationNote,
    confirmationAttachmentId: values.confirmationAttachmentId || undefined,
  };
}

/** 专用海运变更命令共用的外部确认事实字段。 */
export default function SeaExternalConfirmationFields({
  orderId,
}: {
  orderId: string;
}) {
  const [loading, setLoading] = useState(false);
  const [attachments, setAttachments] = useState<API.OrderAttachment[]>([]);

  useEffect(() => {
    let active = true;
    setLoading(true);
    orderAttachmentServiceListAttachments({ orderId })
      .then((response) => {
        if (active) setAttachments(response.data ?? []);
      })
      .catch(() => {
        if (active) setAttachments([]);
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
    };
  }, [orderId]);

  return (
    <>
      <Form.Item
        name="confirmedByParty"
        label="外部确认方"
        rules={[
          {
            required: true,
            whitespace: true,
            message: '请输入船代、船司或承运方名称',
          },
        ]}
      >
        <Input maxLength={200} placeholder="例如：XX 船代操作部" />
      </Form.Item>
      <Form.Item
        name="confirmedAt"
        label="确认时间"
        rules={[{ required: true, message: '请选择外部确认时间' }]}
      >
        <DatePicker showTime style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item
        name="confirmationNote"
        label="确认说明"
        rules={[
          { required: true, whitespace: true, message: '请输入确认说明' },
        ]}
      >
        <Input.TextArea
          rows={2}
          maxLength={500}
          showCount
          placeholder="记录确认渠道、联系人和确认结论"
        />
      </Form.Item>
      <Form.Item name="confirmationAttachmentId" label="确认附件（可选）">
        <Select
          allowClear
          showSearch
          loading={loading}
          optionFilterProp="label"
          placeholder="仅可选择当前订单附件"
          options={attachments
            .filter((attachment) => attachment.id)
            .map((attachment) => ({
              value: attachment.id as string,
              label: attachment.fileName || attachment.docType || attachment.id,
            }))}
        />
      </Form.Item>
    </>
  );
}
