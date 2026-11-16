import { HistoryOutlined, StopOutlined } from '@ant-design/icons';
import {
  useInfiniteQuery,
  useMutation,
  useQueryClient,
} from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Descriptions,
  Drawer,
  Form,
  Input,
  Modal,
  Space,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import React, { useEffect, useRef, useState } from 'react';
import { useAccess } from '@/app/access';
import { useUnsavedChangesGuard } from '@/components/layout/useUnsavedChangesGuard';
import { ResponsiveTable as Table } from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import {
  OrderBusinessType,
  SeaDocumentEventType,
  SeaDocumentStructure,
  SeaDocumentType,
  SeaDocumentVersionSource,
  SeaHouseBillStatus,
} from '@/enums.generated';
import {
  seaDocumentServiceExecuteSeaDocumentAmendment,
  seaDocumentServiceExecuteSeaDocumentVoid,
  seaDocumentServiceListSeaDocumentEvents,
  seaDocumentServiceListSeaHouseBillVersions,
  seaDocumentServiceListSeaMasterBillVersions,
  seaDocumentServicePreviewSeaDocumentAmendment,
  seaDocumentServicePreviewSeaDocumentVoid,
} from '@/services/roncin/seaDocumentService';
import { ensureListResponse, isWriteResponse, unwrapList } from '@/utils/api';
import { formatDate } from '@/utils/format';
import { generateUUID } from '@/utils/uuid';
import SeaExternalConfirmationFields, {
  buildSeaExternalConfirmation,
  type SeaExternalConfirmationFormValues,
} from './SeaExternalConfirmationFields';

type ActionMode = 'amendment' | 'void';
type ChangePreview =
  | API.SeaDocumentAmendmentPreview
  | API.SeaDocumentVoidPreview;

interface SeaDocumentHistoryActionsProps {
  businessType?: number;
  orderId: string;
  orderVersion: string;
  documentType: SeaDocumentType;
  documentId: string;
  documentNo: string;
  documentVersion: string;
  currentVersionId?: string;
  documentStatus?: string;
  currentHouseBill?: API.SeaHouseBill;
  getAmendmentInput: () => API.SeaDocumentAmendmentInput;
  onSuccess: () => Promise<void> | void;
  disabled?: boolean;
}

interface ActionFormValues extends SeaExternalConfirmationFormValues {
  reason: string;
}

interface PreviewPayload {
  values: ActionFormValues;
  amendmentInput?: API.SeaDocumentAmendmentInput;
}

const sourceText: Record<number, string> = {
  [SeaDocumentVersionSource.SEA_DOCUMENT_VERSION_SOURCE_ORDER_LOCK]: '订单锁定',
  [SeaDocumentVersionSource.SEA_DOCUMENT_VERSION_SOURCE_AMENDMENT]: '改单',
  [SeaDocumentVersionSource.SEA_DOCUMENT_VERSION_SOURCE_VOID]: '作废',
  [SeaDocumentVersionSource.SEA_DOCUMENT_VERSION_SOURCE_MODE_CHANGE]:
    '模式切换',
  [SeaDocumentVersionSource.SEA_DOCUMENT_VERSION_SOURCE_DRAFT_SAVE]: '草稿保存',
};

const eventText: Record<number, string> = {
  [SeaDocumentEventType.SEA_DOCUMENT_EVENT_TYPE_AMENDMENT]: '改单',
  [SeaDocumentEventType.SEA_DOCUMENT_EVENT_TYPE_VOID]: '作废',
  [SeaDocumentEventType.SEA_DOCUMENT_EVENT_TYPE_MODE_CHANGE]: '模式切换',
};

/** 历史列表单页条数，与后端列表分页上限一致。 */
const HISTORY_PAGE_SIZE = 200;

/** 只展示本单证的事件；模式切换影响全部单证，始终保留。 */
function isRelevantEvent(event: API.SeaDocumentEvent, documentId: string) {
  return (
    event.documentId === documentId ||
    event.eventType === SeaDocumentEventType.SEA_DOCUMENT_EVENT_TYPE_MODE_CHANGE
  );
}

function createIdempotencyKey() {
  return `sea-document-${generateUUID()}`;
}

function documentModeText(mode?: number) {
  if (mode === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE)
    return 'HOUSE';
  if (mode === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT)
    return 'DIRECT';
  return '-';
}

function PreviewResult({ preview }: { preview: ChangePreview }) {
  const differences = preview.differences ?? [];
  const impacts = preview.impacts ?? [];

  const differenceColumns: ColumnsType<API.SeaDocumentFieldDifference> = [
    { title: '字段', dataIndex: 'label', width: 140 },
    { title: '变更前', dataIndex: 'beforeValue' },
    { title: '变更后', dataIndex: 'afterValue' },
  ];
  const differenceSettings = useColumnSettings<
    ColumnsType<API.SeaDocumentFieldDifference>[number]
  >({
    tableKey: 'orders:doc-field-differences',
    columns: differenceColumns,
  });

  const impactColumns: ColumnsType<API.SeaDocumentDownstreamImpact> = [
    { title: '事实类型', dataIndex: 'factType', width: 130 },
    { title: '编号', dataIndex: 'referenceNo', width: 160 },
    { title: '影响', dataIndex: 'message' },
    {
      title: '结论',
      dataIndex: 'blocksExecution',
      width: 90,
      render: (blocked: boolean) => (
        <Tag color={blocked ? 'error' : 'success'}>
          {blocked ? '阻断' : '可执行'}
        </Tag>
      ),
    },
  ];
  const impactSettings = useColumnSettings<
    ColumnsType<API.SeaDocumentDownstreamImpact>[number]
  >({
    tableKey: 'orders:doc-downstream-impacts',
    columns: impactColumns,
  });

  return (
    <Space orientation="vertical" size={12} style={{ width: '100%' }}>
      <Alert
        showIcon
        type={preview.executable ? 'success' : 'error'}
        title={
          preview.executable ? '预览通过，可以执行' : '存在阻断事实，不能执行'
        }
        description={`基线：${preview.baseVersion?.documentNo ?? '-'} / v${preview.baseVersion?.versionNo ?? '-'}`}
      />
      <div
        style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}
      >
        {differenceSettings.entry}
      </div>
      <Table<API.SeaDocumentFieldDifference>
        size="small"
        rowKey={(row) => `${row.field ?? ''}-${row.label ?? ''}`}
        pagination={false}
        dataSource={differences}
        columns={differenceSettings.columns}
      />
      {impacts.length > 0 ? (
        <>
          <div
            style={{
              display: 'flex',
              justifyContent: 'flex-end',
              marginBottom: 8,
            }}
          >
            {impactSettings.entry}
          </div>
          <Table<API.SeaDocumentDownstreamImpact>
            size="small"
            rowKey={(row) => `${row.factType}-${row.referenceId}`}
            pagination={false}
            dataSource={impacts}
            columns={impactSettings.columns}
          />
        </>
      ) : null}
    </Space>
  );
}

export default function SeaDocumentHistoryActions({
  businessType = OrderBusinessType.BUSINESS_TYPE_SE,
  orderId,
  orderVersion,
  documentType,
  documentId,
  documentNo,
  documentVersion,
  currentVersionId,
  documentStatus,
  currentHouseBill,
  getAmendmentInput,
  onSuccess,
  disabled = false,
}: SeaDocumentHistoryActionsProps) {
  const access = useAccess();
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [previewRevision, setPreviewRevision] = useState(0);
  const [mode, setMode] = useState<ActionMode | null>(null);
  const { markDirty } = useUnsavedChangesGuard({
    enabled: mode !== null,
    identity: `${documentId}:${mode}`,
  });
  const previewIdentity = JSON.stringify([
    orderId,
    documentType,
    documentId,
    orderVersion,
    documentVersion,
    currentVersionId,
  ]);
  const previewMutation = useMutation({
    mutationFn: async ({
      action,
      payload,
      revision,
    }: {
      action: ActionMode;
      payload: PreviewPayload;
      revision: number;
    }) => {
      let response:
        | Awaited<
            ReturnType<typeof seaDocumentServicePreviewSeaDocumentAmendment>
          >
        | Awaited<ReturnType<typeof seaDocumentServicePreviewSeaDocumentVoid>>;
      if (action === 'amendment') {
        if (!payload.amendmentInput) throw new Error('缺少改单预览内容');
        response = await seaDocumentServicePreviewSeaDocumentAmendment(
          { orderId },
          {
            ...buildCommon(payload.values.reason),
            input: payload.amendmentInput,
          },
        );
      } else {
        response = await seaDocumentServicePreviewSeaDocumentVoid(
          { orderId },
          buildCommon(payload.values.reason),
        );
      }
      if (!response?.data || response.success === false)
        throw new Error('预览失败');
      return {
        preview: response.data as ChangePreview,
        payload,
        revision,
        identity: previewIdentity,
      };
    },
    meta: { silent: true },
  });
  const acceptedPreview =
    previewMutation.data?.revision === previewRevision &&
    previewMutation.data.identity === previewIdentity
      ? previewMutation.data
      : undefined;
  const sessionRef = useRef({
    orderId,
    documentId,
    revision: previewRevision,
    identity: previewIdentity,
    mode,
    active: true,
  });
  sessionRef.current = {
    orderId,
    documentId,
    revision: previewRevision,
    identity: previewIdentity,
    mode,
    active: true,
  };
  useEffect(() => {
    sessionRef.current.active = true;
    return () => {
      sessionRef.current.active = false;
    };
  }, []);
  const preview = acceptedPreview?.preview;
  const previewPayload = acceptedPreview?.payload;
  const previewing = previewMutation.isPending;
  const [executing, setExecuting] = useState(false);
  const [idempotencyKey, setIdempotencyKey] = useState('');

  const isHouse = documentType === SeaDocumentType.SEA_DOCUMENT_TYPE_HOUSE_BILL;
  const terminal =
    documentStatus === 'VOIDED' ||
    currentHouseBill?.status ===
      SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_VOIDED;
  const commandReady = Boolean(
    orderVersion && documentVersion && currentVersionId && !terminal,
  );

  const queryClient = useQueryClient();
  const versionsKey = [
    'orders',
    'sea-document-versions',
    { orderId, documentId, documentType, open: drawerOpen },
  ] as const;
  const eventsKey = [
    'orders',
    'sea-document-events',
    { orderId, documentId, open: drawerOpen },
  ] as const;
  const versionQuery = useInfiniteQuery({
    queryKey: versionsKey,
    enabled: drawerOpen && Boolean(orderId && documentId),
    initialPageParam: 1,
    meta: { silent: true },
    queryFn: async ({ pageParam, signal }) => {
      const response = isHouse
        ? await seaDocumentServiceListSeaHouseBillVersions(
            {
              orderId,
              houseBillId: documentId,
              page: pageParam,
              pageSize: HISTORY_PAGE_SIZE,
            },
            { signal, skipErrorHandler: true },
          )
        : await seaDocumentServiceListSeaMasterBillVersions(
            { orderId, page: pageParam, pageSize: HISTORY_PAGE_SIZE },
            { signal, skipErrorHandler: true },
          );
      const rows = unwrapList(ensureListResponse(response, '读取单证历史失败'));
      return { rows, total: Number(response.total ?? rows.length) };
    },
    getNextPageParam: (lastPage, pages) =>
      lastPage.rows.length &&
      pages.reduce((count, page) => count + page.rows.length, 0) <
        lastPage.total
        ? pages.length + 1
        : undefined,
  });
  const eventQuery = useInfiniteQuery({
    queryKey: eventsKey,
    enabled: drawerOpen && Boolean(orderId && documentId),
    initialPageParam: 1,
    meta: { silent: true },
    queryFn: async ({ pageParam, signal }) => {
      const response = await seaDocumentServiceListSeaDocumentEvents(
        { orderId, page: pageParam, pageSize: HISTORY_PAGE_SIZE },
        { signal, skipErrorHandler: true },
      );
      const rows = unwrapList(ensureListResponse(response, '读取单证历史失败'));
      return { rows, total: Number(response.total ?? rows.length) };
    },
    getNextPageParam: (lastPage, pages) =>
      lastPage.rows.length &&
      pages.reduce((count, page) => count + page.rows.length, 0) <
        lastPage.total
        ? pages.length + 1
        : undefined,
  });
  const versions = versionQuery.data?.pages.flatMap((page) => page.rows) ?? [];
  const versionsTotal = versionQuery.data?.pages.at(-1)?.total ?? 0;
  const events =
    eventQuery.data?.pages.flatMap((page) =>
      page.rows.filter((event) => isRelevantEvent(event, documentId)),
    ) ?? [];
  const eventsTotal = eventQuery.data?.pages.at(-1)?.total ?? 0;
  const historyLoading = versionQuery.isFetching || eventQuery.isFetching;
  const versionsLoading = versionQuery.isFetchingNextPage;
  const eventsLoading = eventQuery.isFetchingNextPage;
  const loadMoreVersions = () => void versionQuery.fetchNextPage();
  const loadMoreEvents = () => void eventQuery.fetchNextPage();
  const openHistory = async () => {
    setDrawerOpen(true);
    await Promise.all([
      queryClient.resetQueries({
        queryKey: [
          ...versionsKey.slice(0, 2),
          { orderId, documentId, documentType, open: true },
        ],
        exact: true,
      }),
      queryClient.resetQueries({
        queryKey: [
          ...eventsKey.slice(0, 2),
          { orderId, documentId, open: true },
        ],
        exact: true,
      }),
    ]);
  };

  const openAction = (nextMode: ActionMode) => {
    setMode(nextMode);
    setPreviewRevision((value) => value + 1);
    previewMutation.reset();
    setIdempotencyKey(createIdempotencyKey());
    form.setFieldsValue({
      reason: undefined,
      confirmedByParty: undefined,
      confirmedAt: dayjs(),
      confirmationNote: undefined,
      confirmationAttachmentId: undefined,
    });
  };

  const closeAction = () => {
    if (previewing || executing) return;
    setMode(null);
    previewMutation.reset();
    form.resetFields();
  };

  const buildCommon = (reason: string) => ({
    orderId,
    documentType,
    documentId,
    expectedOrderVersion: orderVersion,
    expectedDocumentVersion: documentVersion,
    expectedCurrentVersionId: currentVersionId as string,
    reason: reason.trim(),
  });

  const handlePreview = async () => {
    if (!mode || !commandReady || disabled) return;
    const revision = previewRevision;
    const stillCurrent = () =>
      sessionRef.current.active &&
      sessionRef.current.orderId === orderId &&
      sessionRef.current.documentId === documentId &&
      sessionRef.current.identity === previewIdentity &&
      sessionRef.current.mode === mode &&
      sessionRef.current.revision === revision;
    const values = (await form.validateFields()) as ActionFormValues;
    if (!stillCurrent()) return;
    previewMutation.reset();
    try {
      await previewMutation.mutateAsync({
        action: mode,
        revision: previewRevision,
        payload: {
          values,
          amendmentInput:
            mode === 'amendment'
              ? structuredClone(getAmendmentInput())
              : undefined,
        },
      });
    } catch (error: unknown) {
      if (stillCurrent())
        message.error(error instanceof Error ? error.message : '预览失败');
    }
  };

  const handleExecute = async (): Promise<false | undefined> => {
    if (!mode || !preview?.executable || !previewPayload || disabled) return;
    const { values, amendmentInput } = previewPayload;
    setExecuting(true);
    try {
      if (mode === 'amendment') {
        if (!amendmentInput) return;
        const writeResponse1 =
          await seaDocumentServiceExecuteSeaDocumentAmendment(
            { orderId },
            {
              ...buildCommon(values.reason),
              idempotencyKey,
              input: amendmentInput,
              confirmation: buildSeaExternalConfirmation(values),
            },
          );
        if (!isWriteResponse(writeResponse1, true)) return false;
      } else {
        const writeResponse2 = await seaDocumentServiceExecuteSeaDocumentVoid(
          { orderId },
          {
            ...buildCommon(values.reason),
            idempotencyKey,
            confirmation: buildSeaExternalConfirmation(values),
          },
        );
        if (!isWriteResponse(writeResponse2, true)) return false;
      }
      message.success(mode === 'amendment' ? '改单版本已发布' : '单证已作废');
      setMode(null);
      previewMutation.reset();
      form.resetFields();
      await onSuccess();
    } catch (error: unknown) {
      message.error(error instanceof Error ? error.message : '执行失败');
    } finally {
      setExecuting(false);
    }

    return undefined;
  };

  const modeTitle =
    mode === 'amendment' ? `改单：${documentNo}` : `作废：${documentNo}`;

  const versionColumns: ColumnsType<API.SeaDocumentVersion> = [
    {
      title: '版本',
      dataIndex: 'versionNo',
      width: 72,
      render: (v) => `v${v}`,
    },
    {
      title: '来源',
      dataIndex: 'source',
      width: 110,
      render: (v) => sourceText[v] ?? '-',
    },
    { title: '状态', dataIndex: 'status', width: 100 },
    { title: '原因', dataIndex: 'reason' },
    {
      title: '形成时间',
      dataIndex: 'createdAt',
      width: 190,
      render: (v: string) => formatDate(v),
    },
  ];
  const versionSettings = useColumnSettings<
    ColumnsType<API.SeaDocumentVersion>[number]
  >({
    tableKey: 'orders:doc-versions',
    columns: versionColumns,
  });

  const eventColumns: ColumnsType<API.SeaDocumentEvent> = [
    {
      title: '类型',
      dataIndex: 'eventType',
      width: 110,
      render: (v) => eventText[v] ?? '-',
    },
    { title: '单证', dataIndex: 'documentNo', width: 150 },
    {
      title: '模式变化',
      key: 'modeChange',
      width: 200,
      render: (_, row) =>
        row.previousMode !== undefined && row.targetMode !== undefined
          ? `${documentModeText(row.previousMode)} → ${documentModeText(row.targetMode)}`
          : '-',
    },
    { title: '原因', dataIndex: 'reason' },
    {
      title: '时间',
      dataIndex: 'createdAt',
      width: 190,
      render: (v: string) => formatDate(v),
    },
  ];
  const eventSettings = useColumnSettings<
    ColumnsType<API.SeaDocumentEvent>[number]
  >({
    tableKey: 'orders:doc-events',
    columns: eventColumns,
  });

  return (
    <>
      <Space wrap>
        <Button size="small" icon={<HistoryOutlined />} onClick={openHistory}>
          版本与事件
        </Button>
        {access.canOrder(businessType, 'amend') ? (
          <Button
            size="small"
            disabled={disabled || !commandReady}
            onClick={() => openAction('amendment')}
          >
            单改
          </Button>
        ) : null}
        {access.canOrder(businessType, 'void') ? (
          <Button
            size="small"
            danger
            icon={<StopOutlined />}
            disabled={disabled || !commandReady}
            onClick={() => openAction('void')}
          >
            作废
          </Button>
        ) : null}
      </Space>

      <Drawer
        size="large"
        title={`${documentNo} · 不可变版本与事件`}
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
      >
        {versionQuery.isError || eventQuery.isError ? (
          <Alert type="error" title="读取单证历史失败" />
        ) : null}
        <Typography.Title level={5}>版本历史</Typography.Title>
        <div
          style={{
            display: 'flex',
            justifyContent: 'flex-end',
            marginBottom: 8,
          }}
        >
          {versionSettings.entry}
        </div>
        <Table<API.SeaDocumentVersion>
          loading={historyLoading}
          size="small"
          rowKey="id"
          pagination={false}
          dataSource={versions}
          columns={versionSettings.columns}
          expandable={{
            expandedRowRender: (row) => (
              <Descriptions size="small" column={2} bordered>
                <Descriptions.Item label="不可变版本 ID" span={2}>
                  {row.id}
                </Descriptions.Item>
                <Descriptions.Item label="单证号">
                  {row.documentNo}
                </Descriptions.Item>
                <Descriptions.Item label="实体版本">
                  v{row.sourceEntityVersion}
                </Descriptions.Item>
                {row.documentType ===
                  SeaDocumentType.SEA_DOCUMENT_TYPE_MASTER_BILL && (
                  <Descriptions.Item label="船公司" span={2}>
                    {row.shippingLineName || row.shippingLineId || '-'}
                  </Descriptions.Item>
                )}
                <Descriptions.Item label="船名">
                  {row.vesselName || '-'}
                </Descriptions.Item>
                <Descriptions.Item label="航次">
                  {row.voyageNo || '-'}
                </Descriptions.Item>
                <Descriptions.Item label="提单内容" span={2}>
                  <Typography.Text code>
                    {JSON.stringify(row.content ?? {}, null, 2)}
                  </Typography.Text>
                </Descriptions.Item>
                {row.printDetails && (
                  <Descriptions.Item label="制单印字资料" span={2}>
                    <Typography.Text code>
                      {JSON.stringify(row.printDetails, null, 2)}
                    </Typography.Text>
                  </Descriptions.Item>
                )}
              </Descriptions>
            ),
          }}
        />
        {versionQuery.hasNextPage ? (
          <Button
            size="small"
            block
            style={{ marginTop: 8 }}
            loading={versionsLoading}
            onClick={loadMoreVersions}
          >
            {`加载更多（共 ${versionsTotal} 条）`}
          </Button>
        ) : null}

        <Typography.Title level={5} style={{ marginTop: 24 }}>
          业务事件
        </Typography.Title>
        <div
          style={{
            display: 'flex',
            justifyContent: 'flex-end',
            marginBottom: 8,
          }}
        >
          {eventSettings.entry}
        </div>
        <Table<API.SeaDocumentEvent>
          loading={historyLoading}
          size="small"
          rowKey="id"
          pagination={false}
          dataSource={events}
          columns={eventSettings.columns}
        />
        {eventQuery.hasNextPage ? (
          <Button
            size="small"
            block
            style={{ marginTop: 8 }}
            loading={eventsLoading}
            onClick={loadMoreEvents}
          >
            {`加载更多（共 ${eventsTotal} 条）`}
          </Button>
        ) : null}
      </Drawer>

      <Modal
        width={820}
        title={modeTitle}
        open={mode !== null}
        onCancel={closeAction}
        destroyOnHidden
        footer={[
          <Button
            key="cancel"
            onClick={closeAction}
            disabled={previewing || executing}
          >
            取消
          </Button>,
          <Button
            key="preview"
            onClick={handlePreview}
            loading={previewing}
            disabled={disabled || executing}
          >
            重新预览最终差异
          </Button>,
          <Button
            key="execute"
            type="primary"
            danger={mode === 'void'}
            onClick={handleExecute}
            loading={executing}
            disabled={disabled || !preview?.executable || previewing}
          >
            确认执行
          </Button>,
        ]}
      >
        <Alert
          type="warning"
          showIcon
          title="执行前必须先预览"
          description="任何输入变化都会使已有预览失效；只有服务端返回可执行并展示最终逐字段差异后，才能执行。"
          style={{ marginBottom: 16 }}
        />
        <Form
          form={form}
          layout="vertical"
          onValuesChange={() => {
            markDirty();
            setPreviewRevision((value) => value + 1);
            previewMutation.reset();
          }}
        >
          <Form.Item
            name="reason"
            label="原因"
            rules={[
              { required: true, whitespace: true, message: '请输入原因' },
            ]}
          >
            <Input.TextArea maxLength={500} showCount rows={3} />
          </Form.Item>
          <SeaExternalConfirmationFields orderId={orderId} />
        </Form>
        {preview ? <PreviewResult preview={preview} /> : null}
      </Modal>
    </>
  );
}
