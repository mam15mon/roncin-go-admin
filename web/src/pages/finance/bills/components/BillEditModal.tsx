import {
  Card,
  DatePicker,
  Form,
  type FormInstance,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
} from 'antd';
import Decimal from 'decimal.js';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { FinanceBillStatus, PartnerRoleType } from '@/enums.generated';
import { partnerServiceGetPartner } from '@/services/roncin/partnerService';
import {
  settlementServiceListBillSettlementAccountUpdateCandidates,
  settlementServiceListBills,
} from '@/services/roncin/settlementService';
import { unwrapList } from '@/utils/api';
import { getCurrencyOptions, type SelectOption } from '@/utils/options';
import BillTermsCreditWarnings from './BillTermsCreditWarnings';
import type { BillFormValues } from './billConstants';

interface BillEditModalProps {
  open: boolean;
  editing?: API.FinanceBill;
  form: FormInstance<BillFormValues>;
  submitting: boolean;
  onCancel: () => void;
  onOk: () => Promise<void>;
}

export default function BillEditModal({
  open,
  editing,
  form,
  submitting,
  onCancel,
  onOk,
}: BillEditModalProps) {
  const [accountOptions, setAccountOptions] = useState<
    API.FinanceSettlementAccountOption[]
  >([]);
  const [accountsLoading, setAccountsLoading] = useState(false);
  const [currencyOptions, setCurrencyOptions] = useState<SelectOption[]>([]);
  const requestSequenceRef = useRef(0);
  // 草稿编辑路径与创建工作台共用同一套预警组装：按对方档案与未核销余额重组散客/超额预警。
  const [counterparty, setCounterparty] = useState<{
    isCasual?: boolean;
    creditLimitAmount?: string;
    creditCurrency?: string;
    unsettledBaseAmount?: string;
  }>();
  const requestIdentity = open ? editing?.id : undefined;
  const paymentTermsDays = Form.useWatch('paymentTermsDays', form);

  useEffect(() => {
    const settlementPartyId = editing?.settlementPartyId;
    const baseCurrency = editing?.baseCurrency;
    if (!open || !requestIdentity || !settlementPartyId || !baseCurrency) {
      setCounterparty(undefined);
      return undefined;
    }
    let cancelled = false;
    void Promise.all([
      partnerServiceGetPartner({ id: settlementPartyId }),
      settlementServiceListBills({
        page: 1,
        pageSize: 1,
        settlementPartyId,
        direction: 'RECEIVABLE',
        status: FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED,
        onlyUnsettled: true,
      }),
    ])
      .then(([partnerResponse, billsResponse]) => {
        if (cancelled) return;
        const partner = partnerResponse.data;
        const customerRule = (partner?.roles ?? []).find(
          (role) => role.type === PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
        )?.settlementRule;
        const unsettledBase = (
          billsResponse.summary?.amountsByBaseCurrency ?? []
        ).find(
          (item) => item.baseCurrency === baseCurrency,
        )?.unverifiedBaseAmount;
        setCounterparty({
          isCasual: Boolean(partner?.isCasual),
          creditLimitAmount: customerRule?.creditLimitMinor
            ? new Decimal(customerRule.creditLimitMinor).div(100).toString()
            : undefined,
          creditCurrency: customerRule?.creditCurrency,
          unsettledBaseAmount: unsettledBase,
        });
      })
      .catch(() => {
        if (!cancelled) setCounterparty(undefined);
      });
    return () => {
      cancelled = true;
    };
  }, [
    editing?.baseCurrency,
    editing?.settlementPartyId,
    open,
    requestIdentity,
  ]);

  const isCreditExceeded = useMemo(() => {
    if (!counterparty?.creditLimitAmount || !counterparty.unsettledBaseAmount) {
      return false;
    }
    const limit = new Decimal(counterparty.creditLimitAmount);
    if (!limit.isPositive()) return false;
    return new Decimal(counterparty.unsettledBaseAmount).greaterThan(limit);
  }, [counterparty?.creditLimitAmount, counterparty?.unsettledBaseAmount]);

  useEffect(() => {
    let cancelled = false;
    void getCurrencyOptions()
      .then((options) => {
        if (!cancelled) setCurrencyOptions(options);
      })
      .catch(() => {
        if (!cancelled) setCurrencyOptions([]);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    const requestSequence = ++requestSequenceRef.current;
    setAccountOptions([]);
    if (!open || !editing?.id) {
      setAccountsLoading(false);
      return undefined;
    }
    setAccountsLoading(true);
    void settlementServiceListBillSettlementAccountUpdateCandidates({
      billId: editing.id,
    })
      .then((response) => {
        if (requestSequence !== requestSequenceRef.current) return;
        const accounts = unwrapList(response);
        setAccountOptions(accounts);
        const selected = form.getFieldValue('settlementAccountId');
        const selectedStillAvailable = accounts.some(
          (account) => account.id === selected,
        );
        if (selected && selectedStillAvailable) return;

        const defaultAccount = accounts.find((account) => account.isDefault);
        form.setFieldValue('settlementAccountId', defaultAccount?.id);
      })
      .catch(() => {
        if (requestSequence === requestSequenceRef.current)
          setAccountOptions([]);
      })
      .finally(() => {
        if (requestSequence === requestSequenceRef.current) {
          setAccountsLoading(false);
        }
      });
    return () => {
      requestSequenceRef.current += 1;
    };
  }, [editing?.id, form, open]);

  return (
    <Modal
      title={`编辑账单 ${editing?.billNo || ''}`}
      open={open}
      width={680}
      destroyOnHidden
      confirmLoading={submitting}
      okText="保存"
      onCancel={onCancel}
      onOk={() => void onOk()}
    >
      <Form form={form} layout="vertical">
        <BillTermsCreditWarnings
          direction={editing?.direction}
          isCasual={counterparty?.isCasual}
          paymentTermsDays={paymentTermsDays}
          creditLimitAmount={counterparty?.creditLimitAmount}
          creditCurrency={counterparty?.creditCurrency}
          currentUnsettledAmount={counterparty?.unsettledBaseAmount}
          isCreditExceeded={isCreditExceeded}
          billAmount={editing?.totalAmount}
          billCurrency={editing?.currency}
        />
        <Space size={16} align="start" wrap style={{ width: '100%' }}>
          <Form.Item
            name="statementTitle"
            label="对账抬头"
            rules={[
              { required: true, whitespace: true, message: '请输入对账抬头' },
              { max: 200, message: '对账抬头不能超过 200 字' },
            ]}
            style={{ minWidth: 260 }}
          >
            <Input maxLength={200} />
          </Form.Item>
          <Form.Item
            name="billDate"
            label="账单日期"
            extra="修改账单日期将自动按新账单日重置 BILL 汇率快照"
            rules={[{ required: true, message: '请选择账单日期' }]}
          >
            <DatePicker allowClear={false} />
          </Form.Item>
          <Form.Item name="paymentTermsDays" label="账期（天，可选）">
            <InputNumber min={0} max={3650} precision={0} />
          </Form.Item>
          <Form.Item
            name="settlementAccountId"
            label="结算账户"
            rules={[{ required: true, message: '请选择结算账户' }]}
            style={{ minWidth: 620 }}
          >
            <Select
              allowClear
              loading={accountsLoading}
              placeholder="请选择可用于该账单的启用账户"
              options={accountOptions.map((account) => ({
                value: account.id,
                label: `${account.name || account.accountHolder || '未命名账户'}｜${account.bankName || '-'}${account.accountNo ? ` · ${account.accountNo}` : ''}｜${account.currency || '-'}`,
              }))}
            />
          </Form.Item>
          <Card
            size="small"
            title="固定账单币种与账单日汇率"
            style={{ width: '100%' }}
          >
            <Space wrap>
              <span>账单币种：{editing?.currency || '-'}</span>
              <span>组织本位币：{editing?.baseCurrency || '-'}</span>
              <span>生效汇率：{editing?.exchangeRate || '服务端未提供'}</span>
              <span>
                汇率日期：{editing?.exchangeRateDate || '服务端未提供'}
              </span>
            </Space>
          </Card>
          <Card
            size="small"
            title="预计开票配置（不改变固定账单币种）"
            style={{ width: '100%' }}
          >
            <Space size={16} align="start" wrap>
              <Form.Item name="estimatedInvoiceCurrency" label="预计开票币种">
                <Select
                  allowClear
                  placeholder="默认使用账单币种"
                  options={currencyOptions}
                  style={{ minWidth: 220 }}
                />
              </Form.Item>
              <Form.Item
                name="estimatedInvoiceRate"
                label="预计开票汇率"
                rules={[
                  ({ getFieldValue }) => ({
                    validator: async (_, rate) => {
                      const estimatedCurrency = getFieldValue(
                        'estimatedInvoiceCurrency',
                      );
                      const billCurrency = editing?.currency;
                      const normalizedRate = rate?.trim();
                      if (!estimatedCurrency) {
                        if (normalizedRate) {
                          throw new Error(
                            '填写预计开票汇率时必须选择预计开票币种',
                          );
                        }
                        return;
                      }
                      if (
                        estimatedCurrency !== billCurrency &&
                        !normalizedRate
                      ) {
                        throw new Error(
                          '预计开票币种与账单币种不同时必须填写预计开票汇率',
                        );
                      }
                      if (normalizedRate) {
                        const num = Number(normalizedRate);
                        if (Number.isNaN(num) || num <= 0) {
                          throw new Error(
                            '预计开票汇率必须为大于 0 的有效数字',
                          );
                        }
                      }
                      if (
                        estimatedCurrency === billCurrency &&
                        normalizedRate &&
                        Number(normalizedRate) !== 1
                      ) {
                        throw new Error(
                          '预计开票币种与账单币种相同时，汇率必须为 1',
                        );
                      }
                    },
                  }),
                ]}
              >
                <Input placeholder="预计汇率" style={{ minWidth: 220 }} />
              </Form.Item>
            </Space>
          </Card>
          <Form.Item name="note" label="备注" style={{ minWidth: 620 }}>
            <Input maxLength={500} />
          </Form.Item>
        </Space>
      </Form>
    </Modal>
  );
}
