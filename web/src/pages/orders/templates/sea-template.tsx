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
import { Col, Form, Input, Select, Space } from 'antd';
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
    currencyOptions,
    searchCustomers,
    searchCarriers,
    searchBookingAgents,
    searchForeignAgents,
  } = props;

  return [
    {
      key: 'basicInfo',
      title: '业务信息',
      content: (
        <>
          {/* 第 1 行：业务核心标识（一行 5 个，紧凑排布） */}
          <ProFormText
            colProps={{ xs: 24, sm: 12, lg: 6, xl: 5 }}
            name="orderNo"
            label="订单编号"
            placeholder="保存后自动生成"
            fieldProps={{ disabled: true }}
          />
          <ProFormDatePicker
            colProps={{ xs: 24, sm: 12, lg: 6, xl: 4 }}
            name="orderDate"
            label="订单日期"
            fieldProps={{ style: { width: '100%' } }}
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 24, lg: 12, xl: 7 }}
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
            colProps={{ xs: 24, sm: 12, lg: 6, xl: 4 }}
            name="shipmentMode"
            label="集运 / 跨境"
            options={shipmentModeOptions}
          />
          <ProFormRadio.Group
            colProps={{ xs: 24, sm: 12, lg: 6, xl: 4 }}
            name="shipmentType"
            label="托运类型"
            options={shipmentTypeOptions}
          />

          {/* 第 2 行：服务类型与货物品类（一行 2 个，双列多选组） */}
          <ProFormCheckbox.Group
            colProps={{ xs: 24, lg: 12 }}
            name="serviceTypeIds"
            label="服务类型"
            options={serviceTypeOptions}
          />
          <ProFormCheckbox.Group
            colProps={{ xs: 24, lg: 12 }}
            name="cargoCategoryIds"
            label="货物品类"
            options={cargoCategoryOptions}
          />

          {/* 第 3 行：商务条款与船公司（一行 5 个） */}
          <ProFormText
            colProps={{ xs: 24, sm: 12, lg: 6, xl: 5 }}
            name="customerReferenceNo"
            label="客户业务编号"
            fieldProps={{ maxLength: 100 }}
            placeholder="请输入客户业务编号"
          />
          <ProFormText
            colProps={{ xs: 24, sm: 12, lg: 6, xl: 4 }}
            name="contractNo"
            label="合约号"
            fieldProps={{ maxLength: 100 }}
            placeholder="请输入合约号"
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, lg: 6, xl: 4 }}
            name="tradeTerm"
            label="贸易条款"
            rules={[{ required: true, message: '请选择贸易条款' }]}
            options={tradeTermOptions}
            placeholder="请选择贸易条款"
          />
          <Col xs={24} sm={12} lg={6} xl={5}>
            <Form.Item label="货值">
              <Space.Compact block>
                <Form.Item
                  noStyle
                  name="cargoValue"
                  dependencies={['cargoCurrency']}
                  rules={[
                    ({ getFieldValue }) => ({
                      validator: async (_, value) => {
                        if (!value && !getFieldValue('cargoCurrency')) return;
                        if (!value) throw new Error('请输入货值');
                        if (!/^(0|[1-9]\d{0,17})(\.\d{1,4})?$/.test(value)) {
                          throw new Error('请输入正确的货值，最多 4 位小数');
                        }
                      },
                    }),
                  ]}
                >
                  <Input placeholder="金额" maxLength={23} />
                </Form.Item>
                <Form.Item
                  noStyle
                  name="cargoCurrency"
                  dependencies={['cargoValue']}
                  rules={[
                    ({ getFieldValue }) => ({
                      validator: async (_, value) => {
                        if (!getFieldValue('cargoValue') || value) return;
                        throw new Error('请选择币种');
                      },
                    }),
                  ]}
                >
                  <Select
                    showSearch
                    optionFilterProp="label"
                    options={currencyOptions}
                    placeholder="币种"
                    style={{ width: 110 }}
                  />
                </Form.Item>
              </Space.Compact>
            </Form.Item>
          </Col>
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, lg: 6, xl: 6 }}
            name="carrierId"
            label="船公司"
            fieldProps={{
              showSearch: true,
              placeholder: '搜索承运人/船东',
            }}
            request={async ({ keyWords }) => searchCarriers(keyWords)}
          />

          {/* 第 4 行：代理协作（一行 2 个，各占 8 栅格） */}
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 8 }}
            name="bookingAgentId"
            label="订舱代理"
            fieldProps={{
              showSearch: true,
              placeholder: '搜索订舱代理',
            }}
            request={async ({ keyWords }) => searchBookingAgents(keyWords)}
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 8 }}
            name="foreignAgentId"
            label="国外代理"
            fieldProps={{
              showSearch: true,
              placeholder: '搜索国外代理',
            }}
            request={async ({ keyWords }) => searchForeignAgents(keyWords)}
          />
        </>
      ),
    },
    {
      key: 'transportInfo',
      title: '配舱信息',
      content: (
        <>
          {/* 第 1 行：航线 4 港口（一行 4 个，各占 6 栅格） */}
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, lg: 6, xl: 6 }}
            name="originLocationId"
            label="起运港"
            options={locationOptions}
            fieldProps={{ showSearch: true }}
            placeholder="请选择起运港或地点"
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, lg: 6, xl: 6 }}
            name="destinationLocationId"
            label="目的港"
            options={locationOptions}
            fieldProps={{ showSearch: true }}
            placeholder="请选择目的港或地点"
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, lg: 6, xl: 6 }}
            name="dischargeLocationId"
            label="卸货港"
            options={locationOptions}
            fieldProps={{ showSearch: true }}
            placeholder="请选择卸货港"
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, lg: 6, xl: 6 }}
            name="transitLocationId"
            label="中转港"
            options={locationOptions}
            fieldProps={{ showSearch: true }}
            placeholder="请选择中转港"
          />

          {/* 第 2 行：箱货与船期（一行 4 个） */}
          <ProFormSelect
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
          <ProFormDatePicker
            colProps={{ xs: 24, sm: 12, lg: 6, xl: 6 }}
            name="etd"
            label="ETD"
            fieldProps={{ style: { width: '100%' } }}
          />
          <ProFormDatePicker
            colProps={{ xs: 24, sm: 12, lg: 6, xl: 6 }}
            name="eta"
            label="ETA"
            fieldProps={{ style: { width: '100%' } }}
          />

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
    },
    {
      key: 'cargoInfo',
      title: '提单与货物',
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

          {/* 第 2 行：件数、单位与付款方式（一行 3 个，各占 8 栅格） */}
          <ProFormDigit
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 8 }}
            name="totalPackages"
            label="委托总件数"
            min={0}
            placeholder="请输入总件数"
          />
          <ProFormText
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 8 }}
            name="totalPackageUnit"
            label="件数单位"
            placeholder="例如: CTNS, PLTS"
          />
          <ProFormSelect
            colProps={{ xs: 24, sm: 12, lg: 8, xl: 8 }}
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
          fieldProps={{ maxLength: 1000, showCount: true, rows: 3 }}
        />
      ),
    },
  ];
}
