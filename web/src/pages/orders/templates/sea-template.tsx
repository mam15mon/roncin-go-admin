import {
  ProFormDatePicker,
  ProFormDateTimePicker,
  ProFormDigit,
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
    statusTemplateOptions,
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
      title: '基本信息',
      content: (
        <>
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="customerId"
            label="客户单位"
            rules={[{ required: true, message: '请选择客户单位' }]}
            fieldProps={{
              showSearch: true,
              placeholder: '搜索客户单位',
            }}
            request={async ({ keyWords }) => searchCustomers(keyWords)}
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="statusTemplateId"
            label="状态流转模板（系统内置）"
            options={statusTemplateOptions}
            placeholder="系统自动匹配"
            fieldProps={{ disabled: true }}
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="carrierId"
            label="承运人 (船公司)"
            fieldProps={{
              showSearch: true,
              placeholder: '搜索承运人/船东',
            }}
            request={async ({ keyWords }) => searchCarriers(keyWords)}
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="bookingAgentId"
            label="订舱代理"
            fieldProps={{
              showSearch: true,
              placeholder: '搜索订舱代理',
            }}
            request={async ({ keyWords }) => searchBookingAgents(keyWords)}
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="tradeTerm"
            label="贸易条款"
            rules={[{ required: true, message: '请选择贸易条款' }]}
            options={tradeTermOptions}
            placeholder="请选择贸易条款"
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="paymentTerm"
            label="付款条款"
            rules={[{ required: true, message: '请选择付款条款' }]}
            options={paymentTermOptions}
            placeholder="请选择付款条款"
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="serviceTypeIds"
            label="服务类型"
            mode="multiple"
            options={serviceTypeOptions}
            placeholder="请选择服务类型 (多选)"
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="cargoCategoryIds"
            label="货类"
            mode="multiple"
            options={cargoCategoryOptions}
            placeholder="请选择货类 (多选)"
          />
          <ProFormDatePicker
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="orderDate"
            label="订单日期"
            fieldProps={{ style: { width: '100%' } }}
          />
        </>
      ),
    },
    {
      key: 'transportInfo',
      title: '运输信息',
      content: (
        <>
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="shipmentType"
            label="装载方式"
            options={shipmentTypeOptions}
            placeholder="请选择装载方式"
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="containerOwnership"
            label="箱属"
            options={containerOwnershipOptions}
            placeholder="请选择箱属 (COC / SOC)"
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="shipmentMode"
            label="运输模式"
            options={shipmentModeOptions}
            placeholder="请选择运输模式 (拼货 / 跨境)"
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="originLocationId"
            label="起运港 / 地点"
            options={locationOptions}
            fieldProps={{ showSearch: true }}
            placeholder="请选择起运港或地点"
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="destinationLocationId"
            label="目的港 / 地点"
            options={locationOptions}
            fieldProps={{ showSearch: true }}
            placeholder="请选择目的港或地点"
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="dischargeLocationId"
            label="卸货港"
            options={locationOptions}
            fieldProps={{ showSearch: true }}
            placeholder="请选择卸货港"
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="transitLocationId"
            label="中转港"
            options={locationOptions}
            fieldProps={{ showSearch: true }}
            placeholder="请选择中转港"
          />
          <ProFormText
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="vesselVoyage"
            label="船名航次"
            placeholder="请输入船名航次"
          />
          <ProFormDatePicker
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="etd"
            label="预计离港时间 (ETD)"
            fieldProps={{ style: { width: '100%' } }}
          />
          <ProFormDatePicker
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="eta"
            label="预计到达时间 (ETA)"
            fieldProps={{ style: { width: '100%' } }}
          />
          <ProFormDateTimePicker
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="siCutoff"
            label="SI截关时间"
            fieldProps={{ style: { width: '100%' } }}
          />
          <ProFormDateTimePicker
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="docCutoff"
            label="单证截关时间"
            fieldProps={{ style: { width: '100%' } }}
          />
          <ProFormDateTimePicker
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="customsCutoff"
            label="报关截关时间"
            fieldProps={{ style: { width: '100%' } }}
          />
          <ProFormDateTimePicker
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="vgmCutoff"
            label="VGM截关时间"
            fieldProps={{ style: { width: '100%' } }}
          />
        </>
      ),
    },
    {
      key: 'cargoInfo',
      title: '货物信息',
      content: (
        <>
          <ProFormTextArea
            colProps={{ span: 24 }}
            name="goodsDescription"
            label="货物描述"
            placeholder="请输入货物描述"
            fieldProps={{ maxLength: 1000, showCount: true }}
          />
          <ProFormDigit
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="totalPackages"
            label="总件数"
            min={0}
            placeholder="请输入总件数"
          />
          <ProFormText
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="totalPackageUnit"
            label="包装单位"
            placeholder="例如: CTNS, PLTS"
          />
          <ProFormTextArea
            colProps={{ span: 24 }}
            name="specialRequirements"
            label="特殊要求"
            placeholder="请输入客户或货物的特殊运输/装卸要求"
            fieldProps={{ maxLength: 1000, showCount: true }}
          />
        </>
      ),
    },
    {
      key: 'internalInfo',
      title: '内部信息',
      content: (
        <ProFormTextArea
          colProps={{ span: 24 }}
          name="notes"
          label="业务备注"
          placeholder="请输入内部业务备注说明"
          fieldProps={{ maxLength: 1000, showCount: true }}
        />
      ),
    },
  ];
}
