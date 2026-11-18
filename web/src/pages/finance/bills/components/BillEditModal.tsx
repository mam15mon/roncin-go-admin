import { QuestionCircleOutlined } from '@ant-design/icons';
import { useMutation, useQuery } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Col,
  Form,
  type FormInstance,
  Input,
  InputNumber,
  Modal,
  Row,
  Select,
  Tooltip,
  Typography,
} from 'antd';
import Decimal from 'decimal.js';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { useAccess } from '@/app/access';
import { useUnsavedChangesGuard } from '@/components/layout/useUnsavedChangesGuard';
import { QuickCreateSelectFooter } from '@/components/ui';
import { FinanceBillStatus, PartnerRoleType } from '@/enums.generated';
import { BillTermsCreditWarnings } from '@/features/finance/bill-creation';
import {
  billFeeLockFactLabels,
  isBillFieldProtected,
} from '@/features/finance/bill-status';
import { currencyOptionsQueryOptions } from '@/features/master-data/currencies';
import {
  SettlementAccountQuickCreateModal,
  settlementAccountLabel,
} from '@/features/partners/settlement-account';
import { partnerServiceGetPartner } from '@/services/roncin/partnerService';
import {
  settlementServiceListBillSettlementAccountUpdateCandidates,
  settlementServiceListBills,
} from '@/services/roncin/settlementService';
import { ensureListResponse, unwrapList } from '@/utils/api';
import { trimDecimal } from '@/utils/format';
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
  const access = useAccess();
  const [quickAccountFor, setQuickAccountFor] = useState<string>();
  const [accountSelectOpen, setAccountSelectOpen] = useState(false);
  const [createdAccount, setCreatedAccount] = useState<{
    identity: string;
    account: API.FinanceSettlementAccountOption;
  }>();
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
  const accountIdentity = [
    requestIdentity,
    editing?.version,
    editing?.organizationId,
    editing?.settlementPartyId,
    editing?.direction,
    editing?.currency,
  ].join(':');
  const accountOptions =
    createdAccount?.identity === accountIdentity
      ? [
          createdAccount.account,
          ...(accountsQuery.data ?? []).filter(
            (account) => account.id !== createdAccount.account.id,
          ),
        ]
      : (accountsQuery.data ?? []);
  const canQuickCreateAccount = Boolean(
    open &&
      !fieldsProtected &&
      !changed &&
      !submitting &&
      !refresh.isPending &&
      access.canUpdateFinanceBills &&
      access.canCreatePartnerAccounts &&
      access.canOperateOrganization(editing?.organizationId) &&
      editing?.settlementPartyId &&
      editing?.currency &&
      (editing.direction === 'RECEIVABLE' || editing.direction === 'PAYABLE'),
  );
  useEffect(() => {
    if (!canQuickCreateAccount) {
      setQuickAccountFor(undefined);
      setAccountSelectOpen(false);
    }
  }, [canQuickCreateAccount]);
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
      width={720}
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
        <Row gutter={16}>
          <Col xs={24} md={12}>
            <Form.Item
              name="statementTitle"
              label="对账抬头"
              rules={[
                { required: true, whitespace: true, message: '请输入对账抬头' },
                { max: 200, message: '对账抬头不能超过 200 字' },
              ]}
            >
              <Input maxLength={200} />
            </Form.Item>
          </Col>
          <Col xs={12} md={6}>
            <Form.Item
              label="创建日期"
              tooltip="由系统维护。未开票时增减费用会更新创建日期，开票后固定。"
            >
              <span style={{ lineHeight: '32px' }}>
                {editing?.billDate || '-'}
              </span>
            </Form.Item>
          </Col>
          <Col xs={12} md={6}>
            <Form.Item name="paymentTermsDays" label="账期（天，可选）">
              <InputNumber
                min={0}
                max={3650}
                precision={0}
                disabled={fieldsProtected}
                style={{ width: '100%' }}
              />
            </Form.Item>
          </Col>
          <Col xs={24}>
            <Form.Item
              name="settlementAccountId"
              label="结算账户"
              rules={[{ required: true, message: '请选择结算账户' }]}
            >
              <Select
                allowClear
                disabled={fieldsProtected}
                open={accountSelectOpen}
                onOpenChange={setAccountSelectOpen}
                loading={accountsLoading}
                placeholder="请选择可用于该账单的启用账户"
                popupRender={
                  canQuickCreateAccount
                    ? (menu) => (
                        <QuickCreateSelectFooter
                          label="新增结算账户"
                          onCreate={() => {
                            setAccountSelectOpen(false);
                            setQuickAccountFor(accountIdentity);
                          }}
                        >
                          {menu}
                        </QuickCreateSelectFooter>
                      )
                    : undefined
                }
                options={accountOptions.map((account) => ({
                  value: account.id,
                  label: settlementAccountLabel(account),
                }))}
              />
            </Form.Item>
          </Col>
          <SettlementAccountQuickCreateModal
            open={canQuickCreateAccount && quickAccountFor === accountIdentity}
            organizationId={editing?.organizationId || ''}
            partnerId={editing?.settlementPartyId || ''}
            partnerName={editing?.settlementPartyName}
            currency={editing?.currency || ''}
            direction={editing?.direction || ''}
            sessionIdentity={accountIdentity}
            disabled={!canQuickCreateAccount}
            onCancel={() => setQuickAccountFor(undefined)}
            onCreated={(account) => {
              setCreatedAccount({ identity: accountIdentity, account });
              form.setFieldValue('settlementAccountId', account.id);
              dirtyRef.current = true;
              markDirty();
              setQuickAccountFor(undefined);
              void accountsQuery.refetch();
            }}
          />
          <Col xs={24}>
            <Form.Item
              label="账单币种与汇率"
              tooltip="账单币种和汇率已固定，不能修改。"
            >
              <Typography.Text>
                账单币种 {editing?.currency || '-'} · 组织本位币{' '}
                {editing?.baseCurrency || '-'} · 生效汇率{' '}
                {editing?.exchangeRate
                  ? trimDecimal(editing.exchangeRate)
                  : '-'}
                {editing?.exchangeRateDate
                  ? `（${editing.exchangeRateDate}）`
                  : ''}
              </Typography.Text>
            </Form.Item>
          </Col>
          <Col xs={24}>
            <Typography.Text
              strong
              style={{ display: 'block', marginBottom: 8 }}
            >
              预计开票{' '}
              <Tooltip title="只用于预计，不改变账单币种。">
                <QuestionCircleOutlined style={{ color: '#8c8c8c' }} />
              </Tooltip>
            </Typography.Text>
          </Col>
          <Col xs={24} md={12}>
            <Form.Item name="estimatedInvoiceCurrency" label="预计开票币种">
              <Select
                allowClear
                disabled={fieldsProtected}
                placeholder="默认使用账单币种"
                options={currencyOptions}
              />
            </Form.Item>
          </Col>
          <Col xs={24} md={12}>
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
                    if (estimatedCurrency !== billCurrency && !normalizedRate) {
                      throw new Error(
                        '预计开票币种与账单币种不同时必须填写预计开票汇率',
                      );
                    }
                    if (normalizedRate) {
                      const num = Number(normalizedRate);
                      if (Number.isNaN(num) || num <= 0) {
                        throw new Error('预计开票汇率必须为大于 0 的有效数字');
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
              <Input disabled={fieldsProtected} placeholder="预计汇率" />
            </Form.Item>
          </Col>
          <Col xs={24}>
            <Form.Item name="note" label="备注">
              <Input.TextArea rows={2} maxLength={500} showCount />
            </Form.Item>
          </Col>
        </Row>
      </Form>
    </Modal>
  );
}
