import { PlusOutlined } from '@ant-design/icons';
import type { ActionType } from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Form, Select } from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { BusinessTagModal } from '@/components/business-tag/BusinessTagModal';
import {
  type FinanceLedgerMetricCard,
  FinanceLedgerTemplate,
  type SearchFilterFieldItem,
  SearchFilterTemplate,
} from '@/components/ui';
import {
  FinanceBillStatus,
  FinanceOrganizationPurpose,
} from '@/enums.generated';
import {
  settlementServiceBatchAssignFinanceBillTags,
  settlementServiceBatchRemoveFinanceBillTags,
  settlementServiceCancelBill,
  settlementServiceConfirmBill,
  settlementServiceGetBill,
  settlementServiceListBills,
  settlementServiceListFinanceBillTagAssignmentOptions,
  settlementServiceListFinanceBillTagOptions,
  settlementServiceListFinanceOrganizationOptions,
  settlementServiceUpdateBill,
} from '@/services/roncin/settlementService';
import { toTableRequest, unwrapPage } from '@/utils/api';
import { getCurrencyOptions, searchPartnerOptions } from '@/utils/options';
import { makeVersionActions } from '@/utils/versionActions';
import BillCreationWorkbench from './components/BillCreationWorkbench';
import BillDetailDrawer from './components/BillDetailDrawer';
import BillEditModal from './components/BillEditModal';
import { getFinanceBillColumns } from './components/billColumns';
import type { BillFormValues } from './components/billConstants';

export default function FinanceBillsPage() {
  const access = useAccess();
  const [tagModalOpen, setTagModalOpen] = useState(false);
  const [tagBillIds, setTagBillIds] = useState<string[]>([]);
  const [tagExisting, setTagExisting] = useState<API.BusinessTagSummary[]>([]);
  const [tagOptions, setTagOptions] = useState<
    { label: string; value: string }[]
  >([]);
  const [tagOptionsLoading, setTagOptionsLoading] = useState(false);
  const [tagFilterIds, setTagFilterIds] = useState<string[]>();
  const [organizationId, setOrganizationId] = useState<string>();
  const [organizationOptions, setOrganizationOptions] = useState<
    API.FinanceOrganizationOption[]
  >([]);
  const tagFilterRequestRef = useRef(0);

  const loadTagFilterOptions = useCallback(
    async (keyword?: string, selectedIds: string[] = []) => {
      if (!organizationId) {
        setTagOptions([]);
        return;
      }
      const requestSequence = ++tagFilterRequestRef.current;
      setTagOptionsLoading(true);
      try {
        const response = await settlementServiceListFinanceBillTagOptions({
          page: 1,
          pageSize: 50,
          keyword: keyword?.trim() || undefined,
          organizationId,
        });
        if (requestSequence !== tagFilterRequestRef.current) return;
        setTagOptions((current) => {
          const selected = new Set(selectedIds);
          const options = new Map(
            current
              .filter((option) => selected.has(option.value))
              .map((option) => [option.value, option]),
          );
          for (const tag of response.tags ?? []) {
            if (tag.id) {
              options.set(tag.id, { label: tag.name ?? '', value: tag.id });
            }
          }
          return [...options.values()];
        });
      } finally {
        if (requestSequence === tagFilterRequestRef.current) {
          setTagOptionsLoading(false);
        }
      }
    },
    [organizationId],
  );

  useEffect(() => {
    if (organizationId) void loadTagFilterOptions();
  }, [loadTagFilterOptions]);
  useEffect(() => {
    void settlementServiceListFinanceOrganizationOptions({
      purpose:
        FinanceOrganizationPurpose.FINANCE_ORGANIZATION_PURPOSE_BILL_READ,
    }).then((response) => setOrganizationOptions(response.data ?? []));
  }, []);

  const { message, modal } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [form] = Form.useForm<BillFormValues>();
  const [workbenchOpen, setWorkbenchOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [editing, setEditing] = useState<API.FinanceBill>();
  const [submitting, setSubmitting] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detail, setDetail] = useState<API.FinanceBill>();

  // 统计指标
  const [metricStats, setMetricStats] = useState({
    totalCount: 0,
    amountsByBaseCurrency: [] as API.FinanceBaseCurrencyAmount[],
  });

  const formatBaseCurrencyAmounts = (
    field:
      | 'receivableBaseAmount'
      | 'payableBaseAmount'
      | 'unverifiedBaseAmount'
      | 'overdueReceivableBaseAmount',
  ) =>
    metricStats.amountsByBaseCurrency
      .map((item) => `${item[field] ?? '0'} ${item.baseCurrency ?? '-'}`)
      .join(' / ') || '-';

  const [searchParams, setSearchParams] = useState<{
    keyword?: string;
    direction?: string;
    status?: number;
    settlementPartyId?: string;
    currency?: string;
    billDateRange?: [Dayjs, Dayjs];
    dueDateRange?: [Dayjs, Dayjs];
    onlyUnsettled?: string;
    onlyOverdue?: string;
  }>({});

  const filterItems: SearchFilterFieldItem[] = [
    {
      name: 'keyword',
      label: '综合搜索',
      placeholder: '输入账单编号/对账抬头/结算单位',
    },
    {
      name: 'direction',
      label: '账单属性',
      type: 'select',
      placeholder: '全部属性',
      options: [
        { label: '应收 (RECEIVABLE)', value: 'RECEIVABLE' },
        { label: '应付 (PAYABLE)', value: 'PAYABLE' },
      ],
    },
    {
      name: 'status',
      label: '账单状态',
      type: 'select',
      placeholder: '全部状态',
      options: [
        { label: '草稿', value: FinanceBillStatus.FINANCE_BILL_STATUS_DRAFT },
        {
          label: '已确认',
          value: FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED,
        },
        {
          label: '已取消',
          value: FinanceBillStatus.FINANCE_BILL_STATUS_CANCELLED,
        },
      ],
    },
    {
      name: 'billDateRange',
      label: '账单日期',
      type: 'date-range',
      placeholder: ['开始日期', '结束日期'],
    },
    {
      name: 'dueDateRange',
      label: '到期日期',
      type: 'date-range',
      placeholder: ['到期开始', '到期结束'],
    },
    {
      name: 'onlyUnsettled',
      label: '结清状态',
      type: 'select',
      placeholder: '全部',
      options: [{ label: '仅看未结清（已确认）', value: 'true' }],
    },
    {
      name: 'onlyOverdue',
      label: '逾期状态',
      type: 'select',
      placeholder: '全部',
      options: [{ label: '仅看已逾期', value: 'true' }],
    },
    {
      name: 'settlementPartyId',
      label: '结算单位',
      type: 'searchable-select',
      placeholder: '输入名称/全拼搜索结算单位',
      request: ({ keyWords }) => searchPartnerOptions(keyWords),
    },
    {
      name: 'currency',
      label: '计价币种',
      type: 'searchable-select',
      placeholder: '全部币种',
      request: getCurrencyOptions,
    },
  ];

  const reload = () => actionRef.current?.reload();
  const billActions = makeVersionActions<API.FinanceBill>({ modal, message });
  const openCreate = () => {
    setWorkbenchOpen(true);
  };

  const openEdit = (bill: API.FinanceBill) => {
    setEditing(bill);
    form.setFieldsValue({
      statementTitle: bill.statementTitle,
      billDate: bill.billDate ? dayjs(bill.billDate) : dayjs(),
      paymentTermsDays: bill.paymentTermsDays,
      note: bill.note,
      settlementAccountId: bill.settlementAccountId,
      estimatedInvoiceCurrency: bill.estimatedInvoiceCurrency || bill.currency,
      estimatedInvoiceRate: bill.estimatedInvoiceRate,
    });
    setEditOpen(true);
  };

  const openDetail = async (bill: API.FinanceBill) => {
    if (!bill.id) return;
    setDetailOpen(true);
    setDetailLoading(true);
    try {
      const response = await settlementServiceGetBill({ id: bill.id });
      setDetail(response.data);
    } catch (error: any) {
      message.error(error.message || '加载账单详情失败');
    } finally {
      setDetailLoading(false);
    }
  };

  const submitBill = async () => {
    if (!editing?.id || !editing.version) return;
    const values = await form.validateFields();
    setSubmitting(true);
    try {
      await settlementServiceUpdateBill(
        { id: editing.id },
        {
          id: editing.id,
          expectedVersion: editing.version,
          statementTitle: values.statementTitle.trim(),
          billDate: values.billDate.format('YYYY-MM-DD'),
          paymentTermsDays: values.paymentTermsDays,
          note: values.note?.trim() || undefined,
          settlementAccountId: values.settlementAccountId,
          estimatedInvoiceCurrency:
            values.estimatedInvoiceCurrency || undefined,
          estimatedInvoiceRate: values.estimatedInvoiceRate || undefined,
        },
      );
      message.success('账单已成功更新并自动刷新汇率快照');
      setEditOpen(false);
      reload();
    } catch (error: any) {
      message.error(error.message || '更新账单失败');
    } finally {
      setSubmitting(false);
    }
  };

  const confirmBill = (bill: API.FinanceBill) =>
    billActions.run(bill, async ({ id, expectedVersion }) => {
      try {
        await settlementServiceConfirmBill({ id }, { id, expectedVersion });
        message.success('账单已确认，进入待开票/待核销流');
        reload();
      } catch (error: any) {
        message.error(error.message || '确认账单失败');
      }
    });

  const cancelBill = (bill: API.FinanceBill) => {
    billActions.confirm(
      bill,
      `取消 ${bill.organizationName || '所属公司未标识'} 的账单并释放关联费用？`,
      async ({ id, expectedVersion }, reason) => {
        await settlementServiceCancelBill(
          { id },
          { id, expectedVersion, reason },
        );
        message.success('账单已取消，关联明细费用已释放并可重新建单');
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
      key: 'total-bills',
      title: '有效账单总记录数',
      value: metricStats.totalCount,
      suffix: '笔',
    },
    {
      key: 'rec-bills',
      title: '应收账单折本币',
      value: formatBaseCurrencyAmounts('receivableBaseAmount'),
      valueColor: '#1677ff',
    },
    {
      key: 'pay-bills',
      title: '应付账单折本币',
      value: formatBaseCurrencyAmounts('payableBaseAmount'),
      valueColor: '#fa8c16',
    },
    {
      key: 'unv-bills',
      title: '未核销总额折本币',
      value: formatBaseCurrencyAmounts('unverifiedBaseAmount'),
      valueColor: metricStats.amountsByBaseCurrency.some(
        (item) => Number(item.unverifiedBaseAmount ?? 0) > 0,
      )
        ? '#cf1322'
        : '#52c41a',
    },
    {
      key: 'overdue-bills',
      title: '逾期应收折本币',
      value: formatBaseCurrencyAmounts('overdueReceivableBaseAmount'),
      valueColor: metricStats.amountsByBaseCurrency.some(
        (item) => Number(item.overdueReceivableBaseAmount ?? 0) > 0,
      )
        ? '#cf1322'
        : '#52c41a',
    },
  ];

  const columns = getFinanceBillColumns({
    access,
    onOpenDetail: openDetail,
    onOpenEdit: openEdit,
    onConfirmBill: confirmBill,
    onCancelBill: cancelBill,
  });

  return (
    <>
      <div style={{ marginBottom: 12 }}>
        <span style={{ marginRight: 8 }}>标签筛选</span>
        <Select
          mode="multiple"
          allowClear
          showSearch={{
            filterOption: false,
            onSearch: (keyword) =>
              void loadTagFilterOptions(keyword, tagFilterIds),
          }}
          disabled={!organizationId}
          loading={tagOptionsLoading}
          style={{ minWidth: 320 }}
          placeholder="命中任一标签即返回"
          options={tagOptions}
          value={tagFilterIds}
          onChange={(value) => {
            setTagFilterIds(value.length ? value : undefined);
            actionRef.current?.reload();
          }}
        />
        <Select
          allowClear
          style={{ minWidth: 220, marginLeft: 12 }}
          placeholder="所属公司"
          options={organizationOptions.map((item) => ({
            value: item.id,
            label: item.name ?? item.code ?? item.id,
          }))}
          value={organizationId}
          onChange={(value) => {
            // 使在切换或清空公司前发出的标签请求立即失效，避免迟到结果回填。
            tagFilterRequestRef.current += 1;
            setTagOptionsLoading(false);
            setOrganizationId(value);
            setTagFilterIds(undefined);
            setTagOptions([]);
            actionRef.current?.reload();
          }}
        />
      </div>

      <FinanceLedgerTemplate<API.FinanceBill>
        headerTitle="账单管理台账"
        actionRef={actionRef}
        columns={columns}
        metricCards={metricCards}
        scrollX={2000}
        search={false}
        customSearch={
          <SearchFilterTemplate
            layout="grid"
            formLayout="horizontal"
            labelWidth={80}
            collapsible={true}
            defaultCollapsed={true}
            defaultVisibleCount={5}
            items={filterItems}
            onSearch={(values) => {
              setSearchParams(values);
              reload();
            }}
            onReset={() => {
              setSearchParams({});
              reload();
            }}
          />
        }
        primaryActionText={
          access.canCreateFinanceBills ? '批量创建账单' : undefined
        }
        batchActions={
          access.canUpdateFinanceBills && organizationId
            ? [
                {
                  key: 'manage-tags',
                  label: '添加/移除标签',
                  onClick: (_keys: React.Key[], rows: API.FinanceBill[]) => {
                    if (!organizationId) {
                      message.warning('请先选择所属公司后再维护账单标签');
                      return;
                    }
                    if (
                      !rows.length ||
                      rows.some((row) => row.organizationId !== organizationId)
                    ) {
                      message.warning('只能维护当前所属公司的账单标签');
                      return;
                    }
                    setTagBillIds(
                      rows.map((row) => row.id ?? '').filter(Boolean),
                    );
                    const seen = new Map<string, API.BusinessTagSummary>();
                    for (const row of rows)
                      for (const tag of row.tags ?? [])
                        if (tag.id) seen.set(tag.id, tag);
                    setTagExisting([...seen.values()]);
                    setTagModalOpen(true);
                  },
                },
              ]
            : []
        }
        primaryActionIcon={<PlusOutlined />}
        onPrimaryAction={openCreate}
        request={async (params) => {
          const billDateFrom = searchParams.billDateRange?.[0]
            ? searchParams.billDateRange[0].format('YYYY-MM-DD')
            : undefined;
          const billDateTo = searchParams.billDateRange?.[1]
            ? searchParams.billDateRange[1].format('YYYY-MM-DD')
            : undefined;
          const dueDateFrom = searchParams.dueDateRange?.[0]
            ? searchParams.dueDateRange[0].format('YYYY-MM-DD')
            : undefined;
          const dueDateTo = searchParams.dueDateRange?.[1]
            ? searchParams.dueDateRange[1].format('YYYY-MM-DD')
            : undefined;

          const response = await settlementServiceListBills({
            page: params.current,
            pageSize: params.pageSize,
            keyword: searchParams.keyword || undefined,
            direction: searchParams.direction || undefined,
            status: searchParams.status,
            settlementPartyId: searchParams.settlementPartyId || undefined,
            currency: searchParams.currency || undefined,
            billDateFrom,
            billDateTo,
            dueDateFrom,
            dueDateTo,
            onlyUnsettled:
              searchParams.onlyUnsettled === 'true' ? true : undefined,
            onlyOverdue: searchParams.onlyOverdue === 'true' ? true : undefined,
            tagIds: tagFilterIds?.length ? tagFilterIds : undefined,
            organizationId,
          });

          const page = unwrapPage(response);
          setMetricStats({
            totalCount: page.total,
            amountsByBaseCurrency:
              response.summary?.amountsByBaseCurrency ?? [],
          });

          return { ...toTableRequest(response), total: page.total };
        }}
      />

      <BillEditModal
        key={editing?.id}
        open={editOpen}
        editing={editing}
        form={form}
        submitting={submitting}
        onCancel={() => setEditOpen(false)}
        onOk={submitBill}
      />

      <BillCreationWorkbench
        open={workbenchOpen}
        onClose={() => setWorkbenchOpen(false)}
        onCreated={() => reload()}
      />

      <BillDetailDrawer
        open={detailOpen}
        loading={detailLoading}
        detail={detail}
        onClose={() => setDetailOpen(false)}
      />
      <BusinessTagModal
        open={tagModalOpen}
        targetCount={tagBillIds.length}
        existingTags={tagExisting}
        // 跨组织财务标签写入不支持在此快捷新建，避免新标签误归当前工作区。
        canQuickCreate={false}
        loadOptions={(params) => {
          if (!organizationId) {
            return Promise.resolve({ tags: [], total: '0' });
          }
          return settlementServiceListFinanceBillTagAssignmentOptions({
            ...params,
            organizationId,
          });
        }}
        onSubmit={async (mode, tagIds) => {
          if (mode === 'assign') {
            await settlementServiceBatchAssignFinanceBillTags({
              billIds: tagBillIds,
              tagIds,
            });
          } else {
            await settlementServiceBatchRemoveFinanceBillTags({
              billIds: tagBillIds,
              tagIds,
            });
          }
          actionRef.current?.reload();
        }}
        onCancel={() => setTagModalOpen(false)}
      />
    </>
  );
}
