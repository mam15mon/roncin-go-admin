import { PlusOutlined } from '@ant-design/icons';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDatePicker,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { Col, Row } from 'antd';
import type { Dayjs } from 'dayjs';
import React, { useRef } from 'react';
import {
  ExchangeRatePreviewCard,
  ProFormSearchableSelect,
} from '@/components/ui';
import { exchangeRatePattern, quantityOrPricePattern } from '@/utils/decimal';

const positiveDecimalRule =
  (pattern: RegExp, messageText: string) => (_: unknown, value?: string) => {
    if (!value) return Promise.resolve();
    const trimmed = value.trim();
    if (!pattern.test(trimmed) || Number(trimmed) <= 0) {
      return Promise.reject(new Error(messageText));
    }
    return Promise.resolve();
  };

export type FeeFormValues = {
  direction?: number;
  feeSettingId: string;
  settlementPartyId: string;
  currency: string;
  unitPrice: string;
  quantity: string;
  billingUnitId: string;
  expenseDate: string | Dayjs;
  exchangeRateOverride?: string;
  note?: string;
};

type FeeFormModalProps = {
  formRef?: React.RefObject<ProFormInstance<FeeFormValues> | undefined>;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  editingFee?: API.OrderFee;
  modalDirection: number; // 1 = RECEIVABLE, 2 = PAYABLE
  isFeeBilled: boolean;
  feeSettings: API.OrderFeeSettingOption[];
  settlementParties: { id?: string; name?: string; code?: string }[];
  currencies: API.Currency[];
  billingUnits: API.BillingUnit[];
  totalPreview?: string;
  exchangeRateStatus: 'idle' | 'loading' | 'resolved' | 'missing' | 'error';
  exchangeRatePreview?: string;
  manualExchangeRate: boolean;
  setManualExchangeRate: (val: boolean) => void;
  onOpenQuickAddFee: () => void;
  onOpenQuickAddPartner: () => void;
  onValuesChange: () => void;
  onFeeSettingSelect: (setting?: API.OrderFeeSettingOption) => void;
  onSubmit: (values: FeeFormValues) => Promise<boolean>;
};

export default function FeeFormModal({
  formRef,
  open,
  onOpenChange,
  editingFee,
  modalDirection,
  isFeeBilled,
  feeSettings,
  settlementParties,
  currencies,
  billingUnits,
  totalPreview,
  exchangeRateStatus,
  exchangeRatePreview,
  manualExchangeRate,
  setManualExchangeRate,
  onOpenQuickAddFee,
  onOpenQuickAddPartner,
  onValuesChange,
  onFeeSettingSelect,
  onSubmit,
}: FeeFormModalProps) {
  const internalFormRef = useRef<ProFormInstance<FeeFormValues> | undefined>(
    undefined,
  );
  const activeFormRef = formRef ?? internalFormRef;

  return (
    <ModalForm<FeeFormValues>
      title={
        editingFee
          ? `编辑${modalDirection === 1 ? '应收' : '应付'}费用`
          : `新增${modalDirection === 1 ? '应收' : '应付'}费用`
      }
      open={open}
      formRef={activeFormRef}
      onOpenChange={onOpenChange}
      onFinish={onSubmit}
      onValuesChange={onValuesChange}
      width={680}
      modalProps={{ destroyOnClose: true }}
    >
      <Row gutter={16}>
        <Col span={12}>
          <ProFormSearchableSelect
            name="feeSettingId"
            label="费用项目"
            rules={[{ required: true, message: '请选择费用项目' }]}
            disabled={isFeeBilled}
            options={feeSettings.map((item) => ({
              label: `${item.nameZh || item.nameEn || item.feeCode} (${item.feeCode})`,
              value: item.id ?? '',
              code: item.feeCode,
              name: item.nameZh,
            }))}
            fieldProps={{
              dropdownRender: (menu) => (
                <>
                  {menu}
                  <div
                    style={{
                      padding: '6px 12px',
                      cursor: 'pointer',
                      color: '#1677ff',
                      fontSize: 12,
                      display: 'flex',
                      alignItems: 'center',
                      gap: 4,
                      background: '#f6faff',
                      borderTop: '1px solid #f0f0f0',
                    }}
                    onMouseDown={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      onOpenQuickAddFee();
                    }}
                  >
                    <PlusOutlined /> 快捷新增费用科目
                  </div>
                </>
              ),
              onChange: (val) => {
                const setting = feeSettings.find((item) => item.id === val);
                onFeeSettingSelect(setting);
                if (setting?.defaultBillingUnitId) {
                  activeFormRef.current?.setFieldValue(
                    'billingUnitId',
                    setting.defaultBillingUnitId,
                  );
                }
                if (setting?.defaultCurrency) {
                  activeFormRef.current?.setFieldValue(
                    'currency',
                    setting.defaultCurrency,
                  );
                }
                onValuesChange();
              },
            }}
          />
        </Col>
        <Col span={12}>
          <ProFormSearchableSelect
            name="settlementPartyId"
            label="结算单位"
            rules={[{ required: true, message: '请选择结算单位' }]}
            disabled={isFeeBilled}
            options={settlementParties.map((item) => ({
              label: item.name ?? '',
              value: item.id ?? '',
              code: item.code,
              name: item.name,
            }))}
            fieldProps={{
              dropdownRender: (menu) => (
                <>
                  {menu}
                  <div
                    style={{
                      padding: '6px 12px',
                      cursor: 'pointer',
                      color: '#1677ff',
                      fontSize: 12,
                      display: 'flex',
                      alignItems: 'center',
                      gap: 4,
                      background: '#f6faff',
                      borderTop: '1px solid #f0f0f0',
                    }}
                    onMouseDown={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      onOpenQuickAddPartner();
                    }}
                  >
                    <PlusOutlined /> 快捷新建往来单位
                  </div>
                </>
              ),
            }}
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
            rules={[
              { required: true, message: '请输入单价' },
              {
                validator: positiveDecimalRule(
                  quantityOrPricePattern,
                  '单价格式不正确',
                ),
              },
            ]}
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
                validator: positiveDecimalRule(
                  quantityOrPricePattern,
                  '数量格式不正确',
                ),
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
            disabled={isFeeBilled}
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
            disabled={isFeeBilled}
            fieldProps={{ style: { width: '100%' } }}
          />
        </Col>

        {/* 汇率与金额计算预览 */}
        <Col span={24}>
          <ExchangeRatePreviewCard
            amountPreview={totalPreview}
            currency={activeFormRef.current?.getFieldValue('currency')}
            amountColor={modalDirection === 1 ? '#1677ff' : '#fa8c16'}
            status={exchangeRateStatus}
            ratePreview={exchangeRatePreview}
            onEnableManual={() => setManualExchangeRate(true)}
          />
        </Col>

        {manualExchangeRate && (
          <Col span={24}>
            <ProFormText
              name="exchangeRateOverride"
              label="手动指定汇率 (对 CNY)"
              rules={[
                { required: true, message: '请输入手动汇率' },
                {
                  validator: positiveDecimalRule(
                    exchangeRatePattern,
                    '汇率格式不正确',
                  ),
                },
              ]}
              placeholder="例如 7.2345"
            />
          </Col>
        )}

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
