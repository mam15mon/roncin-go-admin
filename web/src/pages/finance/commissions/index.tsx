import {
  CheckOutlined,
  CloseCircleOutlined,
  DollarOutlined,
  DownloadOutlined,
  EyeOutlined,
  PlusOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { PageContainer, ProTable } from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import {
  App,
  Button,
  DatePicker,
  Segmented,
  Space,
  Tag,
  Typography,
} from 'antd';
import React, { useEffect, useRef, useState } from 'react';
import { SearchFilterTemplate } from '@/components/ui';
import {
  FinanceCommissionStatus,
  FinanceOrganizationPurpose,
} from '@/enums.generated';
import { financeErrorReasons } from '@/errorReasons.generated';
import {
  settlementServiceCancelCommission,
  settlementServiceCancelCommissionAdjustment,
  settlementServiceConfirmCommission,
  settlementServiceConfirmCommissionAdjustment,
  settlementServiceExportCommissions,
  settlementServiceGetCommission,
  settlementServiceListCommissions,
  settlementServiceListFinanceOrganizationOptions,
  settlementServiceMarkCommissionAdjustmentPaid,
  settlementServiceMarkCommissionPaid,
} from '@/services/roncin/settlementService';
import { toTableRequest } from '@/utils/api';
import { makeVersionActions } from '@/utils/versionActions';
import {
  buildCommissionExportFileName,
  type CommissionQueryFilters,
  type CommissionSearchValues,
  normalizeCommissionFilters,
  serializeCommissionCsv,
} from './commissionExport';
import CommissionAdjustmentModal from './components/CommissionAdjustmentModal';
import CommissionApplicationsPanel from './components/CommissionApplicationsPanel';
import CommissionCreateModal from './components/CommissionCreateModal';
import CommissionDetailDrawer from './components/CommissionDetailDrawer';
import CommissionRulesDrawer from './components/CommissionRulesDrawer';
import PendingDecreasePanel from './components/PendingDecreasePanel';
import {
  calculationBasisMeta,
  calculationBasisText,
  commissionSourceNo,
  commissionStatusMeta,
  decimalText,
  getBusinessReason,
  isReversalAdjustment,
  personnelRoleMeta,
  personnelRoleText,
} from './types';

const { RangePicker } = DatePicker;

type CommissionView = 'ledger' | 'pending-decrease' | 'applications';

export default function FinanceCommissionsPage() {
  const access = useAccess();
  const { message, modal } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [view, setView] = useState<CommissionView>('ledger');

  const searchFiltersRef = useRef<CommissionQueryFilters>({});
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [rulesDrawerOpen, setRulesDrawerOpen] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detail, setDetail] = useState<API.FinanceCommission>();
  const [adjustmentModalOpen, setAdjustmentModalOpen] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [organizationOptions, setOrganizationOptions] = useState<
    API.FinanceOrganizationOption[]
  >([]);

  useEffect(() => {
    void settlementServiceListFinanceOrganizationOptions({
      purpose:
        FinanceOrganizationPurpose.FINANCE_ORGANIZATION_PURPOSE_COMMISSION_READ,
    }).then((response) => setOrganizationOptions(response.data ?? []));
  }, []);

  const reload = () => actionRef.current?.reload();

  const exportCommissions = async () => {
    try {
      setExporting(true);
      const filters = searchFiltersRef.current;
      const response = await settlementServiceExportCommissions(filters);
      const content = serializeCommissionCsv(response.data ?? []);
      if (!content) {
        message.warning('当前筛选条件下没有可导出的提成');
        return;
      }

      const url = URL.createObjectURL(
        new Blob([content], { type: 'text/csv;charset=utf-8' }),
      );
      const link = document.createElement('a');
      link.href = url;
      link.download = buildCommissionExportFileName(filters);
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
      message.success(`成功导出 ${response.data?.length ?? 0} 条提成`);
    } catch (error: any) {
      message.error(error.message || '提成导出失败');
    } finally {
      setExporting(false);
    }
  };
  const commissionActions = makeVersionActions<API.FinanceCommission>({
    modal,
    message,
  });
  const adjustmentActions = makeVersionActions<API.FinanceCommissionAdjustment>(
    {
      modal,
      message,
    },
  );

  const openDetailById = async (commissionId?: string) => {
    if (!commissionId) return;
    setDetailOpen(true);
    setDetailLoading(true);
    setDetail(undefined);
    try {
      const response = await settlementServiceGetCommission({
        id: commissionId,
      });
      setDetail(response.data);
    } catch (error: any) {
      message.error(error.message || '提成明细加载失败');
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  };

  const openDetail = (record: API.FinanceCommission) =>
    openDetailById(record.id);

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
    const isReversal = isReversalAdjustment(record.sourceType);
    const isDecrease = record.direction === 'DECREASE';

    let action = '确认调整';
    let content =
      '确认后该增减金额会计入有效提成；冲减不会被允许把有效提成降到零以下。';
    if (target === 'PAID') {
      if (isReversal) {
        action = '标记已追回';
        content =
          '该操作表示来源反转（反核销或反对冲）冲减款项已实际追回，完成后不可取消。';
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
      content: `所属公司：${record.organizationName || '所属公司未标识'}。${content}`,
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
    adjustmentActions.confirm(
      record,
      `取消 ${record.organizationName || '所属公司未标识'} 的调整 ${record.adjustmentNo}？`,
      async ({ id, expectedVersion }, reason) => {
        try {
          await settlementServiceCancelCommissionAdjustment(
            { id },
            { id, expectedVersion, reason },
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
      title: `${action} ${record.organizationName || '所属公司未标识'} 的提成 ${record.commissionNo}？`,
      content:
        target === 'CONFIRMED'
          ? '系统会重新核对来源单（核销或对冲）、账单费用、提成规则和客户人员归属；来源发生变化时将拒绝确认。'
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
                '请取消当前草稿，然后根据最新来源单、费用和人员归属重新生成。',
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
    commissionActions.confirm(
      record,
      `取消 ${record.organizationName || '所属公司未标识'} 的提成 ${record.commissionNo}？`,
      async ({ id, expectedVersion }, reason) => {
        try {
          await settlementServiceCancelCommission(
            { id },
            { id, expectedVersion, reason },
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
      title: '所属公司',
      dataIndex: 'organizationName',
      width: 150,
      search: false,
      renderText: (value) => value || '-',
    },
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
        const meta =
          commissionStatusMeta[
            record.status ??
              FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT
          ];
        return <Tag color={meta.color}>{meta.text}</Tag>;
      },
    },
    {
      title: '来源单号',
      dataIndex: 'verificationNo',
      width: 170,
      copyable: true,
      search: false,
      render: (_, record) => commissionSourceNo(record),
    },
    {
      title: '归属日期',
      dataIndex: 'commissionDate',
      width: 110,
      search: false,
      renderText: (value) => value || '-',
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
        if (
          record.status ===
          FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CANCELLED
        ) {
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
      title: 'CNY 提成',
      dataIndex: 'cnyEffectiveCommissionAmount',
      width: 180,
      align: 'right',
      search: false,
      render: (_, record) => {
        if (
          record.status ===
          FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CANCELLED
        ) {
          return (
            <Space vertical size={0}>
              <Typography.Text type="secondary" delete>
                {`有效快照 ${decimalText(record.cnyEffectiveCommissionAmount)} CNY`}
              </Typography.Text>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {`原始 ${decimalText(record.cnyCommissionAmount)} · 调整 ${decimalText(record.cnyAdjustmentAmount)}，不计入应发`}
              </Typography.Text>
            </Space>
          );
        }
        return (
          <Space vertical size={0}>
            <strong style={{ color: '#1677ff' }}>
              {`有效 ${decimalText(record.cnyEffectiveCommissionAmount)} CNY`}
            </strong>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {`原始 ${decimalText(record.cnyCommissionAmount)} · 调整 ${decimalText(record.cnyAdjustmentAmount)}`}
            </Typography.Text>
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
          record.status ===
          FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT ? (
            access.canManageFinanceCommissions ? (
              <a key="confirm" onClick={() => transition(record, 'CONFIRMED')}>
                <CheckOutlined /> 确认
              </a>
            ) : null
          ) : null,
          record.status ===
            FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED &&
          access.canManageFinanceCommissions ? (
            <a key="paid" onClick={() => transition(record, 'PAID')}>
              <DollarOutlined /> 已发放
            </a>
          ) : null,
          (record.status ===
            FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT ||
            record.status ===
              FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED) &&
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
    <PageContainer
      title="提成管理"
      subTitle="业务人员业绩提成核算、规则配置与发放台账"
      extra={[
        <Segmented
          key="commission-view"
          value={view}
          onChange={(value) => setView(value as CommissionView)}
          options={[
            { label: '提成台账', value: 'ledger' },
            { label: '月度申请', value: 'applications' },
            { label: '待处理冲减', value: 'pending-decrease' },
          ]}
        />,
      ]}
      style={{ minHeight: '100vh', backgroundColor: '#f5f7fa' }}
    >
      {view === 'pending-decrease' ? (
        <PendingDecreasePanel
          onOpenCommissionDetail={(commissionId) =>
            void openDetailById(commissionId)
          }
        />
      ) : view === 'applications' ? (
        <CommissionApplicationsPanel />
      ) : (
        <>
          <SearchFilterTemplate<CommissionSearchValues>
            layout="grid"
            collapsible={false}
            colSpan={6}
            items={[
              {
                name: 'keyword',
                label: '关键词',
                placeholder: '提成单号、员工名称或业务单号',
                span: 6,
              },
              {
                name: 'status',
                label: '状态',
                type: 'select',
                placeholder: '全部状态',
                span: 4,
                options: [
                  {
                    label: '草稿',
                    value:
                      FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT,
                  },
                  {
                    label: '已确认',
                    value:
                      FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED,
                  },
                  {
                    label: '已发放',
                    value:
                      FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_PAID,
                  },
                  {
                    label: '已取消',
                    value:
                      FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CANCELLED,
                  },
                ],
              },
              {
                name: 'commissionMonth',
                label: '归属月份',
                type: 'custom',
                span: 8,
                render: () => (
                  <RangePicker
                    picker="month"
                    allowEmpty={[true, true]}
                    placeholder={['开始月份', '结束月份']}
                    style={{ width: '100%' }}
                  />
                ),
              },
              {
                name: 'organizationId',
                label: '所属公司',
                type: 'select',
                placeholder: '全部公司',
                span: 4,
                options: organizationOptions.map((item) => ({
                  value: item.id ?? '',
                  label: item.name ?? item.code ?? item.id ?? '',
                })),
              },
            ]}
            onSearch={(values) => {
              searchFiltersRef.current = normalizeCommissionFilters(values);
              actionRef.current?.reload();
            }}
            onReset={() => {
              searchFiltersRef.current = {};
              actionRef.current?.reload();
            }}
            extraRight={
              <Space size={8}>
                {access.canExportFinanceCommissions && (
                  <Button
                    key="export"
                    icon={<DownloadOutlined />}
                    loading={exporting}
                    onClick={exportCommissions}
                  >
                    导出提成
                  </Button>
                )}
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
            cardProps={{
              style: {
                borderRadius: 8,
                border: '1px solid #f0f0f0',
              },
            }}
            size="small"
            scroll={{ x: 1900 }}
            search={false}
            toolBarRender={false}
            request={async (params) => {
              const response = await settlementServiceListCommissions({
                page: params.current ?? 1,
                pageSize: params.pageSize ?? 20,
                ...searchFiltersRef.current,
              });
              return toTableRequest(response);
            }}
          />

          <CommissionCreateModal
            open={createModalOpen}
            onOpenChange={setCreateModalOpen}
            onSuccess={reload}
          />
        </>
      )}

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
    </PageContainer>
  );
}
