import {
  ProFormDateTimePicker,
  ProFormRadio,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { Form } from 'antd';
import {
  CurrencyAmountInput,
  FormCell,
  FormRow,
  ProFormSearchableSelect,
} from '@/components/ui';
import { PartnerRoleType } from '@/enums.generated';
import {
  paymentTermOptions,
  shipmentModeOptions,
  tradeTermOptions,
} from '../../../common';
import PartnerQuickAddSelect from '../../../components/PartnerQuickAddSelect';
import type { TemplateProps } from '../../types';
import { TemplateControlGroup } from '../FieldGroup';
import { SeaCollapsibleNotesField } from '../sea/SeaCollapsibleNotesField';

/**
 * 空运本票经营字段片段：与海运 `getSeaBaseInfoFields` 同一写法，
 * 由模板按海运同序摆放到对应卡片，字段 name 与校验不随位置变化；
 * 这些字段只写本票，不写入共享主单或单证印字。
 */
export function getAirCommercialFields(props: TemplateProps) {
  return {
    shipmentMode: (
      <FormCell span={2}>
        <TemplateControlGroup title="集运/跨境">
          <ProFormRadio.Group
            name="shipmentMode"
            fieldProps={{ 'aria-label': '集运/跨境' }}
            options={shipmentModeOptions}
            disabled={props.readonly}
          />
        </TemplateControlGroup>
      </FormCell>
    ),
    trade: (
      <ProFormSearchableSelect
        name="tradeTerm"
        label="贸易条款"
        options={tradeTermOptions}
        disabled={props.readonly}
      />
    ),
    payment: (
      <ProFormSearchableSelect
        name="paymentTerm"
        label="业务付款方式"
        options={paymentTermOptions}
        rules={[{ required: true, message: '请选择业务付款方式' }]}
        fieldProps={{ allowClear: false }}
        disabled={props.readonly}
      />
    ),
    contract: (
      <FormCell span={2}>
        <ProFormText
          name="contractNo"
          label="合约号"
          fieldProps={{ maxLength: 100 }}
          disabled={props.readonly}
        />
      </FormCell>
    ),
    agents: (
      <>
        <FormCell span={2}>
          <PartnerQuickAddSelect
            name="bookingAgentId"
            displayName="订舱代理"
            role={PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER}
            createRoute="/partners/suppliers/create"
            searchPartners={props.searchBookingAgents}
            disabled={props.readonly}
          />
        </FormCell>
        <FormCell span={2}>
          <PartnerQuickAddSelect
            name="foreignAgentId"
            displayName="国外代理"
            role={PartnerRoleType.PARTNER_ROLE_TYPE_FOREIGN_AGENT}
            createRoute="/partners/foreign-agents/create"
            searchPartners={props.searchForeignAgents}
            disabled={props.readonly}
          />
        </FormCell>
      </>
    ),
    times: (
      <>
        <ProFormDateTimePicker
          name="receivedAt"
          label="接单时间"
          fieldProps={{ style: { width: '100%' } }}
          disabled={props.readonly}
        />
        <ProFormDateTimePicker
          name="cargoReadyAt"
          label="货好时间"
          fieldProps={{ style: { width: '100%' } }}
          disabled={props.readonly}
        />
      </>
    ),
    // 货值、保费保持独立的金额/币种对，可显式清空。
    cargoValue: (
      <FormCell span={2}>
        <Form.Item label="货值" htmlFor="cargoValue">
          <CurrencyAmountInput
            currencyName="cargoCurrency"
            amountName="cargoValue"
            currencyOptions={props.currencyOptions}
            disabled={props.readonly}
            requireAmountWhenCurrency
            amountRuleMessage="请输入正确的货值，最多 4 位小数"
            emptyAmountMessage="请输入货值"
            emptyCurrencyMessage="请选择币种"
          />
        </Form.Item>
      </FormCell>
    ),
    insurance: (
      <FormCell span={2}>
        <Form.Item label="保费" htmlFor="insurancePremium">
          <CurrencyAmountInput
            currencyName="insuranceCurrency"
            amountName="insurancePremium"
            currencyOptions={props.currencyOptions}
            disabled={props.readonly}
            requireAmountWhenCurrency
            amountRuleMessage="请输入正确的保费，最多 4 位小数"
            emptyAmountMessage="请输入保费"
            emptyCurrencyMessage="请选择币种"
          />
        </Form.Item>
      </FormCell>
    ),
    factory: (
      <ProFormText
        name="factoryName"
        label="工厂"
        fieldProps={{ maxLength: 200 }}
        disabled={props.readonly}
      />
    ),
  };
}

/** 委托品名与特殊要求对半，与海运货物描述行同规格。 */
export function AirCargoDescriptionFields({
  disabled,
}: {
  disabled?: boolean;
}) {
  return (
    <FormRow cols={2}>
      <ProFormTextArea
        name="goodsDescription"
        label="委托品名 / 货物描述"
        placeholder="请输入客户委托申报的中英文品名或货物描述"
        fieldProps={{
          maxLength: 1000,
          showCount: true,
          autoSize: { minRows: 2, maxRows: 6 },
        }}
        disabled={disabled}
      />
      <ProFormTextArea
        name="specialRequirements"
        label="特殊要求"
        placeholder="请输入客户或货物的特殊运输、装卸要求"
        fieldProps={{
          maxLength: 1000,
          showCount: true,
          autoSize: { minRows: 2, maxRows: 6 },
        }}
        disabled={disabled}
      />
    </FormRow>
  );
}

/**
 * 本票四类备注：与海运「订舱与运输 → 备注」同一折叠交互，一行四列。
 * 主单备注写入共享主单，随主单字段呈现，不在此处。
 */
export function AirNotesFields({ disabled }: { disabled?: boolean }) {
  return (
    <FormRow cols={4} style={{ rowGap: 4 }}>
      <SeaCollapsibleNotesField
        name="bookingNotes"
        label="订舱备注"
        placeholder="请输入订舱备注"
        disabled={disabled}
      />
      <SeaCollapsibleNotesField
        name="allocationNotes"
        label="配舱备注"
        placeholder="请输入配舱备注"
        disabled={disabled}
      />
      <SeaCollapsibleNotesField
        name="operationNotes"
        label="操作备注"
        placeholder="请输入操作备注"
        disabled={disabled}
      />
      <SeaCollapsibleNotesField
        name="notes"
        label="普通备注"
        placeholder="请输入普通备注"
        disabled={disabled}
      />
    </FormRow>
  );
}
