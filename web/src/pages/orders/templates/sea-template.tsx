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
import { Button, Col, Form, Row, Select, Space } from 'antd';
import dayjs from 'dayjs';
import React from 'react';
import { TooltipInput } from '@/components/ui/tooltip-input';
import {
  OrderContainerRequestFields,
  OrderShippingDocumentFields,
} from '../order-plan-fields';
import {
  containerOwnershipOptions,
  paymentTermOptions,
  shipmentModeOptions,
  shipmentTypeOptions,
  tradeTermOptions,
} from '../common';
import type { SelectOption, TemplateProps, TemplateSection } from './types';

type PersonnelAssignmentFieldsProps = {
  label: string;
  userField: string;
  organizationField: string;
  options: API.OrderPersonnelOption[];
  disabled?: boolean;
};

function PersonnelAssignmentFields({
  label,
  userField,
  organizationField,
  options,
  disabled = false,
}: PersonnelAssignmentFieldsProps) {
  const form = Form.useFormInstance();
  const selectedUserID = Form.useWatch(userField);
  const selectedOrganizationID = Form.useWatch(organizationField);
  const organizationOptions = Array.from(
    new Map(
      options
        .filter((option) => option.organizationId)
        .map((option) => [
          option.organizationId as string,
          {
            label: option.organizationName || option.organizationId,
            value: option.organizationId as string,
          },
        ]),
    ).values(),
  );
  const userOptions = Array.from(
    new Map(
      options
        .filter(
          (option) =>
            option.userId &&
            (!selectedOrganizationID ||
              option.organizationId === selectedOrganizationID),
        )
        .map((option) => [
          option.userId as string,
          {
            label: option.displayName || option.userId,
            value: option.userId as string,
          },
        ]),
    ).values(),
  );

  return (
    <Col xs={24} md={12} xl={8}>
      <Form.Item label={label} style={{ marginInline: 8 }}>
        <Space.Compact style={{ width: '100%' }}>
          <Form.Item
            noStyle
            name={userField}
            dependencies={[organizationField]}
            rules={[
              ({ getFieldValue }) => ({
                validator: async (_, value) => {
                  if (!getFieldValue(organizationField) || value) return;
                  throw new Error(`请选择${label}`);
                },
              }),
            ]}
          >
            <Select
              allowClear
              disabled={disabled}
              showSearch
              optionFilterProp="label"
              options={userOptions}
              placeholder="请选择人员"
              style={{ width: '50%' }}
              onChange={(userID) => {
                if (!userID) return;
                const memberships = options.filter(
                  (option) => option.userId === userID,
                );
                if (
                  !memberships.some(
                    (option) =>
                      option.organizationId === selectedOrganizationID,
                  )
                ) {
                  form.setFieldValue(
                    organizationField,
                    memberships[0]?.organizationId,
                  );
                }
              }}
            />
          </Form.Item>
          <Form.Item
            noStyle
            name={organizationField}
            dependencies={[userField]}
            rules={[
              ({ getFieldValue }) => ({
                validator: async (_, value) => {
                  if (!getFieldValue(userField) || value) return;
                  throw new Error(`请选择${label}所属公司`);
                },
              }),
            ]}
          >
            <Select
              allowClear
              disabled={disabled}
              showSearch
              optionFilterProp="label"
              options={organizationOptions}
              placeholder="所属公司"
              style={{ width: '50%' }}
              onChange={(organizationID) => {
                if (
                  selectedUserID &&
                  !options.some(
                    (option) =>
                      option.userId === selectedUserID &&
                      option.organizationId === organizationID,
                  )
                ) {
                  form.setFieldValue(userField, undefined);
                }
              }}
            />
          </Form.Item>
        </Space.Compact>
      </Form.Item>
    </Col>
  );
}

function SeaScheduleDateFields() {
  const etd = Form.useWatch('etd');
  const eta = Form.useWatch('eta');
  const scheduleInvalid =
    dayjs.isDayjs(etd) && dayjs.isDayjs(eta) && eta.isBefore(etd, 'day');

  return (
    <>
      <ProFormDatePicker
        colProps={{ xs: 24, sm: 12, lg: 6, xl: 6 }}
        name="etd"
        label="ETD"
        fieldProps={{
          status: scheduleInvalid ? 'error' : undefined,
          style: { width: '100%' },
        }}
      />
      <ProFormDatePicker
        colProps={{ xs: 24, sm: 12, lg: 6, xl: 6 }}
        name="eta"
        label="ETA"
        dependencies={['etd']}
        rules={[
          ({ getFieldValue }: { getFieldValue: (name: string) => unknown }) => ({
            validator: async (_: unknown, value: unknown) => {
              const currentEtd = getFieldValue('etd');
              if (
                dayjs.isDayjs(currentEtd) &&
                dayjs.isDayjs(value) &&
                value.isBefore(currentEtd, 'day')
              ) {
                throw new Error('ETA 不能早于 ETD');
              }
            },
          }),
        ]}
        formItemProps={{
          help: scheduleInvalid ? 'ETA 不能早于 ETD' : undefined,
          validateStatus: scheduleInvalid ? 'error' : undefined,
        }}
        fieldProps={{
          status: scheduleInvalid ? 'error' : undefined,
          style: { width: '100%' },
        }}
      />
    </>
  );
}

export function getSeaTemplateSections(
  props: TemplateProps,
): TemplateSection[] {
  const {
    serviceTypeOptions,
    cargoCategoryOptions,
    locationOptions,
    currencyOptions,
    containerSpecOptions,
    searchCustomers,
    searchCarriers,
    searchBookingAgents,
    searchForeignAgents,
    searchShippingAgents,
    setCustomerCode,
    checkCustomerReferenceNo,
    checkInternalReferenceNo,
    personnelOptions,
    creator,
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
                      setCustomerCode(
                        (option as SelectOption | undefined)?.code,
                      ),
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
                <TooltipInput
                  placeholder="请输入"
                  maxLength={100}
                  suffix={
                    <Button
                      type="link"
                      size="small"
                      htmlType="button"
                      style={{ fontSize: 12, height: 21, padding: '0 2px' }}
                      onClick={() => void checkCustomerReferenceNo()}
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
                <TooltipInput
                  placeholder="请输入"
                  maxLength={100}
                  suffix={
                    <Button
                      type="link"
                      size="small"
                      htmlType="button"
                      style={{ fontSize: 12, height: 21, padding: '0 2px' }}
                      onClick={() => void checkInternalReferenceNo()}
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
            <Form.Item label="合约号" style={{ marginInline: 8 }}>
              <Form.Item noStyle name="contractNo">
                <TooltipInput
                  placeholder="请输入"
                  maxLength={100}
                />
              </Form.Item>
            </Form.Item>
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
                <TooltipInput
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
                <TooltipInput
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
                            if (!getFieldValue('insurancePremium') || value)
                              return;
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
            <Form.Item
              label="UN NO."
              name="unNumber"
              style={{ marginInline: 8 }}
              rules={[
                {
                  pattern: /^\d{4}$/,
                  message: 'UN NO. 应为 4 位数字',
                },
              ]}
            >
              <TooltipInput placeholder="4位数字" maxLength={4} />
            </Form.Item>
          </Col>
          <Col className="col-5">
            <Form.Item label="CLASS NO." name="hazardClass" style={{ marginInline: 8 }}>
              <TooltipInput placeholder="类别" maxLength={16} />
            </Form.Item>
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
            <Form.Item label="工厂" name="factoryName" style={{ marginInline: 8 }}>
              <TooltipInput placeholder="请输入工厂" maxLength={200} />
            </Form.Item>
          </Col>

          {/* 第 8 行：一行 5 列（委托单位代码、货好时间、后 3 列留白） */}
          <Col className="col-5">
            <Form.Item label="委托单位代码" name="customerCode" style={{ marginInline: 8 }}>
              <TooltipInput disabled placeholder="选择委托单位后自动带出" />
            </Form.Item>
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
          <SeaScheduleDateFields />

          <OrderContainerRequestFields options={containerSpecOptions} />

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
      title: '提单信息',
      content: (
        <>
          <OrderShippingDocumentFields />

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
      key: 'remarks',
      title: '备注',
      content: (
        <>
          <ProFormTextArea
            colProps={{ xs: 24, lg: 8 }}
            name="bookingNotes"
            label="订舱备注"
            placeholder="请输入订舱备注"
            fieldProps={{ maxLength: 1000, showCount: true, rows: 3 }}
          />
          <ProFormTextArea
            colProps={{ xs: 24, lg: 8 }}
            name="allocationNotes"
            label="配舱备注"
            placeholder="请输入配舱备注"
            fieldProps={{ maxLength: 1000, showCount: true, rows: 3 }}
          />
          <ProFormTextArea
            colProps={{ xs: 24, lg: 8 }}
            name="operationNotes"
            label="操作备注"
            placeholder="请输入操作备注"
            fieldProps={{ maxLength: 1000, showCount: true, rows: 3 }}
          />
        </>
      ),
    },
    {
      key: 'internalInfo',
      title: '内部信息',
      content: (
        <>
          <PersonnelAssignmentFields
            label="创建人员"
            userField="creatorUserId"
            organizationField="creatorOrganizationId"
            options={
              creator
                ? [
                    {
                      userId: creator.userId,
                      displayName: creator.displayName,
                      organizationId: creator.organizationId,
                      organizationName: creator.organizationName,
                    },
                  ]
                : []
            }
            disabled
          />
          <PersonnelAssignmentFields label="操作人员" userField="operatorUserId" organizationField="operatorOrganizationId" options={personnelOptions} />
          <PersonnelAssignmentFields label="业务人员" userField="salesUserId" organizationField="salesOrganizationId" options={personnelOptions} />
          <PersonnelAssignmentFields label="客服人员" userField="customerServiceUserId" organizationField="customerServiceOrganizationId" options={personnelOptions} />
          <PersonnelAssignmentFields label="关联人员" userField="associateUserId" organizationField="associateOrganizationId" options={personnelOptions} />
          <PersonnelAssignmentFields label="单证人员" userField="documentUserId" organizationField="documentOrganizationId" options={personnelOptions} />
          <PersonnelAssignmentFields label="商务人员" userField="commercialUserId" organizationField="commercialOrganizationId" options={personnelOptions} />
          <PersonnelAssignmentFields label="关联人员 2" userField="associate2UserId" organizationField="associate2OrganizationId" options={personnelOptions} />
        </>
      ),
    },
  ];
}
