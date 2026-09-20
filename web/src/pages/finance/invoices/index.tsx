import {
  CheckOutlined,
  CloseCircleOutlined,
  EyeOutlined,
  PlusOutlined,
  SwapOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { useQuery } from '@tanstack/react-query';
import { useAccess } from '@/app/access';
import { App, Form, Select, Space, Tag } from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, { useRef, useState } from 'react';
import {
  type FinanceLedgerMetricCard,
  FinanceLedgerTemplate,
} from '@/components/ui';
import {
  FinanceInvoiceStatus,
  FinanceOrganizationPurpose,
} from '@/enums.generated';
import { financeErrorReasons } from '@/errorReasons.generated';
import {
  settlementServiceCancelInvoice,
  settlementServiceCreateInvoice,
  settlementServiceGetInvoice,
  settlementServiceIssueInvoice,
  settlementServiceListFinanceOrganizationOptions,
  settlementServiceListInvoices,
  settlementServiceRedFlushInvoice,
} from '@/services/roncin/settlementService';
import { toTableRequest, unwrapPage } from '@/utils/api';
import { getErrorMessage } from '@/utils/errorMessage';
import { generateUUID } from '@/utils/uuid';
import InvoiceCancelModal from './components/InvoiceCancelModal';
import InvoiceCreateModal from './components/InvoiceCreateModal';
import InvoiceDetailDrawer from './components/InvoiceDetailDrawer';
import {
  InvoiceIssueModal,
  InvoiceRedFlushModal,
} from './components/InvoiceIssueAndRedFlushModals';
import {
  invoiceCancelSuccessText,
  invoiceIssueActionText,
  invoiceStates,
  invoiceStateText,
  invoiceVoidSuccessText,
  isReceivableInvoice,
} from './components/invoiceConstants';

/** 台账内置搜索表单提交的筛选字段（keyword 等分页字段由模板统一注入，不在此声明） */
type InvoiceLedgerFilterParams = {
  direction?: string;
  status?: string;
};

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
type CancelValues = { reason: string };

/** 读取请求错误对象上的 `reason` 字段（仅接受 truthy 字符串）。 */
function readReasonText(source: unknown): string | undefined {
  if (typeof source !== 'object' || source === null) return undefined;
  if (!('reason' in source)) return undefined;
  const reason = (source as { reason?: unknown }).reason;
  return typeof reason === 'string' && reason ? reason : undefined;
}

/** 提取开票失败的业务 reason：兼容 data 与 response.data 两层包裹。 */
function readInvoiceErrorReason(error: unknown): string | undefined {
  if (typeof error !== 'object' || error === null) return undefined;
  const { data, response } = error as { data?: unknown; response?: unknown };
  const fromData = readReasonText(data);
  if (fromData !== undefined) return fromData;
  if (typeof response === 'object' && response !== null) {
    return readReasonText((response as { data?: unknown }).data);
  }
  return undefined;
}

export default function FinanceInvoicesPage() {
  const access = useAccess();
  const { message } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [createForm] = Form.useForm<CreateValues>();
  const [issueForm] = Form.useForm<IssueValues>();
  const [redFlushForm] = Form.useForm<RedFlushValues>();
  const [cancelForm] = Form.useForm<CancelValues>();
  const [createOpen, setCreateOpen] = useState(false);
  const [issueTarget, setIssueTarget] = useState<API.FinanceInvoice>();
  const [redFlushTarget, setRedFlushTarget] = useState<API.FinanceInvoice>();
  const [cancelTarget, setCancelTarget] = useState<API.FinanceInvoice>();
  const [selectedIDs, setSelectedIDs] = useState<React.Key[]>([]);
  const [selectedBills, setSelectedBills] = useState<API.FinanceBill[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [detail, setDetail] = useState<API.FinanceInvoice>();
  const [organizationId, setOrganizationId] = useState<string>();
  // 顶部筛选所属公司候选：历史无错误兜底，保持静默（仅请求层通知）。
  const organizationQuery = useQuery({
    queryKey: [
      'finance',
      'organization-options',
      {
        purpose:
          FinanceOrganizationPurpose.FINANCE_ORGANIZATION_PURPOSE_INVOICE_READ,
      },
    ],
    meta: { silent: true },
    queryFn: async () => {
      const response = await settlementServiceListFinanceOrganizationOptions({
        purpose:
          FinanceOrganizationPurpose.FINANCE_ORGANIZATION_PURPOSE_INVOICE_READ,
      });
      return response.data ?? [];
    },
  });
  const organizationOptions = organizationQuery.data ?? [];
  const [metricStats, setMetricStats] = useState({
    totalCount: 0,
    issuedCount: 0,
    amountsByBaseCurrency: [] as API.FinanceBaseCurrencyAmount[],
  });
  const formatBaseCurrencyAmounts = (
    field: 'receivableBaseAmount' | 'payableBaseAmount',
  ) =>
    metricStats.amountsByBaseCurrency
      .map((item) => `${item[field] ?? '0'} ${item.baseCurrency ?? '-'}`)
      .join(' / ') || '-';
  const reload = () => actionRef.current?.reload();

  const showDetail = async (row: API.FinanceInvoice) => {
    if (!row.id) return;
    try {
      setDetail((await settlementServiceGetInvoice({ id: row.id })).data);
    } catch (error) {
      message.error(getErrorMessage(error, '加载开票详情失败'));
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
        idempotencyKey: generateUUID(),
      });
      message.success('开票记录已创建，账单已占用');
      setCreateOpen(false);
      reload();
    } catch (error) {
      const reason = readInvoiceErrorReason(error);
      if (reason === financeErrorReasons.FINANCE_INVOICE_PROFILE_REQUIRED) {
        message.error('请选择该结算单位下启用且完整的开票抬头');
      } else {
        message.error(getErrorMessage(error, '创建开票记录失败'));
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
      message.success(
        isReceivableInvoice(issueTarget.direction)
          ? '发票已确认开具'
          : '发票已确认收票',
      );
      setIssueTarget(undefined);
      reload();
    } catch (error) {
      message.error(
        getErrorMessage(
          error,
          isReceivableInvoice(issueTarget.direction)
            ? '确认开具失败'
            : '确认收票失败',
        ),
      );
    } finally {
      setSubmitting(false);
    }
  };

  const cancelInvoice = async () => {
    if (!cancelTarget?.id || !cancelTarget.version) return;
    const issued =
      cancelTarget.status ===
      FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_ISSUED;
    const values = await cancelForm.validateFields();
    setSubmitting(true);
    try {
      await settlementServiceCancelInvoice(
        { id: cancelTarget.id },
        {
          id: cancelTarget.id,
          expectedVersion: cancelTarget.version,
          reason: values.reason,
        },
      );
      message.success(
        issued
          ? invoiceVoidSuccessText(cancelTarget.direction)
          : invoiceCancelSuccessText(cancelTarget.direction),
      );
      setCancelTarget(undefined);
      reload();
    } catch (error) {
      message.error(
        getErrorMessage(error, issued ? '作废发票失败' : '取消开票记录失败'),
      );
    } finally {
      setSubmitting(false);
    }
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
    } catch (error) {
      message.error(getErrorMessage(error, '发票红冲失败'));
    } finally {
      setSubmitting(false);
    }
  };

  const columns: ProColumns<API.FinanceInvoice>[] = [
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
        Object.entries(invoiceStates).map(([k, v]) => [
          k,
          {
            text:
              Number(k) === FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_ISSUED
                ? '已开具 / 已收票'
                : v.text,
          },
        ]),
      ),
      render: (_, r) => {
        const v =
          invoiceStates[
            r.status ?? FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_DRAFT
          ];
        return (
          <Tag color={v?.color}>{invoiceStateText(r.status, r.direction)}</Tag>
        );
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
          return r.status ===
            FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_DRAFT ? (
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
        access.canUpdateFinanceInvoices &&
        access.canOperateOrganization(r.organizationId) &&
        r.status === FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_DRAFT ? (
          <a
            key="issue"
            onClick={() => {
              setIssueTarget(r);
              issueForm.setFieldsValue({ invoiceDate: dayjs() });
            }}
          >
            <CheckOutlined /> {invoiceIssueActionText(r.direction)}
          </a>
        ) : null,
        access.canUpdateFinanceInvoices &&
        access.canOperateOrganization(r.organizationId) &&
        (r.status === FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_DRAFT ||
          r.status === FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_ISSUED) ? (
          <a
            key="cancel"
            style={{ color: '#ff4d4f' }}
            onClick={() => {
              cancelForm.resetFields();
              setCancelTarget(r);
            }}
          >
            <CloseCircleOutlined />{' '}
            {r.status === FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_ISSUED
              ? '作废'
              : '取消'}
          </a>
        ) : null,
        access.canUpdateFinanceInvoices &&
        access.canOperateOrganization(r.organizationId) &&
        r.status === FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_ISSUED ? (
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
      value: formatBaseCurrencyAmounts('receivableBaseAmount'),
      valueColor: '#1677ff',
    },
    {
      key: 'pay-invoices',
      title: '进项发票金额',
      value: formatBaseCurrencyAmounts('payableBaseAmount'),
      valueColor: '#fa8c16',
    },
    {
      key: 'issued-count',
      title: '已开具 / 已收票',
      value: metricStats.issuedCount,
      suffix: '笔',
      valueColor: '#52c41a',
    },
  ];

  return (
    <>
      <FinanceLedgerTemplate<API.FinanceInvoice, InvoiceLedgerFilterParams>
        pageTitle="发票管理"
        pageSubTitle="进项与销项发票开具、收票核验与发票状态跟踪"
        topBar={
          <Space size={8} align="center">
            <span style={{ fontSize: 13, color: 'rgba(0, 0, 0, 0.65)' }}>
              所属公司：
            </span>
            <Select
              allowClear
              placeholder="请选择所属公司"
              style={{ minWidth: 220 }}
              value={organizationId}
              options={organizationOptions.map((item) => ({
                value: item.id,
                label: item.name ?? item.code ?? item.id,
              }))}
              onChange={(value) => {
                setOrganizationId(value);
                actionRef.current?.reload();
              }}
            />
          </Space>
        }
        headerTitle="发票明细列表"
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
          createForm.resetFields();
          setCreateOpen(true);
        }}
        request={async (p) => {
          const r = await settlementServiceListInvoices({
            page: p.current,
            pageSize: p.pageSize,
            keyword: p.keyword,
            direction: p.direction,
            status: p.status ? Number(p.status) : undefined,
            organizationId,
          });
          const page = unwrapPage(r);
          setMetricStats({
            totalCount: page.total,
            issuedCount: Number(r.summary?.issuedCount || 0),
            amountsByBaseCurrency: r.summary?.amountsByBaseCurrency ?? [],
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
        onOk={createInvoice}
      />

      <InvoiceIssueModal
        open={Boolean(issueTarget)}
        submitting={submitting}
        issueForm={issueForm}
        issueTarget={issueTarget}
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

      <InvoiceCancelModal
        open={Boolean(cancelTarget)}
        submitting={submitting}
        cancelForm={cancelForm}
        cancelTarget={cancelTarget}
        onCancel={() => setCancelTarget(undefined)}
        onOk={cancelInvoice}
      />

      <InvoiceDetailDrawer
        detail={detail}
        onClose={() => setDetail(undefined)}
      />
    </>
  );
}
