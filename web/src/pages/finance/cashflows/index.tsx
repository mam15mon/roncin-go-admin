import {
  CheckOutlined,
  CloseCircleOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDatePicker,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import {
  FinanceLedgerTemplate,
  ProFormSearchableSelect,
  type FinanceLedgerMetricCard,
} from '@/components/ui';
import { useAccess } from '@umijs/max';
import { App, Popconfirm, Space, Tag } from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, { useRef, useState } from 'react';
import { FinanceCashflowStatus } from '@/enums.generated';
import {
  settlementServiceCancelCashflow,
  settlementServiceConfirmCashflow,
  settlementServiceCreateCashflow,
  settlementServiceListCashflows,
} from '@/services/roncin/settlementService';
import { toTableRequest } from '@/utils/api';
import { getCurrencyOptions, searchPartnerOptions } from '@/utils/options';
import { makeVersionActions } from '@/utils/versionActions';

type Values = {
  direction: string;
  settlementPartyId: string;
  currency: string;
  amount: string;
  exchangeRate: string;
  baseCurrency: string;
  transactionDate: Dayjs;
  ourAccount: string;
  counterpartyAccount?: string;
  paymentMethod: string;
  bankReferenceNo?: string;
  note?: string;
};
const states: Record<number, { text: string; color: string }> = {
  [FinanceCashflowStatus.FINANCE_CASHFLOW_STATUS_DRAFT]: { text: '草稿', color: 'gold' },
  [FinanceCashflowStatus.FINANCE_CASHFLOW_STATUS_CONFIRMED]: { text: '已确认', color: 'green' },
  [FinanceCashflowStatus.FINANCE_CASHFLOW_STATUS_CANCELLED]: { text: '已取消', color: 'default' },
};
const decimalRule = {
  pattern: /^(0|[1-9][0-9]{0,19})(\.[0-9]{1,8})?$/,
  message: '请输入大于 0 且最多 8 位小数的金额',
};

export default function FinanceCashflowsPage() {
  const access = useAccess();
  const { message, modal } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [open, setOpen] = useState(false);
  const [metricStats, setMetricStats] = useState({
    totalCount: 0,
    incomeBase: 0,
    payoutBase: 0,
    unverifiedBase: 0,
    baseCurrency: '',
  });

  const reload = () => actionRef.current?.reload();
  const cashflowActions = makeVersionActions<API.FinanceCashflow>({
    modal,
    message,
  });
  const confirm = (r: API.FinanceCashflow) =>
    cashflowActions.run(r, async ({ id, expectedVersion }) => {
      try {
        await settlementServiceConfirmCashflow({ id }, { id, expectedVersion });
        message.success('资金流水已确认，可进入核销');
        reload();
      } catch (e: any) {
        message.error(e.message || '确认失败');
      }
    });
  const cancel = (r: API.FinanceCashflow) => {
    cashflowActions.confirm(
      r,
      '取消资金流水？',
      async ({ id, expectedVersion }, reason) => {
        await settlementServiceCancelCashflow(
          { id },
          { id, expectedVersion, reason },
        );
        message.success('资金流水已取消');
        reload();
      },
      {
        danger: true,
        placeholder: '请输入取消原因（必填）',
        requiredMessage: '请输入取消原因',
      },
    );
  };

  const metricCards: FinanceLedgerMetricCard[] = [
    {
      key: 'total-cashflows',
      title: '资金流水总笔数',
      value: metricStats.totalCount,
      suffix: '笔',
    },
    {
      key: 'income-cashflows',
      title: '收款流水折本币',
      value: metricStats.incomeBase,
      precision: 2,
      suffix: metricStats.baseCurrency || '-',
      valueColor: '#1677ff',
    },
    {
      key: 'payout-cashflows',
      title: '付款流水折本币',
      value: metricStats.payoutBase,
      precision: 2,
      suffix: metricStats.baseCurrency || '-',
      valueColor: '#fa8c16',
    },
    {
      key: 'unverified-cashflows',
      title: '未核销可用资金',
      value: metricStats.unverifiedBase,
      precision: 2,
      suffix: metricStats.baseCurrency || '-',
      valueColor: '#52c41a',
    },
  ];

  const columns: ProColumns<API.FinanceCashflow>[] = [
    {
      title: '序号',
      dataIndex: 'index',
      valueType: 'index',
      width: 55,
      fixed: 'left',
    },
    {
      title: '方向',
      dataIndex: 'direction',
      width: 75,
      fixed: 'left',
      valueType: 'select',
      valueEnum: { RECEIVABLE: { text: '收款' }, PAYABLE: { text: '付款' } },
      render: (_, r) => (
        <Tag
          color={r.direction === 'RECEIVABLE' ? 'green' : 'volcano'}
          style={{ margin: 0 }}
        >
          {r.direction === 'RECEIVABLE' ? '收款' : '付款'}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 85,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        Object.entries(states).map(([k, v]) => [k, { text: v.text }]),
      ),
      render: (_, r) => {
        const v = states[
          r.status ?? FinanceCashflowStatus.FINANCE_CASHFLOW_STATUS_DRAFT
        ];
        return (
          <Tag color={v.color} style={{ margin: 0 }}>
            {v.text}
          </Tag>
        );
      },
    },
    {
      title: '流水编号',
      dataIndex: 'flowNo',
      width: 170,
      copyable: true,
      search: false,
    },
    {
      title: '结算单位',
      dataIndex: 'settlementPartyName',
      width: 220,
      ellipsis: true,
      search: false,
    },
    {
      title: '金额',
      dataIndex: 'amount',
      width: 140,
      align: 'right',
      search: false,
      render: (_, r) => (
        <strong style={{ color: '#262626' }}>
          {r.amount} {r.currency}
        </strong>
      ),
    },
    {
      title: '结算汇率',
      dataIndex: 'exchangeRate',
      width: 140,
      align: 'right',
      search: false,
      render: (_, r) => {
        if (!r.exchangeRate) return '-';
        const sourceLabel =
          r.exchangeRateSource === 'MANUAL'
            ? '手工'
            : r.exchangeRateSource === 'BASE_CURRENCY'
            ? '本币'
            : '系统';
        const sourceColor =
          r.exchangeRateSource === 'MANUAL' ? 'purple' : 'default';
        return (
          <Space size={4}>
            <span>{r.exchangeRate}</span>
            <Tag color={sourceColor} style={{ margin: 0, fontSize: 10 }}>
              {sourceLabel}
            </Tag>
          </Space>
        );
      },
    },
    {
      title: '折本币',
      dataIndex: 'baseAmount',
      width: 140,
      align: 'right',
      search: false,
      render: (_, r) => (
        <strong style={{ color: r.direction === 'RECEIVABLE' ? '#1677ff' : '#fa8c16' }}>
          {r.baseAmount} {r.baseCurrency}
        </strong>
      ),
    },
    {
      title: '已核销',
      dataIndex: 'verifiedAmount',
      width: 135,
      align: 'right',
      search: false,
      render: (_, r) => `${r.verifiedAmount || '0.00000000'} ${r.currency}`,
    },
    {
      title: '未核销',
      dataIndex: 'unverifiedAmount',
      width: 135,
      align: 'right',
      search: false,
      render: (_, r) => (
        <strong
          style={{
            color: Number(r.unverifiedAmount || 0) > 0 ? '#cf1322' : '#389e0d',
          }}
        >
          {r.unverifiedAmount || '0.00000000'} {r.currency}
        </strong>
      ),
    },
    {
      title: '交易日期',
      dataIndex: 'transactionDate',
      width: 110,
      search: false,
    },
    {
      title: '我方账户',
      dataIndex: 'ourAccount',
      width: 180,
      ellipsis: true,
      search: false,
    },
    {
      title: '支付方式',
      dataIndex: 'paymentMethod',
      width: 95,
      search: false,
    },
    {
      title: '银行水单号',
      dataIndex: 'bankReferenceNo',
      width: 150,
      search: false,
      renderText: (v) => v || '-',
    },
    {
      title: '操作',
      valueType: 'option',
      fixed: 'right',
      width: 150,
      render: (_, r) => [
        access.canUpdateFinanceCashflows &&
        r.status === FinanceCashflowStatus.FINANCE_CASHFLOW_STATUS_DRAFT ? (
          <Popconfirm
            key="confirm"
            title="确认该笔真实资金流水？"
            onConfirm={() => void confirm(r)}
          >
            <a>
              <CheckOutlined /> 确认
            </a>
          </Popconfirm>
        ) : null,
        access.canUpdateFinanceCashflows &&
        r.status !== FinanceCashflowStatus.FINANCE_CASHFLOW_STATUS_CANCELLED ? (
          <a
            key="cancel"
            style={{ color: '#ff4d4f' }}
            onClick={() => cancel(r)}
          >
            <CloseCircleOutlined /> 取消
          </a>
        ) : null,
      ],
    },
  ];

  return (
    <>
      <FinanceLedgerTemplate<API.FinanceCashflow>
        headerTitle="资金流水管理"
        actionRef={actionRef}
        columns={columns}
        metricCards={metricCards}
        scrollX={1900}
        primaryActionText="登记流水"
        primaryActionIcon={<PlusOutlined />}
        onPrimaryAction={
          access.canCreateFinanceCashflows ? () => setOpen(true) : undefined
        }
        request={async (p) => {
          const r = await settlementServiceListCashflows({
            page: p.current,
            pageSize: p.pageSize,
            keyword: p.keyword,
            direction: p.direction,
            status: p.status ? Number(p.status) : undefined,
          });
          setMetricStats({
            totalCount: Number(r.total ?? 0),
            incomeBase: Number(r.summary?.receivableBaseAmount ?? 0),
            payoutBase: Number(r.summary?.payableBaseAmount ?? 0),
            unverifiedBase: Number(r.summary?.unverifiedBaseAmount ?? 0),
            baseCurrency: r.summary?.baseCurrency ?? '',
          });
          return { ...toTableRequest(r), total: Number(r.total ?? 0) };
        }}
      />
      <ModalForm<Values>
        title="登记资金流水"
        open={open}
        width={760}
        modalProps={{ destroyOnHidden: true, onCancel: () => setOpen(false) }}
        initialValues={{
          direction: 'RECEIVABLE',
          currency: 'CNY',
          transactionDate: dayjs(),
          paymentMethod: 'BANK_TRANSFER',
        }}
        onFinish={async (v) => {
          try {
            await settlementServiceCreateCashflow({
              direction: v.direction,
              settlementPartyId: v.settlementPartyId,
              currency: v.currency,
              amount: v.amount,
              exchangeRate: v.exchangeRate ? v.exchangeRate.trim() : undefined,
              transactionDate: dayjs(v.transactionDate).format('YYYY-MM-DD'),
              ourAccount: v.ourAccount,
              counterpartyAccount: v.counterpartyAccount,
              paymentMethod: v.paymentMethod,
              bankReferenceNo: v.bankReferenceNo,
              note: v.note,
              idempotencyKey: globalThis.crypto.randomUUID(),
            });
            message.success('资金流水已登记');
            setOpen(false);
            reload();
            return true;
          } catch (e: any) {
            message.error(e.message || '登记失败');
            return false;
          }
        }}
      >
        <ProFormSearchableSelect
          name="direction"
          label="流水方向"
          rules={[{ required: true }]}
          options={[
            { label: '收款（流入）', value: 'RECEIVABLE' },
            { label: '付款（流出）', value: 'PAYABLE' },
          ]}
        />
        <ProFormSearchableSelect
          name="settlementPartyId"
          label="往来结算单位"
          rules={[{ required: true }]}
          request={({ keyWords }) =>
            searchPartnerOptions(keyWords, { enabled: true })
          }
        />
        <ProFormSearchableSelect
          name="currency"
          label="原币币种"
          rules={[{ required: true }]}
          request={getCurrencyOptions}
        />
        <ProFormText
          name="amount"
          label="发生金额"
          rules={[{ required: true }, decimalRule]}
        />
        <ProFormText
          name="exchangeRate"
          label="结算汇率（可选）"
          placeholder="留空默认根据交易日期自动获取系统结算汇率"
          extra="外币流水若不填则自动按交易日期匹配 SETTLEMENT 汇率；手动输入需具备财务汇率覆盖权限"
          rules={[
            {
              validator: (_, val) => {
                if (!val) return Promise.resolve();
                if (!decimalRule.pattern.test(val)) {
                  return Promise.reject(new Error('请输入大于 0 且最多 8 位小数的汇率'));
                }
                return Promise.resolve();
              },
            },
          ]}
        />
        <ProFormDatePicker
          name="transactionDate"
          label="交易日期"
          rules={[{ required: true }]}
        />
        <ProFormText
          name="ourAccount"
          label="我方账户"
          rules={[{ required: true }]}
        />
        <ProFormText name="counterpartyAccount" label="对方账户" />
        <ProFormSearchableSelect
          name="paymentMethod"
          label="支付方式"
          rules={[{ required: true }]}
          options={[
            { label: '银行转账', value: 'BANK_TRANSFER' },
            { label: '支票', value: 'CHECK' },
            { label: '现金', value: 'CASH' },
            { label: '第三方支付', value: 'THIRD_PARTY' },
          ]}
        />
        <ProFormText name="bankReferenceNo" label="银行水单号" />
        <ProFormTextArea
          name="note"
          label="备注"
          fieldProps={{ maxLength: 500 }}
        />
      </ModalForm>
    </>
  );
}
