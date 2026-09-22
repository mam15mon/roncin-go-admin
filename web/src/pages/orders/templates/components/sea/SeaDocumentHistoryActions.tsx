import { HistoryOutlined, StopOutlined } from '@ant-design/icons';
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
  Table,
  Tag,
  Typography,
} from 'antd';
import dayjs from 'dayjs';
import React, { useCallback, useRef, useState } from 'react';
import { useAccess } from '@/app/access';
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
      <Table<API.SeaDocumentFieldDifference>
        size="small"
        rowKey={(row) => `${row.field ?? ''}-${row.label ?? ''}`}
        pagination={false}
        dataSource={differences}
        columns={[
          { title: '字段', dataIndex: 'label', width: 140 },
          { title: '变更前', dataIndex: 'beforeValue' },
          { title: '变更后', dataIndex: 'afterValue' },
        ]}
      />
      {impacts.length > 0 ? (
        <Table<API.SeaDocumentDownstreamImpact>
          size="small"
          rowKey={(row) => `${row.factType}-${row.referenceId}`}
          pagination={false}
          dataSource={impacts}
          columns={[
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
          ]}
        />
      ) : null}
    </Space>
  );
}

export default function SeaDocumentHistoryActions({
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
  const [historyLoading, setHistoryLoading] = useState(false);
  const [versions, setVersions] = useState<API.SeaDocumentVersion[]>([]);
  const [versionsTotal, setVersionsTotal] = useState(0);
  const [versionPage, setVersionPage] = useState(1);
  const [versionsLoading, setVersionsLoading] = useState(false);
  const [events, setEvents] = useState<API.SeaDocumentEvent[]>([]);
  // 事件按 documentId 过滤后展示，加载进度按接口原始返回条数计算。
  const [eventsLoadedCount, setEventsLoadedCount] = useState(0);
  const [eventsTotal, setEventsTotal] = useState(0);
  const [eventPage, setEventPage] = useState(1);
  const [eventsLoading, setEventsLoading] = useState(false);
  const [mode, setMode] = useState<ActionMode | null>(null);
  const [preview, setPreview] = useState<ChangePreview | null>(null);
  const [previewPayload, setPreviewPayload] = useState<PreviewPayload | null>(
    null,
  );
  const [previewing, setPreviewing] = useState(false);
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

  // 历史重置代次：loadHistory 递增后，进行中的加载更多响应全部作废。
  const historyGenerationRef = useRef(0);

  const fetchVersions = useCallback(
    (page: number) =>
      isHouse
        ? seaDocumentServiceListSeaHouseBillVersions({
            orderId,
            houseBillId: documentId,
            page,
            pageSize: HISTORY_PAGE_SIZE,
          })
        : seaDocumentServiceListSeaMasterBillVersions({
            orderId,
            page,
            pageSize: HISTORY_PAGE_SIZE,
          }),
    [documentId, isHouse, orderId],
  );

  const fetchEvents = useCallback(
    (page: number) =>
      seaDocumentServiceListSeaDocumentEvents({
        orderId,
        page,
        pageSize: HISTORY_PAGE_SIZE,
      }),
    [orderId],
  );

  const loadHistory = useCallback(async () => {
    const generation = ++historyGenerationRef.current;
    setHistoryLoading(true);
    setVersionPage(1);
    setEventPage(1);
    try {
      const [versionResult, eventResult] = await Promise.all([
        fetchVersions(1),
        fetchEvents(1),
      ]);
      if (generation !== historyGenerationRef.current) return;
      const versionRows = versionResult.data ?? [];
      setVersions(versionRows);
      setVersionsTotal(Number(versionResult.total ?? versionRows.length));
      const eventRows = eventResult.data ?? [];
      setEvents(
        eventRows.filter((event) => isRelevantEvent(event, documentId)),
      );
      setEventsLoadedCount(eventRows.length);
      setEventsTotal(Number(eventResult.total ?? eventRows.length));
    } catch (error: unknown) {
      message.error(
        error instanceof Error ? error.message : '读取单证历史失败',
      );
    } finally {
      if (generation === historyGenerationRef.current) {
        setHistoryLoading(false);
      }
    }
  }, [documentId, fetchEvents, fetchVersions, message]);

  const loadMoreVersions = async () => {
    const generation = historyGenerationRef.current;
    const nextPage = versionPage + 1;
    setVersionsLoading(true);
    try {
      const result = await fetchVersions(nextPage);
      if (generation !== historyGenerationRef.current) return;
      const rows = result.data ?? [];
      setVersions((previous) => [...previous, ...rows]);
      setVersionsTotal(Number(result.total ?? 0));
      setVersionPage(nextPage);
    } catch (error: unknown) {
      message.error(
        error instanceof Error ? error.message : '读取单证历史失败',
      );
    } finally {
      if (generation === historyGenerationRef.current) {
        setVersionsLoading(false);
      }
    }
  };

  const loadMoreEvents = async () => {
    const generation = historyGenerationRef.current;
    const nextPage = eventPage + 1;
    setEventsLoading(true);
    try {
      const result = await fetchEvents(nextPage);
      if (generation !== historyGenerationRef.current) return;
      const rows = result.data ?? [];
      setEvents((previous) => [
        ...previous,
        ...rows.filter((event) => isRelevantEvent(event, documentId)),
      ]);
      setEventsLoadedCount((previous) => previous + rows.length);
      setEventsTotal(Number(result.total ?? 0));
      setEventPage(nextPage);
    } catch (error: unknown) {
      message.error(
        error instanceof Error ? error.message : '读取单证历史失败',
      );
    } finally {
      if (generation === historyGenerationRef.current) {
        setEventsLoading(false);
      }
    }
  };

  const openHistory = async () => {
    setDrawerOpen(true);
    await loadHistory();
  };

  const openAction = (nextMode: ActionMode) => {
    setMode(nextMode);
    setPreview(null);
    setPreviewPayload(null);
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
    setPreview(null);
    setPreviewPayload(null);
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
    const values = (await form.validateFields()) as ActionFormValues;
    setPreviewing(true);
    setPreview(null);
    setPreviewPayload(null);
    try {
      if (mode === 'amendment') {
        const amendmentInput = getAmendmentInput();
        const result = await seaDocumentServicePreviewSeaDocumentAmendment(
          { orderId },
          { ...buildCommon(values.reason), input: amendmentInput },
        );
        setPreview(result.data ?? null);
        setPreviewPayload({ values, amendmentInput });
      } else {
        const result = await seaDocumentServicePreviewSeaDocumentVoid(
          { orderId },
          buildCommon(values.reason),
        );
        setPreview(result.data ?? null);
        setPreviewPayload({ values });
      }
    } catch (error: unknown) {
      message.error(error instanceof Error ? error.message : '预览失败');
    } finally {
      setPreviewing(false);
    }
  };

  const handleExecute = async () => {
    if (!mode || !preview?.executable || !previewPayload || disabled) return;
    const { values, amendmentInput } = previewPayload;
    setExecuting(true);
    try {
      if (mode === 'amendment') {
        if (!amendmentInput) return;
        await seaDocumentServiceExecuteSeaDocumentAmendment(
          { orderId },
          {
            ...buildCommon(values.reason),
            idempotencyKey,
            input: amendmentInput,
            confirmation: buildSeaExternalConfirmation(values),
          },
        );
      } else {
        await seaDocumentServiceExecuteSeaDocumentVoid(
          { orderId },
          {
            ...buildCommon(values.reason),
            idempotencyKey,
            confirmation: buildSeaExternalConfirmation(values),
          },
        );
      }
      message.success(mode === 'amendment' ? '改单版本已发布' : '单证已作废');
      setMode(null);
      setPreview(null);
      setPreviewPayload(null);
      form.resetFields();
      await onSuccess();
    } catch (error: unknown) {
      message.error(error instanceof Error ? error.message : '执行失败');
    } finally {
      setExecuting(false);
    }
  };

  const modeTitle =
    mode === 'amendment' ? `改单：${documentNo}` : `作废：${documentNo}`;

  return (
    <>
      <Space wrap>
        <Button size="small" icon={<HistoryOutlined />} onClick={openHistory}>
          版本与事件
        </Button>
        {access.canOrder(OrderBusinessType.BUSINESS_TYPE_SE, 'amend') ? (
          <Button
            size="small"
            disabled={disabled || !commandReady}
            onClick={() => openAction('amendment')}
          >
            单改
          </Button>
        ) : null}
        {access.canOrder(OrderBusinessType.BUSINESS_TYPE_SE, 'void') ? (
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
        <Typography.Title level={5}>版本历史</Typography.Title>
        <Table<API.SeaDocumentVersion>
          loading={historyLoading}
          size="small"
          rowKey="id"
          pagination={false}
          dataSource={versions}
          columns={[
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
          ]}
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
              </Descriptions>
            ),
          }}
        />
        {versions.length < versionsTotal ? (
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
        <Table<API.SeaDocumentEvent>
          loading={historyLoading}
          size="small"
          rowKey="id"
          pagination={false}
          dataSource={events}
          columns={[
            {
              title: '类型',
              dataIndex: 'eventType',
              width: 110,
              render: (v) => eventText[v] ?? '-',
            },
            { title: '单证', dataIndex: 'documentNo', width: 150 },
            {
              title: '模式变化',
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
          ]}
        />
        {eventsLoadedCount < eventsTotal ? (
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
            setPreview(null);
            setPreviewPayload(null);
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
