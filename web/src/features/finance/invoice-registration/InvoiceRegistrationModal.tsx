import type { ProColumns } from '@ant-design/pro-components';
import { useQueries, useQuery, useQueryClient } from '@tanstack/react-query';
import { Alert, App, Button, Form, Input, Space, Typography } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useRef, useState } from 'react';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import { GuardedModalForm, ResponsiveProTable } from '@/components/ui';
import { useAsyncGuard } from '@/hooks/useAsyncGuard';
import { usePageForeground } from '@/hooks/usePageForeground';
import {
  settlementServiceGetInvoiceRevisionImpact,
  settlementServiceListInvoiceCreationBills,
  settlementServiceRegisterInvoices,
  settlementServiceReviseInvoiceRegistrations,
} from '@/services/roncin/settlementService';
import { toTableRequest } from '@/utils/api';
import { getErrorMessage } from '@/utils/errorMessage';
import { formatAmount } from '@/utils/format';
import { generateUUID } from '@/utils/uuid';
import InvoiceRegistrationEditor from './InvoiceRegistrationEditor';
import {
  type InvoiceRegistrationDraft,
  invoiceBillSource,
  invoiceRegistrationPayload,
} from './types';

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
  bills?: API.FinanceBill[];
  reviseInvoice?: API.FinanceInvoice;
};
type Values = {
  records: (InvoiceRegistrationDraft & { sourceInvoiceId?: string })[];
  reason?: string;
};

/** 每次打开重建会话；关闭和切公司后旧写响应不能影响新表单。 */
export default function InvoiceRegistrationModal(props: Props) {
  const { initialState } = useInitialState();
  const org = initialState?.currentUser?.currentOrganization?.id;
  const intentKeys = useRef(new Map<string, string>());
  if (!props.open) return null;
  return (
    <RegistrationSession
      key={`${initialState?.currentUser?.id}:${org}:${props.reviseInvoice?.id ?? ''}:${props.bills?.map((bill) => bill.id).join(',') ?? ''}`}
      {...props}
      organizationId={org ?? ''}
      intentKeys={intentKeys.current}
    />
  );
}
function RegistrationSession({
  onOpenChange,
  onSuccess,
  bills: initialBills,
  reviseInvoice,
  organizationId,
  intentKeys,
}: Props & { organizationId: string; intentKeys: Map<string, string> }) {
  const access = useAccess();
  const { message, modal } = App.useApp();
  const queryClient = useQueryClient();
  const guard = useAsyncGuard();
  const [form] = Form.useForm<Values>();
  const [bills, setBills] = useState(initialBills ?? []);
  const [baseline, setBaseline] = useState<API.FinanceInvoiceRevisionImpact>();
  const [failure, setFailure] = useState<string>();
  const [changed, setChanged] = useState(false);
  const inputRevision = useRef(0);
  const resume = useRef<() => void>(() => undefined);
  const foreground = usePageForeground(() => resume.current());
  const authorized =
    access.canOperateOrganization(organizationId) &&
    (reviseInvoice
      ? access.canUpdateFinanceInvoices
      : access.canCreateFinanceInvoices);
  const validBills = bills.every(
    (bill) => bill.organizationId === organizationId,
  );
  const impact = useQuery({
    queryKey: [
      'finance',
      'invoice-revision-initial',
      organizationId,
      reviseInvoice?.id,
    ],
    enabled: authorized && Boolean(reviseInvoice?.id),
    staleTime: 0,
    gcTime: 0,
    meta: { errorMessage: '加载全部发票分配与结算影响失败' },
    queryFn: async ({ signal }) => {
      const response = await settlementServiceGetInvoiceRevisionImpact(
        { id: reviseInvoice?.id ?? '' },
        { signal, skipErrorHandler: true },
      );
      if (
        !response?.data?.invoice?.id ||
        response.success === false ||
        response.data.invoice.organizationId !== organizationId ||
        response.data.bills?.some(
          (bill) => bill.organizationId !== organizationId,
        )
      )
        throw new Error('发票影响范围不正确');
      return response.data;
    },
  });
  useEffect(() => {
    if (!impact.data || baseline) return;
    const invoice = impact.data.invoice;
    if (!invoice) return;
    setBaseline(impact.data);
    setBills(impact.data.bills ?? []);
    form.setFieldsValue({
      records: [
        {
          sourceInvoiceId: invoice.id,
          invoiceProfileId: invoice.invoiceProfileId ?? '',
          invoiceType: invoice.invoiceType ?? '',
          taxInvoiceNo: invoice.taxInvoiceNo ?? '',
          invoiceDate: dayjs(invoice.invoiceDate),
          totalAmount: invoice.totalAmount ?? '',
          note: invoice.note,
          allocations: (invoice.billLinks ?? [])
            .filter((link) => link.active === true)
            .map((link) => ({
              billId: link.billId ?? '',
              amount: link.amount ?? '',
              fees: (link.fees ?? []).map((fee) => ({
                orderFeeId: fee.orderFeeId ?? '',
                amount: fee.amount ?? '',
                taxAmount: fee.taxAmount ?? '0',
              })),
            })),
        },
      ],
    });
  }, [impact.data, baseline, form]);
  const probes = useQueries({
    queries: bills.map((bill) => ({
      queryKey: [
        'finance',
        'invoice-registration-bill-probe',
        organizationId,
        bill.id,
      ],
      enabled:
        authorized &&
        validBills &&
        foreground &&
        !reviseInvoice &&
        Boolean(bill.id),
      refetchInterval: foreground ? 15_000 : false,
      refetchIntervalInBackground: false,
      meta: { silent: true },
      queryFn: async ({ signal }: { signal: AbortSignal }) => {
        const response = await settlementServiceListInvoiceCreationBills(
          { organizationId, billId: bill.id, page: 1, pageSize: 1 },
          { signal, skipErrorHandler: true },
        );
        if (
          !response ||
          response.success === false ||
          (response.data === undefined && response.success !== true)
        )
          throw new Error('账单变动检查失败');
        return response.data?.[0]?.version ?? 'unavailable';
      },
    })),
  });
  const revisionProbe = useQuery({
    queryKey: [
      'finance',
      'invoice-revision-probe',
      organizationId,
      reviseInvoice?.id,
    ],
    enabled: authorized && foreground && Boolean(baseline?.invoice?.id),
    refetchInterval: foreground ? 15_000 : false,
    meta: { silent: true },
    queryFn: async ({ signal }) => {
      const response = await settlementServiceGetInvoiceRevisionImpact(
        { id: reviseInvoice?.id ?? '' },
        { signal, skipErrorHandler: true },
      );
      if (!response?.data?.invoice?.version || response.success === false)
        throw new Error('发票变动检查失败');
      return response.data;
    },
  });
  resume.current = () => {
    if (!reviseInvoice) for (const probe of probes) void probe.refetch();
    if (baseline) void revisionProbe.refetch();
  };
  const detected =
    probes.some(
      (probe, index) =>
        probe.data !== undefined &&
        bills[index].version !== undefined &&
        (probe.data === 'unavailable' ||
          BigInt(probe.data) > BigInt(bills[index].version ?? '0')),
    ) ||
    (revisionProbe.data !== undefined &&
      (revisionProbe.data.invoice?.version !== baseline?.invoice?.version ||
        bills.some(
          (bill) =>
            revisionProbe.data?.bills?.find((latest) => latest.id === bill.id)
              ?.version !== bill.version,
        )));
  useEffect(() => {
    if (detected) setChanged(true);
  }, [detected]);
  const sources = bills.map(invoiceBillSource);
  const columns: ProColumns<API.FinanceBill>[] = [
    {
      title: '关键字',
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: { placeholder: '账单号、结算单位、拼音' },
    },
    { title: '账单编号', dataIndex: 'billNo', search: false },
    { title: '结算单位', dataIndex: 'settlementPartyName', search: false },
    {
      title: '可登记余额',
      dataIndex: 'invoiceableAmount',
      search: false,
      render: (_, bill) =>
        `${formatAmount(bill.invoiceableAmount)} ${bill.currency ?? ''}`,
    },
  ];
  const canSubmit =
    authorized &&
    validBills &&
    bills.length > 0 &&
    !changed &&
    !detected &&
    (!reviseInvoice || baseline?.canRevise === true);
  return (
    <GuardedModalForm<Values>
      title={reviseInvoice ? '修订已有发票登记' : '登记已有发票'}
      open
      onOpenChange={onOpenChange}
      onValuesChange={() => {
        inputRevision.current += 1;
      }}
      form={form}
      width={1100}
      dateFormatter={false}
      initialValues={{ records: [] }}
      modalProps={{ destroyOnHidden: true }}
      submitter={{
        searchConfig: { submitText: '保存登记' },
        submitButtonProps: { disabled: !canSubmit },
      }}
      onFinish={async (values) => {
        if (!canSubmit) return false;
        if (!reviseInvoice && !values.records?.length) {
          setFailure('请至少添加一张发票');
          return false;
        }
        let records: API.InvoiceRegistrationInput[];
        try {
          records = invoiceRegistrationPayload(values.records ?? [], sources);
        } catch (error) {
          setFailure(getErrorMessage(error, '请核对明确填写的分配金额'));
          return false;
        }
        const previous = baseline?.invoice;
        const body = previous
          ? {
              reason: values.reason?.trim() ?? '',
              billVersions: sources.map((bill) => ({
                billId: bill.id,
                expectedVersion: bill.version,
              })),
              entries: [
                ...(values.records ?? []).map((record, index) => ({
                  id: record.sourceInvoiceId,
                  expectedVersion: record.sourceInvoiceId
                    ? previous.version
                    : undefined,
                  registration: records[index],
                })),
                ...(!(values.records ?? []).some(
                  (record) => record.sourceInvoiceId === previous.id,
                )
                  ? [
                      {
                        id: previous.id,
                        expectedVersion: previous.version,
                        remove: true,
                      },
                    ]
                  : []),
              ],
            }
          : { records };
        const fingerprint = JSON.stringify({ organizationId, body });
        let key = intentKeys.get(fingerprint);
        if (!key) {
          key = generateUUID();
          intentKeys.set(fingerprint, key);
        }
        const submittingRevision = inputRevision.current;
        let saved = false;
        try {
          await guard.run(
            () =>
              previous
                ? settlementServiceReviseInvoiceRegistrations({
                    ...body,
                    idempotencyKey: key,
                  } as API.ReviseInvoiceRegistrationsRequest)
                : settlementServiceRegisterInvoices({
                    records,
                    idempotencyKey: key,
                  }),
            (response) => {
              if (
                !response ||
                response.success === false ||
                !response.data?.length ||
                response.data.some(
                  (item) => !item.id || item.organizationId !== organizationId,
                )
              ) {
                setFailure('登记结果未确认，请保留输入并重试');
                return;
              }
              if (inputRevision.current !== submittingRevision) {
                setFailure(
                  '原提交已保存，当前新增输入已保留。请核对登记结果后重新打开。',
                );
                setChanged(true);
                onSuccess();
                return;
              }
              intentKeys.delete(fingerprint);
              saved = true;
              message.success('发票登记已保存');
              void queryClient.invalidateQueries({ queryKey: ['finance'] });
              onSuccess();
              onOpenChange(false);
            },
          );
        } catch (error) {
          setFailure(getErrorMessage(error, '登记失败，输入已保留'));
        }
        return saved;
      }}
    >
      {failure && (
        <Alert
          type="error"
          showIcon
          title={failure}
          style={{ marginBottom: 12 }}
        />
      )}
      {(changed || detected) && (
        <Alert
          type="warning"
          showIcon
          title="这个账单或发票已经发生变动，请重新刷新。"
          action={
            <Button
              onClick={() =>
                modal.confirm({
                  title: '重新载入会覆盖未保存内容，是否继续？',
                  onOk: () => {
                    onOpenChange(false);
                  },
                })
              }
            >
              关闭后重新打开
            </Button>
          }
          style={{ marginBottom: 12 }}
        />
      )}
      {!authorized && <Alert type="warning" title="当前公司或权限不允许登记" />}
      {reviseInvoice && !baseline && (
        <Typography.Paragraph>正在加载完整分配与结算影响…</Typography.Paragraph>
      )}
      {baseline && (
        <>
          <Typography.Paragraph>
            本次修订影响全部 {bills.length} 张账单：
            {bills.map((bill) => bill.billNo).join('、')}
          </Typography.Paragraph>
          {baseline.canRevise !== true && (
            <Alert
              type="warning"
              showIcon
              title="已有有效核销或已确认对冲，须先核对并按原权限完成必要反向处理，再重新打开修订。"
              description={
                <Space orientation="vertical">
                  {(baseline.dependencies ?? []).map((item) => (
                    <Typography.Text
                      key={`${item.kind}:${item.id}:${item.billId}`}
                    >
                      {item.kind === 'VERIFICATION' ? '核销' : '对冲'}{' '}
                      {item.documentNo} · 账单 {item.billNo} ·{' '}
                      {formatAmount(item.amount)} {item.currency}
                    </Typography.Text>
                  ))}
                </Space>
              }
            />
          )}
          <Form.Item
            name="reason"
            label="修订原因"
            rules={[
              { required: true, whitespace: true, message: '请填写修订原因' },
              { max: 500 },
            ]}
          >
            <Input.TextArea />
          </Form.Item>
        </>
      )}
      {!reviseInvoice && !initialBills && (
        <ResponsiveProTable<API.FinanceBill>
          rowKey="id"
          columns={columns}
          options={false}
          size="small"
          pagination={{ defaultPageSize: 20 }}
          rowSelection={{
            selectedRowKeys: bills.map((bill) => bill.id ?? ''),
            preserveSelectedRowKeys: true,
            onChange: (keys, rows) => {
              inputRevision.current += 1;
              const all = new Map(
                [...bills, ...rows].map((bill) => [bill.id, bill]),
              );
              setBills(
                keys
                  .map((key) => all.get(String(key)))
                  .filter((bill): bill is API.FinanceBill => Boolean(bill)),
              );
            },
            getCheckboxProps: (bill) => ({
              disabled:
                !authorized ||
                bill.organizationId !== organizationId ||
                Boolean(
                  bills[0] &&
                    (bill.direction !== bills[0].direction ||
                      bill.settlementPartyId !== bills[0].settlementPartyId ||
                      bill.currency !== bills[0].currency ||
                      bill.baseCurrency !== bills[0].baseCurrency),
                ),
            }),
          }}
          request={async (params) => {
            if (!authorized) return { data: [], total: 0, success: false };
            return toTableRequest(
              await settlementServiceListInvoiceCreationBills({
                organizationId,
                page: params.current,
                pageSize: params.pageSize,
                keyword: params.keyword,
              }),
            );
          }}
        />
      )}
      {bills.length > 0 && authorized && validBills && (
        <InvoiceRegistrationEditor
          name={['records']}
          bills={sources}
          organizationId={organizationId}
          settlementPartyId={bills[0]?.settlementPartyId ?? ''}
          allowAdd={!reviseInvoice || access.canCreateFinanceInvoices}
          profiles={
            baseline?.invoice
              ? [
                  {
                    id: baseline.invoice.invoiceProfileId,
                    invoiceTitle: baseline.invoice.invoiceTitle,
                    taxpayerIdentificationNo:
                      baseline.invoice.taxpayerIdentificationNo,
                  },
                ]
              : undefined
          }
        />
      )}
    </GuardedModalForm>
  );
}
