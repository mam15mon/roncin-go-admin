import {
  type ActionType,
  ProFormCheckbox,
  ProFormDateTimePicker,
  ProFormDependency,
  type ProFormInstance,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Collapse,
  Form,
  Space,
  Tag,
  Typography,
} from 'antd';
import { useEffect, useRef, useState } from 'react';
import { useLocation } from 'react-router';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import {
  FormRow,
  GuardedModalForm,
  ProFormSearchableSelect,
  ResponsiveProTable,
  ResponsiveTable,
} from '@/components/ui';
import {
  OrderBusinessType,
  OrderClosureStatus,
  OrderTerminationStatus,
  PartnerRoleType,
} from '@/enums.generated';
import { searchPartnerOptions } from '@/features/partners';
import { formatPersonnelLabel } from '@/features/personnel';
import { usePageForeground } from '@/hooks/usePageForeground';
import { orderAttachmentServiceListAttachments } from '@/services/roncin/orderAttachmentService';
import {
  orderServiceGetImportHandling,
  orderServiceListImportHandlingEvents,
  orderServiceListPersonnelOptions,
  orderServiceSaveImportHandling,
} from '@/services/roncin/orderService';
import { ensureListResponse } from '@/utils/api';
import { formatDate } from '@/utils/format';
import { getOrderKindDefinitionByBusinessType } from '../order-kinds/registry';
import { TemplateFieldGroup } from '../templates/components/FieldGroup';
import {
  cancelledServices,
  type HandlingFormValues,
  type HandlingSnapshot,
  handlingChanged,
  handlingFormValues,
  handlingPayload,
  handlingSnapshot,
  IMPORT_SERVICES,
  needsHandlingReason,
  serviceStatus,
  validVersion,
} from './model';

export type ImportHandlingSaveResult = {
  orderId: string;
  previousVersion: string;
  saved: API.OrderImportHandling;
};
type Props = {
  order: API.Order;
  expectedVersion?: string;
  readonly?: boolean;
  blockedReason?: string;
  onSaved: (result: ImportHandlingSaveResult) => Promise<void>;
};
type ScopeGuard = {
  current: { key: string; generation: number; alive: boolean };
};

function SnapshotTable({
  snapshot,
  attachments,
}: {
  snapshot: HandlingSnapshot;
  attachments?: API.OrderAttachment[];
}) {
  const files = new Map(attachments?.map((item) => [item.id, item.fileName]));
  return (
    <ResponsiveTable<API.ImportHandlingStep>
      rowKey={(step) => String(step.kind)}
      size="small"
      pagination={false}
      dataSource={snapshot.steps}
      columns={[
        {
          title: '服务',
          width: 90,
          render: (_, step) =>
            IMPORT_SERVICES.find((item) => item.kind === step.kind)?.label,
        },
        {
          title: '状态',
          width: 90,
          render: (_, step) => (
            <Tag
              color={
                step.undertaken !== true
                  ? undefined
                  : step.occurredAt
                    ? 'success'
                    : 'processing'
              }
            >
              {serviceStatus(step)}
            </Tag>
          ),
        },
        {
          title: '实际完成时间',
          width: 170,
          render: (_, step) => formatDate(step.occurredAt, 'minute'),
        },
        {
          title: '责任人',
          width: 110,
          render: (_, step) =>
            step.responsibleUserId
              ? step.responsibleUserName || '未命名人员'
              : '-',
        },
        { title: '参考号', dataIndex: 'referenceNo', width: 150 },
        {
          title: '备注',
          dataIndex: 'notes',
          render: (_, step) => (
            <span style={{ whiteSpace: 'pre-wrap' }}>{step.notes || '-'}</span>
          ),
        },
        {
          title: '附件',
          width: 150,
          render: (_, step) =>
            step.attachmentIds?.length
              ? step.attachmentIds
                  .map((id) => files.get(id))
                  .filter(Boolean)
                  .join('、') || `已关联 ${step.attachmentIds.length} 份附件`
              : '-',
        },
      ]}
    />
  );
}

function StepFields({
  index,
  snapshot,
  guard,
  canSearchPersonnel,
  attachments,
  canReadAttachments,
  businessType,
}: {
  index: number;
  snapshot: HandlingSnapshot;
  guard: ScopeGuard;
  canSearchPersonnel: boolean;
  attachments: API.OrderAttachment[];
  canReadAttachments: boolean;
  businessType: number;
}) {
  const form = Form.useFormInstance();
  const undertaken = Form.useWatch(['steps', index, 'undertaken'], form);
  const service = IMPORT_SERVICES[index];
  const step = snapshot.steps[index];
  return (
    <TemplateFieldGroup title={service.label}>
      <ProFormText name={['steps', index, 'kind']} hidden />
      <ProFormCheckbox
        name={['steps', index, 'undertaken']}
        fieldProps={{
          onChange: (event) => {
            if (!event.target.checked)
              form.setFieldValue(['steps', index, 'occurredAt'], undefined);
          },
        }}
      >
        承接{service.label}
      </ProFormCheckbox>
      {undertaken && (
        <>
          <FormRow cols={3}>
            <ProFormDateTimePicker
              name={['steps', index, 'occurredAt']}
              label={`${service.label}实际完成时间`}
              tooltip="留空表示本项尚未完成。"
              fieldProps={{ style: { width: '100%' } }}
            />
            <ProFormSearchableSelect
              name={['steps', index, 'responsibleUserId']}
              label={`${service.label}责任人`}
              disabled={!canSearchPersonnel}
              options={
                step.responsibleUserId
                  ? [
                      {
                        value: step.responsibleUserId,
                        label: step.responsibleUserName || '未命名人员',
                      },
                    ]
                  : []
              }
              fieldProps={{ filterOption: false }}
              request={async ({ keyWords }) => {
                if (!canSearchPersonnel || !guard.current.alive) return [];
                const generation = guard.current.generation;
                const response = ensureListResponse(
                  await orderServiceListPersonnelOptions({
                    businessType,
                    keyword: keyWords,
                    page: 1,
                    pageSize: 50,
                  }),
                  '责任人候选加载失败',
                );
                if (
                  !guard.current.alive ||
                  generation !== guard.current.generation
                )
                  return [];
                return (response.data ?? []).map((item) => ({
                  value: item.userId || '',
                  label: formatPersonnelLabel(item),
                }));
              }}
            />
            <ProFormText
              name={['steps', index, 'referenceNo']}
              label={`${service.label}参考号`}
              fieldProps={{ maxLength: 200 }}
            />
          </FormRow>
          <ProFormSearchableSelect
            name={['steps', index, 'attachmentIds']}
            label={`${service.label}关联附件`}
            mode="multiple"
            disabled={!canReadAttachments}
            options={[
              ...attachments.map((item) => ({
                value: item.id || '',
                label: item.fileName || '未命名附件',
              })),
              ...(step.attachmentIds ?? [])
                .filter((id) => !attachments.some((item) => item.id === id))
                .map((id) => ({
                  value: id,
                  label: '已关联附件（暂不可读取）',
                })),
            ]}
            fieldProps={{ maxCount: 50 }}
          />
          <ProFormTextArea
            name={['steps', index, 'notes']}
            label={`${service.label}备注`}
            fieldProps={{
              maxLength: 2000,
              autoSize: { minRows: 1, maxRows: 5 },
            }}
          />
        </>
      )}
    </TemplateFieldGroup>
  );
}

export default function ImportHandlingPanel({
  order,
  expectedVersion,
  readonly,
  blockedReason,
  onSaved,
}: Props) {
  const { message, modal } = App.useApp();
  const { initialState } = useInitialState();
  const access = useAccess();
  const location = useLocation();
  const queryClient = useQueryClient();
  const organizationId = initialState?.currentUser?.currentOrganization?.id;
  const type = order.businessType ?? 0;
  const kind = getOrderKindDefinitionByBusinessType(type)?.kind;
  const scope = `${initialState?.currentUser?.id}:${organizationId}:${type}:${order.id}`;
  const guard = useRef({ key: scope, generation: 0, alive: true });
  if (guard.current.key !== scope)
    guard.current = {
      key: scope,
      generation: guard.current.generation + 1,
      alive: true,
    };
  const sessionScope = `${scope}:${guard.current.generation}`;
  useEffect(() => {
    guard.current.alive = true;
    return () => {
      guard.current.alive = false;
      guard.current.generation += 1;
    };
  }, []);
  const isImport =
    type === OrderBusinessType.BUSINESS_TYPE_SI ||
    type === OrderBusinessType.BUSINESS_TYPE_AI;
  const enabled = Boolean(
    isImport &&
      organizationId &&
      organizationId === order.organizationId &&
      order.id &&
      access.canOrder(type, 'read'),
  );
  const resume = useRef<() => void>(() => undefined);
  const foreground = usePageForeground(() => resume.current());
  const active =
    enabled &&
    foreground &&
    location.pathname.replace(/\/$/, '') === `/orders/${kind}/${order.id}`;
  const query = useQuery<HandlingSnapshot>({
    queryKey: [
      'orders',
      'import-handling',
      { scope: sessionScope, expectedVersion },
    ],
    enabled: active,
    // 本人保存推进订单版本即换查询键；保留同一订单的上一份快照，避免面板闪回
    // 「正在读取」。仅作展示占位，openReady 仍要求非 fetching 且版本一致。
    placeholderData: (previous: HandlingSnapshot | undefined) =>
      previous?.orderId === order.id ? previous : undefined,
    refetchInterval: active ? 15_000 : false,
    refetchIntervalInBackground: false,
    queryFn: async ({ signal }) => {
      const response = await orderServiceGetImportHandling(
        { orderId: order.id || '' },
        { signal, skipErrorHandler: true },
      );
      if (response?.success === false) throw new Error('进口服务加载失败');
      return handlingSnapshot(response?.data, order.id || '');
    },
    meta: { silent: true },
  });
  resume.current = () => {
    if (active) void query.refetch({ cancelRefetch: false });
  };
  const canReadAttachments =
    enabled && access.canOrder(type, 'attachment.read');
  const files = useQuery({
    queryKey: ['orders', 'import-handling-attachments', sessionScope],
    enabled: active && canReadAttachments,
    queryFn: async ({ signal }) =>
      ensureListResponse(
        await orderAttachmentServiceListAttachments(
          { orderId: order.id || '' },
          { signal },
        ),
        '本票附件加载失败',
      ).data ?? [],
  });
  const [edit, setEdit] = useState<{
    scope: string;
    serial: number;
    snapshot: HandlingSnapshot;
  }>();
  const serial = useRef(0);
  const editor = edit?.scope === sessionScope ? edit : undefined;
  const [changedSerial, setChangedSerial] = useState<number>();
  // 本人写入在途（含写后订单与服务重读）：不比较版本，避免把本人写入误报为外部变动。
  // 成功后保持至弹窗关闭，期间重读已取得新快照，关闭后比较不再出现先现后消的提醒。
  const [ownWritePending, setOwnWritePending] = useState(false);
  const formRef = useRef<ProFormInstance<HandlingFormValues> | undefined>(
    undefined,
  );
  const eventsRef = useRef<ActionType | undefined>(undefined);
  useEffect(() => {
    if (
      editor &&
      !ownWritePending &&
      (expectedVersion !== editor.snapshot.orderVersion ||
        (query.isSuccess &&
          query.data &&
          handlingChanged(editor.snapshot, query.data)))
    )
      setChangedSerial(editor.serial);
  }, [editor, ownWritePending, expectedVersion, query.data, query.isSuccess]);
  const changed = Boolean(
    editor &&
      (changedSerial === editor.serial ||
        (!ownWritePending &&
          (expectedVersion !== editor.snapshot.orderVersion ||
            (query.data && handlingChanged(editor.snapshot, query.data))))),
  );
  const closed =
    order.closureStatus === OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED;
  const terminated =
    order.terminationStatus !==
    OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE;
  const blocked =
    readonly ||
    !enabled ||
    !access.canOrder(type, 'update') ||
    closed ||
    terminated;
  const writable = useRef(!blocked);
  writable.current = !blocked;
  const changedRef = useRef(changed);
  changedRef.current = changed;
  const editRevision = useRef(0);
  const expectedVersionRef = useRef(expectedVersion);
  expectedVersionRef.current = expectedVersion;
  const serialRef = useRef(editor?.serial);
  serialRef.current = editor?.serial;
  const openReady =
    !blocked &&
    query.isSuccess &&
    !query.isFetching &&
    query.data?.orderVersion === expectedVersion &&
    validVersion(expectedVersion);
  const confirmCancel = (names: string[]) =>
    new Promise<boolean>((resolve) =>
      modal.confirm({
        title: `确认取消承接：${names.join('、')}？`,
        content: '已完成事实会保留在服务历史中。',
        okText: '确认取消承接',
        cancelText: '保留承接',
        onOk: () => resolve(true),
        onCancel: () => resolve(false),
      }),
    );
  const refreshEditor = async () => {
    if (!editor) return;
    const captured = guard.current.generation;
    const revision = editRevision.current;
    const next = await query.refetch();
    if (
      !guard.current.alive ||
      captured !== guard.current.generation ||
      !next.isSuccess ||
      !next.data ||
      serialRef.current !== editor.serial ||
      editRevision.current !== revision
    )
      return;
    if (next.data.orderVersion !== expectedVersionRef.current) {
      message.warning('请先刷新订单资料，再编辑进口服务。');
      return;
    }
    setChangedSerial(undefined);
    setEdit({
      scope: sessionScope,
      serial: ++serial.current,
      snapshot: structuredClone(next.data),
    });
  };
  const askRefresh = () =>
    modal.confirm({
      title: '重新读取服务资料？',
      content: '当前未保存的服务修改将被覆盖。',
      onOk: refreshEditor,
    });
  if (!enabled) return null;
  return (
    <>
      <div style={{ display: 'grid', gap: 8 }}>
        <Space style={{ justifyContent: 'flex-end' }}>
          <Button onClick={() => void query.refetch()}>刷新服务</Button>
          <Button
            type="primary"
            disabled={!openReady}
            onClick={() =>
              query.data &&
              setEdit({
                scope: sessionScope,
                serial: ++serial.current,
                snapshot: structuredClone(query.data),
              })
            }
          >
            维护承接与完成情况
          </Button>
        </Space>
        <Typography.Paragraph type="secondary">
          按本票承接范围分别维护，允许并行办理；运输到达与本票服务完成分别记录。
        </Typography.Paragraph>
        {closed && (
          <Alert
            type="info"
            showIcon
            title="订单已完结，请先重开订单再变更进口服务。"
          />
        )}
        {blockedReason && <Alert type="info" title={blockedReason} />}
        {query.isError ? (
          <Alert type="error" showIcon title="进口服务加载失败，请重试" />
        ) : query.data ? (
          <>
            <Typography.Paragraph>
              换单代理：{query.data.exchangeAgentName || '未指定'}
            </Typography.Paragraph>
            <SnapshotTable snapshot={query.data} attachments={files.data} />
            {!ownWritePending &&
              query.data.orderVersion !== expectedVersion && (
                <Alert
                  type="warning"
                  title="这个订单已经发生变动，请重新刷新。"
                />
              )}
          </>
        ) : (
          <Typography.Text>正在读取进口服务…</Typography.Text>
        )}
        <Collapse
          size="small"
          style={{ marginTop: 12 }}
          items={[
            {
              key: 'history',
              label: '服务变更历史',
              children: (
                <ResponsiveProTable<API.ImportHandlingEvent>
                  rowKey="id"
                  actionRef={eventsRef}
                  search={false}
                  options={false}
                  size="small"
                  pagination={{ defaultPageSize: 10 }}
                  request={async ({ current, pageSize }) => {
                    const generation = guard.current.generation;
                    const response = await orderServiceListImportHandlingEvents(
                      {
                        orderId: order.id || '',
                        page: current || 1,
                        pageSize: pageSize || 10,
                      },
                    );
                    if (
                      !guard.current.alive ||
                      generation !== guard.current.generation
                    )
                      return { data: [], success: false, total: 0 };
                    if (!response?.data || response.success === false)
                      throw new Error('服务历史加载失败');
                    return {
                      data: response.data.items ?? [],
                      total: response.data.total ?? 0,
                      success: true,
                    };
                  }}
                  columns={[
                    {
                      title: '时间',
                      render: (_, item) =>
                        formatDate(item.occurredAt, 'datetime'),
                      width: 180,
                    },
                    {
                      title: '操作人',
                      render: (_, item) => item.actorName || '未命名人员',
                      width: 110,
                    },
                    {
                      title: '原因',
                      render: (_, item) =>
                        item.reason || '承接范围或服务进度登记',
                    },
                  ]}
                  expandable={{
                    expandedRowRender: (event) => (
                      <Space orientation="vertical" style={{ width: '100%' }}>
                        <Typography.Text strong>
                          变更前 · 换单代理：
                          {event.before?.exchangeAgentName || '未指定'}
                        </Typography.Text>
                        <SnapshotTable
                          snapshot={handlingSnapshot(
                            event.before,
                            order.id || '',
                          )}
                          attachments={files.data}
                        />
                        <Typography.Text strong>
                          变更后 · 换单代理：
                          {event.after?.exchangeAgentName || '未指定'}
                        </Typography.Text>
                        <SnapshotTable
                          snapshot={handlingSnapshot(
                            event.after,
                            order.id || '',
                          )}
                          attachments={files.data}
                        />
                      </Space>
                    ),
                  }}
                />
              ),
            },
          ]}
        />
      </div>
      {editor && (
        <GuardedModalForm<HandlingFormValues>
          key={editor.serial}
          title="维护本票进口服务"
          open
          formRef={formRef}
          width={1000}
          guardIdentity={`${scope}:${editor.serial}`}
          modalProps={{ destroyOnHidden: true }}
          initialValues={handlingFormValues(editor.snapshot)}
          onValuesChange={() => {
            editRevision.current += 1;
          }}
          onOpenChange={(open) => {
            if (!open) {
              setEdit(undefined);
              setOwnWritePending(false);
            }
          }}
          submitter={{
            submitButtonProps: {
              disabled: blocked || changed || query.isError || query.isFetching,
            },
            searchConfig: { submitText: '保存服务资料' },
          }}
          onFinish={async (values) => {
            if (
              !writable.current ||
              changed ||
              query.isError ||
              query.isFetching
            )
              return false;
            const generation = guard.current.generation;
            const revision = editRevision.current;
            const payload = handlingPayload(editor.snapshot, values);
            if (
              needsHandlingReason(editor.snapshot, payload) &&
              !payload.reason
            ) {
              message.warning('撤回、取消或纠正已完成服务必须填写原因。');
              return false;
            }
            const cancelled = cancelledServices(editor.snapshot, payload);
            if (cancelled.length && !(await confirmCancel(cancelled)))
              return false;
            if (
              !guard.current.alive ||
              generation !== guard.current.generation ||
              !writable.current ||
              changedRef.current ||
              serialRef.current !== editor.serial ||
              editRevision.current !== revision
            )
              return false;
            setOwnWritePending(true);
            let closing = false;
            try {
              const response = await orderServiceSaveImportHandling(
                { orderId: editor.snapshot.orderId },
                payload,
              );
              if (
                !response?.data ||
                response.success === false ||
                !guard.current.alive ||
                generation !== guard.current.generation ||
                serialRef.current !== editor.serial
              )
                return false;
              const saved = handlingSnapshot(
                response.data,
                editor.snapshot.orderId,
              );
              message.success('进口服务资料已保存');
              try {
                await onSaved({
                  orderId: saved.orderId,
                  previousVersion: editor.snapshot.orderVersion,
                  saved,
                });
              } catch {
                message.warning('服务已保存，订单资料刷新失败，请重新刷新。');
              }
              if (
                !guard.current.alive ||
                generation !== guard.current.generation ||
                serialRef.current !== editor.serial
              )
                return false;
              await queryClient.invalidateQueries({
                queryKey: ['orders', 'import-handling'],
              });
              if (
                !guard.current.alive ||
                generation !== guard.current.generation ||
                serialRef.current !== editor.serial
              )
                return false;
              eventsRef.current?.reload();
              if (editRevision.current !== revision) {
                setChangedSerial(editor.serial);
                message.warning(
                  '服务已保存，保存期间的新输入已保留，请重新读取后再提交。',
                );
                return false;
              }
              closing = true;
              return true;
            } finally {
              // 成功时弹窗随即关闭，由 onOpenChange 与 setEdit 同批复位。
              if (!closing) setOwnWritePending(false);
            }
          }}
        >
          <Form.Item name="steps" hidden>
            <span />
          </Form.Item>
          {(changed || query.isError) && (
            <Alert
              type="warning"
              showIcon
              title={
                changed
                  ? '这个订单已经发生变动，请重新刷新。'
                  : '服务资料检查失败，请重试后保存。'
              }
              description="当前输入已保留。"
              action={<Button onClick={askRefresh}>重新读取</Button>}
              style={{ marginBottom: 12 }}
            />
          )}
          {blocked && (
            <Alert
              type="warning"
              title={
                closed
                  ? '订单已完结，请先重开订单。'
                  : blockedReason || '当前订单不允许修改服务资料。'
              }
            />
          )}
          <ProFormSearchableSelect
            name="exchangeAgentId"
            label="换单代理"
            disabled={!access.canReadPartners}
            options={
              editor.snapshot.exchangeAgentId
                ? [
                    {
                      value: editor.snapshot.exchangeAgentId,
                      label:
                        editor.snapshot.exchangeAgentName || '未命名换单代理',
                    },
                  ]
                : []
            }
            fieldProps={{ filterOption: false }}
            request={async ({ keyWords }) => {
              if (!access.canReadPartners || !guard.current.alive) return [];
              const generation = guard.current.generation;
              const options = await searchPartnerOptions(keyWords, {
                role: PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER,
                enabled: true,
              });
              return guard.current.alive &&
                generation === guard.current.generation
                ? options
                : [];
            }}
          />
          <Space orientation="vertical" style={{ width: '100%' }} size={12}>
            {IMPORT_SERVICES.map((service, index) => (
              <StepFields
                key={service.kind}
                index={index}
                snapshot={editor.snapshot}
                businessType={type}
                guard={guard}
                canSearchPersonnel={access.canOrder(type, 'create')}
                attachments={files.data ?? []}
                canReadAttachments={
                  canReadAttachments &&
                  files.isSuccess &&
                  !files.isFetching &&
                  !files.isError
                }
              />
            ))}
          </Space>
          <ProFormDependency name={['steps', 'exchangeAgentId']}>
            {(values) => {
              const needsReason = needsHandlingReason(
                editor.snapshot,
                handlingPayload(editor.snapshot, values as HandlingFormValues),
              );
              return (
                <ProFormTextArea
                  name="reason"
                  label="变更原因"
                  rules={[
                    {
                      required: needsReason,
                      whitespace: true,
                      message: '撤回、取消或纠正已完成服务必须填写原因',
                    },
                  ]}
                  fieldProps={{
                    maxLength: 500,
                    autoSize: { minRows: 2, maxRows: 4 },
                  }}
                />
              );
            }}
          </ProFormDependency>
        </GuardedModalForm>
      )}
    </>
  );
}
