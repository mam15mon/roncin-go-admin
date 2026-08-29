import { PlusOutlined } from '@ant-design/icons';
import type { ActionType } from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Form, Input, Select } from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, { useEffect, useRef, useState } from 'react';
import {
  FinanceLedgerTemplate,
  SearchFilterTemplate,
  type FinanceLedgerMetricCard,
  type SearchFilterFieldItem,
} from '@/components/ui';
import { partnerServiceListPartners } from '@/services/roncin/partnerService';
import {
  settlementServiceCancelBill,
  settlementServiceConfirmBill,
  settlementServiceGetBill,
  settlementServiceListBills,
  settlementServiceUpdateBill,
} from '@/services/roncin/settlementService';
import {
  settlementServiceListFinanceBillTagOptions,
  settlementServiceBatchAssignFinanceBillTags,
  settlementServiceBatchRemoveFinanceBillTags,
} from '@/services/roncin/settlementService';
import { BusinessTagModal } from '@/components/business-tag/BusinessTagModal';
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
  const [tagOptions, setTagOptions] = useState<{ label: string; value: string }[]>([]);
  const [tagFilterIds, setTagFilterIds] = useState<string[]>();

  useEffect(() => {
    void settlementServiceListFinanceBillTagOptions({ page: 1, pageSize: 200 }).then((response) => {
      setTagOptions((response.tags ?? []).map((tag) => ({ label: tag.name ?? '', value: tag.id ?? '' })));
    });
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
    receivableBase: 0,
    payableBase: 0,
    unverifiedBase: 0,
  });

  const [searchParams, setSearchParams] = useState<{
    keyword?: string;
    direction?: string;
    status?: string;
    settlementPartyId?: string;
    currency?: string;
    billDateRange?: [Dayjs, Dayjs];
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
        { label: '草稿 (DRAFT)', value: 'DRAFT' },
        { label: '已确认 (CONFIRMED)', value: 'CONFIRMED' },
        { label: '已取消 (CANCELLED)', value: 'CANCELLED' },
      ],
    },
    {
      name: 'billDateRange',
      label: '账单日期',
      type: 'date-range',
      placeholder: ['开始日期', '结束日期'],
    },
    {
      name: 'settlementPartyId',
      label: '结算单位',
      type: 'searchable-select',
      placeholder: '输入名称/全拼搜索结算单位',
      request: async ({ keyWords }) => {
        const res = await partnerServiceListPartners({
          page: 1,
          pageSize: 200,
          keyword: keyWords,
        });
        return (res.data || []).map((p) => ({
          label: p.legalName || p.code || p.id || '',
          value: p.id || '',
        }));
      },
    },
    {
      name: 'currency',
      label: '计价币种',
      type: 'select',
      placeholder: '全部币种',
      options: [
        { label: 'CNY - 人民币', value: 'CNY' },
        { label: 'USD - 美元', value: 'USD' },
        { label: 'EUR - 欧元', value: 'EUR' },
        { label: 'HKD - 港币', value: 'HKD' },
      ],
    },
  ];

  const reload = () => actionRef.current?.reload();
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

  const confirmBill = async (bill: API.FinanceBill) => {
    if (!bill.id || !bill.version) return;
    try {
      await settlementServiceConfirmBill(
        { id: bill.id },
        { id: bill.id, expectedVersion: bill.version },
      );
      message.success('账单已确认，进入待开票/待核销流');
      reload();
    } catch (error: any) {
      message.error(error.message || '确认账单失败');
    }
  };

  const cancelBill = (bill: API.FinanceBill) => {
    const id = bill.id;
    const version = bill.version;
    if (!id || !version) return;
    let reason = '';
    modal.confirm({
      title: '取消账单并释放关联费用？',
      content: (
        <Input.TextArea
          placeholder="请输入取消原因（必填）"
          maxLength={500}
          onChange={(e) => {
            reason = e.target.value.trim();
          }}
        />
      ),
      okButtonProps: { danger: true },
      onOk: async () => {
        if (!reason) {
          message.warning('请输入取消原因');
          throw new Error('取消原因不能为空');
        }
        await settlementServiceCancelBill(
          { id },
          { id, expectedVersion: version, reason },
        );
        message.success('账单已取消，关联明细费用已释放并可重新建单');
        reload();
      },
    });
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
      value: metricStats.receivableBase,
      precision: 2,
      suffix: 'CNY',
      valueColor: '#1677ff',
    },
    {
      key: 'pay-bills',
      title: '应付账单折本币',
      value: metricStats.payableBase,
      precision: 2,
      suffix: 'CNY',
      valueColor: '#fa8c16',
    },
    {
      key: 'unv-bills',
      title: '未核销总额折本币',
      value: metricStats.unverifiedBase,
      precision: 2,
      suffix: 'CNY',
      valueColor: metricStats.unverifiedBase > 0 ? '#cf1322' : '#52c41a',
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
          showSearch
          optionFilterProp="label"
          style={{ minWidth: 320 }}
          placeholder="命中任一标签即返回"
          options={tagOptions}
          value={tagFilterIds}
          onChange={(value) => {
            setTagFilterIds(value.length ? value : undefined);
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
        primaryActionIcon={<PlusOutlined />}
        onPrimaryAction={openCreate}
        request={async (params) => {
          const billDateFrom = searchParams.billDateRange?.[0]
            ? searchParams.billDateRange[0].format('YYYY-MM-DD')
            : undefined;
          const billDateTo = searchParams.billDateRange?.[1]
            ? searchParams.billDateRange[1].format('YYYY-MM-DD')
            : undefined;

          const response = await settlementServiceListBills({
            page: params.current,
            pageSize: params.pageSize,
            keyword: searchParams.keyword || undefined,
            direction: searchParams.direction || undefined,
            status: searchParams.status || undefined,
            settlementPartyId: searchParams.settlementPartyId || undefined,
            currency: searchParams.currency || undefined,
            billDateFrom,
            billDateTo,
            tagIds: tagFilterIds?.length ? tagFilterIds : undefined,
          });

          const list = response.data || [];
          const recBase = list
            .filter((x) => x.direction === 'RECEIVABLE')
            .reduce((s, x) => s + Number(x.baseCurrencyAmount || 0), 0);
          const payBase = list
            .filter((x) => x.direction === 'PAYABLE')
            .reduce((s, x) => s + Number(x.baseCurrencyAmount || 0), 0);
          const unvBase = list.reduce(
            (s, x) => s + Number(x.unverifiedAmount || 0),
            0,
          );

          setMetricStats({
            totalCount: Number(response.total || list.length),
            receivableBase: recBase,
            payableBase: payBase,
            unverifiedBase: unvBase,
          });

          return {
            data: list,
            total: Number(response.total || 0),
            success: response.success ?? true,
          };
        }}
      />

      <BillEditModal
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
        canQuickCreate={Boolean(access?.canCreateEnterpriseResources)}
        loadOptions={settlementServiceListFinanceBillTagOptions}
        onSubmit={async (mode, tagIds) => {
          if (mode === 'assign') {
            await settlementServiceBatchAssignFinanceBillTags({ billIds: tagBillIds, tagIds });
          } else {
            await settlementServiceBatchRemoveFinanceBillTags({ billIds: tagBillIds, tagIds });
          }
          actionRef.current?.reload();
        }}
        onCancel={() => setTagModalOpen(false)}
      />
    </>
  );
}
