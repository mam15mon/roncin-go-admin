import {
  ProFormCheckbox,
  ProFormDatePicker,
  ProFormDateTimePicker,
  ProFormDigit,
  ProFormRadio,
  ProFormSelect,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import React from 'react';
import {
  containerOwnershipOptions,
  paymentTermOptions,
  shipmentModeOptions,
  shipmentTypeOptions,
  tradeTermOptions,
} from '../common';
import type { TemplateProps, TemplateSection } from './types';

export function getSeaTemplateSections(props: TemplateProps): TemplateSection[] {
  const {
    serviceTypeOptions,
    cargoCategoryOptions,
    locationOptions,
    searchCustomers,
    searchCarriers,
    searchBookingAgents,
  } = props;

  return [
    {
      key: 'basicInfo',
      title: '业务信息',
      content: (
        <>
          <ProFormText
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 6 }}
            name="orderNo"
            label="订单编号"
            placeholder="保存后自动生成"
            fieldProps={{ disabled: true }}
          />
          <ProFormDatePicker
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 6 }}
            name="orderDate"
            label="订单日期"
            fieldProps={{ style: { width: '100%' } }}
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 6 }}
            name="customerId"
            label="委托单位"
            rules={[{ required: true, message: '请选择客户单位' }]}
            fieldProps={{
              showSearch: true,
              placeholder: '搜索客户单位',
            }}
            request={async ({ keyWords }) => searchCustomers(keyWords)}
          />
          <ProFormRadio.Group
            colProps={{ xs: 24, lg: 12, xl: 8 }}
            name="shipmentMode"
            label="集运 / 跨境"
            options={shipmentModeOptions}
          />
          <ProFormRadio.Group
            colProps={{ xs: 24, lg: 12, xl: 8 }}
            name="shipmentType"
            label="托运类型"
            options={shipmentTypeOptions}
          />
          <ProFormCheckbox.Group
            colProps={{ span: 24 }}
            name="serviceTypeIds"
            label="服务类型"
            options={serviceTypeOptions}
          />
          <ProFormCheckbox.Group
            colProps={{ span: 24 }}
            name="cargoCategoryIds"
            label="货物品类"
            options={cargoCategoryOptions}
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 6 }}
            name="carrierId"
            label="船公司"
            fieldProps={{
              showSearch: true,
              placeholder: '搜索承运人/船东',
            }}
            request={async ({ keyWords }) => searchCarriers(keyWords)}
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 6 }}
            name="bookingAgentId"
            label="订舱代理"
            fieldProps={{
              showSearch: true,
              placeholder: '搜索订舱代理',
            }}
            request={async ({ keyWords }) => searchBookingAgents(keyWords)}
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 6 }}
            name="tradeTerm"
            label="贸易条款"
            rules={[{ required: true, message: '请选择贸易条款' }]}
            options={tradeTermOptions}
            placeholder="请选择贸易条款"
          />
        </>
      ),
    },
    {
      key: 'transportInfo',
      title: '配舱信息',
      content: (
        <>
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 6 }}
            name="originLocationId"
            label="起运港"
            options={locationOptions}
            fieldProps={{ showSearch: true }}
            placeholder="请选择起运港或地点"
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 6 }}
            name="destinationLocationId"
            label="目的港"
            options={locationOptions}
            fieldProps={{ showSearch: true }}
            placeholder="请选择目的港或地点"
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 6 }}
            name="dischargeLocationId"
            label="卸货港"
            options={locationOptions}
            fieldProps={{ showSearch: true }}
            placeholder="请选择卸货港"
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 6 }}
            name="transitLocationId"
            label="中转港"
            options={locationOptions}
            fieldProps={{ showSearch: true }}
            placeholder="请选择中转港"
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 6 }}
            name="containerOwnership"
            label="货主箱标记"
            options={containerOwnershipOptions}
            placeholder="请选择 COC / SOC"
          />
          <ProFormText
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 6 }}
            name="vesselVoyage"
            label="船名航次"
            placeholder="请输入船名航次"
          />
          <ProFormDatePicker
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 6 }}
            name="etd"
            label="ETD"
            fieldProps={{ style: { width: '100%' } }}
          />
          <ProFormDatePicker
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 6 }}
            name="eta"
            label="ETA"
            fieldProps={{ style: { width: '100%' } }}
          />
          <ProFormDateTimePicker
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 6 }}
            name="siCutoff"
            label="SI截关时间"
            fieldProps={{ style: { width: '100%' } }}
          />
          <ProFormDateTimePicker
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 6 }}
            name="docCutoff"
            label="单证截关时间"
            fieldProps={{ style: { width: '100%' } }}
          />
          <ProFormDateTimePicker
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 6 }}
            name="customsCutoff"
            label="报关截关时间"
            fieldProps={{ style: { width: '100%' } }}
          />
          <ProFormDateTimePicker
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 6 }}
            name="vgmCutoff"
            label="VGM截关时间"
            fieldProps={{ style: { width: '100%' } }}
          />
        </>
      ),
    },
    {
      key: 'cargoInfo',
      title: '提单与货物',
      content: (
        <>
          <ProFormTextArea
            colProps={{ xs: 24, lg: 12 }}
            name="goodsDescription"
            label="品名 / 货物描述"
            placeholder="请输入中英文品名或货物描述"
            fieldProps={{ maxLength: 1000, showCount: true }}
          />
          <ProFormTextArea
            colProps={{ xs: 24, lg: 12 }}
            name="specialRequirements"
            label="特殊要求"
            placeholder="请输入客户或货物的特殊运输、装卸要求"
            fieldProps={{ maxLength: 1000, showCount: true }}
          />
          <ProFormDigit
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 6 }}
            name="totalPackages"
            label="委托总件数"
            min={0}
            placeholder="请输入总件数"
          />
          <ProFormText
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 6 }}
            name="totalPackageUnit"
            label="件数单位"
            placeholder="例如: CTNS, PLTS"
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 6 }}
            name="paymentTerm"
            label="付款方式"
            rules={[{ required: true, message: '请选择付款方式' }]}
            options={paymentTermOptions}
            placeholder="请选择预付 / 到付"
          />
        </>
      ),
    },
    {
      key: 'internalInfo',
      title: '订单备注',
      content: (
        <ProFormTextArea
          colProps={{ span: 24 }}
          name="notes"
          label="内部操作备注"
          placeholder="请输入订舱、配舱或操作过程中需要内部协作的信息"
          fieldProps={{ maxLength: 1000, showCount: true }}
        />
      ),
    },
  ];
}
