import {
  Button,
  DatePicker,
  Form,
  InputNumber,
  Radio,
  Space,
  Typography,
} from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, { useState } from 'react';
import { billDueResult } from './billDueSettings';

type BillDueSettingsProps = {
  billDate?: string;
  namePrefix?: (string | number)[];
  termsRequired?: boolean;
  fieldsProtected?: boolean;
  hasPaymentTerms?: boolean;
};

/** 建账与编辑共用的到期结果及按需展开字段；折叠保留注册和当前草稿。 */
export default function BillDueSettings({
  billDate,
  namePrefix = [],
  termsRequired = false,
  fieldsProtected = false,
  hasPaymentTerms = false,
}: BillDueSettingsProps) {
  const form = Form.useFormInstance();
  const [expanded, setExpanded] = useState(false);
  const modeName = [...namePrefix, 'dueMode'];
  const termsName = [...namePrefix, 'paymentTermsDays'];
  const dateName = [...namePrefix, 'dueDate'];
  const dueMode = Form.useWatch(modeName, { form, preserve: true }) ?? 'terms';
  const paymentTermsDays = Form.useWatch(termsName, { form, preserve: true });
  const dueDate = Form.useWatch(dateName, { form, preserve: true });
  const due = billDueResult(billDate, { dueMode, paymentTermsDays, dueDate });
  const locked = fieldsProtected && hasPaymentTerms;

  const toggle = async () => {
    if (expanded) {
      try {
        const currentMode = form.getFieldValue(modeName);
        await form.validateFields([
          currentMode === 'date' ? dateName : termsName,
        ]);
      } catch {
        return;
      }
    }
    setExpanded(!expanded);
  };

  return (
    <div style={{ marginBottom: 24 }}>
      <Space wrap style={{ marginBottom: 4 }}>
        <Typography.Text>到期日</Typography.Text>
        <Button
          type="link"
          size="small"
          disabled={locked || undefined}
          onClick={() => void toggle()}
        >
          {expanded ? '收起到期设置' : '调整到期日'}
        </Button>
      </Space>
      <Typography.Text strong style={{ display: 'block', fontSize: 15 }}>
        {due.text}
      </Typography.Text>
      <fieldset
        hidden={!expanded}
        style={{ border: 0, padding: 0, margin: '8px 0' }}
      >
        <legend style={{ fontSize: 14, marginBottom: 4 }}>到期方式</legend>
        <Form.Item name={modeName} style={{ marginBottom: 0 }}>
          <Radio.Group
            aria-label="到期方式"
            disabled={locked || undefined}
            options={[
              {
                label: '按账期天数',
                value: 'terms',
                disabled: fieldsProtected && !hasPaymentTerms,
              },
              { label: '指定日期', value: 'date' },
            ]}
          />
        </Form.Item>
      </fieldset>
      <Form.Item
        name={termsName}
        label="账期（天）"
        hidden={!expanded || dueMode !== 'terms'}
        style={{ marginBottom: 0 }}
        rules={[
          ({ getFieldValue }) => ({
            required: getFieldValue(modeName) !== 'date' && termsRequired,
            message: '请填写账期天数，或改为指定到期日',
          }),
        ]}
      >
        <InputNumber
          min={0}
          max={3650}
          precision={0}
          disabled={fieldsProtected || undefined}
          aria-label="账期（天）"
          placeholder="未约定"
          style={{ width: '100%' }}
        />
      </Form.Item>
      <Form.Item
        name={dateName}
        label="指定到期日"
        hidden={!expanded || dueMode !== 'date'}
        style={{ marginBottom: 0 }}
        rules={[
          {
            validator: async (_, value: Dayjs | null | undefined) => {
              if (form.getFieldValue(modeName) !== 'date') return;
              if (!value?.isValid()) throw new Error('请选择指定到期日');
              if (billDate && value.format('YYYY-MM-DD') < billDate) {
                throw new Error('到期日不能早于创建日期');
              }
            },
          },
        ]}
      >
        <DatePicker
          format="YYYY-MM-DD"
          minDate={billDate ? dayjs(billDate) : undefined}
          aria-label="指定到期日"
          disabled={locked || undefined}
          style={{ width: '100%' }}
        />
      </Form.Item>
    </div>
  );
}
