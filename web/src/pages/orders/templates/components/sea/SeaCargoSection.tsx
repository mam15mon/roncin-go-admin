import { ProFormDigit, ProFormTextArea } from '@ant-design/pro-components';
import { Alert, Form, Input } from 'antd';
import React from 'react';
import { FormRow, PackageCountInput } from '@/components/ui';
import { resolveSeaOrderFormPolicy } from '../../../sea-order-policy';

export function SeaRevenueTonAlert() {
  const shipmentType = Form.useWatch('shipmentType');
  const grossWeightKg = Number(Form.useWatch('totalGrossWeightKg') ?? 0);
  const volumeCbm = Number(Form.useWatch('totalVolumeCbm') ?? 0);
  const policy = resolveSeaOrderFormPolicy({ shipmentType });
  const revenueTon = Math.max(grossWeightKg / 1000, volumeCbm);

  if (!policy.showRevenueTon) return null;

  return (
    <Alert
      type="info"
      showIcon
      title={`散杂计费吨 (RT)：${revenueTon.toFixed(3)}`}
      description="按 max(总毛重 TON, 总体积 CBM) 实时计算；这是舱位与运费测算口径，不会覆盖原始件重尺。"
    />
  );
}

export function SeaCargoMeasurementFields() {
  return <SeaRevenueTonAlert />;
}

/** 委托品名与特殊要求行；隐藏的付款方式字段随货物区常驻注册。 */
export function SeaCargoDescriptionFields() {
  return (
    <>
      <Form.Item name="paymentTerm" hidden initialValue={1}>
        <Input />
      </Form.Item>
      {/* 品名与特殊要求（FormRow row-2 对半，与后续行共享列边界） */}
      <FormRow cols={2}>
        <ProFormTextArea
          name="goodsDescription"
          label="委托品名 / 货物描述"
          placeholder="请输入客户委托申报的中英文品名或货物描述"
          fieldProps={{ maxLength: 1000, showCount: true, rows: 3 }}
        />
        <ProFormTextArea
          name="specialRequirements"
          label="特殊要求"
          placeholder="请输入客户或货物的特殊运输、装卸要求"
          fieldProps={{ maxLength: 1000, showCount: true, rows: 3 }}
        />
      </FormRow>
    </>
  );
}

/** 客户委托件重尺：订单级汇总字段（件数/单位/毛重/体积）与散杂计费吨提示。 */
export function SeaConsignedMeasurementFields({
  disabled,
}: {
  disabled?: boolean;
}) {
  return (
    <>
      <FormRow cols={4}>
        <Form.Item label="委托件数" style={{ marginBottom: 0 }}>
          <PackageCountInput
            countName="totalPackages"
            unitName="totalPackageUnit"
            countPlaceholder="件数"
            unitPlaceholder="请选择单位"
            unitWidth={104}
            disabled={disabled}
          />
        </Form.Item>
        <ProFormDigit
          name="totalGrossWeightKg"
          label="委托毛重（KGS）"
          placeholder="0"
          disabled={disabled}
          min={0}
          fieldProps={{ precision: 3, style: { width: '100%' } }}
        />
        <ProFormDigit
          name="totalVolumeCbm"
          label="委托体积（CBM）"
          placeholder="0"
          disabled={disabled}
          min={0}
          fieldProps={{ precision: 3, style: { width: '100%' } }}
        />
      </FormRow>
      <SeaRevenueTonAlert />
    </>
  );
}
