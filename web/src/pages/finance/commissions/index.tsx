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
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
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
  amount: number;
  reason: string;
  note?: string;
};

const statusMeta: Record<string, { text: string; color: string }> = {
  DRAFT: { text: '草稿', color: 'processing' },
  CONFIRMED: { text: '已确认', color: 'success' },
  PAID: { text: '已发放', color: 'blue' },
  CANCELLED: { text: '已取消', color: 'default' },
};

const calculationBasisText = (value?: string) =>
  value === 'REALIZED_REVENUE' ? '已实现收入' : '已实现毛利';

const personnelRoleText = (value?: string) => {
  if (value === 'OPERATOR') return '操作';
  if (value === 'CUSTOMER_SERVICE') return '客服';
  return '销售';
};

const decimalText = (value?: string) => {
  if (!value) return '0';
  return value.replace(/(\.\d*?[1-9])0+$|\.0+$/, '$1');
};

const calculationSignature = (values: Partial<CreateValues>) =>
  [values.verificationId, values.ruleId, values.employeeId].join('|');

export default function FinanceCommissionsPage() {
  const access = useAccess();
  const { message, modal } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const ruleActionRef = useRef<ActionType | undefined>(undefined);
  const createFormRef = useRef<ProFormInstance<CreateValues>>(undefined);
  const [open, setOpen] = useState(false);
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
  const reload = () => actionRef.current?.reload();

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
    const response = await settlementServiceGetCommission({ id: detail.id });
    setDetail(response.data);
    reload();
  };

  const transitionAdjustment = (
    record: API.FinanceCommissionAdjustment,
    target: 'CONFIRMED' | 'PAID',
  ) => {
    if (!record.id || !record.version) return;
    const adjustmentID = record.id;
    const adjustmentVersion = record.version;
    const action = target === 'CONFIRMED' ? '确认调整' : '标记调整已发放';
    modal.confirm({
      title: `${action} ${record.adjustmentNo}？`,
      content:
        target === 'CONFIRMED'
          ? '确认后该增减金额会计入有效提成；冲减不会被允许把有效提成降到零以下。'
          : '该操作表示本笔调整已实际发放或扣回，完成后不可取消。',
      onOk: async () => {
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
        await settlementServiceCancelCommissionAdjustment(
          { id: adjustmentID },
          { id: adjustmentID, expectedVersion: adjustmentVersion, reason },
        );
        message.success('调整已取消');
        await refreshDetail();
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
          ? '系统会重新校验核销、账单费用、提成规则和客户人员归属；来源发生变化时将拒绝确认。'
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
      width: 150,
      search: false,
      renderText: (_, record) =>
        record.personnelRole
          ? `${personnelRoleText(record.personnelRole)} · ${calculationBasisText(record.calculationBasis)}`
          : '-',
    },
    {
      title: '业务覆盖',
      width: 140,
      search: false,
      render: (_, record) => (
        <Space size={4}>
          <Tag color="blue">{record.customerCount || 1} 客户</Tag>
          <Tag color="cyan">{record.orderCount || 1} 订单</Tag>
        </Space>
      ),
    },
    {
      title: '已实现收入',
      dataIndex: 'realizedRevenue',
      width: 140,
      align: 'right',
      search: false,
      renderText: (value, record) => `${decimalText(value)} ${record.baseCurrency}`,
    },
    {
      title: '分摊成本',
      dataIndex: 'allocatedCost',
      width: 140,
      align: 'right',
      search: false,
      renderText: (value, record) => `${decimalText(value)} ${record.baseCurrency}`,
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
      width: 170,
      align: 'right',
      search: false,
      render: (_, record) => (
        <Space direction="vertical" size={0}>
          <span>{`${decimalText(record.commissionAmount)} ${record.baseCurrency}`}</span>
          <strong style={{ color: '#1677ff' }}>
            {`有效 ${decimalText(record.effectiveCommissionAmount || record.commissionAmount)} ${record.baseCurrency}`}
          </strong>
        </Space>
      ),
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
    { title: '调整编号', dataIndex: 'adjustmentNo', key: 'adjustmentNo', width: 190 },
    { title: '订单编号', dataIndex: 'orderNo', key: 'orderNo', width: 180 },
    {
      title: '方向',
      dataIndex: 'direction',
      key: 'direction',
      width: 90,
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
        <Typography.Text type={record.direction === 'DECREASE' ? 'danger' : 'success'} strong>
          {`${record.direction === 'DECREASE' ? '-' : '+'}${decimalText(value)} ${record.baseCurrency}`}
        </Typography.Text>
      ),
    },
    { title: '调整原因', dataIndex: 'reason', key: 'reason', ellipsis: true },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (value: string) => {
        const meta = statusMeta[value] || statusMeta.DRAFT;
        return <Tag color={meta.color}>{meta.text}</Tag>;
      },
    },
    {
      title: '操作',
      key: 'action',
      width: 180,
      render: (_: unknown, record: API.FinanceCommissionAdjustment) => (
        <Space size={8}>
          {record.status === 'DRAFT' && access.canManageFinanceCommissions && (
            <a onClick={() => transitionAdjustment(record, 'CONFIRMED')}>确认</a>
          )}
          {record.status === 'CONFIRMED' && access.canManageFinanceCommissions && (
            <a onClick={() => transitionAdjustment(record, 'PAID')}>已发放</a>
          )}
          {['DRAFT', 'CONFIRMED'].includes(record.status || '') &&
            access.canManageFinanceCommissions && (
              <a style={{ color: '#ff4d4f' }} onClick={() => cancelAdjustment(record)}>
                取消
              </a>
            )}
        </Space>
      ),
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
        SALES: { text: '客户销售' },
        OPERATOR: { text: '客户操作' },
        CUSTOMER_SERVICE: { text: '客户客服' },
      },
      renderText: (value) => personnelRoleText(value),
    },
    {
      title: '计算口径',
      dataIndex: 'calculationBasis',
      width: 130,
      search: false,
      renderText: (value) =>
        value === 'REALIZED_PROFIT' ? '已实现毛利' : '已实现收入',
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

  return (
    <>
      <ProTable<API.FinanceCommission>
        headerTitle="提成管理"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        bordered
        size="small"
        scroll={{ x: 2150 }}
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
                <Button
                  key="rules"
                  icon={<SettingOutlined />}
                  onClick={() => setRulesOpen(true)}
                >
                  考核规则
                </Button>,
              ]
            : [
                <Button
                  key="rules"
                  icon={<SettingOutlined />}
                  onClick={() => setRulesOpen(true)}
                >
                  查看规则
                </Button>,
              ]
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
              idempotencyKey: globalThis.crypto.randomUUID(),
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
        <ProFormSelect
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
        <ProFormSelect
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
              label: `${item.name}｜${personnelRoleText(item.personnelRole)}｜${item.calculationBasis === 'REALIZED_PROFIT' ? '毛利' : '收入'} × ${decimalText(item.ratePercent)}%`,
              value: item.id,
            }));
          }}
        />
        <ProFormDependency name={['verificationId', 'ruleId']}>
          {({ verificationId, ruleId }) => (
            <ProFormSelect
              key={`${verificationId || ''}-${ruleId || ''}`}
              name="employeeId"
              label="符合规则的客户归属人员"
              rules={[{ required: true, message: '请选择符合角色的客户归属人员' }]}
              disabled={!verificationId || !ruleId}
              request={async () => {
                if (!verificationId || !ruleId) return [];
                const response =
                  await settlementServiceListCommissionCandidates({
                    verificationId,
                    ruleId,
                  });
                return (response.data || []).map((item) => ({
                  label: item.displayName,
                  value: item.id,
                }));
              }}
              extra="人员来自本次核销涉及订单的客户档案人员（按客户销售/指定角色归属匹配）。"
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
                      rowExpandable: (record) => Boolean(record.fees && record.fees.length > 0),
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
              onClick={() => setAdjustmentOpen(true)}
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
                    <Tag color={statusMeta[detail.status || 'DRAFT']?.color}>
                      {statusMeta[detail.status || 'DRAFT']?.text}
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
                  children: `${detail.customerCount || 1} 个客户 · ${detail.orderCount || 1} 票订单 · ${detail.feeCount || 0} 笔费用`,
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
                  children: (
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
                rowExpandable: (record) => Boolean(record.fees && record.fees.length > 0),
              }}
            />
            <Space style={{ width: '100%', justifyContent: 'space-between' }}>
              <Typography.Title level={5} style={{ margin: 0 }}>
                提成调整记录
              </Typography.Title>
              <Typography.Text type="secondary">
                草稿不计入有效提成，确认后计入；已发放调整不可取消。
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
                idempotencyKey: globalThis.crypto.randomUUID(),
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
          description="请选择产生差异的具体订单。增提或冲减会形成独立编号，并保留确认、发放和取消轨迹。"
          style={{ marginBottom: 16 }}
        />
        <ProFormSelect
          name="orderId"
          label="归属订单"
          rules={[{ required: true, message: '请选择调整归属订单' }]}
          options={(detail?.lines || []).map((line) => ({
            label: `${line.orderNo}｜原始提成 ${decimalText(line.commissionAmount)} ${line.baseCurrency}`,
            value: line.orderId,
          }))}
        />
        <ProFormSelect
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
          fieldProps={{ precision: 8, stringMode: false }}
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
              page: params.current,
              pageSize: params.pageSize,
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
        }}
      >
        <ProFormText
          name="name"
          label="规则名称"
          rules={[{ required: true, message: '请输入规则名称' }]}
        />
        <ProFormSelect
          name="personnelRole"
          label="适用客户人员角色"
          rules={[{ required: true, message: '请选择适用角色' }]}
          options={[
            { value: 'SALES', label: '客户销售人员' },
            { value: 'OPERATOR', label: '客户操作人员' },
            { value: 'CUSTOMER_SERVICE', label: '客户客服人员' },
          ]}
        />
        <ProFormSelect
          name="calculationBasis"
          label="计提口径"
          rules={[{ required: true, message: '请选择计提口径' }]}
          options={[
            { value: 'REALIZED_PROFIT', label: '已实现毛利（收入配比分摊成本）' },
            { value: 'REALIZED_REVENUE', label: '已实现收入（回款全额）' },
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
