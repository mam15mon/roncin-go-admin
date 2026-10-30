import {
  ProFormCheckbox,
  ProFormDatePicker,
  ProFormDateTimePicker,
  ProFormRadio,
  ProFormSelect,
  ProFormText,
} from '@ant-design/pro-components';
import { useQuery } from '@tanstack/react-query';
import { Alert, Button, Form, Input, Tooltip } from 'antd';
import { createStyles } from 'antd-style';
import React from 'react';
import {
  CurrencyAmountInput,
  FormRow,
  ProFormSearchableSelect,
} from '@/components/ui';
import {
  DangerousPackingGroup,
  OrderReferenceType,
  PartnerAssignmentRole,
  PartnerRoleType,
} from '@/enums.generated';
import { orderServiceCheckOrderReference } from '@/services/roncin/orderService';
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
import { TemplateControlGroup } from '../FieldGroup';

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

/** 查重交给表单原生异步校验调度，防抖和过期结果由 Form.Item 管理。 */
export function SeaOrderReferenceField({
  name,
  isDetail = false,
}: {
  name: 'customerReferenceNo' | 'internalReferenceNo';
  isDetail?: boolean;
}) {
  const form = Form.useFormInstance();
  const value = Form.useWatch(name, form);
  const isCustomer = name === 'customerReferenceNo';
  const label = isCustomer ? '客户业务编号' : '企业内部编号';

  return (
    <Form.Item
      label={label}
      name={name}
      dependencies={isCustomer ? ['customerId'] : undefined}
      validateDebounce={300}
      hasFeedback={!isDetail && Boolean(String(value ?? '').trim())}
      rules={
        isDetail
          ? undefined
          : [
              ({ getFieldValue }) => ({
                warningOnly: true,
                async validator(_, rawValue) {
                  const referenceNo = String(rawValue ?? '').trim();
                  if (!referenceNo) return;
                  const selectedCustomerId = getFieldValue('customerId');
                  if (isCustomer && !selectedCustomerId) {
                    throw new Error('请先选择委托单位');
                  }
                  const response = await orderServiceCheckOrderReference(
                    {
                      referenceType: isCustomer
                        ? OrderReferenceType.ORDER_REFERENCE_TYPE_CUSTOMER
                        : OrderReferenceType.ORDER_REFERENCE_TYPE_INTERNAL,
                      referenceNo,
                      customerId: isCustomer ? selectedCustomerId : undefined,
                    },
                    { skipErrorHandler: true },
                  );
                  if (!response?.data || response.success === false) {
                    throw new Error(`${label}查重失败，请重试`);
                  }
                  if (response.data.duplicate) {
                    throw new Error(
                      `${label}已用于订单 ${response.data.orderNo || response.data.orderId}`,
                    );
                  }
                },
              }),
            ]
      }
    >
      <TooltipInput placeholder="请输入" maxLength={100} suffix={<span />} />
    </Form.Item>
  );
}

const useDangerousGoodsStyles = createStyles(({ css }) => ({
  reveal: css`
    display: grid;
    grid-template-rows: 0fr;
    opacity: 0;
    visibility: hidden;
    transition:
      grid-template-rows 180ms ease,
      opacity 180ms ease,
      visibility 0s 180ms;

    &[data-open='true'] {
      grid-template-rows: 1fr;
      opacity: 1;
      visibility: visible;
      transition-delay: 0s;
    }
    .dangerous-goods-clip {
      min-height: 0;
      overflow: hidden;
    }
    .dangerous-goods-content {
      padding-top: 12px;
    }

    @media (prefers-reduced-motion: reduce) {
      transition: none;
    }
  `,
}));

export function SeaDangerousGoodsFields({
  options,
  disabled,
  showPackingGroup = false,
}: {
  options: SelectOption[];
  disabled?: boolean;
  /** 仅空运出口需要包装类别（IATA DGR）；海运不传，渲染保持三字段不变。 */
  showPackingGroup?: boolean;
}) {
  const { styles } = useDangerousGoodsStyles();
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
        dangerousPackingGroup: undefined,
      });
    }
    prevIsDgRef.current = isDangerousGoodsSelected;
  }, [isDangerousGoodsSelected, form]);

  return (
    <div
      className={styles.reveal}
      data-testid="sea-dangerous-goods-motion"
      data-open={isDangerousGoodsSelected}
      aria-hidden={!isDangerousGoodsSelected}
      inert={!isDangerousGoodsSelected}
    >
      <div className="dangerous-goods-clip">
        <div className="dangerous-goods-content">
          <FormRow cols={showPackingGroup ? 4 : 3}>
            <div>
              <Form.Item
                label="UN NO."
                name="unNumber"
                rules={
                  isDangerousGoodsSelected
                    ? [
                        {
                          pattern: /^\d{4}$/,
                          message: 'UN NO. 应为 4 位数字',
                        },
                      ]
                    : undefined
                }
              >
                <TooltipInput
                  placeholder="4位数字"
                  maxLength={4}
                  disabled={disabled || !isDangerousGoodsSelected || undefined}
                />
              </Form.Item>
            </div>
            <div>
              <Form.Item label="CLASS NO." name="hazardClass">
                <TooltipInput
                  placeholder="类别"
                  maxLength={16}
                  disabled={disabled || !isDangerousGoodsSelected || undefined}
                />
              </Form.Item>
            </div>
            <div>
              <ProFormDateTimePicker
                name="declarationCutoffAt"
                label="截申报时间"
                tooltip="主要监管或舱单申报截止时间；VGM、SI、截关仍使用各自独立节点"
                readonly={disabled}
                disabled={disabled || !isDangerousGoodsSelected || undefined}
                fieldProps={{ style: { width: '100%' } }}
              />
            </div>
            {showPackingGroup && (
              <div>
                <ProFormSelect
                  name="dangerousPackingGroup"
                  label="包装类别"
                  tooltip="IATA DGR 包装类别：Ⅰ类危险性最大、Ⅲ类最小；Class 1/2/7 无包装类别，可留空"
                  options={dangerousPackingGroupOptions}
                  disabled={disabled || !isDangerousGoodsSelected || undefined}
                  fieldProps={{
                    style: { width: '100%' },
                    allowClear: true,
                  }}
                />
              </div>
            )}
          </FormRow>
        </div>
      </div>
    </div>
  );
}

const dangerousPackingGroupOptions = [
  { label: 'Ⅰ类', value: DangerousPackingGroup.DANGEROUS_PACKING_GROUP_I },
  { label: 'Ⅱ类', value: DangerousPackingGroup.DANGEROUS_PACKING_GROUP_II },
  { label: 'Ⅲ类', value: DangerousPackingGroup.DANGEROUS_PACKING_GROUP_III },
];

function SeaCarrierField({
  seaHouseOrderCreateSource,
  isDetail,
  readonly,
  searchShippingLines,
}: {
  isDetail?: boolean;
  readonly?: boolean;
  searchShippingLines: (keyword?: string) => Promise<SelectOption[]>;
} & Pick<TemplateProps, 'seaHouseOrderCreateSource'>) {
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
    : seaHouseOrderCreateSource
      ? {
          value: seaHouseOrderCreateSource.shippingLineId,
          label:
            seaHouseOrderCreateSource.shippingLineName || '船公司名称读取失败',
        }
      : undefined;

  return (
    <ProFormSearchableSelect
      name="shippingLineId"
      params={{
        currentShippingLineId: currentShippingLineOption?.value,
        currentShippingLineName: currentShippingLineOption?.label,
      }}
      label="船公司"
      placeholder="请选择"
      options={
        currentShippingLineOption ? [currentShippingLineOption] : undefined
      }
      disabled={readonly || isMultiMemberLocked || undefined}
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
  personnelOptions?: Array<{ userId?: string; displayName?: string }>;
  onCustomerChange?: (option?: PartnerSelectOption) => void;
}) {
  const form = Form.useFormInstance();
  const [selection, setSelection] = React.useState<{
    id?: string;
    revision: number;
    preserveExisting?: boolean;
  }>({ revision: 0 });
  const restoredCustomerId = Form.useWatch('customerId', {
    form,
    preserve: true,
  });
  const restoredSalesId = Form.useWatch('salesUserId', {
    form,
    preserve: true,
  });
  const restoredServiceId = Form.useWatch('customerServiceUserId', {
    form,
    preserve: true,
  });
  const instanceId = React.useId();
  const appliedRevision = React.useRef<number | undefined>(undefined);
  const personnelFields = [
    'operatorUserId',
    'salesUserId',
    'customerServiceUserId',
    'commercialUserId',
    'associateUserId',
    'associate2UserId',
    'documentUserId',
  ];
  const customerQuery = useQuery({
    queryKey: ['orders', 'customer-personnel', { instanceId, ...selection }],
    enabled: Boolean(selection.id && !readonly),
    queryFn: async () => {
      if (!selection.id) throw new Error('请选择客户');
      const response = await partnerServiceGetPartner(
        { id: selection.id },
        { skipErrorHandler: true },
      );
      if (!response?.data || response.success === false)
        throw new Error('客户人员资料加载失败');
      return response.data;
    },
    meta: { silent: true },
  });
  React.useEffect(() => {
    if (
      readonly ||
      !customerQuery.isSuccess ||
      customerQuery.isFetching ||
      appliedRevision.current === selection.revision
    )
      return;
    appliedRevision.current = selection.revision;
    const updates = extractPersonnelFromPartnerAssignments(
      customerQuery.data.assignments,
      personnelOptions,
    );
    form.setFields(
      Object.entries(updates)
        .filter(
          ([name]) =>
            !form.isFieldTouched(name) &&
            (!selection.preserveExisting || !form.getFieldValue(name)),
        )
        .map(([name, value]) => ({ name, value, touched: false })),
    );
  }, [
    customerQuery.data,
    customerQuery.isSuccess,
    customerQuery.isFetching,
    selection.revision,
    selection.preserveExisting,
    readonly,
    form,
    personnelOptions,
  ]);

  const handlePartnerChange = (option?: PartnerSelectOption) => {
    setCustomerCode(option?.code);
    onCustomerChange?.(option);
    // 新客户意图清理旧默认值，同时重新开始追踪用户实际修改（包括主动清空）。
    form.setFields(
      personnelFields.map((name) => ({
        name,
        value: undefined,
        touched: false,
      })),
    );
    setSelection((previous) => ({
      id: option?.value ? String(option.value) : undefined,
      revision: previous.revision + 1,
    }));
  };
  const missingStaff =
    customerQuery.isSuccess &&
    !customerQuery.isFetching &&
    (!customerQuery.data.assignments?.some(
      (item) =>
        item.role === PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_SALES &&
        item.userId,
    ) ||
      !customerQuery.data.assignments?.some(
        (item) =>
          item.role ===
            PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_CUSTOMER_SERVICE &&
          item.userId,
      ));
  return (
    <div>
      <PartnerQuickAddSelect
        name="customerId"
        displayName="委托单位"
        role={PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER}
        createRoute="/partners/customers/create"
        searchPartners={searchCustomers}
        staffOptions={personnelOptions}
        required
        disabled={readonly}
        onPartnerChange={handlePartnerChange}
      />
      {!selection.id &&
        restoredCustomerId &&
        !readonly &&
        (!restoredSalesId || !restoredServiceId) && (
          <Alert
            type="warning"
            title="客户业务或客服人员尚未带入，可重新加载客户人员。"
            action={
              <Button
                size="small"
                onClick={() =>
                  setSelection((previous) => ({
                    id: String(restoredCustomerId),
                    revision: previous.revision + 1,
                    preserveExisting: true,
                  }))
                }
              >
                重新加载客户人员
              </Button>
            }
          />
        )}
      {selection.id && !readonly && customerQuery.isFetching && (
        <div role="status">正在加载客户默认人员…</div>
      )}
      {selection.id && !readonly && customerQuery.isError && (
        <Alert
          type="error"
          title="客户人员加载失败，请重试"
          action={
            <Button size="small" onClick={() => void customerQuery.refetch()}>
              重试人员加载
            </Button>
          }
        />
      )}
      {selection.id && !readonly && missingStaff && (
        <Alert
          type="warning"
          title="客户尚未配置业务或客服人员，请联系客户档案维护人员补齐后重新选择客户。"
        />
      )}
    </div>
  );
}

export function getSeaBaseInfoFields(props: TemplateProps) {
  const {
    cargoCategoryOptions,
    currencyOptions,
    searchCustomers,
    searchShippingLines,
    searchBookingAgents,
    searchForeignAgents,
    searchShippingAgents,
    setCustomerCode,
  } = props;
  return {
    orderIdentity: (
      <>
        <ProFormText
          name="orderNo"
          label="订单编号"
          placeholder={props.isDetail ? '订单编号' : '保存后自动生成'}
          fieldProps={{ disabled: true }}
        />
        {!props.isDetail && (
          <ProFormDatePicker
            name="orderDate"
            label="订单编号时间"
            tooltip="编号时间仅在订单初次保存前可修改"
            fieldProps={{ style: { width: '100%' } }}
          />
        )}
      </>
    ),
    customer: (
      <FormRow cols={4}>
        <SeaCustomerField
          searchCustomers={searchCustomers}
          readonly={props.readonly}
          setCustomerCode={setCustomerCode}
          personnelOptions={props.personnelOptions}
        />
        <Form.Item label="委托单位代码" name="customerCode">
          <TooltipInput disabled placeholder="选择委托单位后自动带出" />
        </Form.Item>
        <TemplateControlGroup title="集运/跨境">
          <ProFormRadio.Group
            name="shipmentMode"
            fieldProps={{ 'aria-label': '集运/跨境' }}
            options={shipmentModeOptions}
          />
        </TemplateControlGroup>
        <TemplateControlGroup title="托运类型">
          <ProFormRadio.Group
            name="shipmentType"
            fieldProps={{ 'aria-label': '托运类型' }}
            options={shipmentTypeOptions}
          />
        </TemplateControlGroup>
      </FormRow>
    ),
    categories: (
      <div style={{ width: '100%' }}>
        <TemplateControlGroup title="货物品类">
          <ProFormCheckbox.Group
            name="cargoCategoryIds"
            fieldProps={{ 'aria-label': '货物品类', name: 'cargoCategoryIds' }}
            options={cargoCategoryOptions}
          />
        </TemplateControlGroup>
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
        <SeaOrderReferenceField
          name="customerReferenceNo"
          isDetail={props.isDetail}
        />
        <SeaOrderReferenceField
          name="internalReferenceNo"
          isDetail={props.isDetail}
        />
      </>
    ),
    booking: (
      <Form.Item label="订舱号" htmlFor="bookingNo">
        <Form.Item noStyle name="bookingNo">
          <TooltipInput
            placeholder="请输入"
            maxLength={100}
            disabled={props.readonly}
          />
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
        <div>
          <PartnerQuickAddSelect
            name="bookingAgentId"
            displayName="订舱代理"
            role={PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER}
            createRoute="/partners/suppliers/create"
            searchPartners={searchBookingAgents}
            disabled={props.readonly}
          />
        </div>
        <div>
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
        seaHouseOrderCreateSource={props.seaHouseOrderCreateSource}
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
        fieldProps={{ filterOption: false }}
        request={async ({ keyWords }: { keyWords?: string }) =>
          searchShippingAgents(keyWords)
        }
      />
    ),
    contract: (
      <Form.Item label="合约号" htmlFor="contractNo">
        <Form.Item noStyle name="contractNo">
          <TooltipInput
            placeholder="请输入"
            maxLength={100}
            disabled={props.readonly}
          />
        </Form.Item>
      </Form.Item>
    ),
    cargoValue: (
      <Form.Item label="货值" htmlFor="cargoValue">
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
    ),
    insurance: (
      <Form.Item label="保费" htmlFor="insurancePremium">
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
    ),
    commercial: (
      <FormRow cols={4}>
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
        <div style={{ gridColumn: 'span 2' }}>
          <Form.Item label="工厂" name="factoryName">
            <TooltipInput
              placeholder="请输入工厂"
              maxLength={200}
              disabled={props.readonly}
            />
          </Form.Item>
        </div>
      </FormRow>
    ),
  };
}
