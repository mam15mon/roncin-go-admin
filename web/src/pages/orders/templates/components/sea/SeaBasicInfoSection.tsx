import {
  ProFormCheckbox,
  ProFormDatePicker,
  ProFormDateTimePicker,
  ProFormRadio,
  ProFormText,
} from '@ant-design/pro-components';
import { Button, Form, Input, Tooltip } from 'antd';
import React from 'react';
import {
  CurrencyAmountInput,
  FormRow,
  ProFormSearchableSelect,
} from '@/components/ui';
import { PartnerAssignmentRole, PartnerRoleType } from '@/enums.generated';
import { partnerServiceGetPartner } from '@/services/roncin/partnerService';
import {
  shipmentModeOptions,
  shipmentTypeOptions,
  tradeTermOptions,
} from '../../../common';
import PartnerQuickAddSelect, {
  type PartnerSelectOption,
} from '../../../components/PartnerQuickAddSelect';
import type { SelectOption, TemplateProps } from '../../types';

export function TooltipInput(props: React.ComponentProps<typeof Input>) {
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
  return (
    <div style={{ width: '100%' }}>
      <ProFormCheckbox.Group
        name="serviceTypeIds"
        label="服务类型"
        options={options.map((option) => ({
          label: option.label,
          value: option.value,
        }))}
      />
    </div>
  );
}

export function SeaDangerousGoodsFields({
  options,
  disabled,
}: {
  options: SelectOption[];
  disabled?: boolean;
}) {
  const form = Form.useFormInstance();
  const cargoCategoryIds = (Form.useWatch('cargoCategoryIds') ??
    form?.getFieldValue('cargoCategoryIds')) as (string | number)[] | undefined;

  const isDangerousGoodsSelected = React.useMemo(() => {
    if (!Array.isArray(cargoCategoryIds) || cargoCategoryIds.length === 0) {
      return false;
    }
    const dgValues = new Set(
      options
        .filter(
          (opt) =>
            opt.code === 'DANGEROUS' ||
            opt.code === 'DG' ||
            opt.label === '危险品',
        )
        .map((opt) => opt.value),
    );

    return cargoCategoryIds.some(
      (val) =>
        dgValues.has(val) ||
        val === 'DANGEROUS' ||
        val === 'DG' ||
        val === '危险品',
    );
  }, [cargoCategoryIds, options]);

  const prevIsDgRef = React.useRef(isDangerousGoodsSelected);
  React.useEffect(() => {
    if (prevIsDgRef.current && !isDangerousGoodsSelected) {
      form?.setFieldsValue({
        unNumber: undefined,
        hazardClass: undefined,
        declarationCutoffAt: undefined,
      });
    }
    prevIsDgRef.current = isDangerousGoodsSelected;
  }, [isDangerousGoodsSelected, form]);

  if (!isDangerousGoodsSelected) {
    return null;
  }

  return (
    <FormRow cols={6}>
      <div style={{ gridColumn: 'span 2' }}>
        <Form.Item
          label="UN NO."
          name="unNumber"
          rules={[
            {
              pattern: /^\d{4}$/,
              message: 'UN NO. 应为 4 位数字',
            },
          ]}
        >
          <TooltipInput
            placeholder="4位数字"
            maxLength={4}
            disabled={disabled}
          />
        </Form.Item>
      </div>
      <div style={{ gridColumn: 'span 2' }}>
        <Form.Item label="CLASS NO." name="hazardClass">
          <TooltipInput placeholder="类别" maxLength={16} disabled={disabled} />
        </Form.Item>
      </div>
      <div style={{ gridColumn: 'span 2' }}>
        <ProFormDateTimePicker
          name="declarationCutoffAt"
          label="截申报时间"
          tooltip="主要监管或舱单申报截止时间；VGM、SI、截关仍使用各自独立节点"
          readonly={disabled}
          fieldProps={{ style: { width: '100%' } }}
        />
      </div>
    </FormRow>
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
      options={
        currentShippingLineOption ? [currentShippingLineOption] : undefined
      }
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

export function extractPersonnelFromPartnerAssignments(
  assignments: API.PartnerAssignment[] | undefined,
  _personnelOptions?: Array<{ userId?: string }>,
): Record<string, string | undefined> {
  if (!assignments || assignments.length === 0) {
    return {};
  }

  const findAssignment = (role: number, index = 0) => {
    const item = assignments
      .filter((a) => a.role === role)
      .sort((a, b) => (a.sortOrder ?? 0) - (b.sortOrder ?? 0))[index];
    if (!item?.userId) return undefined;
    return { userId: item.userId };
  };

  const operator = findAssignment(
    PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_OPERATOR,
  );
  const sales = findAssignment(
    PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_SALES,
  );
  const customerService = findAssignment(
    PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_CUSTOMER_SERVICE,
  );
  const commercial = findAssignment(
    PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_COMMERCIAL,
  );
  const contact1 = findAssignment(
    PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_INTERNAL_CONTACT,
    0,
  );
  const contact2 = findAssignment(
    PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_INTERNAL_CONTACT,
    1,
  );
  const doc = findAssignment(
    PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_DOCUMENT,
  );

  const updates: Record<string, string | undefined> = {};
  if (operator?.userId) {
    updates.operatorUserId = operator.userId;
  }
  if (sales?.userId) {
    updates.salesUserId = sales.userId;
  }
  if (customerService?.userId) {
    updates.customerServiceUserId = customerService.userId;
  }
  if (commercial?.userId) {
    updates.commercialUserId = commercial.userId;
  }
  if (contact1?.userId) {
    updates.associateUserId = contact1.userId;
  }
  if (contact2?.userId) {
    updates.associate2UserId = contact2.userId;
  }
  if (doc?.userId) {
    updates.documentUserId = doc.userId;
  }

  return updates;
}

export function SeaCustomerField({
  searchCustomers,
  readonly,
  setCustomerCode,
  personnelOptions,
  onCustomerChange,
}: {
  searchCustomers: (keyword?: string) => Promise<PartnerSelectOption[]>;
  readonly?: boolean;
  setCustomerCode: (code?: string) => void;
  personnelOptions?: Array<{ userId?: string }>;
  onCustomerChange?: (option?: PartnerSelectOption) => void;
}) {
  const form = Form.useFormInstance();
  const latestCustomerIdRef = React.useRef<string | number | undefined>(
    undefined,
  );

  const handlePartnerChange = async (option?: PartnerSelectOption) => {
    setCustomerCode(option?.code);
    onCustomerChange?.(option);

    const customerId = option?.value;
    latestCustomerIdRef.current = customerId;
    if (!customerId) {
      return;
    }

    try {
      const res = await partnerServiceGetPartner({ id: String(customerId) });
      if (latestCustomerIdRef.current !== customerId) {
        return;
      }
      const partner = res.data;
      if (!partner) return;

      const updates = extractPersonnelFromPartnerAssignments(
        partner.assignments,
        personnelOptions,
      );
      if (Object.keys(updates).length > 0 && form) {
        form.setFieldsValue(updates);
      }
    } catch {
      // 容错处理：获取客商内部信息失败时不阻断主表单操作
    }
  };

  return (
    <PartnerQuickAddSelect
      name="customerId"
      displayName="委托单位"
      role={PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER}
      createRoute="/partners/customers/create"
      searchPartners={searchCustomers}
      required
      disabled={readonly}
      onPartnerChange={handlePartnerChange}
    />
  );
}

export function getSeaBaseInfoFields(
  props: TemplateProps,
  createLayout = false,
) {
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
    orderIdentity: (
      <FormRow cols={6}>
        <ProFormText
          name="orderNo"
          label="订单编号"
          placeholder={props.isDetail ? '订单编号' : '保存后自动生成'}
          fieldProps={{ disabled: true }}
        />
        {!props.isDetail ? (
          <ProFormDatePicker
            name="orderDate"
            label="订单编号时间"
            tooltip="编号时间仅在订单初次保存前可修改"
            fieldProps={{ style: { width: '100%' } }}
          />
        ) : (
          <div style={{ gridColumn: 'span 5' }} />
        )}
      </FormRow>
    ),
    customer: (
      <FormRow cols={6}>
        <div style={{ gridColumn: 'span 2' }}>
          <SeaCustomerField
            searchCustomers={searchCustomers}
            readonly={props.readonly}
            setCustomerCode={setCustomerCode}
            personnelOptions={props.personnelOptions}
          />
        </div>
        <Form.Item label="委托单位代码" name="customerCode">
          <TooltipInput disabled placeholder="选择委托单位后自动带出" />
        </Form.Item>
        <ProFormRadio.Group
          name="shipmentMode"
          label="集运/跨境"
          options={shipmentModeOptions}
        />
        <div style={{ gridColumn: 'span 2' }}>
          <ProFormRadio.Group
            name="shipmentType"
            label="托运类型"
            options={shipmentTypeOptions}
          />
        </div>
      </FormRow>
    ),
    services: <SeaServiceTypeFields options={serviceTypeOptions} />,
    categories: (
      <div style={{ width: '100%' }}>
        <ProFormCheckbox.Group
          name="cargoCategoryIds"
          label="货物品类"
          options={cargoCategoryOptions}
        />
      </div>
    ),
    dangerous: (
      <SeaDangerousGoodsFields
        options={cargoCategoryOptions}
        disabled={props.readonly}
      />
    ),
    references: (
      <>
        <div style={{ gridColumn: 'span 2' }}>
          <Form.Item label="客户业务编号">
            <Form.Item noStyle name="customerReferenceNo">
              <TooltipInput
                placeholder="请输入"
                maxLength={100}
                suffix={
                  <Button
                    type="link"
                    size="small"
                    htmlType="button"
                    tabIndex={-1}
                    style={{ fontSize: 12, height: 21, padding: '0 2px' }}
                    onClick={() => void checkCustomerReferenceNo()}
                  >
                    重复校验
                  </Button>
                }
              />
            </Form.Item>
          </Form.Item>
        </div>
        <div style={{ gridColumn: 'span 2' }}>
          <Form.Item label="企业内部编号">
            <Form.Item noStyle name="internalReferenceNo">
              <TooltipInput
                placeholder="请输入"
                maxLength={100}
                suffix={
                  <Button
                    type="link"
                    size="small"
                    htmlType="button"
                    tabIndex={-1}
                    style={{ fontSize: 12, height: 21, padding: '0 2px' }}
                    onClick={() => void checkInternalReferenceNo()}
                  >
                    重复校验
                  </Button>
                }
              />
            </Form.Item>
          </Form.Item>
        </div>
      </>
    ),
    booking: (
      <Form.Item label="订舱号">
        <Form.Item noStyle name="bookingNo">
          <TooltipInput placeholder="请输入" maxLength={100} />
        </Form.Item>
      </Form.Item>
    ),
    trade: (
      <ProFormSearchableSelect
        name="tradeTerm"
        label="贸易条款"
        options={tradeTermOptions}
        placeholder="请选择"
      />
    ),
    agents: (
      <>
        <div style={{ gridColumn: 'span 2' }}>
          <PartnerQuickAddSelect
            name="bookingAgentId"
            displayName="订舱代理"
            role={PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER}
            createRoute="/partners/suppliers/create"
            searchPartners={searchBookingAgents}
            disabled={props.readonly}
          />
        </div>
        <div style={{ gridColumn: 'span 2' }}>
          <PartnerQuickAddSelect
            name="foreignAgentId"
            displayName="国外代理"
            role={PartnerRoleType.PARTNER_ROLE_TYPE_FOREIGN_AGENT}
            createRoute="/partners/foreign-agents/create"
            searchPartners={searchForeignAgents}
            disabled={props.readonly}
          />
        </div>
      </>
    ),
    carrier: (
      <SeaCarrierField
        isDetail={props.isDetail}
        readonly={props.readonly}
        searchShippingLines={searchShippingLines}
      />
    ),
    shippingAgent: (
      <ProFormSearchableSelect
        name="shippingAgentId"
        label="船代"
        placeholder="请选择"
        request={async ({ keyWords }: { keyWords?: string }) =>
          searchShippingAgents(keyWords)
        }
      />
    ),
    commercial: (
      <FormRow cols={createLayout ? 3 : 6}>
        <Form.Item label="合约号">
          <Form.Item noStyle name="contractNo">
            <TooltipInput placeholder="请输入" maxLength={100} />
          </Form.Item>
        </Form.Item>
        <Form.Item label="货值">
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
        <Form.Item label="保费">
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
        <ProFormDateTimePicker
          name="receivedAt"
          label="接单时间"
          fieldProps={{ style: { width: '100%' } }}
        />
        <ProFormDateTimePicker
          name="cargoReadyAt"
          label="货好时间"
          fieldProps={{ style: { width: '100%' } }}
        />
        <Form.Item label="工厂" name="factoryName">
          <TooltipInput placeholder="请输入工厂" maxLength={200} />
        </Form.Item>
      </FormRow>
    ),
  };
}

export function buildSeaBaseInfoSection(props: TemplateProps) {
  const fields = getSeaBaseInfoFields(props);
  return {
    key: 'basicInfo',
    title: '业务信息',
    content: (
      <div style={{ display: 'grid', gap: 12, width: '100%' }}>
        {fields.orderIdentity}
        {fields.customer}
        {fields.services}
        {fields.categories}
        {fields.dangerous}
        <FormRow cols={6}>
          {fields.references}
          {fields.booking}
          {fields.trade}
        </FormRow>
        <FormRow cols={6}>
          {fields.agents}
          {fields.carrier}
          {fields.shippingAgent}
        </FormRow>
        {fields.commercial}
      </div>
    ),
  };
}
