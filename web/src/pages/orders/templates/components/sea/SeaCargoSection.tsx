import {
  ProFormDigit,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { Alert, Col, Form } from 'antd';
import React from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import { paymentTermOptions } from '../../../common';
import { resolveSeaOrderFormPolicy } from '../../../sea-order-policy';

export function SeaCargoMeasurementFields() {
  const shipmentType = Form.useWatch('shipmentType');
  const grossWeightKg = Number(Form.useWatch('totalGrossWeightKg') ?? 0);
  const volumeCbm = Number(Form.useWatch('totalVolumeCbm') ?? 0);
  const policy = resolveSeaOrderFormPolicy({ shipmentType });
  const revenueTon = Math.max(grossWeightKg / 1000, volumeCbm);

  return (
    <>
      <ProFormDigit
        colProps={{ xs: 24, sm: 12, lg: 6 }}
        name="totalPackages"
        label="委托总件数"
        min={0}
        placeholder="请输入总件数"
      />
      <ProFormText
        colProps={{ xs: 24, sm: 12, lg: 6 }}
        name="totalPackageUnit"
        label="件数单位"
        placeholder="例如: CTNS, PLTS"
      />
      <ProFormDigit
        colProps={{ xs: 24, sm: 12, lg: 6 }}
        name="totalGrossWeightKg"
        label="委托总毛重 (KGS)"
        min={0}
        fieldProps={{ precision: 3 }}
        placeholder="请输入毛重"
      />
      <ProFormDigit
        colProps={{ xs: 24, sm: 12, lg: 6 }}
        name="totalVolumeCbm"
        label="委托总体积 (CBM)"
        min={0}
        fieldProps={{ precision: 3 }}
        placeholder="请输入体积"
      />
      {policy.showRevenueTon && (
        <Col span={24}>
          <Alert
            type="info"
            showIcon
            title={`散杂计费吨 (RT)：${revenueTon.toFixed(3)}`}
            description="按 max(总毛重 TON, 总体积 CBM) 实时计算；这是舱位与运费测算口径，不会覆盖原始件重尺。"
            style={{ marginBottom: 16 }}
          />
        </Col>
      )}
      <ProFormSearchableSelect
        colProps={{ xs: 24, sm: 12, lg: 8 }}
        name="paymentTerm"
        label="付款方式"
        rules={[{ required: true, message: '请选择付款方式' }]}
        options={paymentTermOptions}
        placeholder="请选择预付 / 到付"
      />
    </>
  );
}

export function buildSeaCargoSection() {
  return {
    key: 'cargoInfo',
    title: '提单信息',
    content: (
      <>
        {/* 第 1 行：品名与特殊要求（一行 2 个，各占 12 栅格） */}
        <ProFormTextArea
          colProps={{ xs: 24, lg: 12 }}
          name="goodsDescription"
          label="品名 / 货物描述"
          placeholder="请输入中英文品名或货物描述"
          fieldProps={{ maxLength: 1000, showCount: true, rows: 3 }}
        />
        <ProFormTextArea
          colProps={{ xs: 24, lg: 12 }}
          name="specialRequirements"
          label="特殊要求"
          placeholder="请输入客户或货物的特殊运输、装卸要求"
          fieldProps={{ maxLength: 1000, showCount: true, rows: 3 }}
        />

        <SeaCargoMeasurementFields />
      </>
    ),
  };
}
