import {
  CheckOutlined,
  CloseCircleOutlined,
  DollarOutlined,
  EyeOutlined,
  PlusOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Button, Space, Tag, Typography } from 'antd';
import React, { useRef, useState } from 'react';
import { SearchFilterTemplate } from '@/components/ui';
import { financeErrorReasons } from '@/errorReasons.generated';
import {
  settlementServiceCancelCommission,
  settlementServiceCancelCommissionAdjustment,
  settlementServiceConfirmCommission,
  settlementServiceConfirmCommissionAdjustment,
  settlementServiceGetCommission,
  settlementServiceListCommissions,
  settlementServiceMarkCommissionAdjustmentPaid,
  settlementServiceMarkCommissionPaid,
} from '@/services/roncin/settlementService';
import { confirmWithReason } from '@/utils/confirmWithReason';
import CommissionAdjustmentModal from './components/CommissionAdjustmentModal';
import CommissionCreateModal from './components/CommissionCreateModal';
import CommissionDetailDrawer from './components/CommissionDetailDrawer';
import CommissionRulesDrawer from './components/CommissionRulesDrawer';
import {
  calculationBasisMeta,
  calculationBasisText,
  commissionStatusMeta,
  decimalText,
  getBusinessReason,
  personnelRoleMeta,
  personnelRoleText,
} from './types';

export default function FinanceCommissionsPage() {
  const access = useAccess();
  const { message, modal } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);

  const [searchParams, setSearchParams] = useState<{
    keyword?: string;
    status?: string;
  }>({});
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [rulesDrawerOpen, setRulesDrawerOpen] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detail, setDetail] = useState<API.FinanceCommission>();
  const [adjustmentModalOpen, setAdjustmentModalOpen] = useState(false);

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
    let content =
      '确认后该增减金额会计入有效提成；冲减不会被允许把有效提成降到零以下。';
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
          if (
            reason === financeErrorReasons.FINANCE_COMMISSION_ADJUSTMENT_EXCEEDS
          ) {
            modal.warning({
              title: '冲减金额超限',
              content: '冲减后的有效提成不能小于零，请检查调整金额。',
            });
            return;
          }
          if (
            reason ===
            financeErrorReasons.FINANCE_COMMISSION_ADJUSTMENT_TRANSITION
          ) {
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
    confirmWithReason(
      { modal, message },
      `取消调整 ${record.adjustmentNo}？`,
      async (reason) => {
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
      {
        placeholder: '请输入取消原因（必填）',
        requiredMessage: '请输入取消原因',
      },
    );
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
          if (
            reason === financeErrorReasons.FINANCE_COMMISSION_SOURCE_CHANGED
          ) {
            modal.warning({
              title: '提成来源已经变化',
              content:
                '请取消当前草稿，然后根据最新核销、费用和人员归属重新生成。',
            });
            return;
          }
          if (
            reason === financeErrorReasons.FINANCE_COMMISSION_UNCONFIRMED_FEES
          ) {
            modal.warning({
              title: '关联订单存在草稿费用',
              content: '请先确认或作废草稿费用，再确认提成。',
            });
            return;
          }
          if (reason === financeErrorReasons.FINANCE_COMMISSION_TRANSITION) {
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
    confirmWithReason(
      { modal, message },
      `取消提成 ${record.commissionNo}？`,
      async (reason) => {
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
      {
        placeholder: '请输入取消原因（必填）',
        requiredMessage: '请输入取消原因',
      },
    );
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
      render: (_, record) => record.verificationNo || '-',
    },
    {
      title: '提成员工',
      dataIndex: 'employeeName',
      width: 120,
      search: false,
    },
    {
      title: '考核角色',
      dataIndex: 'personnelRole',
      width: 110,
      valueType: 'select',
      valueEnum: personnelRoleMeta,
      renderText: (value) => personnelRoleText(value),
    },
    {
      title: '规则名称',
      dataIndex: 'ruleName',
      width: 160,
      search: false,
      render: (_, record) => (
        <Space size={4}>
          <span>{record.ruleName || '-'}</span>
          <Tag style={{ fontSize: 11 }}>{`v${record.ruleVersion || 0}`}</Tag>
        </Space>
      ),
    },
    {
      title: '计提口径',
      dataIndex: 'calculationBasis',
      width: 110,
      valueType: 'select',
      valueEnum: calculationBasisMeta,
      renderText: (value) => calculationBasisText(value),
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
            <Space vertical size={0}>
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
          <Space vertical size={0}>
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

  return (
    <>
      <SearchFilterTemplate
        layout="bar"
        keywordPlaceholder="搜索提成单号、员工名称或业务单号..."
        quickFilters={[
          {
            name: 'status',
            placeholder: '全部状态',
            width: 120,
            options: [
              { label: '草稿', value: 'DRAFT' },
              { label: '已确认', value: 'CONFIRMED' },
              { label: '已发放', value: 'PAID' },
              { label: '已取消', value: 'CANCELLED' },
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
                onClick={() => setCreateModalOpen(true)}
              >
                生成提成
              </Button>
            )}
            <Button
              key="rules"
              icon={<SettingOutlined />}
              onClick={() => setRulesDrawerOpen(true)}
            >
              {access.canManageFinanceCommissions ? '考核规则' : '查看规则'}
            </Button>
          </Space>
        }
      />
      <ProTable<API.FinanceCommission>
        headerTitle="提成结算列表"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        bordered
        size="small"
        scroll={{ x: 1600 }}
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

      <CommissionCreateModal
        open={createModalOpen}
        onOpenChange={setCreateModalOpen}
        onSuccess={reload}
      />

      <CommissionDetailDrawer
        open={detailOpen}
        onClose={() => {
          setDetailOpen(false);
          setDetail(undefined);
        }}
        detail={detail}
        loading={detailLoading}
        canManage={access.canManageFinanceCommissions}
        onOpenAdjustment={() => setAdjustmentModalOpen(true)}
        onTransitionAdjustment={transitionAdjustment}
        onCancelAdjustment={cancelAdjustment}
      />

      <CommissionAdjustmentModal
        open={adjustmentModalOpen}
        onOpenChange={setAdjustmentModalOpen}
        detail={detail}
        onSuccess={refreshDetail}
      />

      <CommissionRulesDrawer
        open={rulesDrawerOpen}
        onClose={() => setRulesDrawerOpen(false)}
        canManage={access.canManageFinanceCommissions}
      />
    </>
  );
}
