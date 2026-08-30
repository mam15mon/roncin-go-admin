import {
  CheckOutlined,
  CloseCircleOutlined,
  EyeOutlined,
  PlusOutlined,
  SwapOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Form, Space, Tag } from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, { useRef, useState } from 'react';
import {
  type FinanceLedgerMetricCard,
  FinanceLedgerTemplate,
} from '@/components/ui';
import { financeErrorReasons } from '@/errorReasons.generated';
import { partnerServiceListPartnerInvoiceProfiles } from '@/services/roncin/partnerService';
import {
  settlementServiceCancelInvoice,
  settlementServiceCreateInvoice,
  settlementServiceGetInvoice,
  settlementServiceIssueInvoice,
  settlementServiceListInvoices,
  settlementServiceRedFlushInvoice,
} from '@/services/roncin/settlementService';
import { toTableRequest, unwrapList, unwrapPage } from '@/utils/api';
import { makeVersionActions } from '@/utils/versionActions';
import InvoiceCreateModal from './components/InvoiceCreateModal';
import InvoiceDetailDrawer from './components/InvoiceDetailDrawer';
import {
  InvoiceIssueModal,
  InvoiceRedFlushModal,
} from './components/InvoiceIssueAndRedFlushModals';
import { invoiceStates } from './components/invoiceConstants';

type CreateValues = {
  invoiceProfileId: string;
  invoiceType: string;
  note?: string;
};
type IssueValues = { taxInvoiceNo: string; invoiceDate: Dayjs };
type RedFlushValues = {
  redInvoiceNo: string;
  redInvoiceDate: Dayjs;
  reason: string;
};

export default function FinanceInvoicesPage() {
  const access = useAccess();
  const { message, modal } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [createForm] = Form.useForm<CreateValues>();
  const [issueForm] = Form.useForm<IssueValues>();
  const [redFlushForm] = Form.useForm<RedFlushValues>();
  const [createOpen, setCreateOpen] = useState(false);
  const [issueTarget, setIssueTarget] = useState<API.FinanceInvoice>();
  const [redFlushTarget, setRedFlushTarget] = useState<API.FinanceInvoice>();
  const [selectedIDs, setSelectedIDs] = useState<React.Key[]>([]);
  const [selectedBills, setSelectedBills] = useState<API.FinanceBill[]>([]);
  const [availableProfiles, setAvailableProfiles] = useState<
    API.PartnerInvoiceProfile[]
  >([]);
  const [selectedProfile, setSelectedProfile] =
    useState<API.PartnerInvoiceProfile>();
  const [submitting, setSubmitting] = useState(false);
  const [detail, setDetail] = useState<API.FinanceInvoice>();
  const [metricStats, setMetricStats] = useState({
    totalCount: 0,
    receivableTotal: 0,
    payableTotal: 0,
    issuedCount: 0,
    baseCurrency: '',
  });
  const reload = () => actionRef.current?.reload();
  const invoiceActions = makeVersionActions<API.FinanceInvoice>({
    modal,
    message,
  });

  const loadSelectedProfiles = async (partnerId?: string) => {
    if (!partnerId) {
      setAvailableProfiles([]);
      setSelectedProfile(undefined);
      createForm.setFieldValue('invoiceProfileId', undefined);
      return;
    }
    try {
      const response = await partnerServiceListPartnerInvoiceProfiles(
        { partnerId },
        { skipErrorHandler: true },
      );
      const profiles = unwrapList(response).filter((item) => item.enabled);
      setAvailableProfiles(profiles);
      const selected = profiles.find((item) => item.isDefault) || profiles[0];
      setSelectedProfile(selected);
      createForm.setFieldValue('invoiceProfileId', selected?.id);
      if (selected?.defaultInvoiceType) {
        createForm.setFieldValue('invoiceType', selected.defaultInvoiceType);
      }
      if (!selected) {
        message.warning(
          '该结算单位尚未配置可用开票抬头，请先到往来单位档案维护',
        );
      }
    } catch (rawError: any) {
      setAvailableProfiles([]);
      setSelectedProfile(undefined);
      createForm.setFieldValue('invoiceProfileId', undefined);
      message.error(rawError.message || '加载开票抬头失败');
    }
  };

  const showDetail = async (row: API.FinanceInvoice) => {
    if (!row.id) return;
    try {
      setDetail((await settlementServiceGetInvoice({ id: row.id })).data);
    } catch (error: any) {
      message.error(error.message || '加载开票详情失败');
    }
  };

  const createInvoice = async () => {
    const values = await createForm.validateFields();
    if (!selectedIDs.length) {
      message.warning('请至少选择一张已确认账单');
      return;
    }
    setSubmitting(true);
    try {
      await settlementServiceCreateInvoice({
        billIds: selectedIDs.map(String),
        invoiceProfileId: values.invoiceProfileId,
        invoiceType: values.invoiceType,
        note: values.note,
        idempotencyKey: globalThis.crypto.randomUUID(),
      });
      message.success('开票记录已创建，账单已占用');
      setCreateOpen(false);
      reload();
    } catch (error: any) {
      const reason = error.data?.reason || error.response?.data?.reason;
      if (reason === financeErrorReasons.FINANCE_INVOICE_PROFILE_REQUIRED) {
        message.error('请选择该结算单位下启用且完整的开票抬头');
      } else {
        message.error(error.message || '创建开票记录失败');
      }
    } finally {
      setSubmitting(false);
    }
  };

  const issueInvoice = async () => {
    if (!issueTarget?.id || !issueTarget.version) return;
    const values = await issueForm.validateFields();
    setSubmitting(true);
    try {
      await settlementServiceIssueInvoice(
        { id: issueTarget.id },
        {
          id: issueTarget.id,
          expectedVersion: issueTarget.version,
          taxInvoiceNo: values.taxInvoiceNo,
          invoiceDate: values.invoiceDate.format('YYYY-MM-DD'),
        },
      );
      message.success('发票已确认开具');
      setIssueTarget(undefined);
      reload();
    } catch (error: any) {
      message.error(error.message || '确认开具失败');
    } finally {
      setSubmitting(false);
    }
  };

  const cancelInvoice = (row: API.FinanceInvoice) => {
    invoiceActions.confirm(
      row,
      '取消开票记录并释放账单？',
      async ({ id, expectedVersion }, reason) => {
        await settlementServiceCancelInvoice(
          { id },
          { id, expectedVersion, reason },
        );
        message.success('开票记录已取消，账单已释放');
        reload();
      },
      {
        danger: true,
        placeholder: '请输入取消原因（必填）',
        requiredMessage: '请输入取消原因',
      },
    );
  };

  const redFlushInvoice = async () => {
    if (!redFlushTarget?.id || !redFlushTarget.version) return;
    const values = await redFlushForm.validateFields();
    setSubmitting(true);
    try {
      await settlementServiceRedFlushInvoice(
        { id: redFlushTarget.id },
        {
          id: redFlushTarget.id,
          expectedVersion: redFlushTarget.version,
          redInvoiceNo: values.redInvoiceNo,
          redInvoiceDate: values.redInvoiceDate.format('YYYY-MM-DD'),
          reason: values.reason,
        },
      );
      message.success('发票已红冲，原账单开票占用已释放');
      setRedFlushTarget(undefined);
      reload();
    } catch (error: any) {
      message.error(error.message || '发票红冲失败');
    } finally {
      setSubmitting(false);
    }
  };

  const columns: ProColumns<API.FinanceInvoice>[] = [
    {
      title: '关键词',
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: { placeholder: '记录编号、发票号或结算单位' },
    },
    {
      title: '记录编号',
      dataIndex: 'recordNo',
      width: 160,
      copyable: true,
      search: false,
    },
    {
      title: '方向',
      dataIndex: 'direction',
      width: 80,
      valueType: 'select',
      valueEnum: { RECEIVABLE: { text: '销项' }, PAYABLE: { text: '进项' } },
      render: (_, r) => (
        <Tag color={r.direction === 'RECEIVABLE' ? 'blue' : 'purple'}>
          {r.direction === 'RECEIVABLE' ? '销项' : '进项'}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        Object.entries(invoiceStates).map(([k, v]) => [k, { text: v.text }]),
      ),
      render: (_, r) => {
        const v = invoiceStates[r.status || 'DRAFT'];
        return <Tag color={v?.color}>{v?.text}</Tag>;
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
      title: '发票类型',
      dataIndex: 'invoiceType',
      width: 100,
      search: false,
      renderText: (v) => (v === 'SPECIAL' ? '专用发票' : '普通发票'),
    },
    {
      title: '金额',
      dataIndex: 'totalAmount',
      width: 140,
      align: 'right',
      search: false,
      render: (_, r) => (
        <strong style={{ color: '#262626' }}>
          {r.totalAmount} {r.currency}
        </strong>
      ),
    },
    {
      title: '开票汇率',
      dataIndex: 'exchangeRate',
      width: 135,
      align: 'right',
      search: false,
      render: (_, r) => {
        if (!r.exchangeRate) {
          return r.status === 'DRAFT' ? (
            <span style={{ color: '#8c8c8c' }}>开票时确定</span>
          ) : (
            '-'
          );
        }
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
      title: '折本币金额',
      dataIndex: 'baseCurrencyAmount',
      width: 145,
      align: 'right',
      search: false,
      render: (_, r) =>
        r.baseCurrencyAmount ? (
          <strong
            style={{
              color: r.direction === 'RECEIVABLE' ? '#1677ff' : '#fa8c16',
            }}
          >
            {r.baseCurrencyAmount} {r.baseCurrency}
          </strong>
        ) : (
          '-'
        ),
    },
    {
      title: '税额',
      dataIndex: 'taxAmount',
      width: 120,
      align: 'right',
      search: false,
    },
    {
      title: '税务发票号',
      dataIndex: 'taxInvoiceNo',
      width: 160,
      search: false,
      renderText: (v) => v || '-',
    },
    {
      title: '开票日期',
      dataIndex: 'invoiceDate',
      width: 110,
      search: false,
      renderText: (v) => v || '-',
    },
    {
      title: '操作',
      valueType: 'option',
      fixed: 'right',
      width: 210,
      render: (_, r) => [
        <a key="detail" onClick={() => void showDetail(r)}>
          <EyeOutlined /> 详情
        </a>,
        access.canUpdateFinanceInvoices && r.status === 'DRAFT' ? (
          <a
            key="issue"
            onClick={() => {
              setIssueTarget(r);
              issueForm.setFieldsValue({ invoiceDate: dayjs() });
            }}
          >
            <CheckOutlined /> 确认开具
          </a>
        ) : null,
        access.canUpdateFinanceInvoices &&
        r.status !== 'CANCELLED' &&
        r.status !== 'RED_FLUSHED' ? (
          <a
            key="cancel"
            style={{ color: '#ff4d4f' }}
            onClick={() => cancelInvoice(r)}
          >
            <CloseCircleOutlined /> {r.status === 'ISSUED' ? '作废' : '取消'}
          </a>
        ) : null,
        access.canUpdateFinanceInvoices && r.status === 'ISSUED' ? (
          <a
            key="red-flush"
            style={{ color: '#cf1322' }}
            onClick={() => {
              setRedFlushTarget(r);
              redFlushForm.setFieldsValue({ redInvoiceDate: dayjs() });
            }}
          >
            <SwapOutlined /> 红冲
          </a>
        ) : null,
      ],
    },
  ];

  const metricCards: FinanceLedgerMetricCard[] = [
    {
      key: 'total-invoices',
      title: '发票总记录数',
      value: metricStats.totalCount,
      suffix: '笔',
    },
    {
      key: 'rec-invoices',
      title: '销项发票金额',
      value: metricStats.receivableTotal,
      precision: 2,
      suffix: metricStats.baseCurrency || '-',
      valueColor: '#1677ff',
    },
    {
      key: 'pay-invoices',
      title: '进项发票金额',
      value: metricStats.payableTotal,
      precision: 2,
      suffix: metricStats.baseCurrency || '-',
      valueColor: '#fa8c16',
    },
    {
      key: 'issued-count',
      title: '已正式开具',
      value: metricStats.issuedCount,
      suffix: '笔',
      valueColor: '#52c41a',
    },
  ];

  return (
    <>
      <FinanceLedgerTemplate<API.FinanceInvoice>
        headerTitle="发票明细管理"
        actionRef={actionRef}
        columns={columns}
        metricCards={metricCards}
        scrollX={1600}
        primaryActionText={
          access.canCreateFinanceInvoices ? '从账单创建开票' : undefined
        }
        primaryActionIcon={<PlusOutlined />}
        onPrimaryAction={() => {
          setSelectedIDs([]);
          setSelectedBills([]);
          setAvailableProfiles([]);
          setSelectedProfile(undefined);
          createForm.resetFields();
          setCreateOpen(true);
        }}
        request={async (p) => {
          const r = await settlementServiceListInvoices({
            page: p.current,
            pageSize: p.pageSize,
            keyword: p.keyword,
            direction: p.direction,
            status: p.status,
          });
          const page = unwrapPage(r);
          setMetricStats({
            totalCount: page.total,
            receivableTotal: Number(r.summary?.receivableBaseAmount || 0),
            payableTotal: Number(r.summary?.payableBaseAmount || 0),
            issuedCount: Number(r.summary?.issuedCount || 0),
            baseCurrency: r.summary?.baseCurrency || '',
          });
          return { ...toTableRequest(r), total: page.total };
        }}
      />

      <InvoiceCreateModal
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        submitting={submitting}
        createForm={createForm}
        selectedBills={selectedBills}
        selectedIDs={selectedIDs}
        setSelectedIDs={setSelectedIDs}
        setSelectedBills={setSelectedBills}
        availableProfiles={availableProfiles}
        selectedProfile={selectedProfile}
        setSelectedProfile={setSelectedProfile}
        loadSelectedProfiles={loadSelectedProfiles}
        onOk={createInvoice}
      />

      <InvoiceIssueModal
        open={Boolean(issueTarget)}
        submitting={submitting}
        issueForm={issueForm}
        onCancel={() => setIssueTarget(undefined)}
        onOk={issueInvoice}
      />

      <InvoiceRedFlushModal
        open={Boolean(redFlushTarget)}
        submitting={submitting}
        redFlushTarget={redFlushTarget}
        redFlushForm={redFlushForm}
        onCancel={() => setRedFlushTarget(undefined)}
        onOk={redFlushInvoice}
      />

      <InvoiceDetailDrawer
        detail={detail}
        onClose={() => setDetail(undefined)}
      />
    </>
  );
}
