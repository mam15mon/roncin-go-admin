import { useMutation, useQuery } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Card,
  Form,
  type FormInstance,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
} from 'antd';
import Decimal from 'decimal.js';
import React, { useEffect, useMemo, useRef } from 'react';
import { useUnsavedChangesGuard } from '@/components/layout/useUnsavedChangesGuard';
import { FinanceBillStatus, PartnerRoleType } from '@/enums.generated';
import { BillTermsCreditWarnings } from '@/features/finance/bill-creation';
import {
  billFeeLockFactLabels,
  isBillFieldProtected,
} from '@/features/finance/bill-status';
import { currencyOptionsQueryOptions } from '@/features/master-data/currencies';
import { partnerServiceGetPartner } from '@/services/roncin/partnerService';
import {
  settlementServiceListBillSettlementAccountUpdateCandidates,
  settlementServiceListBills,
} from '@/services/roncin/settlementService';
import { ensureListResponse, unwrapList } from '@/utils/api';
import type { BillFormValues } from './billConstants';
import { useBillChangeNotice } from './use-bill-change-notice';

interface BillEditModalProps {
  open: boolean;
  editing?: API.FinanceBill;
  form: FormInstance<BillFormValues>;
  submitting: boolean;
  onCancel: () => void;
  onOk: () => Promise<unknown>;
  onRefresh: () => Promise<boolean>;
}

export default function BillEditModal({
  open,
  editing,
  form,
  submitting,
  onCancel,
  onOk,
  onRefresh,
}: BillEditModalProps) {
  const { modal } = App.useApp();
  const { markDirty, resetDirty } = useUnsavedChangesGuard({
    enabled: open,
    identity: editing?.id,
  });
  const dirtyRef = useRef(false);
  const initializedAccount = useRef<string | undefined>(undefined);
  const requestIdentity = open ? editing?.id : undefined;
  const fieldsProtected = isBillFieldProtected(editing);
  const protectedFactText = billFeeLockFactLabels(editing)
    .filter((label) => label !== '草稿对冲')
    .join('、');
  const paymentTermsDays = Form.useWatch('paymentTermsDays', form);
  const { changed } = useBillChangeNotice({
    billId: editing?.id,
    version: editing?.version,
    enabled: open,
    ownWritePending: submitting,
  });
  const refresh = useMutation({
    meta: { silent: true },
    mutationFn: async () => {
      if (dirtyRef.current) {
        const confirmed = await new Promise<boolean>((resolve) => {
          modal.confirm({
            title: '有未保存的修改，是否放弃并刷新账单？',
            okText: '放弃并刷新',
            cancelText: '继续编辑',
            onOk: () => resolve(true),
            onCancel: () => resolve(false),
          });
        });
        if (!confirmed) return false;
      }
      const refreshed = await onRefresh();
      if (refreshed) {
        dirtyRef.current = false;
        resetDirty();
      }
      return refreshed;
    },
  });
  const currencyQuery = useQuery({
    ...currencyOptionsQueryOptions(),
    enabled: open,
  });
  const currencyOptions = currencyQuery.data ?? [];
  const counterpartyQuery = useQuery({
    queryKey: [
      'finance',
      'bill-edit-counterparty',
      {
        billId: requestIdentity,
        settlementPartyId: editing?.settlementPartyId,
        baseCurrency: editing?.baseCurrency,
      },
    ],
    enabled: Boolean(
      requestIdentity && editing?.settlementPartyId && editing?.baseCurrency,
    ),
    meta: { errorMessage: '账单往来单位及未核销余额加载失败' },
    queryFn: async ({ signal }) => {
      const settlementPartyId = editing?.settlementPartyId;
      const baseCurrency = editing?.baseCurrency;
      if (!settlementPartyId || !baseCurrency)
        throw new Error('缺少账单结算单位或本位币');
      const [partnerResponse, billsResponse] = await Promise.all([
        partnerServiceGetPartner(
          { id: settlementPartyId },
          { signal, skipErrorHandler: true },
        ),
        settlementServiceListBills(
          {
            page: 1,
            pageSize: 1,
            settlementPartyId,
            direction: 'RECEIVABLE',
            status: FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED,
            onlyUnsettled: true,
          },
          { signal, skipErrorHandler: true },
        ),
      ]);
      if (!partnerResponse?.data || partnerResponse.success === false)
        throw new Error('账单往来单位加载失败');
      ensureListResponse(billsResponse, '账单未核销余额加载失败');
      const partner = partnerResponse.data;
      const customerRule = (partner.roles ?? []).find(
        (role) => role.type === PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
      )?.settlementRule;
      return {
        isCasual: Boolean(partner.isCasual),
        creditLimitAmount: customerRule?.creditLimitMinor
          ? new Decimal(customerRule.creditLimitMinor).div(100).toString()
          : undefined,
        creditCurrency: customerRule?.creditCurrency,
        unsettledBaseAmount: (
          billsResponse.summary?.amountsByBaseCurrency ?? []
        ).find((item) => item.baseCurrency === baseCurrency)
          ?.unverifiedBaseAmount,
      };
    },
  });
  const counterparty = counterpartyQuery.data;
  const isCreditExceeded = useMemo(() => {
    if (!counterparty?.creditLimitAmount || !counterparty.unsettledBaseAmount)
      return false;
    const limit = new Decimal(counterparty.creditLimitAmount);
    return (
      limit.isPositive() &&
      new Decimal(counterparty.unsettledBaseAmount).greaterThan(limit)
    );
  }, [counterparty?.creditLimitAmount, counterparty?.unsettledBaseAmount]);
  const accountsQuery = useQuery({
    queryKey: ['finance', 'bill-edit-accounts', { billId: requestIdentity }],
    enabled: Boolean(requestIdentity),
    meta: { errorMessage: '结算账户加载失败' },
    queryFn: async ({ signal }) => {
      if (!requestIdentity) throw new Error('缺少账单标识');
      const response =
        await settlementServiceListBillSettlementAccountUpdateCandidates(
          {
            billId: requestIdentity,
          },
          { signal, skipErrorHandler: true },
        );
      return unwrapList(ensureListResponse(response, '结算账户加载失败'));
    },
  });
  const accountOptions = accountsQuery.data ?? [];
  const accountsLoading = accountsQuery.isFetching;
  useEffect(() => {
    initializedAccount.current = undefined;
    dirtyRef.current = false;
  }, [requestIdentity, editing?.version]);
  useEffect(() => {
    if (
      !requestIdentity ||
      !accountsQuery.isSuccess ||
      initializedAccount.current === requestIdentity ||
      fieldsProtected
    )
      return;
    initializedAccount.current = requestIdentity;
    // 仅首次候选就绪时补默认账户；后台重查不覆盖人工选择。
    const selected = form.getFieldValue('settlementAccountId');
    if (selected && accountOptions.some((account) => account.id === selected))
      return;
    if (!dirtyRef.current)
      form.setFieldValue(
        'settlementAccountId',
        accountOptions.find((account) => account.isDefault)?.id,
      );
  }, [
    accountOptions,
    accountsQuery.isSuccess,
    fieldsProtected,
    form,
    requestIdentity,
  ]);

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
      okButtonProps={{ disabled: changed || refresh.isPending }}
      closable={!submitting && !refresh.isPending}
      mask={{ closable: !submitting && !refresh.isPending }}
      keyboard={!submitting && !refresh.isPending}
      cancelButtonProps={{ disabled: submitting || refresh.isPending }}
    >
      <Form
        form={form}
        disabled={refresh.isPending}
        layout="vertical"
        onValuesChange={() => {
          dirtyRef.current = true;
          markDirty();
        }}
      >
        {changed && (
          <Alert
            type="warning"
            showIcon
            style={{ marginBottom: 12 }}
            title="这个账单已经发生变动，请重新刷新。"
            action={
              <Button
                size="small"
                loading={refresh.isPending}
                onClick={() => void refresh.mutateAsync()}
              >
                刷新账单
              </Button>
            }
          />
        )}
        {fieldsProtected ? (
          <Alert
            type="warning"
            showIcon
            style={{ marginBottom: 12 }}
            title={`账单已有${protectedFactText}，只能修改对账抬头和备注；如需调整账期、结算账户或预计开票，请先撤销相关财务记录。`}
          />
        ) : null}
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
            label="创建日期"
            tooltip="由系统维护。未开票时增减费用会更新创建日期，开票后固定。"
          >
            <span>{editing?.billDate || '-'}</span>
          </Form.Item>
          <Form.Item name="paymentTermsDays" label="账期（天，可选）">
            <InputNumber
              min={0}
              max={3650}
              precision={0}
              disabled={fieldsProtected}
            />
          </Form.Item>
          <Form.Item
            name="settlementAccountId"
            label="结算账户"
            rules={[{ required: true, message: '请选择结算账户' }]}
            style={{ minWidth: 620 }}
          >
            <Select
              allowClear
              disabled={fieldsProtected}
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
            title="固定账单币种与汇率"
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
                  disabled={fieldsProtected}
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
                <Input
                  disabled={fieldsProtected}
                  placeholder="预计汇率"
                  style={{ minWidth: 220 }}
                />
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
