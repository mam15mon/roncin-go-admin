import { PlusOutlined, RollbackOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Input, Tag } from 'antd';
import { useRef, useState } from 'react';
import {
  FinanceLedgerTemplate,
  type FinanceLedgerMetricCard,
} from '@/components/ui';
import {
  settlementServiceListVerifications,
  settlementServiceReverseVerification,
} from '@/services/roncin/settlementService';
import VerificationWorkbench from './VerificationWorkbench';

export default function FinanceVerificationsPage() {
  const access = useAccess();
  const { message, modal } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [open, setOpen] = useState(false);
  const [metricStats, setMetricStats] = useState({
    totalCount: 0,
    receivableTotal: 0,
    payableTotal: 0,
  });

  const reload = () => actionRef.current?.reload();
  const reverse = (r: API.FinanceVerification) => {
    const id = r.id;
    const v = r.version;
    if (!id || !v) return;
    let reason = '';
    modal.confirm({
      title: '反核销该批分配？',
      content: (
        <Input.TextArea
          placeholder="请输入反核销原因（必填）"
          onChange={(e) => {
            reason = e.target.value.trim();
          }}
        />
      ),
      onOk: async () => {
        if (!reason) {
          message.warning('请输入原因');
          throw new Error('原因不能为空');
        }
        await settlementServiceReverseVerification(
          { id },
          { id, expectedVersion: v, reason },
        );
        message.success('反核销成功，资金和账单余额已释放');
        reload();
      },
    });
  };

  const metricCards: FinanceLedgerMetricCard[] = [
    {
      key: 'total-verifications',
      title: '有效核销总笔数',
      value: metricStats.totalCount,
      suffix: '笔',
    },
    {
      key: 'rec-verifications',
      title: '应收核销总金额',
      value: metricStats.receivableTotal,
      precision: 2,
      suffix: 'CNY',
      valueColor: '#1677ff',
    },
    {
      key: 'pay-verifications',
      title: '应付核销总金额',
      value: metricStats.payableTotal,
      precision: 2,
      suffix: 'CNY',
      valueColor: '#fa8c16',
    },
  ];

  const columns: ProColumns<API.FinanceVerification>[] = [
    {
      title: '序号',
      dataIndex: 'index',
      valueType: 'index',
      width: 55,
      fixed: 'left',
    },
    {
      title: '核销编号',
      dataIndex: 'verificationNo',
      width: 170,
      copyable: true,
      search: false,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      valueType: 'select',
      valueEnum: { ACTIVE: { text: '有效' }, REVERSED: { text: '已反核销' } },
      render: (_, r) => (
        <Tag color={r.status === 'ACTIVE' ? 'green' : 'default'} style={{ margin: 0 }}>
          {r.status === 'ACTIVE' ? '有效' : '已反核销'}
        </Tag>
      ),
    },
    {
      title: '方向',
      dataIndex: 'direction',
      width: 90,
      search: false,
      renderText: (v) => (v === 'RECEIVABLE' ? '应收核销' : '应付核销'),
    },
    {
      title: '结算单位',
      dataIndex: 'settlementPartyName',
      width: 220,
      ellipsis: true,
      search: false,
    },
    {
      title: '核销金额',
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
      title: '分配数',
      search: false,
      render: (_, r) => r.allocations?.length || 0,
      width: 75,
      align: 'center',
    },
    {
      title: '核销日期',
      dataIndex: 'verificationDate',
      width: 110,
      search: false,
    },
    {
      title: '关联明细',
      search: false,
      width: 280,
      ellipsis: true,
      render: (_, r) =>
        (r.allocations || [])
          .map((x) => `${x.cashflowNo} → ${x.billNo}: ${x.amount}`)
          .join('；'),
    },
    {
      title: '操作',
      valueType: 'option',
      fixed: 'right',
      width: 110,
      render: (_, r) =>
        access.canReverseFinanceVerifications && r.status === 'ACTIVE'
          ? [
              <a key="reverse" onClick={() => reverse(r)}>
                <RollbackOutlined /> 反核销
              </a>,
            ]
          : [],
    },
  ];

  return (
    <>
      <FinanceLedgerTemplate<API.FinanceVerification>
        headerTitle="核销台账管理"
        actionRef={actionRef}
        columns={columns}
        metricCards={metricCards}
        scrollX={1500}
        primaryActionText="新建核销"
        primaryActionIcon={<PlusOutlined />}
        onPrimaryAction={
          access.canCreateFinanceVerifications ? () => setOpen(true) : undefined
        }
        request={async (p) => {
          const r = await settlementServiceListVerifications({
            page: p.current,
            pageSize: p.pageSize,
            keyword: p.keyword,
            status: p.status,
          });
          const list = r.data || [];
          let recTotal = 0;
          let payTotal = 0;
          for (const item of list) {
            const amount = Number(item.amount || 0);
            if (item.direction === 'RECEIVABLE') {
              recTotal += amount;
            } else if (item.direction === 'PAYABLE') {
              payTotal += amount;
            }
          }
          setMetricStats({
            totalCount: Number(r.total || 0),
            receivableTotal: recTotal,
            payableTotal: payTotal,
          });
          return {
            data: list,
            total: Number(r.total || 0),
            success: r.success ?? true,
          };
        }}
      />
      <VerificationWorkbench
        open={open}
        onClose={() => setOpen(false)}
        onCreated={reload}
      />
    </>
  );
}
