import {
  ProFormDateTimePicker,
  ProFormText,
} from '@ant-design/pro-components';
import { Alert, Button, Col, Form } from 'antd';
import React from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import { containerOwnershipOptions } from '../../../common';
import {
  OrderContainerRequestFields,
  OrderShippingDocumentFields,
  type SelectOption,
} from '../../../order-plan-fields';
import { resolveSeaOrderFormPolicy } from '../../../sea-order-policy';

export function SeaContainerPlanFields({
  options,
}: {
  options: SelectOption[];
}) {
  const form = Form.useFormInstance();
  const shipmentType = Form.useWatch('shipmentType');
  const containerRequests = (Form.useWatch('containerRequests') ??
    []) as API.OrderContainerRequestInput[];
  const policy = resolveSeaOrderFormPolicy({ shipmentType });

  if (!policy.showContainerPlan) {
    return (
      <Col span={24}>
        <Alert
          type={containerRequests.length > 0 ? 'warning' : 'info'}
          showIcon
          title="散杂货不使用箱型箱量、箱号或封号配置"
          description={
            containerRequests.length > 0
              ? '切换托运类型前已经录入箱型箱量，请确认后清空，系统不会静默删除已有数据。'
              : '页面已隐藏集装箱专属配置，货物将按件数、毛重、体积和计费吨管理。'
          }
          action={
            containerRequests.length > 0 ? (
              <Button
                danger
                size="small"
                htmlType="button"
                onClick={() => form.setFieldValue('containerRequests', [])}
              >
                清空箱量计划
              </Button>
            ) : undefined
          }
          style={{ marginBottom: 16 }}
        />
      </Col>
    );
  }
  return <OrderContainerRequestFields options={options} />;
}

export function SeaScheduleDateFields() {
  const form = Form.useFormInstance();
  return (
    <>
      <ProFormDateTimePicker
        colProps={{ xs: 24, sm: 12, lg: 6, xl: 6 }}
        name="etd"
        label="ETD (预计开航)"
        fieldProps={{
          style: { width: '100%' },
          onChange: (date) => {
            if (date) {
              const currentEta = form?.getFieldValue('eta');
              if (currentEta?.isBefore(date)) {
                form?.setFieldValue('eta', undefined);
              }
            }
          },
        }}
      />
      <ProFormDateTimePicker
        colProps={{ xs: 24, sm: 12, lg: 6, xl: 6 }}
        name="eta"
        label="ETA (预计到达)"
        dependencies={['etd']}
        rules={[
          ({ getFieldValue }) => ({
            validator(_, value) {
              const etd = getFieldValue('etd');
              if (!value || !etd || value.isAfter(etd) || value.isSame(etd)) {
                return Promise.resolve();
              }
              return Promise.reject(new Error('ETA 不能早于 ETD'));
            },
          }),
        ]}
        fieldProps={{ style: { width: '100%' } }}
      />
    </>
  );
}

export function buildSeaTransportSection(
  locationOptions: SelectOption[],
  containerSpecOptions: SelectOption[],
) {
  return {
    key: 'transportInfo',
    title: '配舱信息',
    content: (
      <>
        <OrderShippingDocumentFields transportMode="sea" />

        {/* 第 1 行：航线 4 港口（一行 4 个，各占 6 栅格） */}
        <ProFormSearchableSelect
          colProps={{ xs: 24, sm: 12, lg: 6, xl: 6 }}
          name="originLocationId"
          label="起运港"
          options={locationOptions}
          placeholder="请选择起运港或地点"
        />
        <ProFormSearchableSelect
          colProps={{ xs: 24, sm: 12, lg: 6, xl: 6 }}
          name="destinationLocationId"
          label="目的港"
          options={locationOptions}
          placeholder="请选择目的港或地点"
        />
        <ProFormSearchableSelect
          colProps={{ xs: 24, sm: 12, lg: 6, xl: 6 }}
          name="dischargeLocationId"
          label="卸货港"
          options={locationOptions}
          placeholder="请选择卸货港"
        />
        <ProFormSearchableSelect
          colProps={{ xs: 24, sm: 12, lg: 6, xl: 6 }}
          name="transitLocationId"
          label="中转港"
          options={locationOptions}
          placeholder="请选择中转港"
        />

        {/* 第 2 行：箱货与船期（一行 4 个） */}
        <ProFormSearchableSelect
          colProps={{ xs: 24, sm: 12, lg: 6, xl: 5 }}
          name="containerOwnership"
          label="货主箱标记"
          options={containerOwnershipOptions}
          placeholder="请选择 COC / SOC"
        />
        <ProFormText
          colProps={{ xs: 24, sm: 12, lg: 6, xl: 7 }}
          name="vesselVoyage"
          label="船名航次"
          placeholder="请输入船名航次"
        />
        <SeaScheduleDateFields />

        <SeaContainerPlanFields options={containerSpecOptions} />

        {/* 第 3 行：4 大截关时间（一行 4 个，各占 6 栅格） */}
        <ProFormDateTimePicker
          colProps={{ xs: 24, sm: 12, lg: 6, xl: 6 }}
          name="siCutoff"
          label="SI截关时间"
          fieldProps={{ style: { width: '100%' } }}
        />
        <ProFormDateTimePicker
          colProps={{ xs: 24, sm: 12, lg: 6, xl: 6 }}
          name="docCutoff"
          label="单证截关时间"
          fieldProps={{ style: { width: '100%' } }}
        />
        <ProFormDateTimePicker
          colProps={{ xs: 24, sm: 12, lg: 6, xl: 6 }}
          name="customsCutoff"
          label="报关截关时间"
          fieldProps={{ style: { width: '100%' } }}
        />
        <ProFormDateTimePicker
          colProps={{ xs: 24, sm: 12, lg: 6, xl: 6 }}
          name="vgmCutoff"
          label="VGM截关时间"
          fieldProps={{ style: { width: '100%' } }}
        />
      </>
    ),
  };
}
