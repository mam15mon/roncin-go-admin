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
  paymentTermOptions,
  tradeDirectionOptions,
  tradeTermOptions,
} from '../common';
import type { TemplateProps, TemplateSection } from './types';

export function getAirTemplateSections(props: TemplateProps): TemplateSection[] {
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
            label="状态流转模板"
            rules={[{ required: true, message: '请选择状态模板' }]}
            options={statusTemplateOptions}
            placeholder="请选择状态模板"
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="carrierId"
            label="承运人 (航司)"
            fieldProps={{
              showSearch: true,
              placeholder: '搜索承运人/航空公司',
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
            name="tradeDirection"
            label="贸易方向"
            rules={[{ required: true, message: '请选择贸易方向' }]}
            options={tradeDirectionOptions}
            placeholder="请选择贸易方向"
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
      key: 'flightInfo',
      title: '航班信息',
      content: (
        <>
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="originLocationId"
            label="起运机场 / 地点"
            options={locationOptions}
            fieldProps={{ showSearch: true }}
            placeholder="请选择起运机场或地点"
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="destinationLocationId"
            label="目的机场 / 地点"
            options={locationOptions}
            fieldProps={{ showSearch: true }}
            placeholder="请选择目的机场或地点"
          />
          <ProFormText
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="vesselVoyage"
            label="航班号"
            placeholder="请输入航班号 (如 CA1234)"
          />
          <ProFormDatePicker
            colProps={{ xs: 24, sm: 12, md: 8 }}
            name="etd"
            label="预计起飞时间 (ETD)"
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
            name="customsCutoff"
            label="报关截关时间"
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
            placeholder="请输入客户或货物的特殊航空运输/装卸要求"
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
