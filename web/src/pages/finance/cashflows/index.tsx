import {
  CheckOutlined,
  CloseCircleOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDatePicker,
  ProFormSelect,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Button, Input, Popconfirm, Tag } from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, { useRef, useState } from 'react';
import { partnerServiceListPartners } from '@/services/roncin/partnerService';
import {
  settlementServiceCancelCashflow,
  settlementServiceConfirmCashflow,
  settlementServiceCreateCashflow,
  settlementServiceListCashflows,
} from '@/services/roncin/settlementService';

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
const states: Record<string, { text: string; color: string }> = {
  DRAFT: { text: '草稿', color: 'gold' },
  CONFIRMED: { text: '已确认', color: 'green' },
  CANCELLED: { text: '已取消', color: 'default' },
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
  const reload = () => actionRef.current?.reload();
  const confirm = async (r: API.FinanceCashflow) => {
    if (!r.id || !r.version) return;
    try {
      await settlementServiceConfirmCashflow(
        { id: r.id },
        { id: r.id, expectedVersion: r.version },
      );
      message.success('资金流水已确认，可进入核销');
      reload();
    } catch (e: any) {
      message.error(e.message || '确认失败');
    }
  };
  const cancel = (r: API.FinanceCashflow) => {
    const id = r.id,
      v = r.version;
    if (!id || !v) return;
    let reason = '';
    modal.confirm({
      title: '取消资金流水？',
      content: (
        <Input.TextArea
          placeholder="请输入取消原因（必填）"
          onChange={(e) => {
            reason = e.target.value.trim();
          }}
        />
      ),
      okButtonProps: { danger: true },
      onOk: async () => {
        if (!reason) {
          message.warning('请输入取消原因');
          throw new Error('原因不能为空');
        }
        await settlementServiceCancelCashflow(
          { id },
          { id, expectedVersion: v, reason },
        );
        message.success('资金流水已取消');
        reload();
      },
    });
  };
  const columns: ProColumns<API.FinanceCashflow>[] = [
    {
      title: '关键词',
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: { placeholder: '流水号、单位或银行水单号' },
    },
    {
      title: '流水编号',
      dataIndex: 'flowNo',
      width: 170,
      copyable: true,
      search: false,
    },
    {
      title: '方向',
      dataIndex: 'direction',
      width: 80,
      valueType: 'select',
      valueEnum: { RECEIVABLE: { text: '收款' }, PAYABLE: { text: '付款' } },
      render: (_, r) => (
        <Tag color={r.direction === 'RECEIVABLE' ? 'green' : 'volcano'}>
          {r.direction === 'RECEIVABLE' ? '收款' : '付款'}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        Object.entries(states).map(([k, v]) => [k, { text: v.text }]),
      ),
      render: (_, r) => {
        const v = states[r.status || 'DRAFT'];
        return <Tag color={v.color}>{v.text}</Tag>;
      },
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
      width: 150,
      align: 'right',
      search: false,
      render: (_, r) => (
        <strong>
          {r.amount} {r.currency}
        </strong>
      ),
    },
    {
      title: '折本币',
      dataIndex: 'baseAmount',
      width: 150,
      align: 'right',
      search: false,
      render: (_, r) => `${r.baseAmount} ${r.baseCurrency}`,
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
      width: 100,
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
        access.canUpdateFinanceCashflows && r.status === 'DRAFT' ? (
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
        access.canUpdateFinanceCashflows && r.status !== 'CANCELLED' ? (
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
      <ProTable<API.FinanceCashflow>
        headerTitle="收付管理"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        bordered
        size="small"
        scroll={{ x: 1550 }}
        toolBarRender={() =>
          access.canCreateFinanceCashflows
            ? [
                <Button
                  key="new"
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => setOpen(true)}
                >
                  录入收付款
                </Button>,
              ]
            : []
        }
        request={async (p) => {
          const r = await settlementServiceListCashflows({
            page: p.current,
            pageSize: p.pageSize,
            keyword: p.keyword,
            direction: p.direction,
            status: p.status,
          });
          return {
            data: r.data || [],
            total: Number(r.total || 0),
            success: r.success ?? true,
          };
        }}
      />
      <ModalForm<Values>
        title="录入真实资金流水"
        open={open}
        width={720}
        modalProps={{ destroyOnHidden: true, onCancel: () => setOpen(false) }}
        initialValues={{
          direction: 'RECEIVABLE',
          currency: 'CNY',
          baseCurrency: 'CNY',
          exchangeRate: '1',
          transactionDate: dayjs(),
          paymentMethod: '银行转账',
        }}
        onFinish={async (v) => {
          try {
            await settlementServiceCreateCashflow({
              direction: v.direction,
              settlementPartyId: v.settlementPartyId,
              currency: v.currency,
              amount: v.amount,
              exchangeRate: v.exchangeRate,
              baseCurrency: v.baseCurrency,
              transactionDate: dayjs(v.transactionDate).format('YYYY-MM-DD'),
              ourAccount: v.ourAccount,
              counterpartyAccount: v.counterpartyAccount,
              paymentMethod: v.paymentMethod,
              bankReferenceNo: v.bankReferenceNo,
              note: v.note,
              idempotencyKey: globalThis.crypto.randomUUID(),
            });
            message.success('资金流水录入成功');
            setOpen(false);
            reload();
            return true;
          } catch (e: any) {
            message.error(e.message || '录入失败');
            return false;
          }
        }}
      >
        <ProFormSelect
          name="direction"
          label="资金方向"
          rules={[{ required: true }]}
          options={[
            { value: 'RECEIVABLE', label: '收款' },
            { value: 'PAYABLE', label: '付款' },
          ]}
        />
        <ProFormSelect
          name="settlementPartyId"
          label="对方单位"
          rules={[{ required: true }]}
          request={async () => {
            const r = await partnerServiceListPartners({
              page: 1,
              pageSize: 200,
              enabled: true,
            });
            return (r.data || []).map((x) => ({
              value: x.id,
              label: `${x.code || ''} ${x.legalName || ''}`,
            }));
          }}
        />
        <ProFormText
          name="amount"
          label="原币金额"
          rules={[{ required: true }, decimalRule]}
        />
        <ProFormText
          name="currency"
          label="原币币种"
          rules={[{ required: true, pattern: /^[A-Za-z]{3}$/ }]}
        />
        <ProFormText
          name="exchangeRate"
          label="折本币汇率"
          rules={[{ required: true }, decimalRule]}
        />
        <ProFormText
          name="baseCurrency"
          label="本位币"
          rules={[{ required: true, pattern: /^[A-Za-z]{3}$/ }]}
        />
        <ProFormDatePicker
          name="transactionDate"
          label="交易日期"
          rules={[{ required: true }]}
        />
        <ProFormText
          name="ourAccount"
          label="我方收付账户"
          rules={[{ required: true, max: 200 }]}
        />
        <ProFormText name="counterpartyAccount" label="对方账户" />
        <ProFormText
          name="paymentMethod"
          label="支付方式"
          rules={[{ required: true, max: 50 }]}
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
