import {
  CheckOutlined,
  CloseCircleOutlined,
  DollarOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDigit,
  ProFormSelect,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Button, Input, Space, Tag } from 'antd';
import React, { useRef, useState } from 'react';
import {
  settlementServiceCancelCommission,
  settlementServiceConfirmCommission,
  settlementServiceCreateCommission,
  settlementServiceListCommissionEmployees,
  settlementServiceListCommissions,
  settlementServiceListVerifications,
  settlementServiceMarkCommissionPaid,
} from '@/services/roncin/settlementService';

type CreateValues = {
  verificationId: string;
  employeeId: string;
  ratePercent: number;
  note?: string;
};

const statusMeta: Record<string, { text: string; color: string }> = {
  DRAFT: { text: '草稿', color: 'processing' },
  CONFIRMED: { text: '已确认', color: 'success' },
  PAID: { text: '已发放', color: 'blue' },
  CANCELLED: { text: '已取消', color: 'default' },
};

export default function FinanceCommissionsPage() {
  const access = useAccess();
  const { message, modal } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [open, setOpen] = useState(false);
  const reload = () => actionRef.current?.reload();

  const transition = async (
    record: API.FinanceCommission,
    target: 'CONFIRMED' | 'PAID',
  ) => {
    if (!record.id || !record.version) return;
    const id = record.id;
    const version = record.version;
    const action = target === 'CONFIRMED' ? '确认' : '标记已发放';
    modal.confirm({
      title: `${action}提成 ${record.commissionNo}？`,
      content:
        target === 'CONFIRMED'
          ? '确认后计算快照将锁定。'
          : '该操作表示提成已实际发放，完成后不可取消。',
      onOk: async () => {
        const body = { id, expectedVersion: version };
        if (target === 'CONFIRMED') {
          await settlementServiceConfirmCommission({ id }, body);
        } else {
          await settlementServiceMarkCommissionPaid({ id }, body);
        }
        message.success(`${action}成功`);
        reload();
      },
    });
  };

  const cancel = (record: API.FinanceCommission) => {
    if (!record.id || !record.version) return;
    const id = record.id;
    const version = record.version;
    let reason = '';
    modal.confirm({
      title: `取消提成 ${record.commissionNo}？`,
      content: (
        <Input.TextArea
          placeholder="请输入取消原因（必填）"
          maxLength={500}
          onChange={(event) => {
            reason = event.target.value.trim();
          }}
        />
      ),
      onOk: async () => {
        if (!reason) {
          message.warning('请输入取消原因');
          throw new Error('取消原因不能为空');
        }
        await settlementServiceCancelCommission(
          { id },
          { id, expectedVersion: version, reason },
        );
        message.success('提成已取消，关联核销可以反核销');
        reload();
      },
    });
  };

  const columns: ProColumns<API.FinanceCommission>[] = [
    {
      title: '关键词',
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: { placeholder: '提成号、核销号或员工' },
    },
    {
      title: '提成编号',
      dataIndex: 'commissionNo',
      width: 260,
      copyable: true,
      search: false,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        Object.entries(statusMeta).map(([key, value]) => [key, value.text]),
      ),
      render: (_, record) => {
        const meta = statusMeta[record.status || 'DRAFT'];
        return <Tag color={meta.color}>{meta.text}</Tag>;
      },
    },
    {
      title: '核销编号',
      dataIndex: 'verificationNo',
      width: 170,
      copyable: true,
      search: false,
    },
    {
      title: '提成员工',
      dataIndex: 'employeeName',
      width: 130,
      search: false,
    },
    {
      title: '已实现收入',
      dataIndex: 'realizedRevenue',
      width: 150,
      align: 'right',
      search: false,
      renderText: (value, record) => `${value} ${record.baseCurrency}`,
    },
    {
      title: '分摊成本',
      dataIndex: 'allocatedCost',
      width: 150,
      align: 'right',
      search: false,
      renderText: (value, record) => `${value} ${record.baseCurrency}`,
    },
    {
      title: '已实现毛利',
      dataIndex: 'realizedProfit',
      width: 150,
      align: 'right',
      search: false,
      render: (_, record) => (
        <strong>{`${record.realizedProfit} ${record.baseCurrency}`}</strong>
      ),
    },
    {
      title: '比例',
      dataIndex: 'ratePercent',
      width: 90,
      align: 'right',
      search: false,
      renderText: (value) => `${value}%`,
    },
    {
      title: '提成金额',
      dataIndex: 'commissionAmount',
      width: 150,
      align: 'right',
      search: false,
      render: (_, record) => (
        <strong style={{ color: '#1677ff' }}>
          {`${record.commissionAmount} ${record.baseCurrency}`}
        </strong>
      ),
    },
    {
      title: '操作',
      valueType: 'option',
      fixed: 'right',
      width: 190,
      render: (_, record) => {
        if (!access.canManageFinanceCommissions) return [];
        return [
          record.status === 'DRAFT' ? (
            <a key="confirm" onClick={() => transition(record, 'CONFIRMED')}>
              <CheckOutlined /> 确认
            </a>
          ) : null,
          record.status === 'CONFIRMED' ? (
            <a key="paid" onClick={() => transition(record, 'PAID')}>
              <DollarOutlined /> 已发放
            </a>
          ) : null,
          ['DRAFT', 'CONFIRMED'].includes(record.status || '') ? (
            <a key="cancel" onClick={() => cancel(record)}>
              <CloseCircleOutlined /> 取消
            </a>
          ) : null,
        ].filter(Boolean);
      },
    },
  ];

  return (
    <>
      <ProTable<API.FinanceCommission>
        headerTitle="提成管理"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        bordered
        size="small"
        scroll={{ x: 1700 }}
        toolBarRender={() =>
          access.canManageFinanceCommissions
            ? [
                <Button
                  key="create"
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => setOpen(true)}
                >
                  生成提成
                </Button>,
              ]
            : []
        }
        request={async (params) => {
          const response = await settlementServiceListCommissions({
            page: params.current,
            pageSize: params.pageSize,
            keyword: params.keyword,
            status: params.status,
          });
          return {
            data: response.data || [],
            total: Number(response.total || 0),
            success: response.success ?? true,
          };
        }}
      />
      <ModalForm<CreateValues>
        title="按已实现毛利生成提成"
        open={open}
        width={640}
        modalProps={{ destroyOnHidden: true, onCancel: () => setOpen(false) }}
        onFinish={async (values) => {
          try {
            await settlementServiceCreateCommission({
              verificationId: values.verificationId,
              employeeId: values.employeeId,
              ratePercent: String(values.ratePercent),
              note: values.note,
              idempotencyKey: globalThis.crypto.randomUUID(),
            });
            message.success('提成计算完成，请核对后确认');
            setOpen(false);
            reload();
            return true;
          } catch (error: any) {
            message.error(error.message || '提成生成失败');
            return false;
          }
        }}
      >
        <ProFormSelect
          name="verificationId"
          label="有效应收核销"
          rules={[{ required: true }]}
          request={async () => {
            const response = await settlementServiceListVerifications({
              page: 1,
              pageSize: 200,
              status: 'ACTIVE',
            });
            return (response.data || [])
              .filter((item) => item.direction === 'RECEIVABLE')
              .map((item) => ({
                label: `${item.verificationNo}｜${item.settlementPartyName}｜${item.amount} ${item.currency}`,
                value: item.id,
              }));
          }}
        />
        <ProFormSelect
          name="employeeId"
          label="提成员工"
          rules={[{ required: true }]}
          request={async () => {
            const response = await settlementServiceListCommissionEmployees({});
            return (response.data || []).map((item) => ({
              label: item.displayName,
              value: item.id,
            }));
          }}
        />
        <ProFormDigit
          name="ratePercent"
          label="提成比例（%）"
          min={0.0001}
          max={100}
          fieldProps={{ precision: 4 }}
          rules={[{ required: true }]}
          extra="提成基数由系统按有效核销对应的已实现毛利计算，不允许手工修改。"
        />
        <ProFormTextArea
          name="note"
          label="备注"
          fieldProps={{ maxLength: 500 }}
        />
        <Space direction="vertical" size={2} style={{ color: '#666' }}>
          <span>计算口径：已实现收入 − 按订单毛利率分摊的成本。</span>
          <span>亏损订单的提成基数按 0 处理，但仍保留真实负毛利快照。</span>
        </Space>
      </ModalForm>
    </>
  );
}
