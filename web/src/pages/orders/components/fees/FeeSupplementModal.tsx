import type { ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDatePicker,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { Alert, Checkbox, Col, Row, Typography } from 'antd';
import dayjs from 'dayjs';
import React, { useEffect, useRef, useState } from 'react';
import { useAccess } from '@/app/access';
import {
  ExchangeRatePreviewCard,
  ProFormSearchableSelect,
} from '@/components/ui';
import { useFeeExchangePreview } from '../../use-fee-exchange-preview';
import type { FeeFormValues } from './FeeFormModal';
import { PAYABLE } from './feeConstants';
import { feeQuantityRuleError } from './feeQuantityRule';

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
  initialRequest?: API.OrderFeeSupplementRequestData;
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
  initialRequest,
}: FeeSupplementModalProps) {
  const access = useAccess();
  const [confirmSystemRate, setConfirmSystemRate] = useState(false);
  const [showRateConfirmationError, setShowRateConfirmationError] =
    useState(false);
  const [manualRateCleared, setManualRateCleared] = useState(false);
  const internalFormRef =
    useRef<ProFormInstance<FeeSupplementFormValues>>(undefined);
  const {
    totalPreview,
    exchangeRatePreview,
    exchangeRateStatus,
    manualExchangeRate,
    setManualExchangeRate,
    resetPreview,
    handleValuesChange,
  } = useFeeExchangePreview(
    orderId,
    internalFormRef as unknown as React.RefObject<
      ProFormInstance<FeeFormValues> | undefined
    >,
  );

  const unavailable: string[] = [];
  const availableValue = (
    value: string | undefined,
    valid: boolean,
    label: string,
  ) => {
    if (initialRequest && value && !valid) {
      unavailable.push(label);
      return undefined;
    }
    return value;
  };
  const feeSettingId = availableValue(
    initialRequest?.feeSettingId,
    feeSettings.some((item) => item.id === initialRequest?.feeSettingId),
    '费用项目',
  );
  const settlementPartyId = availableValue(
    initialRequest?.settlementPartyId,
    settlementParties.some(
      (item) => item.id === initialRequest?.settlementPartyId,
    ),
    '结算单位',
  );
  const billingUnitId = availableValue(
    initialRequest?.billingUnitId,
    billingUnits.some((item) => item.id === initialRequest?.billingUnitId),
    '计费单位',
  );
  const currency = availableValue(
    initialRequest?.currency,
    currencies.some((item) => item.code === initialRequest?.currency),
    '币种',
  );
  const manualRate = initialRequest?.exchangeRateSource === 'MANUAL';
  const canReuseManualRate = manualRate && access.canOverrideFeeExchangeRate;
  const requiresSystemRateConfirmation =
    manualRate && !access.canOverrideFeeExchangeRate;
  const prefill: Partial<FeeSupplementFormValues> = initialRequest
    ? {
        direction: PAYABLE,
        feeSettingId,
        settlementPartyId,
        billingUnitId,
        currency,
        quantity: initialRequest.quantity,
        unitPrice: initialRequest.unitPrice,
        expenseDate: initialRequest.expenseDate
          ? dayjs(initialRequest.expenseDate)
          : undefined,
        reason: initialRequest.reason,
        note: initialRequest.note,
        exchangeRateOverride: canReuseManualRate
          ? initialRequest.exchangeRate
          : undefined,
      }
    : { direction: PAYABLE, currency: 'CNY', quantity: '1' };

  useEffect(() => {
    if (!open) return;
    setConfirmSystemRate(false);
    setShowRateConfirmationError(false);
    setManualRateCleared(false);
    setManualExchangeRate(Boolean(canReuseManualRate));
    if (requiresSystemRateConfirmation) {
      internalFormRef.current?.setFieldValue('exchangeRateOverride', undefined);
    }
  }, [
    open,
    initialRequest?.id,
    canReuseManualRate,
    requiresSystemRateConfirmation,
    setManualExchangeRate,
  ]);

  return (
    <ModalForm<FeeSupplementFormValues>
      title="补录费用（应付）"
      open={open}
      formRef={internalFormRef}
      initialValues={prefill}
      onOpenChange={(next) => {
        if (next) {
          internalFormRef.current?.resetFields();
          internalFormRef.current?.setFieldsValue(prefill);
        } else {
          resetPreview();
        }
        onOpenChange(next);
      }}
      onFinish={(values) => {
        if (requiresSystemRateConfirmation && !confirmSystemRate) {
          setShowRateConfirmationError(true);
          return Promise.resolve(false);
        }
        return onSubmit(values);
      }}
      onValuesChange={(changedValues) => {
        const rateBasisChanged =
          'feeSettingId' in changedValues ||
          'currency' in changedValues ||
          'expenseDate' in changedValues;
        if (rateBasisChanged && manualExchangeRate) {
          setManualRateCleared(true);
        }
        if (
          rateBasisChanged ||
          'quantity' in changedValues ||
          'unitPrice' in changedValues
        ) {
          const retainedManualRate =
            manualExchangeRate && !rateBasisChanged
              ? internalFormRef.current?.getFieldValue('exchangeRateOverride')
              : undefined;
          handleValuesChange();
          if (retainedManualRate !== undefined) {
            internalFormRef.current?.setFieldValue(
              'exchangeRateOverride',
              retainedManualRate,
            );
            setManualExchangeRate(true);
          }
        }
        if (internalFormRef.current?.getFieldValue('quantity')) {
          void internalFormRef.current
            .validateFields(['quantity'])
            .catch(() => undefined);
        }
      }}
      width={680}
      modalProps={{ destroyOnHidden: true }}
    >
      <Alert
        type="info"
        showIcon
        title="补录仅用于锁定订单追加真实发生的应付成本；审批通过后生成一条新的未建账应付费用，不会修改原有费用。应收方向不支持补录。"
        style={{ marginBottom: 16 }}
      />
      {initialRequest && (
        <Alert
          type="info"
          showIcon
          title="已预填撤回申请，请核对后作为新申请提交"
          style={{ marginBottom: 16 }}
        />
      )}
      {unavailable.length > 0 && (
        <Alert
          type="warning"
          showIcon
          title={`${unavailable.join('、')}已不可用`}
          description="原值未带入新申请，请重新选择后再提交。"
          style={{ marginBottom: 16 }}
        />
      )}
      {requiresSystemRateConfirmation && (
        <Alert
          type="warning"
          showIcon
          title="原申请使用手动汇率，当前已无覆盖权限"
          description={
            <div>
              <p>原手动汇率不会沿用；新申请将按当前系统汇率重新计算。</p>
              <Checkbox
                checked={confirmSystemRate}
                onChange={(event) => {
                  setConfirmSystemRate(event.target.checked);
                  setShowRateConfirmationError(false);
                }}
              >
                我确认改用当前系统汇率
              </Checkbox>
              {showRateConfirmationError && (
                <Typography.Text type="danger" style={{ display: 'block' }}>
                  请先确认汇率变化
                </Typography.Text>
              )}
            </div>
          }
          style={{ marginBottom: 16 }}
        />
      )}
      {manualRateCleared && (
        <Alert
          type="warning"
          showIcon
          title="费用项目、币种或发生日期已变化，原手动汇率已清除"
          description="请重新指定手动汇率，或核对当前系统汇率后提交。"
          style={{ marginBottom: 16 }}
        />
      )}
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
                  void internalFormRef.current
                    ?.validateFields(['quantity'])
                    .catch(() => undefined);
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
            rules={[
              { required: true, message: '请输入数量' },
              {
                validator: (_, value) => {
                  const error = feeQuantityRuleError(
                    value,
                    internalFormRef.current?.getFieldValue('billingUnitId'),
                    billingUnits,
                  );
                  return error
                    ? Promise.reject(new Error(error))
                    : Promise.resolve();
                },
              },
            ]}
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
            onEnableManual={
              access.canOverrideFeeExchangeRate
                ? () => {
                    setManualRateCleared(false);
                    setManualExchangeRate(true);
                  }
                : undefined
            }
          />
        </Col>

        {manualExchangeRate && access.canOverrideFeeExchangeRate && (
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
