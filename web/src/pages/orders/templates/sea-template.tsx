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
import { Button, Col, Form, Input, Row, Select, message } from 'antd';
import React from 'react';
import {
  containerOwnershipOptions,
  paymentTermOptions,
  shipmentModeOptions,
  shipmentTypeOptions,
  tradeTermOptions,
} from '../common';
import type { SelectOption, TemplateProps, TemplateSection } from './types';

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
    searchShippingAgents,
    setCustomerCode,
  } = props;

  return [
    {
      key: 'basicInfo',
      title: '业务信息',
      content: (
        <>
          {/* 第 1 行：订单编号、订单编号时间及提示语 */}
          <Col span={24}>
            <Row gutter={16} align="middle">
              <Col className="col-5">
                <ProFormText
                  name="orderNo"
                  label="订单编号"
                  placeholder="保存后自动生成"
                  fieldProps={{ disabled: true }}
                />
              </Col>
              <Col className="col-5">
                <ProFormDatePicker
                  name="orderDate"
                  label="订单编号时间"
                  fieldProps={{ style: { width: '100%' } }}
                />
              </Col>
              <Col flex="auto">
                <div
                  style={{
                    color: '#ff4d4f',
                    fontSize: 12,
                    lineHeight: '32px',
                    marginBottom: 24,
                  }}
                >
                  *订单编号时间依据为订单编号生成时定义的时间，只可在该订单初次保存前修改
                </div>
              </Col>
            </Row>
          </Col>

          {/* 第 2 行：委托单位、集运/跨境、托运类型（第二行整体左移） */}
          <Col span={24}>
            <Row gutter={16} align="middle">
              <Col className="col-5">
                <ProFormSelect
                  name="customerId"
                  label="委托单位"
                  rules={[{ required: true, message: '请选择客户单位' }]}
                  fieldProps={{
                    showSearch: true,
                    placeholder: '请选择',
                    onChange: (_, option) =>
                      setCustomerCode((option as SelectOption | undefined)?.code),
                  }}
                  request={async ({ keyWords }) => searchCustomers(keyWords)}
                />
              </Col>
              <Col className="col-5">
                <ProFormRadio.Group
                  name="shipmentMode"
                  label="集运/跨境"
                  options={shipmentModeOptions}
                />
              </Col>
              <Col className="col-5">
                <ProFormRadio.Group
                  name="shipmentType"
                  label="托运类型"
                  options={shipmentTypeOptions}
                />
              </Col>
            </Row>
          </Col>

          {/* 第 3 行：服务类型（整行复选框） */}
          <Col span={24}>
            <ProFormCheckbox.Group
              name="serviceTypeIds"
              label="服务类型"
              options={serviceTypeOptions}
            />
          </Col>

          {/* 第 4 行：货物品类（整行复选框） */}
          <Col span={24}>
            <ProFormCheckbox.Group
              name="cargoCategoryIds"
              label="货物品类"
              options={cargoCategoryOptions}
            />
          </Col>

          {/* 第 5 行：一行 5 列（客户业务编号、企业内部编号、贸易条款、订舱代理、国外代理） */}
          <Col className="col-5">
            <Form.Item label="客户业务编号" style={{ marginInline: 8 }}>
              <Form.Item noStyle name="customerReferenceNo">
                <Input
                  placeholder="请输入"
                  maxLength={100}
                  suffix={
                    <Button
                      type="link"
                      size="small"
                      style={{ fontSize: 12, height: 21, padding: '0 2px' }}
                      onClick={() => message.info('正在校验客户业务编号')}
                    >
                      重复校验
                    </Button>
                  }
                />
              </Form.Item>
            </Form.Item>
          </Col>
          <Col className="col-5">
            <Form.Item label="企业内部编号" style={{ marginInline: 8 }}>
              <Form.Item noStyle name="internalReferenceNo">
                <Input
                  placeholder="请输入"
                  maxLength={100}
                  suffix={
                    <Button
                      type="link"
                      size="small"
                      style={{ fontSize: 12, height: 21, padding: '0 2px' }}
                      onClick={() => message.info('正在校验企业内部编号')}
                    >
                      重复校验
                    </Button>
                  }
                />
              </Form.Item>
            </Form.Item>
          </Col>
          <Col className="col-5">
            <ProFormSelect
              name="tradeTerm"
              label="贸易条款"
              rules={[{ required: true, message: '请选择贸易条款' }]}
              options={tradeTermOptions}
              placeholder="请选择"
            />
          </Col>
          <Col className="col-5">
            <ProFormSelect
              name="bookingAgentId"
              label="订舱代理"
              fieldProps={{
                showSearch: true,
                placeholder: '请选择',
              }}
              request={async ({ keyWords }) => searchBookingAgents(keyWords)}
            />
          </Col>
          <Col className="col-5">
            <ProFormSelect
              name="foreignAgentId"
              label="国外代理"
              fieldProps={{
                showSearch: true,
                placeholder: '请选择',
              }}
              request={async ({ keyWords }) => searchForeignAgents(keyWords)}
            />
          </Col>

          {/* 第 6 行：一行 5 列（合约号、船公司、船代、货值、保费） */}
          <Col className="col-5">
            <ProFormText
              name="contractNo"
              label="合约号"
              fieldProps={{ maxLength: 100 }}
              placeholder="请输入"
            />
          </Col>
          <Col className="col-5">
            <ProFormSelect
              name="carrierId"
              label="船公司"
              fieldProps={{
                showSearch: true,
                placeholder: '请选择',
              }}
              request={async ({ keyWords }) => searchCarriers(keyWords)}
            />
          </Col>
          <Col className="col-5">
            <ProFormSelect
              name="shippingAgentId"
              label="船代"
              fieldProps={{
                showSearch: true,
                placeholder: '请选择',
              }}
              request={async ({ keyWords }) => searchShippingAgents(keyWords)}
            />
          </Col>
          <Col className="col-5">
            <Form.Item label="货值" style={{ marginInline: 8 }}>
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
                <Input
                  placeholder="金额"
                  maxLength={23}
                  suffix={
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
                        size="small"
                        variant="borderless"
                        style={{ width: 72, height: 21 }}
                      />
                    </Form.Item>
                  }
                />
              </Form.Item>
            </Form.Item>
          </Col>
          <Col className="col-5">
            <Form.Item label="保费" style={{ marginInline: 8 }}>
              <Form.Item
                noStyle
                name="insurancePremium"
                dependencies={['insuranceCurrency']}
                rules={[
                  ({ getFieldValue }) => ({
                    validator: async (_, value) => {
                      if (!value && !getFieldValue('insuranceCurrency')) return;
                      if (!value) throw new Error('请输入保费');
                      if (!/^(0|[1-9]\d{0,17})(\.\d{1,4})?$/.test(value)) {
                        throw new Error('请输入正确的保费，最多 4 位小数');
                      }
                    },
                  }),
                ]}
              >
                <Input
                  placeholder="金额"
                  maxLength={23}
                  suffix={
                    <Form.Item
                      noStyle
                      name="insuranceCurrency"
                      dependencies={['insurancePremium']}
                      rules={[
                        ({ getFieldValue }) => ({
                          validator: async (_, value) => {
                            if (!getFieldValue('insurancePremium') || value) return;
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
                        size="small"
                        variant="borderless"
                        style={{ width: 72, height: 21 }}
                      />
                    </Form.Item>
                  }
                />
              </Form.Item>
            </Form.Item>
          </Col>

          {/* 第 7 行：一行 5 列（UN NO.、CLASS NO.、截申报时间、接单时间、工厂） */}
          <Col className="col-5">
            <ProFormText
              name="unNumber"
              label="UN NO."
              rules={[
                {
                  pattern: /^\d{4}$/,
                  message: 'UN NO. 应为 4 位数字',
                },
              ]}
              fieldProps={{ maxLength: 4 }}
              placeholder="4位数字"
            />
          </Col>
          <Col className="col-5">
            <ProFormText
              name="hazardClass"
              label="CLASS NO."
              fieldProps={{ maxLength: 16 }}
              placeholder="类别"
            />
          </Col>
          <Col className="col-5">
            <ProFormDateTimePicker
              name="loadingTerms"
              label="截申报时间"
              fieldProps={{ style: { width: '100%' } }}
            />
          </Col>
          <Col className="col-5">
            <ProFormDateTimePicker
              name="receivedAt"
              label="接单时间"
              fieldProps={{ style: { width: '100%' } }}
            />
          </Col>
          <Col className="col-5">
            <ProFormText
              name="factoryName"
              label="工厂"
              fieldProps={{ maxLength: 200 }}
              placeholder="请输入工厂"
            />
          </Col>

          {/* 第 8 行：一行 5 列（委托单位代码、货好时间、后 3 列留白） */}
          <Col className="col-5">
            <ProFormText
              name="customerCode"
              label="委托单位代码"
              fieldProps={{ disabled: true }}
              placeholder="选择委托单位后自动带出"
            />
          </Col>
          <Col className="col-5">
            <ProFormDateTimePicker
              name="cargoReadyAt"
              label="货好时间"
              fieldProps={{ style: { width: '100%' } }}
            />
          </Col>
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
