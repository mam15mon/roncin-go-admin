import {
  HistoryOutlined,
  RetweetOutlined,
  StopOutlined,
} from '@ant-design/icons';
import { useAccess } from '@umijs/max';
import {
  Alert,
  App,
  Button,
  Descriptions,
  Drawer,
  Form,
  Input,
  Modal,
  Radio,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import React, { useCallback, useState } from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import {
  OrderBusinessType,
  SeaDocumentEventType,
  SeaDocumentType,
  SeaDocumentVersionSource,
  SeaHouseBillIssuerSource,
  SeaHouseBillStatus,
} from '@/enums.generated';
import {
  seaDocumentServiceExecuteSeaDocumentAmendment,
  seaDocumentServiceExecuteSeaDocumentVoid,
  seaDocumentServiceExecuteSeaHouseBillSwitch,
  seaDocumentServiceListSeaDocumentEvents,
  seaDocumentServiceListSeaHouseBillVersions,
  seaDocumentServiceListSeaMasterBillVersions,
  seaDocumentServicePreviewSeaDocumentAmendment,
  seaDocumentServicePreviewSeaDocumentVoid,
  seaDocumentServicePreviewSeaHouseBillSwitch,
} from '@/services/roncin/seaDocumentService';
import { searchPartnerOptions } from '@/utils/options';

type ActionMode = 'amendment' | 'void' | 'switch';
type ChangePreview =
  | API.SeaDocumentAmendmentPreview
  | API.SeaDocumentVoidPreview
  | API.SeaHouseBillSwitchPreview;

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

interface SwitchFormValues {
  reason: string;
  surrenderInfo?: string;
  houseNo: string;
  issuerSource: number;
  issuerPartnerId?: string;
  note?: string;
}

interface PreviewPayload {
  values: SwitchFormValues;
  amendmentInput?: API.SeaDocumentAmendmentInput;
  switchHouseBill?: API.SeaHouseBillInput;
}

const sourceText: Record<number, string> = {
  [SeaDocumentVersionSource.SEA_DOCUMENT_VERSION_SOURCE_ORDER_LOCK]: '订单锁定',
  [SeaDocumentVersionSource.SEA_DOCUMENT_VERSION_SOURCE_AMENDMENT]: '改单',
  [SeaDocumentVersionSource.SEA_DOCUMENT_VERSION_SOURCE_SWITCH]: 'Switch B/L',
  [SeaDocumentVersionSource.SEA_DOCUMENT_VERSION_SOURCE_VOID]: '作废',
};

const eventText: Record<number, string> = {
  [SeaDocumentEventType.SEA_DOCUMENT_EVENT_TYPE_AMENDMENT]: '改单',
  [SeaDocumentEventType.SEA_DOCUMENT_EVENT_TYPE_VOID]: '作废',
  [SeaDocumentEventType.SEA_DOCUMENT_EVENT_TYPE_SWITCH]: 'Switch B/L',
};

function createIdempotencyKey() {
  return `sea-document-${crypto.randomUUID()}`;
}

function PreviewResult({ preview }: { preview: ChangePreview }) {
  const differences = preview.differences ?? [];
  const impacts = preview.impacts ?? [];
  return (
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
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
  const [events, setEvents] = useState<API.SeaDocumentEvent[]>([]);
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
    documentStatus === 'REPLACED' ||
    currentHouseBill?.status ===
      SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_VOIDED ||
    currentHouseBill?.status ===
      SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_REPLACED;
  const commandReady = Boolean(
    orderVersion && documentVersion && currentVersionId && !terminal,
  );

  const loadHistory = useCallback(async () => {
    setHistoryLoading(true);
    try {
      const [versionResult, eventResult] = await Promise.all([
        isHouse
          ? seaDocumentServiceListSeaHouseBillVersions({
              orderId,
              houseBillId: documentId,
              page: 1,
              pageSize: 200,
            })
          : seaDocumentServiceListSeaMasterBillVersions({
              orderId,
              page: 1,
              pageSize: 200,
            }),
        seaDocumentServiceListSeaDocumentEvents({
          orderId,
          page: 1,
          pageSize: 200,
        }),
      ]);
      setVersions(versionResult.data ?? []);
      setEvents(
        (eventResult.data ?? []).filter(
          (event) =>
            event.documentId === documentId ||
            event.oldHouseBillId === documentId ||
            event.newHouseBillId === documentId,
        ),
      );
    } catch (error: unknown) {
      message.error(
        error instanceof Error ? error.message : '读取单证历史失败',
      );
    } finally {
      setHistoryLoading(false);
    }
  }, [documentId, isHouse, message, orderId]);

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
      surrenderInfo: undefined,
      houseNo: undefined,
      issuerSource: currentHouseBill?.issuerSource,
      issuerPartnerId: currentHouseBill?.issuerPartnerId,
      note: currentHouseBill?.note,
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
    reason,
  });

  const buildSwitchHouseBill = (
    values: SwitchFormValues,
  ): API.SeaHouseBillInput => ({
    houseNo: values.houseNo,
    issuerSource: values.issuerSource,
    issuerPartnerId:
      values.issuerSource ===
      SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_OTHER_PARTNER
        ? values.issuerPartnerId
        : undefined,
    note: values.note?.trim() || undefined,
    content: currentHouseBill?.content ?? {},
  });

  const handlePreview = async () => {
    if (!mode || !commandReady || disabled) return;
    const values = (await form.validateFields()) as SwitchFormValues;
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
      } else if (mode === 'void') {
        const result = await seaDocumentServicePreviewSeaDocumentVoid(
          { orderId },
          buildCommon(values.reason),
        );
        setPreview(result.data ?? null);
        setPreviewPayload({ values });
      } else {
        const switchHouseBill = buildSwitchHouseBill(values);
        const result = await seaDocumentServicePreviewSeaHouseBillSwitch(
          { orderId },
          {
            orderId,
            oldHouseBillId: documentId,
            expectedOrderVersion: orderVersion,
            expectedHouseBillVersion: documentVersion,
            expectedCurrentVersionId: currentVersionId as string,
            reason: values.reason,
            surrenderInfo: values.surrenderInfo?.trim() || undefined,
            newHouseBill: switchHouseBill,
          },
        );
        setPreview(result.data ?? null);
        setPreviewPayload({ values, switchHouseBill });
      }
    } catch (error: unknown) {
      message.error(error instanceof Error ? error.message : '预览失败');
    } finally {
      setPreviewing(false);
    }
  };

  const handleExecute = async () => {
    if (!mode || !preview?.executable || !previewPayload || disabled) return;
    const { values, amendmentInput, switchHouseBill } = previewPayload;
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
          },
        );
      } else if (mode === 'void') {
        await seaDocumentServiceExecuteSeaDocumentVoid(
          { orderId },
          { ...buildCommon(values.reason), idempotencyKey },
        );
      } else {
        if (!switchHouseBill) return;
        await seaDocumentServiceExecuteSeaHouseBillSwitch(
          { orderId },
          {
            orderId,
            oldHouseBillId: documentId,
            expectedOrderVersion: orderVersion,
            expectedHouseBillVersion: documentVersion,
            expectedCurrentVersionId: currentVersionId as string,
            reason: values.reason,
            surrenderInfo: values.surrenderInfo?.trim() || undefined,
            idempotencyKey,
            newHouseBill: switchHouseBill,
          },
        );
      }
      message.success(
        mode === 'amendment'
          ? '改单版本已发布'
          : mode === 'void'
            ? '单证已作废'
            : 'Switch B/L 已完成',
      );
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
    mode === 'amendment'
      ? `改单：${documentNo}`
      : mode === 'void'
        ? `作废：${documentNo}`
        : `Switch B/L：${documentNo}`;

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
        {isHouse &&
        access.canOrder(OrderBusinessType.BUSINESS_TYPE_SE, 'switch') ? (
          <Button
            size="small"
            icon={<RetweetOutlined />}
            disabled={disabled || !commandReady}
            onClick={() => openAction('switch')}
          >
            Switch B/L
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
            { title: '形成时间', dataIndex: 'createdAt', width: 190 },
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
              title: '替代链',
              width: 200,
              render: (_, row) =>
                row.oldHouseNo && row.newHouseNo
                  ? `${row.oldHouseNo} → ${row.newHouseNo}`
                  : '-',
            },
            { title: '原因', dataIndex: 'reason' },
            { title: '时间', dataIndex: 'createdAt', width: 190 },
          ]}
        />
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
          {mode === 'switch' ? (
            <>
              <Form.Item name="surrenderInfo" label="交回/作废信息">
                <Input maxLength={500} />
              </Form.Item>
              <Form.Item
                name="houseNo"
                label="新 HBL 号"
                rules={[
                  {
                    required: true,
                    whitespace: true,
                    message: '请输入新 HBL 号',
                  },
                ]}
              >
                <Input maxLength={128} />
              </Form.Item>
              <Form.Item
                name="issuerSource"
                label="新 HBL 签发主体"
                rules={[{ required: true, message: '请选择签发主体' }]}
              >
                <Radio.Group>
                  <Radio
                    value={
                      SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_SELF_ORGANIZATION
                    }
                  >
                    本公司
                  </Radio>
                  <Radio
                    value={
                      SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_CUSTOMER_PARTNER
                    }
                  >
                    委托单位
                  </Radio>
                  <Radio
                    value={
                      SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_OTHER_PARTNER
                    }
                  >
                    其他主体
                  </Radio>
                </Radio.Group>
              </Form.Item>
              <Form.Item noStyle shouldUpdate>
                {({ getFieldValue }) =>
                  getFieldValue('issuerSource') ===
                  SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_OTHER_PARTNER ? (
                    <ProFormSearchableSelect
                      name="issuerPartnerId"
                      label="签发合作伙伴"
                      rules={[
                        { required: true, message: '请选择签发合作伙伴' },
                      ]}
                      request={async ({ keyWords }) =>
                        searchPartnerOptions(keyWords)
                      }
                      fieldProps={{ filterOption: false }}
                    />
                  ) : null
                }
              </Form.Item>
              <Form.Item name="note" label="新 HBL 备注">
                <Input maxLength={500} />
              </Form.Item>
            </>
          ) : null}
        </Form>
        {preview ? <PreviewResult preview={preview} /> : null}
      </Modal>
    </>
  );
}
