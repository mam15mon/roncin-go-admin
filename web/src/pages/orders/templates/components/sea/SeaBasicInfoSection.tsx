import {
  ProFormCheckbox,
  ProFormDatePicker,
  ProFormDateTimePicker,
  ProFormRadio,
  ProFormText,
} from '@ant-design/pro-components';
import { Button, Col, Form, Input, Row, Tag, Tooltip } from 'antd';
import React from 'react';
import { ProFormSearchableSelect, SearchableSelect } from '@/components/ui';
import {
  loadingTermsOptions,
  shipmentModeOptions,
  shipmentTypeOptions,
  tradeTermOptions,
} from '../../../common';
import { resolveSeaOrderFormPolicy } from '../../../sea-order-policy';
import type { SelectOption, TemplateProps } from '../../types';

export function TooltipInput(props: any) {
  const [value, setValue] = React.useState(props.value || '');
  React.useEffect(() => {
    setValue(props.value || '');
  }, [props.value]);

  return (
    <Tooltip title={value} placement="topLeft">
      <Input
        {...props}
        value={value}
        onChange={(e) => {
          setValue(e.target.value);
          props.onChange?.(e);
        }}
      />
    </Tooltip>
  );
}

export function SeaServiceTypeFields({
  options,
}: {
  options: SelectOption[];
}) {
  const shipmentMode = Form.useWatch('shipmentMode');
  const policy = resolveSeaOrderFormPolicy({ shipmentMode });
  const recommendedCodes = new Set(policy.recommendedServiceCodes);

  return (
    <Col span={24}>
      <ProFormCheckbox.Group
        name="serviceTypeIds"
        label="服务类型"
        options={options.map((option) => ({
          label: (
            <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
              <span>{option.label}</span>
              {option.code && recommendedCodes.has(option.code) && (
                <Tag
                  variant="filled"
                  color="blue"
                  style={{ marginInlineEnd: 0 }}
                >
                  推荐
                </Tag>
              )}
            </span>
          ),
          value: option.value,
        }))}
      />
    </Col>
  );
}

export function buildSeaBaseInfoSection(props: TemplateProps) {
  const {
    serviceTypeOptions,
    cargoCategoryOptions,
    currencyOptions,
    searchCustomers,
    searchCarriers,
    searchBookingAgents,
    searchForeignAgents,
    searchShippingAgents,
    setCustomerCode,
    checkCustomerReferenceNo,
    checkInternalReferenceNo,
  } = props;

  return {
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
                placeholder={props.isDetail ? '订单编号' : '保存后自动生成'}
                fieldProps={{ disabled: true }}
              />
            </Col>
            {!props.isDetail && (
              <>
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
              </>
            )}
          </Row>
        </Col>

        {/* 第 2 行：委托单位、集运/跨境、托运类型 */}
        <Col span={24}>
          <Row gutter={16} align="middle">
            <Col className="col-5">
              <ProFormSearchableSelect
                name="customerId"
                label="委托单位"
                rules={[{ required: true, message: '请选择客户单位' }]}
                fieldProps={{
                  placeholder: '请选择',
                  onChange: (_: any, option: any) =>
                    setCustomerCode(
                      (option as SelectOption | undefined)?.code,
                    ),
                }}
                request={async ({ keyWords }: { keyWords?: string }) =>
                  searchCustomers(keyWords)
                }
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
        <SeaServiceTypeFields options={serviceTypeOptions} />

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
          <ProFormSearchableSelect
            name="tradeTerm"
            label="贸易条款"
            rules={[{ required: true, message: '请选择贸易条款' }]}
            options={tradeTermOptions}
            placeholder="请选择"
          />
        </Col>
        <Col className="col-5">
          <ProFormSearchableSelect
            name="bookingAgentId"
            label="订舱代理"
            placeholder="请选择"
            request={async ({ keyWords }: { keyWords?: string }) =>
              searchBookingAgents(keyWords)
            }
          />
        </Col>
        <Col className="col-5">
          <ProFormSearchableSelect
            name="foreignAgentId"
            label="国外代理"
            placeholder="请选择"
            request={async ({ keyWords }: { keyWords?: string }) =>
              searchForeignAgents(keyWords)
            }
          />
        </Col>

        {/* 第 6 行：一行 5 列（合约号、船公司、船代、货值、保费） */}
        <Col className="col-5">
          <Form.Item label="合约号" style={{ marginInline: 8 }}>
            <Form.Item noStyle name="contractNo">
              <TooltipInput placeholder="请输入" maxLength={100} />
            </Form.Item>
          </Form.Item>
        </Col>
        <Col className="col-5">
          <ProFormSearchableSelect
            name="carrierId"
            label="船公司"
            placeholder="请选择"
            request={async ({ keyWords }: { keyWords?: string }) =>
              searchCarriers(keyWords)
            }
          />
        </Col>
        <Col className="col-5">
          <ProFormSearchableSelect
            name="shippingAgentId"
            label="船代"
            placeholder="请选择"
            request={async ({ keyWords }: { keyWords?: string }) =>
              searchShippingAgents(keyWords)
            }
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
                  <span onMouseDown={(e) => e.stopPropagation()}>
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
                      <SearchableSelect
                        popupMatchSelectWidth={false}
                        options={currencyOptions}
                        placeholder="币种"
                        size="small"
                        variant="borderless"
                        style={{ width: 72, height: 21 }}
                      />
                    </Form.Item>
                  </span>
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
                  <span onMouseDown={(e) => e.stopPropagation()}>
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
                      <SearchableSelect
                        popupMatchSelectWidth={false}
                        options={currencyOptions}
                        placeholder="币种"
                        size="small"
                        variant="borderless"
                        style={{ width: 72, height: 21 }}
                      />
                    </Form.Item>
                  </span>
                }
              />
            </Form.Item>
          </Form.Item>
        </Col>

        {/* 第 7 行：危险品、运输条款与合规时间 */}
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
          <Form.Item
            label="CLASS NO."
            name="hazardClass"
            style={{ marginInline: 8 }}
          >
            <TooltipInput placeholder="类别" maxLength={16} />
          </Form.Item>
        </Col>
        <Col className="col-5">
          <ProFormSearchableSelect
            name="loadingTerms"
            label="运输条款"
            options={loadingTermsOptions}
            placeholder="请选择 CY / CFS / DOOR 条款"
          />
        </Col>
        <Col className="col-5">
          <ProFormDateTimePicker
            name="declarationCutoffAt"
            label="截申报时间"
            tooltip="主要监管或舱单申报截止时间；VGM、SI、截关仍使用各自独立节点"
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
          <Form.Item
            label="工厂"
            name="factoryName"
            style={{ marginInline: 8 }}
          >
            <TooltipInput placeholder="请输入工厂" maxLength={200} />
          </Form.Item>
        </Col>

        {/* 第 8 行：一行 5 列（委托单位代码、货好时间、后 3 列留白） */}
        <Col className="col-5">
          <Form.Item
            label="委托单位代码"
            name="customerCode"
            style={{ marginInline: 8 }}
          >
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
  };
}
