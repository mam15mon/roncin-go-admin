import {
  ProFormDatePicker,
  ProFormDateTimePicker,
  ProFormDigit,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import React from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import { OrderShippingDocumentFields } from '../order-plan-fields';
import { paymentTermOptions, tradeTermOptions } from '../common';
import type { TemplateProps, TemplateSection } from './types';

export function getAirTemplateSections(props: TemplateProps): TemplateSection[] {
  const {
    serviceTypeOptions,
    cargoCategoryOptions,
    locationOptions,
    searchLocations,
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
          <OrderShippingDocumentFields transportMode="air" />

          {/* 第 1 行：核心商务信息（一行 5 个） */}
          <ProFormSearchableSelect
            colProps={{ xs: 24, sm: 24, lg: 12, xl: 7 }}
            name="customerId"
            label="客户单位"
            rules={[{ required: true, message: '请选择客户单位' }]}
            placeholder="搜索客户单位"
            request={async ({ keyWords }) => searchCustomers(keyWords)}
          />
          <ProFormSearchableSelect
            colProps={{ xs: 24, sm: 12, lg: 6, xl: 5 }}
            name="carrierId"
            label="承运人 (航司)"
            placeholder="搜索承运人/航空公司"
            request={async ({ keyWords }) => searchCarriers(keyWords)}
          />
          <ProFormSearchableSelect
            colProps={{ xs: 24, sm: 12, lg: 6, xl: 4 }}
            name="bookingAgentId"
            label="订舱代理"
            placeholder="搜索订舱代理"
            request={async ({ keyWords }) => searchBookingAgents(keyWords)}
          />
          <ProFormSearchableSelect
            colProps={{ xs: 24, sm: 12, lg: 6, xl: 4 }}
            name="tradeTerm"
            label="贸易条款"
            rules={[{ required: true, message: '请选择贸易条款' }]}
            options={tradeTermOptions}
            placeholder="请选择贸易条款"
          />
          <ProFormSearchableSelect
            colProps={{ xs: 24, sm: 12, lg: 6, xl: 4 }}
            name="paymentTerm"
            label="付款条款"
            rules={[{ required: true, message: '请选择付款条款' }]}
            options={paymentTermOptions}
            placeholder="请选择付款条款"
          />

          {/* 第 2 行：服务与分类（一行 3 个） */}
          <ProFormSearchableSelect
            colProps={{ xs: 24, sm: 12, lg: 10 }}
            name="serviceTypeIds"
            label="服务类型"
            mode="multiple"
            options={serviceTypeOptions}
            placeholder="请选择服务类型 (多选)"
          />
          <ProFormSearchableSelect
            colProps={{ xs: 24, sm: 12, lg: 10 }}
            name="cargoCategoryIds"
            label="货类"
            mode="multiple"
            options={cargoCategoryOptions}
            placeholder="请选择货类 (多选)"
          />
          <ProFormDatePicker
            colProps={{ xs: 24, sm: 12, lg: 4 }}
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
          {/* 第 1 行：航线与航班（一行 3 个，各占 8 栅格） */}
          <ProFormSearchableSelect
            colProps={{ xs: 24, sm: 12, lg: 8 }}
            name="originLocationId"
            label="起运机场 / 地点"
            options={locationOptions}
            request={async ({ keyWords }) => searchLocations(keyWords)}
            fieldProps={{ filterOption: false }}
            placeholder="请选择起运机场或地点"
          />
          <ProFormSearchableSelect
            colProps={{ xs: 24, sm: 12, lg: 8 }}
            name="destinationLocationId"
            label="目的机场 / 地点"
            options={locationOptions}
            request={async ({ keyWords }) => searchLocations(keyWords)}
            fieldProps={{ filterOption: false }}
            placeholder="请选择目的机场或地点"
          />
          <ProFormText
            colProps={{ xs: 24, sm: 12, lg: 8 }}
            name="vesselVoyage"
            label="航班号"
            placeholder="请输入航班号 (如 CA1234)"
          />

          {/* 第 2 行：时间节点（一行 3 个，各占 8 栅格） */}
          <ProFormDatePicker
            colProps={{ xs: 24, sm: 12, lg: 8 }}
            name="etd"
            label="预计起飞时间 (ETD)"
            fieldProps={{ style: { width: '100%' } }}
          />
          <ProFormDatePicker
            colProps={{ xs: 24, sm: 12, lg: 8 }}
            name="eta"
            label="预计到达时间 (ETA)"
            fieldProps={{ style: { width: '100%' } }}
          />
          <ProFormDateTimePicker
            colProps={{ xs: 24, sm: 12, lg: 8 }}
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
          {/* 第 1 行：品名与特殊要求（一行 2 个，各占 12 栅格） */}
          <ProFormTextArea
            colProps={{ xs: 24, lg: 12 }}
            name="goodsDescription"
            label="货物描述"
            placeholder="请输入货物描述"
            fieldProps={{ maxLength: 1000, showCount: true, rows: 3 }}
          />
          <ProFormTextArea
            colProps={{ xs: 24, lg: 12 }}
            name="specialRequirements"
            label="特殊要求"
            placeholder="请输入客户或货物的特殊航空运输/装卸要求"
            fieldProps={{ maxLength: 1000, showCount: true, rows: 3 }}
          />

          {/* 第 2 行：件数与包装（一行 2 个，各占 8 栅格） */}
          <ProFormDigit
            colProps={{ xs: 24, sm: 12, lg: 8 }}
            name="totalPackages"
            label="总件数"
            min={0}
            placeholder="请输入总件数"
          />
          <ProFormText
            colProps={{ xs: 24, sm: 12, lg: 8 }}
            name="totalPackageUnit"
            label="包装单位"
            placeholder="例如: CTNS, PLTS"
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
          fieldProps={{ maxLength: 1000, showCount: true, rows: 3 }}
        />
      ),
    },
  ];
}
