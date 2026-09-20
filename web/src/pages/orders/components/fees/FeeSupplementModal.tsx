import type { ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDatePicker,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { Alert, Col, Row } from 'antd';
import React, { useRef } from 'react';
import {
  ExchangeRatePreviewCard,
  ProFormSearchableSelect,
} from '@/components/ui';
import { useFeeExchangePreview } from '../../use-fee-exchange-preview';
import type { FeeFormValues } from './FeeFormModal';
import { PAYABLE } from './feeConstants';

export type FeeSupplementFormValues = FeeFormValues & {
  /** 补录原因必填，参与服务端 request_fingerprint。 */
  reason: string;
};

export type FeeSupplementOptions = {
  feeSettings: API.OrderFeeSettingOption[];
  settlementParties: API.OrderFeeSettlementPartyOption[];
  currencies: API.OrderFeeCurrencyOption[];
  billingUnits: API.OrderFeeBillingUnitOption[];
};

type FeeSupplementModalProps = FeeSupplementOptions & {
  orderId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (values: FeeSupplementFormValues) => Promise<boolean>;
};

/** 锁后费用补录表单：复用普通费用字段，固定应付方向并要求补录原因。 */
export default function FeeSupplementModal({
  orderId,
  open,
  onOpenChange,
  feeSettings,
  settlementParties,
  currencies,
  billingUnits,
  onSubmit,
}: FeeSupplementModalProps) {
  const internalFormRef =
    useRef<ProFormInstance<FeeSupplementFormValues>>(undefined);
  const {
    totalPreview,
    exchangeRatePreview,
    exchangeRateStatus,
    manualExchangeRate,
    setManualExchangeRate,
    inheritedLastWeek,
    resetPreview,
    handleValuesChange,
  } = useFeeExchangePreview(
    orderId,
    internalFormRef as unknown as React.RefObject<
      ProFormInstance<FeeFormValues> | undefined
    >,
  );

  return (
    <ModalForm<FeeSupplementFormValues>
      title="补录费用（应付）"
      open={open}
      formRef={internalFormRef}
      onOpenChange={(next) => {
        if (next) {
          internalFormRef.current?.setFieldsValue({
            direction: PAYABLE,
            currency: 'CNY',
            quantity: '1',
            expenseDate: undefined,
          });
        } else {
          resetPreview();
        }
        onOpenChange(next);
      }}
      onFinish={onSubmit}
      onValuesChange={handleValuesChange}
      width={680}
      modalProps={{ destroyOnHidden: true }}
    >
      <Alert
        type="info"
        showIcon
        title="补录仅用于锁定订单追加真实发生的应付成本；审批通过后生成一条新的已确认应付费用，不会修改原有费用。应收方向不支持补录。"
        style={{ marginBottom: 16 }}
      />
      <Row gutter={16}>
        <Col span={12}>
          <ProFormSearchableSelect
            name="feeSettingId"
            label="费用项目"
            rules={[{ required: true, message: '请选择费用项目' }]}
            options={feeSettings.map((item) => ({
              label: `${item.nameZh || item.nameEn || item.feeCode} (${item.feeCode})`,
              value: item.id ?? '',
              code: item.feeCode,
              name: item.nameZh,
            }))}
            fieldProps={{
              onChange: (val) => {
                const setting = feeSettings.find((item) => item.id === val);
                if (setting?.defaultBillingUnitId) {
                  internalFormRef.current?.setFieldValue(
                    'billingUnitId',
                    setting.defaultBillingUnitId,
                  );
                }
                if (setting?.defaultCurrency) {
                  internalFormRef.current?.setFieldValue(
                    'currency',
                    setting.defaultCurrency,
                  );
                }
                handleValuesChange();
              },
            }}
          />
        </Col>
        <Col span={12}>
          <ProFormSearchableSelect
            name="settlementPartyId"
            label="结算单位"
            rules={[{ required: true, message: '请选择结算单位' }]}
            options={settlementParties.map((item) => ({
              label: item.name ?? '',
              value: item.id ?? '',
              code: item.code,
              name: item.name,
            }))}
          />
        </Col>
        <Col span={8}>
          <ProFormSearchableSelect
            name="currency"
            label="币种"
            rules={[{ required: true, message: '请选择币种' }]}
            options={currencies.map((c) => ({
              label: `${c.code} (${c.name})`,
              value: c.code ?? '',
              code: c.code,
              name: c.name,
            }))}
          />
        </Col>
        <Col span={8}>
          <ProFormText
            name="unitPrice"
            label="单价"
            rules={[{ required: true, message: '请输入单价' }]}
            placeholder="0.00"
          />
        </Col>
        <Col span={8}>
          <ProFormText
            name="quantity"
            label="数量"
            rules={[{ required: true, message: '请输入数量' }]}
            placeholder="1"
          />
        </Col>
        <Col span={12}>
          <ProFormSearchableSelect
            name="billingUnitId"
            label="计费单位"
            rules={[{ required: true, message: '请选择计费单位' }]}
            options={billingUnits.map((u) => ({
              label: `${u.name} (${u.code})`,
              value: u.id ?? '',
              code: u.code,
              name: u.name,
            }))}
          />
        </Col>
        <Col span={12}>
          <ProFormDatePicker
            name="expenseDate"
            label="发生日期"
            rules={[{ required: true, message: '请选择发生日期' }]}
            fieldProps={{ style: { width: '100%' } }}
          />
        </Col>

        <Col span={24}>
          <ExchangeRatePreviewCard
            amountPreview={totalPreview}
            currency={internalFormRef.current?.getFieldValue('currency')}
            amountColor="#fa8c16"
            status={exchangeRateStatus}
            ratePreview={exchangeRatePreview}
            inherited={inheritedLastWeek}
            onEnableManual={() => setManualExchangeRate(true)}
          />
        </Col>

        {manualExchangeRate && (
          <Col span={24}>
            <ProFormText
              name="exchangeRateOverride"
              label="手动指定汇率 (对 CNY)"
              rules={[{ required: true, message: '请输入手动汇率' }]}
              placeholder="例如 7.2345"
            />
          </Col>
        )}

        <Col span={24}>
          <ProFormTextArea
            name="reason"
            label="补录原因"
            rules={[
              { required: true, message: '请填写补录原因' },
              { max: 500, message: '补录原因不能超过 500 个字符' },
            ]}
            placeholder="请说明漏录成本的原因与业务背景（必填）"
            fieldProps={{ rows: 3, maxLength: 500, showCount: true }}
          />
        </Col>

        <Col span={24}>
          <ProFormTextArea
            name="note"
            label="备注说明"
            placeholder="请输入费用相关备注（可选）"
            fieldProps={{ rows: 2, maxLength: 500, showCount: true }}
          />
        </Col>
      </Row>
    </ModalForm>
  );
}
