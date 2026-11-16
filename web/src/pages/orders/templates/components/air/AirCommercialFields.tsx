import {
  ProFormDateTimePicker,
  ProFormRadio,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { Form } from 'antd';
import {
  CurrencyAmountInput,
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

/** 空运本票经营字段，不写入共享主单或单证印字。 */
export function AirBusinessFields({ props }: { props: TemplateProps }) {
  return (
    <>
      <FormRow cols={3}>
        <ProFormRadio.Group
          name="shipmentMode"
          label="集运/跨境"
          options={shipmentModeOptions}
          disabled={props.readonly}
        />
        <ProFormSearchableSelect
          name="tradeTerm"
          label="贸易条款"
          options={tradeTermOptions}
          disabled={props.readonly}
        />
        <ProFormSearchableSelect
          name="paymentTerm"
          label="业务付款方式"
          options={paymentTermOptions}
          rules={[{ required: true, message: '请选择业务付款方式' }]}
          fieldProps={{ allowClear: false }}
          disabled={props.readonly}
        />
      </FormRow>
      <FormRow cols={3}>
        <PartnerQuickAddSelect
          name="bookingAgentId"
          displayName="订舱代理"
          role={PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER}
          createRoute="/partners/suppliers/create"
          searchPartners={props.searchBookingAgents}
          disabled={props.readonly}
        />
        <PartnerQuickAddSelect
          name="foreignAgentId"
          displayName="国外代理"
          role={PartnerRoleType.PARTNER_ROLE_TYPE_FOREIGN_AGENT}
          createRoute="/partners/foreign-agents/create"
          searchPartners={props.searchForeignAgents}
          disabled={props.readonly}
        />
        <ProFormText
          name="contractNo"
          label="合约号"
          fieldProps={{ maxLength: 100 }}
          disabled={props.readonly}
        />
      </FormRow>
    </>
  );
}

/** 货值、保费及本票时间保持独立，可显式清空。 */
export function AirCargoCommercialFields({ props }: { props: TemplateProps }) {
  return (
    <>
      <FormRow cols={3}>
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
        <ProFormText
          name="factoryName"
          label="工厂"
          fieldProps={{ maxLength: 200 }}
          disabled={props.readonly}
        />
      </FormRow>
      <FormRow cols={2}>
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
      </FormRow>
      <FormRow cols={2}>
        <ProFormTextArea
          name="goodsDescription"
          label="委托品名 / 货物描述"
          fieldProps={{ maxLength: 1000, autoSize: { minRows: 2, maxRows: 6 } }}
          disabled={props.readonly}
        />
        <ProFormTextArea
          name="specialRequirements"
          label="特殊要求"
          fieldProps={{ maxLength: 1000, autoSize: { minRows: 2, maxRows: 6 } }}
          disabled={props.readonly}
        />
      </FormRow>
      <FormRow cols={3}>
        <ProFormTextArea
          name="bookingNotes"
          label="订舱备注"
          fieldProps={{ maxLength: 1000, autoSize: { minRows: 1, maxRows: 4 } }}
          disabled={props.readonly}
        />
        <ProFormTextArea
          name="allocationNotes"
          label="配舱备注"
          fieldProps={{ maxLength: 1000, autoSize: { minRows: 1, maxRows: 4 } }}
          disabled={props.readonly}
        />
        <ProFormTextArea
          name="notes"
          label="普通备注"
          fieldProps={{ maxLength: 1000, autoSize: { minRows: 1, maxRows: 4 } }}
          disabled={props.readonly}
        />
      </FormRow>
    </>
  );
}
