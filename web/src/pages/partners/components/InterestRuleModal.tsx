import {
  ModalForm,
  ProFormDigit,
  ProFormSelect,
  ProFormSwitch,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { Alert, Col, Form } from 'antd';
import React, { useEffect } from 'react';

export interface InterestRuleValues {
  enabled?: boolean;
  dailyRateBp?: number; // 万分比基点，例如 5 表示 0.05%/日
  graceDays?: number; // 宽限期天数
  calcMode?: string; // daily_simple / monthly_compound
  remark?: string;
}

interface InterestRuleModalProps {
  open: boolean;
  value?: InterestRuleValues;
  onOpenChange: (open: boolean) => void;
  onFinish: (values: InterestRuleValues) => Promise<boolean | undefined>;
}

export default function InterestRuleModal({
  open,
  value,
  onOpenChange,
  onFinish,
}: InterestRuleModalProps) {
  const [form] = Form.useForm();

  useEffect(() => {
    if (open) {
      form.setFieldsValue({
        enabled: value?.enabled ?? false,
        dailyRateBp: value?.dailyRateBp ?? 5,
        graceDays: value?.graceDays ?? 3,
        calcMode: value?.calcMode ?? 'daily_simple',
        remark: value?.remark ?? '',
      });
    }
  }, [open, value, form]);

  const handleSubmit = async (values: any) => {
    await onFinish({
      enabled: Boolean(values.enabled),
      dailyRateBp: Number(values.dailyRateBp || 0),
      graceDays: Number(values.graceDays || 0),
      calcMode: values.calcMode || 'daily_simple',
      remark: values.remark?.trim(),
    });
    return true;
  };

  return (
    <ModalForm
      title="配置结算利息与逾期规则"
      open={open}
      form={form}
      onOpenChange={onOpenChange}
      onFinish={handleSubmit}
      modalProps={{
        destroyOnClose: true,
        maskClosable: false,
        width: 500,
      }}
      layout="horizontal"
      labelAlign="right"
      labelCol={{ flex: '120px' }}
      wrapperCol={{ flex: 'auto' }}
      grid
    >
      <Col span={24}>
        <Alert
          message="利息规则将在财务对账逾期时自动计算滞纳金与违约金"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />
      </Col>
      <Col span={24}>
        <ProFormSwitch
          name="enabled"
          label="启用逾期计息"
          extra="开启后，超期未核销应收账款将按照以下规则产生滞纳利息"
        />
      </Col>
      <Col span={24}>
        <ProFormDigit
          name="dailyRateBp"
          label="逾期日利率 (‱)"
          placeholder="例如输入 5 表示万分之五/天 (0.05%)"
          min={0}
          max={100}
          addonAfter="‱ / 天"
        />
      </Col>
      <Col span={24}>
        <ProFormDigit
          name="graceDays"
          label="宽限期天数"
          placeholder="超期后免息宽限天数"
          min={0}
          max={30}
          addonAfter="天"
        />
      </Col>
      <Col span={24}>
        <ProFormSelect
          name="calcMode"
          label="计息方式"
          options={[
            { label: '按日单利计算', value: 'daily_simple' },
            { label: '按月复利计算', value: 'monthly_compound' },
          ]}
        />
      </Col>
      <Col span={24}>
        <ProFormTextArea
          name="remark"
          label="规则说明"
          placeholder="例如：经双方协议，逾期超过7天后加收滞纳金"
          fieldProps={{ rows: 2 }}
        />
      </Col>
    </ModalForm>
  );
}
