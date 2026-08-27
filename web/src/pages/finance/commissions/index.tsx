import {
  CheckOutlined,
  CloseCircleOutlined,
  DollarOutlined,
  EditOutlined,
  EyeOutlined,
  PlusOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDateRangePicker,
  ProFormDependency,
  ProFormDigit,
  ProFormSwitch,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { ProFormSearchableSelect, SearchFilterTemplate } from '@/components/ui';
import { useAccess } from '@umijs/max';
import {
  Alert,
  App,
  Button,
  Descriptions,
  Drawer,
  Input,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, { useRef, useState } from 'react';
import {
  settlementServiceCancelCommissionAdjustment,
  settlementServiceCancelCommission,
  settlementServiceConfirmCommissionAdjustment,
  settlementServiceConfirmCommission,
  settlementServiceCreateCommissionAdjustment,
  settlementServiceCreateCommission,
  settlementServiceCreateCommissionRule,
  settlementServiceGetCommission,
  settlementServiceListCommissionCandidates,
  settlementServiceListCommissionRules,
  settlementServiceListCommissions,
  settlementServiceListVerifications,
  settlementServiceMarkCommissionPaid,
  settlementServiceMarkCommissionAdjustmentPaid,
  settlementServicePreviewCommission,
  settlementServiceUpdateCommissionRule,
} from '@/services/roncin/settlementService';

type CreateValues = {
  verificationId: string;
  employeeId: string;
  ruleId: string;
  note?: string;
};

type RuleValues = {
  name: string;
  personnelRole: 'SALES' | 'OPERATOR' | 'CUSTOMER_SERVICE';
  calculationBasis: 'REALIZED_PROFIT' | 'REALIZED_REVENUE';
  ratePercent: number;
  effectiveRange?: [Dayjs, Dayjs];
  enabled: boolean;
  note?: string;
};

type AdjustmentValues = {
  orderId: string;
  direction: 'INCREASE' | 'DECREASE';
  amount: string;
  reason: string;
  note?: string;
};

const commissionStatusMeta: Record<string, { text: string; color: string }> = {
  DRAFT: { text: '草稿', color: 'processing' },
  CONFIRMED: { text: '已确认', color: 'success' },
  PAID: { text: '已发放', color: 'blue' },
  CANCELLED: { text: '已取消', color: 'default' },
};

const personnelRoleMeta: Record<string, string> = {
  SALES: '业务人员',
  OPERATOR: '操作人员',
  CUSTOMER_SERVICE: '客服人员',
};

const personnelRoleText = (value?: string) =>
  personnelRoleMeta[value || ''] || '业务人员';

const calculationBasisMeta: Record<string, string> = {
  REALIZED_PROFIT: '已实现毛利',
  REALIZED_REVENUE: '已实现收入',
};

const calculationBasisText = (value?: string) =>
  calculationBasisMeta[value || ''] || '已实现毛利';

const decimalText = (value?: string) => {
  if (!value) return '0';
  return value.replace(/(\.\d*?[1-9])0+$|\.0+$/, '$1');
};

const calculationSignature = (values: Partial<CreateValues>) =>
  [values.verificationId, values.ruleId, values.employeeId].join('|');

function getBusinessReason(error: any): string {
  return (
    error?.data?.reason ??
    error?.response?.data?.reason ??
    error?.reason ??
    ''
  );
}

function getAdjustmentStatusInfo(adjustment: API.FinanceCommissionAdjustment) {
  const isReversal = adjustment.sourceType === 'VERIFICATION_REVERSAL';
  const isDecrease = adjustment.direction === 'DECREASE';

  if (isReversal) {
    if (adjustment.status === 'CONFIRMED') return { text: '待追回', color: 'warning' };
    if (adjustment.status === 'PAID') return { text: '已追回', color: 'purple' };
    if (adjustment.status === 'CANCELLED') return { text: '已取消', color: 'default' };
    return { text: '反核销草稿', color: 'processing' };
  }

  if (isDecrease) {
    if (adjustment.status === 'DRAFT') return { text: '冲减草稿', color: 'processing' };
    if (adjustment.status === 'CONFIRMED') return { text: '待扣回', color: 'warning' };
    if (adjustment.status === 'PAID') return { text: '已扣回', color: 'purple' };
    return { text: '已取消', color: 'default' };
  }

  if (adjustment.status === 'DRAFT') return { text: '增提草稿', color: 'processing' };
  if (adjustment.status === 'CONFIRMED') return { text: '待发放', color: 'success' };
  if (adjustment.status === 'PAID') return { text: '已发放', color: 'blue' };
  return { text: '已取消', color: 'default' };
}

export default function FinanceCommissionsPage() {
  const access = useAccess();
  const { message, modal } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const ruleActionRef = useRef<ActionType | undefined>(undefined);
  const createFormRef = useRef<ProFormInstance<CreateValues>>(undefined);

  const [open, setOpen] = useState(false);
  const [createIdempotencyKey, setCreateIdempotencyKey] = useState(() =>
    globalThis.crypto.randomUUID(),
  );

  const [rulesOpen, setRulesOpen] = useState(false);
  const [ruleFormOpen, setRuleFormOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<API.FinanceCommissionRule>();

  const [preview, setPreview] = useState<API.CommissionCalculation>();
  const [previewSignature, setPreviewSignature] = useState('');
  const [previewLoading, setPreviewLoading] = useState(false);

  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detail, setDetail] = useState<API.FinanceCommission>();

  const [adjustmentOpen, setAdjustmentOpen] = useState(false);
  const [adjustmentIdempotencyKey, setAdjustmentIdempotencyKey] = useState(() =>
    globalThis.crypto.randomUUID(),
  );

  const reload = () => actionRef.current?.reload();

  const openCreateModal = () => {
    setCreateIdempotencyKey(globalThis.crypto.randomUUID());
    setPreview(undefined);
    setPreviewSignature('');
    createFormRef.current?.resetFields();
    setOpen(true);
  };

  const openAdjustmentModal = () => {
    setAdjustmentIdempotencyKey(globalThis.crypto.randomUUID());
    setAdjustmentOpen(true);
  };

  const openDetail = async (record: API.FinanceCommission) => {
    if (!record.id) return;
    setDetailOpen(true);
    setDetailLoading(true);
    setDetail(undefined);
    try {
      const response = await settlementServiceGetCommission({ id: record.id });
      setDetail(response.data);
    } catch (error: any) {
      message.error(error.message || '提成明细加载失败');
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  };

  const refreshDetail = async () => {
    if (!detail?.id) return;
    try {
      const response = await settlementServiceGetCommission({ id: detail.id });
      setDetail(response.data);
    } catch {
      // ignore
    }
    reload();
  };

  const transitionAdjustment = (
    record: API.FinanceCommissionAdjustment,
    target: 'CONFIRMED' | 'PAID',
  ) => {
    if (!record.id || !record.version) return;
    const adjustmentID = record.id;
    const adjustmentVersion = record.version;
    const isReversal = record.sourceType === 'VERIFICATION_REVERSAL';
    const isDecrease = record.direction === 'DECREASE';

    let action = '确认调整';
    let content = '确认后该增减金额会计入有效提成；冲减不会被允许把有效提成降到零以下。';
    if (target === 'PAID') {
      if (isReversal) {
        action = '标记已追回';
        content = '该操作表示反核销冲减款项已实际追回，完成后不可取消。';
      } else if (isDecrease) {
        action = '标记已扣回';
        content = '该操作表示冲减金额已实际扣回，完成后不可取消。';
      } else {
        action = '标记已发放';
        content = '该操作表示增提金额已实际发放，完成后不可取消。';
      }
    }

    modal.confirm({
      title: `${action} ${record.adjustmentNo}？`,
      content,
      onOk: async () => {
        try {
          const body = { id: adjustmentID, expectedVersion: adjustmentVersion };
          if (target === 'CONFIRMED') {
            await settlementServiceConfirmCommissionAdjustment(
              { id: adjustmentID },
              body,
            );
          } else {
            await settlementServiceMarkCommissionAdjustmentPaid(
              { id: adjustmentID },
              body,
            );
          }
          message.success(`${action}成功`);
          await refreshDetail();
        } catch (error: any) {
          const reason = getBusinessReason(error);
          if (reason === 'FINANCE_COMMISSION_ADJUSTMENT_EXCEEDS') {
            modal.warning({
              title: '冲减金额超限',
              content: '冲减后的有效提成不能小于零，请检查调整金额。',
            });
            return;
          }
          if (reason === 'FINANCE_COMMISSION_ADJUSTMENT_TRANSITION') {
            message.warning('调整状态已变化，已刷新详情');
            await refreshDetail();
            return;
          }
          message.error(error.message || `${action}失败`);
        }
      },
    });
  };

  const cancelAdjustment = (record: API.FinanceCommissionAdjustment) => {
    if (!record.id || !record.version) return;
    const adjustmentID = record.id;
    const adjustmentVersion = record.version;
    let reason = '';
    modal.confirm({
      title: `取消调整 ${record.adjustmentNo}？`,
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
        try {
          await settlementServiceCancelCommissionAdjustment(
            { id: adjustmentID },
            { id: adjustmentID, expectedVersion: adjustmentVersion, reason },
          );
          message.success('调整已取消');
          await refreshDetail();
        } catch (error: any) {
          message.error(error.message || '取消调整失败');
        }
      },
    });
  };

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
          ? '系统会重新核对核销、账单费用、提成规则和客户人员归属；来源发生变化时将拒绝确认。'
          : '该操作表示提成已实际发放，完成后不可取消。',
      onOk: async () => {
        try {
          const body = { id, expectedVersion: version };
          if (target === 'CONFIRMED') {
            await settlementServiceConfirmCommission({ id }, body);
          } else {
            await settlementServiceMarkCommissionPaid({ id }, body);
          }
          message.success(`${action}成功`);
          reload();
          if (detail?.id === id) {
            await refreshDetail();
          }
        } catch (error: any) {
          const reason = getBusinessReason(error);
          if (reason === 'FINANCE_COMMISSION_SOURCE_CHANGED') {
            modal.warning({
              title: '提成来源已经变化',
              content: '请取消当前草稿，然后根据最新核销、费用和人员归属重新生成。',
            });
            return;
          }
          if (reason === 'FINANCE_COMMISSION_UNCONFIRMED_FEES') {
            modal.warning({
              title: '关联订单存在草稿费用',
              content: '请先确认或作废草稿费用，再确认提成。',
            });
            return;
          }
          if (reason === 'FINANCE_COMMISSION_TRANSITION') {
            message.warning('提成状态已变化，页面已刷新');
            reload();
            if (detail?.id === id) await refreshDetail();
            return;
          }
          message.error(error.message || `${action}失败`);
        }
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
        try {
          await settlementServiceCancelCommission(
            { id },
            { id, expectedVersion: version, reason },
          );
          message.success('提成已取消');
          reload();
          if (detail?.id === id) {
            await refreshDetail();
          }
        } catch (error: any) {
          message.error(error.message || '取消提成失败');
        }
      },
    });
  };

  const columns: ProColumns<API.FinanceCommission>[] = [
    {
      title: '关键词',
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: { placeholder: '提成号、员工或规则' },
    },
    {
      title: '提成编号',
      dataIndex: 'commissionNo',
      width: 240,
      copyable: true,
      search: false,
      render: (_, record) => (
        <a onClick={() => openDetail(record)}>{record.commissionNo}</a>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        Object.entries(commissionStatusMeta).map(([key, value]) => [
          key,
          value.text,
        ]),
      ),
      render: (_, record) => {
        const meta = commissionStatusMeta[record.status || 'DRAFT'];
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
      width: 120,
      search: false,
    },
    {
      title: '考核规则',
      dataIndex: 'ruleName',
      width: 160,
      search: false,
      renderText: (value) => value || '历史手工计提',
    },
    {
      title: '角色/口径',
      width: 160,
      search: false,
      render: (_, record) => (
        <Space direction="vertical" size={0}>
          <span>{personnelRoleText(record.personnelRole)}</span>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {calculationBasisText(record.calculationBasis)}
          </Typography.Text>
        </Space>
      ),
    },
    {
      title: '业务覆盖',
      width: 140,
      search: false,
      render: (_, record) => (
        <Space size={4}>
          <Tag color="blue">{record.customerCount ?? 1} 客户</Tag>
          <Tag color="cyan">{record.orderCount ?? 1} 订单</Tag>
        </Space>
      ),
    },
    {
      title: '已实现收入',
      dataIndex: 'realizedRevenue',
      width: 140,
      align: 'right',
      search: false,
      renderText: (value, record) =>
        `${decimalText(value)} ${record.baseCurrency}`,
    },
    {
      title: '分摊成本',
      dataIndex: 'allocatedCost',
      width: 140,
      align: 'right',
      search: false,
      renderText: (value, record) =>
        `${decimalText(value)} ${record.baseCurrency}`,
    },
    {
      title: '已实现毛利',
      dataIndex: 'realizedProfit',
      width: 140,
      align: 'right',
      search: false,
      render: (_, record) => (
        <strong>{`${decimalText(record.realizedProfit)} ${record.baseCurrency}`}</strong>
      ),
    },
    {
      title: '比例',
      dataIndex: 'ratePercent',
      width: 80,
      align: 'right',
      search: false,
      renderText: (value) => `${decimalText(value)}%`,
    },
    {
      title: '原始/有效提成',
      dataIndex: 'commissionAmount',
      width: 180,
      align: 'right',
      search: false,
      render: (_, record) => {
        if (record.status === 'CANCELLED') {
          return (
            <Space direction="vertical" size={0}>
              <Typography.Text type="secondary" delete>
                {`快照 ${decimalText(record.commissionAmount)} ${record.baseCurrency}`}
              </Typography.Text>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                已取消，不计入应发
              </Typography.Text>
            </Space>
          );
        }
        return (
          <Space direction="vertical" size={0}>
            <span>{`原始 ${decimalText(record.commissionAmount)} ${record.baseCurrency}`}</span>
            <strong style={{ color: '#1677ff' }}>
              {`有效 ${decimalText(record.effectiveCommissionAmount || record.commissionAmount)} ${record.baseCurrency}`}
            </strong>
          </Space>
        );
      },
    },
    {
      title: '操作',
      valueType: 'option',
      fixed: 'right',
      width: 190,
      render: (_, record) => {
        return [
          <a key="detail" onClick={() => openDetail(record)}>
            <EyeOutlined /> 明细
          </a>,
          record.status === 'DRAFT' ? (
            access.canManageFinanceCommissions ? (
              <a key="confirm" onClick={() => transition(record, 'CONFIRMED')}>
                <CheckOutlined /> 确认
              </a>
            ) : null
          ) : null,
          record.status === 'CONFIRMED' &&
          access.canManageFinanceCommissions ? (
            <a key="paid" onClick={() => transition(record, 'PAID')}>
              <DollarOutlined /> 已发放
            </a>
          ) : null,
          ['DRAFT', 'CONFIRMED'].includes(record.status || '') &&
          access.canManageFinanceCommissions ? (
            <a key="cancel" onClick={() => cancel(record)}>
              <CloseCircleOutlined /> 取消
            </a>
          ) : null,
        ].filter(Boolean);
      },
    },
  ];

  const previewColumns = [
    {
      title: '客户名称',
      dataIndex: 'customerName',
      key: 'customerName',
      width: 180,
      render: (val: string, line: API.FinanceCommissionLine) => (
        <Space direction="vertical" size={0}>
          <Typography.Text strong>{val || '-'}</Typography.Text>
          {line.customerCode && (
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {line.customerCode}
            </Typography.Text>
          )}
        </Space>
      ),
    },
    {
      title: '订单编号',
      dataIndex: 'orderNo',
      key: 'orderNo',
      width: 170,
      render: (val: string, line: API.FinanceCommissionLine) => (
        <Space direction="vertical" size={0}>
          <Typography.Text strong>{val}</Typography.Text>
          {line.orderDate && (
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {line.orderDate}
            </Typography.Text>
          )}
        </Space>
      ),
    },
    {
      title: '客户人员归属',
      key: 'personnelSnapshot',
      width: 170,
      render: (_: unknown, line: API.FinanceCommissionLine) => (
        <Space direction="vertical" size={0}>
          <span>{`${line.employeeName} · ${personnelRoleText(line.personnelRole)}`}</span>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {line.customerAssignedAt || line.personnelAssignedAt
              ? `归属于 ${dayjs(line.customerAssignedAt || line.personnelAssignedAt).format('YYYY-MM-DD')}`
              : '-'}
          </Typography.Text>
        </Space>
      ),
    },
    {
      title: '已实现收入',
      dataIndex: 'realizedRevenue',
      key: 'realizedRevenue',
      align: 'right' as const,
      render: (value: string, line: API.FinanceCommissionLine) =>
        `${decimalText(value)} ${line.baseCurrency}`,
    },
    {
      title: '分摊成本',
      dataIndex: 'allocatedCost',
      key: 'allocatedCost',
      align: 'right' as const,
      render: (value: string, line: API.FinanceCommissionLine) =>
        `${decimalText(value)} ${line.baseCurrency}`,
    },
    {
      title: '已实现毛利',
      dataIndex: 'realizedProfit',
      key: 'realizedProfit',
      align: 'right' as const,
      render: (value: string, line: API.FinanceCommissionLine) => (
        <strong>{`${decimalText(value)} ${line.baseCurrency}`}</strong>
      ),
    },
    {
      title: '提成金额',
      dataIndex: 'commissionAmount',
      key: 'commissionAmount',
      align: 'right' as const,
      render: (value: string, line: API.FinanceCommissionLine) => (
        <Typography.Text strong type="success">
          {`${decimalText(value)} ${line.baseCurrency}`}
        </Typography.Text>
      ),
    },
  ];

  const renderExpandedFees = (record: API.FinanceCommissionLine) => {
    if (!record.fees || record.fees.length === 0) {
      return (
        <Typography.Text
          type="secondary"
          style={{ padding: '8px 16px', display: 'block' }}
        >
          暂无关联费用明细
        </Typography.Text>
      );
    }
    const feeColumns = [
      {
        title: '费用项目',
        key: 'feeName',
        render: (_: unknown, fee: API.CommissionFeeDetail) => (
          <Space size={4}>
            <Typography.Text strong>{fee.feeName || '-'}</Typography.Text>
            {fee.feeCode && <Tag>{fee.feeCode}</Tag>}
          </Space>
        ),
      },
      {
        title: '收付方向',
        dataIndex: 'direction',
        key: 'direction',
        width: 90,
        render: (dir: string) => (
          <Tag color={dir === 'RECEIVABLE' ? 'blue' : 'orange'}>
            {dir === 'RECEIVABLE' ? '应收' : '应付'}
          </Tag>
        ),
      },
      {
        title: '结算单位',
        dataIndex: 'settlementPartyName',
        key: 'settlementPartyName',
        render: (val?: string) => val || '-',
      },
      {
        title: '原币金额',
        key: 'totalAmount',
        align: 'right' as const,
        render: (_: unknown, fee: API.CommissionFeeDetail) =>
          `${decimalText(fee.totalAmount)} ${fee.currency || ''}`,
      },
      {
        title: '汇率',
        dataIndex: 'exchangeRate',
        key: 'exchangeRate',
        align: 'right' as const,
        width: 90,
        render: (val?: string) => decimalText(val),
      },
      {
        title: '折本币金额',
        key: 'baseCurrencyAmount',
        align: 'right' as const,
        render: (_: unknown, fee: API.CommissionFeeDetail) => (
          <Typography.Text strong>
            {`${decimalText(fee.baseCurrencyAmount)} ${fee.baseCurrency || record.baseCurrency || ''}`}
          </Typography.Text>
        ),
      },
      {
        title: '费用日期',
        dataIndex: 'expenseDate',
        key: 'expenseDate',
        width: 110,
        render: (val?: string) => val || '-',
      },
    ];

    return (
      <div
        style={{
          padding: '10px 16px',
          backgroundColor: '#fafbfc',
          borderRadius: 4,
          border: '1px solid #f0f0f0',
          margin: '4px 0',
        }}
      >
        <Typography.Text
          type="secondary"
          style={{ fontSize: 12, marginBottom: 6, display: 'block' }}
        >
          订单费用构成核算快照（共 {record.fees.length} 笔）
        </Typography.Text>
        <Table<API.CommissionFeeDetail>
          rowKey={(item) =>
            item.feeId || `${item.feeCode}-${item.expenseDate}-${item.totalAmount}`
          }
          columns={feeColumns}
          dataSource={record.fees}
          pagination={false}
          size="small"
          bordered
        />
      </div>
    );
  };

  const adjustmentColumns = [
    { title: '调整编号', dataIndex: 'adjustmentNo', key: 'adjustmentNo', width: 180 },
    { title: '归属订单', dataIndex: 'orderNo', key: 'orderNo', width: 170 },
    {
      title: '调整来源',
      dataIndex: 'sourceType',
      key: 'sourceType',
      width: 110,
      render: (value: string) =>
        value === 'VERIFICATION_REVERSAL' ? (
          <Tag color="volcano">核销撤销</Tag>
        ) : (
          <Tag>手工调整</Tag>
        ),
    },
    {
      title: '方向',
      dataIndex: 'direction',
      key: 'direction',
      width: 80,
      render: (value: string) => (
        <Tag color={value === 'INCREASE' ? 'green' : 'red'}>
          {value === 'INCREASE' ? '增提' : '冲减'}
        </Tag>
      ),
    },
    {
      title: '金额',
      dataIndex: 'amount',
      key: 'amount',
      width: 140,
      align: 'right' as const,
      render: (value: string, record: API.FinanceCommissionAdjustment) => (
        <Typography.Text
          type={record.direction === 'DECREASE' ? 'danger' : 'success'}
          strong
        >
          {`${record.direction === 'DECREASE' ? '-' : '+'}${decimalText(value)} ${record.baseCurrency}`}
        </Typography.Text>
      ),
    },
    { title: '调整原因', dataIndex: 'reason', key: 'reason', ellipsis: true },
    {
      title: '状态',
      key: 'status',
      width: 100,
      render: (_: unknown, record: API.FinanceCommissionAdjustment) => {
        const info = getAdjustmentStatusInfo(record);
        return <Tag color={info.color}>{info.text}</Tag>;
      },
    },
    {
      title: '操作',
      key: 'action',
      width: 160,
      render: (_: unknown, record: API.FinanceCommissionAdjustment) => {
        const isReversal = record.sourceType === 'VERIFICATION_REVERSAL';
        const isDecrease = record.direction === 'DECREASE';

        return (
          <Space size={8}>
            {/* 手工调整草稿确认 */}
            {!isReversal &&
              record.status === 'DRAFT' &&
              access.canManageFinanceCommissions && (
                <a onClick={() => transitionAdjustment(record, 'CONFIRMED')}>
                  确认
                </a>
              )}

            {/* 反核销冲减已确认 -> 标记已追回 */}
            {isReversal &&
              record.status === 'CONFIRMED' &&
              access.canManageFinanceCommissions && (
                <a onClick={() => transitionAdjustment(record, 'PAID')}>
                  标记已追回
                </a>
              )}

            {/* 手工调整已确认 -> 标记已发放 / 标记已扣回 */}
            {!isReversal &&
              record.status === 'CONFIRMED' &&
              access.canManageFinanceCommissions && (
                <a onClick={() => transitionAdjustment(record, 'PAID')}>
                  {isDecrease ? '已扣回' : '已发放'}
                </a>
              )}

            {/* 仅手工调整允许取消；反核销自动记录不允许取消 */}
            {!isReversal &&
              ['DRAFT', 'CONFIRMED'].includes(record.status || '') &&
              access.canManageFinanceCommissions && (
                <a
                  style={{ color: '#ff4d4f' }}
                  onClick={() => cancelAdjustment(record)}
                >
                  取消
                </a>
              )}
          </Space>
        );
      },
    },
  ];

  const ruleColumns: ProColumns<API.FinanceCommissionRule>[] = [
    { title: '规则名称', dataIndex: 'name', width: 180 },
    {
      title: '适用角色',
      dataIndex: 'personnelRole',
      width: 120,
      valueType: 'select',
      valueEnum: {
        SALES: { text: '业务人员' },
        OPERATOR: { text: '操作人员' },
        CUSTOMER_SERVICE: { text: '客服人员' },
      },
      renderText: (value) => personnelRoleText(value),
    },
    {
      title: '计算口径',
      dataIndex: 'calculationBasis',
      width: 130,
      search: false,
      renderText: (value) => calculationBasisText(value),
    },
    {
      title: '比例',
      dataIndex: 'ratePercent',
      width: 90,
      align: 'right',
      search: false,
      renderText: (value) => `${decimalText(value)}%`,
    },
    {
      title: '有效期',
      width: 190,
      search: false,
      renderText: (_, record) =>
        `${record.effectiveFrom || '不限'} ～ ${record.effectiveTo || '不限'}`,
    },
    {
      title: '启用',
      dataIndex: 'enabled',
      width: 80,
      valueType: 'select',
      valueEnum: {
        true: { text: '启用' },
        false: { text: '停用' },
      },
      render: (_, record) => (
        <Tag color={record.enabled ? 'green' : 'default'}>
          {record.enabled ? '启用' : '停用'}
        </Tag>
      ),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 80,
      render: (_, record) =>
        access.canManageFinanceCommissions
          ? [
              <a
                key="edit"
                onClick={() => {
                  setEditingRule(record);
                  setRuleFormOpen(true);
                }}
              >
                <EditOutlined /> 编辑
              </a>,
            ]
          : [],
    },
  ];

  const [searchParams, setSearchParams] = useState<{ keyword?: string; status?: string }>({});

  return (
    <>
      <SearchFilterTemplate
        layout="bar"
        keywordPlaceholder="搜索提成单号、人员名称或业务单号..."
        quickFilters={[
          {
            name: 'status',
            placeholder: '全部状态',
            width: 120,
            options: [
              { label: '草稿', value: 'DRAFT' },
              { label: '已确认', value: 'CONFIRMED' },
              { label: '已作废', value: 'CANCELLED' },
            ],
          },
        ]}
        onSearch={(values) => {
          setSearchParams(values);
          actionRef.current?.reload();
        }}
        onReset={() => {
          setSearchParams({});
          actionRef.current?.reload();
        }}
        extraRight={
          <Space size={8}>
            {access.canManageFinanceCommissions && (
              <Button
                key="create"
                type="primary"
                icon={<PlusOutlined />}
                onClick={openCreateModal}
              >
                生成提成
              </Button>
            )}
            <Button
              key="rules"
              icon={<SettingOutlined />}
              onClick={() => setRulesOpen(true)}
            >
              {access.canManageFinanceCommissions ? '考核规则' : '查看规则'}
            </Button>
          </Space>
        }
      />
      <ProTable<API.FinanceCommission>
        headerTitle="提成管理"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        bordered
        size="small"
        scroll={{ x: 2150 }}
        search={false}
        toolBarRender={false}
        request={async (params) => {
          const response = await settlementServiceListCommissions({
            page: params.current ?? 1,
            pageSize: params.pageSize ?? 20,
            keyword: searchParams.keyword,
            status: searchParams.status,
          });
          return {
            data: response.data || [],
            total: Number(response.total || 0),
            success: response.success ?? true,
          };
        }}
      />

      {/* 生成提成弹窗 */}
      <ModalForm<CreateValues>
        formRef={createFormRef}
        title="生成提成"
        open={open}
        width={980}
        submitter={{
          searchConfig: { submitText: '生成草稿' },
          submitButtonProps: { disabled: !preview },
        }}
        modalProps={{
          destroyOnHidden: true,
          onCancel: () => {
            setOpen(false);
            setPreview(undefined);
            setPreviewSignature('');
          },
        }}
        onValuesChange={(changedValues) => {
          if (
            'verificationId' in changedValues ||
            'ruleId' in changedValues ||
            'employeeId' in changedValues
          ) {
            setPreview(undefined);
            setPreviewSignature('');
            setCreateIdempotencyKey(globalThis.crypto.randomUUID());
          }
        }}
        onFinish={async (values) => {
          if (!preview || previewSignature !== calculationSignature(values)) {
            message.warning('请先计算并核对当前选择的提成预览');
            return false;
          }
          try {
            await settlementServiceCreateCommission({
              verificationId: values.verificationId,
              employeeId: values.employeeId,
              ruleId: values.ruleId,
              note: values.note,
              idempotencyKey: createIdempotencyKey,
            });
            message.success('提成草稿已按预览结果生成');
            setOpen(false);
            setPreview(undefined);
            setPreviewSignature('');
            reload();
            return true;
          } catch (error: any) {
            message.error(error.message || '提成生成失败');
            return false;
          }
        }}
      >
        <ProFormSearchableSelect
          name="verificationId"
          label="有效应收核销"
          rules={[{ required: true, message: '请选择有效应收核销单' }]}
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
        <ProFormSearchableSelect
          name="ruleId"
          label="考核规则"
          rules={[{ required: true, message: '请选择考核规则' }]}
          request={async () => {
            const response = await settlementServiceListCommissionRules({
              page: 1,
              pageSize: 200,
              enabled: true,
            });
            return (response.data || []).map((item) => ({
              label: `${item.name}｜${personnelRoleText(item.personnelRole)}｜${calculationBasisText(item.calculationBasis)} × ${decimalText(item.ratePercent)}%`,
              value: item.id,
            }));
          }}
        />
        <ProFormDependency name={['verificationId', 'ruleId']}>
          {({ verificationId, ruleId }) => (
            <ProFormSearchableSelect
              key={`${verificationId || ''}-${ruleId || ''}`}
              name="employeeId"
              label="符合规则的候选人员"
              rules={[{ required: true, message: '请选择符合角色的候选人员' }]}
              disabled={!verificationId || !ruleId}
              request={async () => {
                if (!verificationId || !ruleId) return [];
                const response =
                  await settlementServiceListCommissionCandidates({
                    verificationId,
                    ruleId,
                    page: 1,
                    pageSize: 200,
                  });
                return (response.data || []).map((item) => ({
                  label: `${item.employeeName}｜${item.customerCount ?? 0}个客户｜${item.orderCount ?? 0}票订单｜预计 ${decimalText(item.commissionAmount)} ${item.baseCurrency}`,
                  value: item.employeeId,
                }));
              }}
              extra="人员来自本次核销涉及订单在创建时固化的业务、操作或客服归属；客户后续换人不会改变历史订单归属。"
            />
          )}
        </ProFormDependency>
        <ProFormTextArea
          name="note"
          label="备注"
          fieldProps={{ maxLength: 500 }}
        />
        <ProFormDependency name={['verificationId', 'ruleId', 'employeeId']}>
          {(values: Partial<CreateValues>) => (
            <Space direction="vertical" size={12} style={{ width: '100%' }}>
              <Button
                type="primary"
                ghost
                loading={previewLoading}
                disabled={
                  !values.verificationId || !values.ruleId || !values.employeeId
                }
                onClick={async () => {
                  const { verificationId, employeeId, ruleId } = values;
                  if (!verificationId || !employeeId || !ruleId) return;
                  try {
                    setPreviewLoading(true);
                    const response = await settlementServicePreviewCommission({
                      verificationId,
                      employeeId,
                      ruleId,
                    });
                    setPreview(response.data);
                    setPreviewSignature(calculationSignature(values));
                  } catch (error: any) {
                    setPreview(undefined);
                    setPreviewSignature('');
                    message.error(error.message || '提成预览计算失败');
                  } finally {
                    setPreviewLoading(false);
                  }
                }}
              >
                计算并核对预览
              </Button>
              {!preview ? (
                <Alert
                  type="info"
                  showIcon
                  message="生成前必须计算预览"
                  description="系统会按本次核销涉及的订单，分别展示已实现收入、分摊成本、毛利和提成金额，并支持下钻展开费用明细。"
                />
              ) : (
                <>
                  <Descriptions
                    size="small"
                    bordered
                    column={4}
                    items={[
                      {
                        key: 'employee',
                        label: '提成员工',
                        children: preview.employeeName,
                      },
                      {
                        key: 'rule',
                        label: '规则',
                        children: `${preview.ruleName}（v${preview.ruleVersion}）`,
                      },
                      {
                        key: 'basis',
                        label: '角色/口径',
                        children: `${personnelRoleText(preview.personnelRole)} · ${calculationBasisText(preview.calculationBasis)}`,
                      },
                      {
                        key: 'rate',
                        label: '比例',
                        children: `${decimalText(preview.ratePercent)}%`,
                      },
                      {
                        key: 'coverage',
                        label: '业务覆盖',
                        children: `${preview.customerCount || 1} 个客户 · ${preview.orderCount || 1} 票订单 · ${preview.feeCount || 0} 笔费用`,
                      },
                      {
                        key: 'revenue',
                        label: '已实现收入',
                        children: `${decimalText(preview.realizedRevenue)} ${preview.baseCurrency}`,
                      },
                      {
                        key: 'cost',
                        label: '分摊成本',
                        children: `${decimalText(preview.allocatedCost)} ${preview.baseCurrency}`,
                      },
                      {
                        key: 'profit',
                        label: '已实现毛利',
                        children: `${decimalText(preview.realizedProfit)} ${preview.baseCurrency}`,
                      },
                      {
                        key: 'amount',
                        label: '提成金额',
                        children: (
                          <Typography.Text strong type="success">
                            {`${decimalText(preview.commissionAmount)} ${preview.baseCurrency}`}
                          </Typography.Text>
                        ),
                      },
                    ]}
                  />
                  <Table<API.FinanceCommissionLine>
                    size="small"
                    bordered
                    pagination={false}
                    rowKey={(line) => line.orderId || line.orderNo || ''}
                    columns={previewColumns}
                    dataSource={preview.lines || []}
                    scroll={{ x: 1080 }}
                    expandable={{
                      expandedRowRender: renderExpandedFees,
                      rowExpandable: (record) =>
                        Boolean(record.fees && record.fees.length > 0),
                    }}
                  />
                </>
              )}
            </Space>
          )}
        </ProFormDependency>
        <Space direction="vertical" size={2} style={{ color: '#666', marginTop: 8 }}>
          <span>
            计算比例、角色与口径均取自已启用且在核销日期生效的考核规则。
          </span>
          <span>亏损订单逐票按 0 计提，但仍保留真实负毛利快照。</span>
          <span>草稿确认时会重新校验客户人员与费用来源；来源变化后必须取消并重新生成。</span>
        </Space>
      </ModalForm>

      {/* 提成明细抽屉 */}
      <Drawer
        title={`提成明细${detail?.commissionNo ? ` · ${detail.commissionNo}` : ''}`}
        width={1120}
        open={detailOpen}
        loading={detailLoading}
        extra={
          detail &&
          ['CONFIRMED', 'PAID'].includes(detail.status || '') &&
          access.canManageFinanceCommissions ? (
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={openAdjustmentModal}
            >
              新增提成调整
            </Button>
          ) : null
        }
        onClose={() => {
          setDetailOpen(false);
          setDetail(undefined);
        }}
      >
        {detail ? (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions
              bordered
              size="small"
              column={4}
              items={[
                {
                  key: 'status',
                  label: '状态',
                  children: (
                    <Tag
                      color={
                        commissionStatusMeta[detail.status || 'DRAFT']?.color
                      }
                    >
                      {commissionStatusMeta[detail.status || 'DRAFT']?.text}
                    </Tag>
                  ),
                },
                {
                  key: 'verification',
                  label: '核销编号',
                  children: detail.verificationNo,
                },
                {
                  key: 'employee',
                  label: '提成员工',
                  children: detail.employeeName,
                },
                {
                  key: 'rule',
                  label: '规则快照',
                  children: `${detail.ruleName || '-'}（v${detail.ruleVersion || 0}）`,
                },
                {
                  key: 'basis',
                  label: '角色/口径',
                  children: `${personnelRoleText(detail.personnelRole)} · ${calculationBasisText(detail.calculationBasis)}`,
                },
                {
                  key: 'rate',
                  label: '比例',
                  children: `${decimalText(detail.ratePercent)}%`,
                },
                {
                  key: 'coverage',
                  label: '业务覆盖',
                  children: `${detail.customerCount ?? 1} 个客户 · ${detail.orderCount ?? 1} 票订单 · ${detail.feeCount ?? 0} 笔费用`,
                },
                {
                  key: 'profit',
                  label: '已实现毛利',
                  children: `${decimalText(detail.realizedProfit)} ${detail.baseCurrency}`,
                },
                {
                  key: 'amount',
                  label: '原始提成',
                  children: (
                    <Typography.Text strong type="success">
                      {`${decimalText(detail.commissionAmount)} ${detail.baseCurrency}`}
                    </Typography.Text>
                  ),
                },
                {
                  key: 'adjustmentAmount',
                  label: '已确认调整',
                  children: `${Number(detail.adjustmentAmount || 0) > 0 ? '+' : ''}${decimalText(detail.adjustmentAmount)} ${detail.baseCurrency}`,
                },
                {
                  key: 'effectiveAmount',
                  label: '有效提成',
                  children: detail.status === 'CANCELLED' ? (
                    <Typography.Text type="secondary">
                      已取消，不计入应发
                    </Typography.Text>
                  ) : (
                    <Typography.Text strong style={{ color: '#1677ff' }}>
                      {`${decimalText(detail.effectiveCommissionAmount || detail.commissionAmount)} ${detail.baseCurrency}`}
                    </Typography.Text>
                  ),
                },
                {
                  key: 'calculationVersion',
                  label: '计算版本',
                  children: detail.calculationVersion || '-',
                },
                {
                  key: 'createdAt',
                  label: '生成时间',
                  children: detail.createdAt
                    ? dayjs(detail.createdAt).format('YYYY-MM-DD HH:mm:ss')
                    : '-',
                },
                {
                  key: 'confirmedAt',
                  label: '确认时间',
                  children: detail.confirmedAt
                    ? dayjs(detail.confirmedAt).format('YYYY-MM-DD HH:mm:ss')
                    : '-',
                },
                {
                  key: 'paidAt',
                  label: '发放时间',
                  children: detail.paidAt
                    ? dayjs(detail.paidAt).format('YYYY-MM-DD HH:mm:ss')
                    : '-',
                },
              ]}
            />
            <Alert
              type="info"
              showIcon
              message="逐订单计算快照与财务锁"
              description="下列数据是生成提成时固化的计算证据。提成确认后，关联订单的原费用会进入财务锁定；支持展开行下钻查看每笔订单的具体费用明细构成。"
            />
            <Table<API.FinanceCommissionLine>
              size="small"
              bordered
              pagination={false}
              rowKey={(line) => line.id || line.orderId || ''}
              columns={previewColumns}
              dataSource={detail.lines || []}
              scroll={{ x: 1080 }}
              expandable={{
                expandedRowRender: renderExpandedFees,
                rowExpandable: (record) =>
                  Boolean(record.fees && record.fees.length > 0),
              }}
            />
            <Space style={{ width: '100%', justifyContent: 'space-between' }}>
              <Typography.Title level={5} style={{ margin: 0 }}>
                提成调整记录
              </Typography.Title>
              <Typography.Text type="secondary">
                草稿不计入有效提成，确认后计入；已发放或已扣回调整不可取消。
              </Typography.Text>
            </Space>
            <Table<API.FinanceCommissionAdjustment>
              size="small"
              bordered
              pagination={false}
              rowKey={(item) => item.id || item.adjustmentNo || ''}
              columns={adjustmentColumns}
              dataSource={detail.adjustments || []}
              scroll={{ x: 1040 }}
              locale={{ emptyText: '暂无调整，当前有效提成等于原始提成' }}
            />
          </Space>
        ) : null}
      </Drawer>

      {/* 新增调整弹窗 */}
      <ModalForm<AdjustmentValues>
        title={`新增提成调整${detail?.commissionNo ? ` · ${detail.commissionNo}` : ''}`}
        open={adjustmentOpen}
        width={620}
        initialValues={{ direction: 'INCREASE' }}
        modalProps={{
          destroyOnHidden: true,
          onCancel: () => setAdjustmentOpen(false),
        }}
        onFinish={async (values) => {
          if (!detail?.id) return false;
          try {
            await settlementServiceCreateCommissionAdjustment(
              { commissionId: detail.id },
              {
                commissionId: detail.id,
                orderId: values.orderId,
                direction: values.direction,
                amount: String(values.amount),
                reason: values.reason,
                note: values.note,
                idempotencyKey: adjustmentIdempotencyKey,
              },
            );
            message.success('提成调整草稿已创建');
            setAdjustmentOpen(false);
            await refreshDetail();
            return true;
          } catch (error: any) {
            message.error(error.message || '提成调整创建失败');
            return false;
          }
        }}
      >
        <Alert
          type="warning"
          showIcon
          message="原始提成不会被修改"
          description="请选择产生差异的具体订单。增提或冲减会形成独立编号，并保留确认、发放/扣回和取消轨迹。冲减金额不能使有效提成小于零。"
          style={{ marginBottom: 16 }}
        />
        <ProFormSearchableSelect
          name="orderId"
          label="归属订单"
          rules={[{ required: true, message: '请选择调整归属订单' }]}
          options={(detail?.lines || []).map((line) => ({
            label: `${line.orderNo}｜原始提成 ${decimalText(line.commissionAmount)} ${line.baseCurrency}`,
            value: line.orderId,
          }))}
        />
        <ProFormSearchableSelect
          name="direction"
          label="调整方向"
          rules={[{ required: true }]}
          options={[
            { label: '增提（增加应发提成）', value: 'INCREASE' },
            { label: '冲减（减少应发提成）', value: 'DECREASE' },
          ]}
        />
        <ProFormDigit
          name="amount"
          label={`调整金额（${detail?.baseCurrency || ''}）`}
          min={0.00000001}
          fieldProps={{
            precision: 8,
            stringMode: true,
          }}
          rules={[{ required: true, message: '请输入大于 0 的调整金额' }]}
        />
        <ProFormTextArea
          name="reason"
          label="调整原因"
          fieldProps={{ maxLength: 500, showCount: true }}
          rules={[{ required: true, message: '请输入调整原因' }]}
        />
        <ProFormTextArea
          name="note"
          label="补充备注"
          fieldProps={{ maxLength: 500, showCount: true }}
        />
      </ModalForm>

      {/* 考核规则抽屉 */}
      <Drawer
        title="提成考核规则"
        width={980}
        open={rulesOpen}
        onClose={() => setRulesOpen(false)}
      >
        <ProTable<API.FinanceCommissionRule>
          actionRef={ruleActionRef}
          rowKey="id"
          columns={ruleColumns}
          bordered
          size="small"
          search={{ defaultCollapsed: false }}
          toolBarRender={() =>
            access.canManageFinanceCommissions
              ? [
                  <Button
                    key="new-rule"
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={() => {
                      setEditingRule(undefined);
                      setRuleFormOpen(true);
                    }}
                  >
                    新建规则
                  </Button>,
                ]
              : []
          }
          request={async (params) => {
            const response = await settlementServiceListCommissionRules({
              page: params.current ?? 1,
              pageSize: params.pageSize ?? 20,
              keyword: params.name,
              personnelRole: params.personnelRole,
              enabled: params.enabled,
            });
            return {
              data: response.data || [],
              total: Number(response.total || 0),
              success: response.success ?? true,
            };
          }}
        />
      </Drawer>

      {/* 新建/编辑规则弹窗 */}
      <ModalForm<RuleValues>
        key={editingRule?.id || 'new-rule'}
        title={editingRule ? '编辑考核规则' : '新建考核规则'}
        open={ruleFormOpen}
        width={620}
        modalProps={{
          destroyOnHidden: true,
          onCancel: () => setRuleFormOpen(false),
        }}
        initialValues={{
          name: editingRule?.name,
          personnelRole: editingRule?.personnelRole || 'SALES',
          calculationBasis: editingRule?.calculationBasis || 'REALIZED_PROFIT',
          ratePercent: editingRule
            ? Number(editingRule.ratePercent)
            : undefined,
          effectiveRange:
            editingRule?.effectiveFrom && editingRule?.effectiveTo
              ? [
                  dayjs(editingRule.effectiveFrom),
                  dayjs(editingRule.effectiveTo),
                ]
              : undefined,
          enabled: editingRule?.enabled ?? true,
          note: editingRule?.note,
        }}
        onFinish={async (values) => {
          const rule = {
            name: values.name,
            personnelRole: values.personnelRole,
            calculationBasis: values.calculationBasis,
            ratePercent: String(values.ratePercent),
            effectiveFrom: values.effectiveRange?.[0]?.format('YYYY-MM-DD'),
            effectiveTo: values.effectiveRange?.[1]?.format('YYYY-MM-DD'),
            enabled: values.enabled,
            note: values.note,
          };
          try {
            if (editingRule?.id && editingRule.version) {
              await settlementServiceUpdateCommissionRule(
                { id: editingRule.id },
                {
                  id: editingRule.id,
                  expectedVersion: editingRule.version,
                  rule,
                },
              );
              message.success('考核规则已更新');
            } else {
              await settlementServiceCreateCommissionRule({ rule });
              message.success('考核规则已创建');
            }
            setRuleFormOpen(false);
            ruleActionRef.current?.reload();
            return true;
          } catch (error: any) {
            const reason = getBusinessReason(error);
            if (reason === 'FINANCE_COMMISSION_RULE_CONFLICT') {
              modal.warning({
                title: '规则名称冲突或版本已变化',
                content: '请检查规则名称是否重复，或刷新后重试。',
              });
              return false;
            }
            message.error(error.message || '保存规则失败');
            return false;
          }
        }}
      >
        <ProFormText
          name="name"
          label="规则名称"
          rules={[{ required: true, message: '请输入规则名称' }]}
        />
        <ProFormSearchableSelect
          name="personnelRole"
          label="适用客户人员角色"
          rules={[{ required: true, message: '请选择适用角色' }]}
          options={[
            { value: 'SALES', label: '业务人员' },
            { value: 'OPERATOR', label: '操作人员' },
            { value: 'CUSTOMER_SERVICE', label: '客服人员' },
          ]}
        />
        <ProFormSearchableSelect
          name="calculationBasis"
          label="计提口径"
          rules={[{ required: true, message: '请选择计提口径' }]}
          options={[
            { value: 'REALIZED_PROFIT', label: '已实现毛利（推荐，按收入配比分摊成本）' },
            { value: 'REALIZED_REVENUE', label: '已实现收入（按核销收入全额）' },
          ]}
        />
        <ProFormDigit
          name="ratePercent"
          label="提成比例（%）"
          min={0.0001}
          max={100}
          fieldProps={{ precision: 4 }}
          rules={[{ required: true, message: '请输入提成比例' }]}
        />
        <ProFormDateRangePicker name="effectiveRange" label="生效区间" />
        <ProFormSwitch name="enabled" label="启用" />
        <ProFormTextArea
          name="note"
          label="规则说明"
          fieldProps={{ maxLength: 500 }}
        />
      </ModalForm>
    </>
  );
}
