import {
  ProFormCheckbox,
  ProFormDatePicker,
  ProFormDateTimePicker,
  ProFormRadio,
  ProFormText,
} from '@ant-design/pro-components';
import { Button, Col, Form, Input, Row, Tag, Tooltip } from 'antd';
import React from 'react';
import { CurrencyAmountInput, ProFormSearchableSelect } from '@/components/ui';
import { PartnerRoleType } from '@/enums.generated';
import {
  shipmentModeOptions,
  shipmentTypeOptions,
  tradeTermOptions,
} from '../../../common';
import PartnerQuickAddSelect from '../../../components/PartnerQuickAddSelect';
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

export function SeaServiceTypeFields({ options }: { options: SelectOption[] }) {
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
            <span
              style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}
            >
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

function SeaCarrierField({
  isDetail,
  readonly,
  searchShippingLines,
}: {
  isDetail?: boolean;
  readonly?: boolean;
  searchShippingLines: (keyword?: string) => Promise<SelectOption[]>;
}) {
  const form = Form.useFormInstance();
  const existingMbl = Form.useWatch('seaMasterBill', {
    form,
    preserve: true,
  }) as API.SeaMasterBillSummary | undefined;
  const isMultiMemberLocked =
    isDetail && !!existingMbl && (existingMbl.memberCount ?? 0) > 1;
  const currentShippingLineOption = existingMbl?.shippingLineId
    ? {
        label: existingMbl.shippingLineName || existingMbl.shippingLineId,
        value: existingMbl.shippingLineId,
      }
    : undefined;

  return (
    <ProFormSearchableSelect
      name="shippingLineId"
      label="船公司"
      placeholder="请选择"
      options={currentShippingLineOption ? [currentShippingLineOption] : []}
      disabled={readonly || isMultiMemberLocked}
      tooltip={
        isMultiMemberLocked
          ? '该主单已关联多票订单，禁止从单票页面修改船公司'
          : undefined
      }
      rules={[{ required: true, message: '请选择船公司' }]}
      request={
        readonly || isMultiMemberLocked
          ? undefined
          : async ({ keyWords }: { keyWords?: string }) => {
              const options = await searchShippingLines(keyWords);
              if (
                !currentShippingLineOption ||
                options.some(
                  (option) => option.value === currentShippingLineOption.value,
                )
              ) {
                return options;
              }
              return [currentShippingLineOption, ...options];
            }
      }
    />
  );
}

export function buildSeaBaseInfoSection(props: TemplateProps) {
  const {
    serviceTypeOptions,
    cargoCategoryOptions,
    currencyOptions,
    searchCustomers,
    searchShippingLines,
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
              <PartnerQuickAddSelect
                name="customerId"
                displayName="委托单位"
                role={PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER}
                createRoute="/partners/customers/create"
                searchPartners={searchCustomers}
                required
                disabled={props.readonly}
                onPartnerChange={(option) => setCustomerCode(option?.code)}
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

        {/* 第 5 行：客户业务编号、企业内部编号、订舱号及业务属性 */}
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
          <Form.Item label="订舱号" style={{ marginInline: 8 }}>
            <Form.Item noStyle name="bookingNo">
              <TooltipInput placeholder="请输入" maxLength={100} />
            </Form.Item>
          </Form.Item>
        </Col>
        <Col className="col-5">
          <ProFormSearchableSelect
            name="tradeTerm"
            label="贸易条款"
            options={tradeTermOptions}
            placeholder="请选择"
          />
        </Col>
        <Col className="col-5">
          <PartnerQuickAddSelect
            name="bookingAgentId"
            displayName="订舱代理"
            role={PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER}
            createRoute="/partners/suppliers/create"
            searchPartners={searchBookingAgents}
            disabled={props.readonly}
          />
        </Col>
        <Col className="col-5">
          <PartnerQuickAddSelect
            name="foreignAgentId"
            displayName="国外代理"
            role={PartnerRoleType.PARTNER_ROLE_TYPE_FOREIGN_AGENT}
            createRoute="/partners/foreign-agents/create"
            searchPartners={searchForeignAgents}
            disabled={props.readonly}
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
          <SeaCarrierField
            isDetail={props.isDetail}
            readonly={props.readonly}
            searchShippingLines={searchShippingLines}
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
            <CurrencyAmountInput
              currencyName="cargoCurrency"
              amountName="cargoValue"
              currencyOptions={currencyOptions}
              disabled={props.readonly}
              amountPlaceholder="金额"
              amountRuleMessage="请输入正确的货值，最多 4 位小数"
              emptyAmountMessage="请输入货值"
              emptyCurrencyMessage="请选择币种"
            />
          </Form.Item>
        </Col>
        <Col className="col-5">
          <Form.Item label="保费" style={{ marginInline: 8 }}>
            <CurrencyAmountInput
              currencyName="insuranceCurrency"
              amountName="insurancePremium"
              currencyOptions={currencyOptions}
              disabled={props.readonly}
              amountPlaceholder="金额"
              amountRuleMessage="请输入正确的保费，最多 4 位小数"
              emptyAmountMessage="请输入保费"
              emptyCurrencyMessage="请选择币种"
            />
          </Form.Item>
        </Col>

        {/* 第 7 行：危险品与合规时间（运输条款在提单信息区块维护） */}
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
