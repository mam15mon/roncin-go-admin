import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Col,
  DatePicker,
  Form,
  Input,
  Modal,
  Radio,
  Row,
  Select,
  Space,
  Spin,
  Tag,
  Typography,
} from 'antd';
import type { DefaultOptionType } from 'antd/es/select';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import React, { useEffect, useRef, useState } from 'react';
import { useUnsavedChangesGuard } from '@/components/layout/useUnsavedChangesGuard';
import { ResponsiveTable as Table } from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import { useAsyncGuard } from '@/hooks/useAsyncGuard';
import { orderServiceMatchSeaMasterBillCandidate } from '@/services/roncin/orderService';
import {
  seaOrderChangeServiceExecuteSeaOrderReassignment,
  seaOrderChangeServicePreviewSeaOrderReassignment,
} from '@/services/roncin/seaOrderChangeService';
import { isWriteResponse } from '@/utils/api';
import { getErrorMessage } from '@/utils/errorMessage';
import { formatDate } from '@/utils/format';
import { canonicalStringify, computeCanonicalSha256 } from '@/utils/hash';
import { generateUUID } from '@/utils/uuid';
import SeaExternalConfirmationFields, {
  buildSeaExternalConfirmation,
} from '../../templates/components/sea/SeaExternalConfirmationFields';
import {
  buildReassignmentTarget,
  type ReassignmentFormValues,
} from './seaOrderReassignmentModels';

const { Text } = Typography;
const { TextArea } = Input;

interface SeaOrderReassignmentModalProps {
  orderId: string;
  orderNo?: string;
  open: boolean;
  disabled?: boolean;
  disabledReason?: string;
  onClose: () => void;
  onSuccess: () => void;
  searchShippingLines?: (keyword?: string) => Promise<DefaultOptionType[]>;
  searchLocations?: (keyword?: string) => Promise<DefaultOptionType[]>;
  initialShippingLineId?: string;
  initialShippingLineName?: string;
}

function ReassignmentForm({
  orderId,
  orderNo,
  disabled = false,
  onClose,
  onSuccess,
  searchShippingLines,
  searchLocations,
  initialShippingLineId,
  initialShippingLineName,
}: SeaOrderReassignmentModalProps) {
  const { markDirty, resetDirty } = useUnsavedChangesGuard({
    identity: orderId,
  });
  const [form] = Form.useForm<ReassignmentFormValues>();
  const values = Form.useWatch([], form) as ReassignmentFormValues | undefined;
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const executionGuard = useAsyncGuard();
  // 每次打开均建立独立编辑会话，关闭后旧预览和写入结果不得影响再次打开的弹窗。
  const [sessionId] = useState(generateUUID);
  const [requestedTargetType, setTargetType] = useState<'candidate' | 'new'>(
    'new',
  );
  const [matchLookup, setMatchLookup] = useState<{
    masterNo: string;
    shippingLineId: string;
    attemptId: string;
  } | null>(null);
  const [selectedTeId, setSelectedTeId] = useState<string>();
  const [confirmation, setConfirmation] =
    useState<API.ExecuteSeaOrderReassignmentRequest | null>(null);

  const [carrierOptions, setCarrierOptions] = useState<DefaultOptionType[]>(
    initialShippingLineId
      ? [
          {
            label: initialShippingLineName || initialShippingLineId,
            value: initialShippingLineId,
          },
        ]
      : [],
  );
  const [originPortOptions, setOriginPortOptions] = useState<
    DefaultOptionType[]
  >([]);
  const [dischargePortOptions, setDischargePortOptions] = useState<
    DefaultOptionType[]
  >([]);
  const [transitPortOptions, setTransitPortOptions] = useState<
    DefaultOptionType[]
  >([]);
  const onCloseRef = useRef(onClose);
  onCloseRef.current = onClose;
  useEffect(() => {
    if (disabled) onCloseRef.current();
  }, [disabled]);

  const matchQuery = useQuery({
    queryKey: [
      'order-reassignment',
      'candidate',
      { orderId, sessionId, matchLookup },
    ],
    enabled: !disabled && Boolean(matchLookup),
    queryFn: async ({ signal }) => {
      if (!matchLookup) throw new Error('请先输入目标提单号并选择船公司');
      const response = await orderServiceMatchSeaMasterBillCandidate(
        {
          masterNo: matchLookup.masterNo,
          shippingLineId: matchLookup.shippingLineId,
        },
        { signal, skipErrorHandler: true },
      );
      if (
        !response ||
        response.success === false ||
        (response.success !== true && typeof response.matched !== 'boolean')
      ) {
        throw new Error('匹配候选母单失败，请重试');
      }
      if (
        response.matched &&
        (!response.candidate?.id ||
          !response.candidate.version ||
          !(response.candidate.transportExecutions ?? []).some(
            (item) => item.id && item.version,
          ))
      ) {
        throw new Error('候选母单没有可选择的运输执行或版本信息');
      }
      return { ...response, matched: response.matched === true };
    },
    meta: { silent: true },
  });
  const candidateMatched =
    matchLookup && matchQuery.data?.matched
      ? (matchQuery.data.candidate ?? null)
      : null;
  const transportExecutions = (
    candidateMatched?.transportExecutions ?? []
  ).filter((item) => item.id && item.version);
  const candidateTe =
    transportExecutions.length === 1
      ? transportExecutions[0]
      : (transportExecutions.find((item) => item.id === selectedTeId) ?? null);
  const targetType = candidateMatched
    ? 'candidate'
    : matchLookup && matchQuery.data?.matched === false
      ? 'new'
      : requestedTargetType;

  // 仅将本次显式匹配/选择的航程回显到表单，候选航程不接受本票手工改写。
  useEffect(() => {
    if (!candidateMatched || !candidateTe) return;
    form.setFieldsValue({
      masterNo: candidateMatched.masterNo,
      shippingLineId: candidateTe.shippingLineId,
      vesselName: candidateTe.vesselName,
      voyageNo: candidateTe.voyageNo,
      originLocationId: candidateTe.originLocationId,
      dischargeLocationId: candidateTe.dischargeLocationId,
      transitLocationId: candidateTe.transitLocationId,
      etd: candidateTe.etd ? dayjs(candidateTe.etd) : undefined,
      eta: candidateTe.eta ? dayjs(candidateTe.eta) : undefined,
    });
    if (candidateTe.shippingLineId) {
      setCarrierOptions((current) => [
        ...current.filter((item) => item.value !== candidateTe.shippingLineId),
        {
          label:
            candidateMatched.shippingLineName || candidateTe.shippingLineId,
          value: candidateTe.shippingLineId,
        },
      ]);
    }
  }, [candidateMatched, candidateTe, form]);

  const targetInput = buildReassignmentTarget(
    values ?? {},
    targetType,
    candidateMatched,
    candidateTe,
  );
  const matching = Boolean(matchLookup) && matchQuery.isFetching;
  const previewEnabled =
    !disabled && !matching && !matchQuery.isError && Boolean(targetInput);
  const previewQuery = useQuery({
    queryKey: [
      'order-reassignment',
      'preview',
      { orderId, sessionId, target: targetInput },
    ],
    enabled: previewEnabled,
    queryFn: async ({ signal }) => {
      if (!targetInput) throw new Error('请填写完整的改配目标');
      const response = await seaOrderChangeServicePreviewSeaOrderReassignment(
        { orderId },
        { orderId, target: targetInput },
        { signal, skipErrorHandler: true },
      );
      if (!response?.data || response.success === false)
        throw new Error('改配预览未返回有效结果，请重试');
      if (
        response.data.isValid &&
        (!/^[1-9]\d*$/.test(String(response.data.orderVersion ?? '')) ||
          !/^[1-9]\d*$/.test(String(response.data.currentLinkVersion ?? '')))
      ) {
        throw new Error('改配预览缺少订单或关联版本，请重新预览');
      }
      return response.data;
    },
    meta: { silent: true },
  });
  const previewData = previewEnabled ? previewQuery.data : undefined;
  const previewing = matching || (previewEnabled && previewQuery.isFetching);
  const previewError =
    matchLookup && matchQuery.isError
      ? getErrorMessage(matchQuery.error, '匹配候选母单失败')
      : previewEnabled && previewQuery.isError
        ? getErrorMessage(previewQuery.error, '改配比对校验失败')
        : previewData?.isValid === false
          ? previewData.errors?.join('；') || '当前目标不满足改配条件'
          : null;
  const canExecute =
    previewEnabled &&
    !previewing &&
    previewQuery.isSuccess &&
    previewData?.isValid === true;
  const currentPreviewRef = useRef({
    canExecute,
    targetInput,
    previewData,
    targetType,
    candidateMatched,
    candidateTe,
    disabled,
  });
  currentPreviewRef.current = {
    canExecute,
    targetInput,
    previewData,
    targetType,
    candidateMatched,
    candidateTe,
    disabled,
  };

  const execution = useMutation({
    mutationFn: (payload: API.ExecuteSeaOrderReassignmentRequest) =>
      seaOrderChangeServiceExecuteSeaOrderReassignment({ orderId }, payload, {
        skipErrorHandler: true,
      }),
    meta: { silent: true },
  });
  const submitting = execution.isPending;

  const handleMatchCandidate = async () => {
    if (disabled || confirmation || submitting) return;
    try {
      const fields = await form.validateFields(['masterNo', 'shippingLineId']);
      if (!fields.masterNo || !fields.shippingLineId) return;
      markDirty();
      setSelectedTeId(undefined);
      setMatchLookup({
        masterNo: fields.masterNo,
        shippingLineId: fields.shippingLineId,
        attemptId: generateUUID(),
      });
    } catch {
      // 字段错误由 Form 呈现，不发送不完整的匹配请求。
    }
  };

  const handleExecute = async () => {
    if (!currentPreviewRef.current.canExecute || submitting || confirmation)
      return;
    try {
      const fields = await form.validateFields();
      const current = currentPreviewRef.current;
      const target = buildReassignmentTarget(
        fields,
        current.targetType,
        current.candidateMatched,
        current.candidateTe,
      );
      if (
        !current.canExecute ||
        current.disabled ||
        !target ||
        canonicalStringify(target) !== canonicalStringify(current.targetInput)
      ) {
        message.warning('改配目标已变化，请等待当前预览通过后再确认');
        return;
      }
      const payload = {
        orderId,
        target,
        reason: fields.reason?.trim() ?? '',
        responsibilityType: fields.responsibilityType ?? '',
        responsiblePartnerId: fields.responsiblePartnerId || undefined,
        confirmation: buildSeaExternalConfirmation(fields),
        expectedOrderVersion: String(current.previewData?.orderVersion),
        expectedLinkVersion: String(current.previewData?.currentLinkVersion),
        expectedCandidateMblVersion: target.candidateVersion,
        expectedCandidateTeVersion: target.candidateTeVersion,
      };
      const hash = computeCanonicalSha256(payload);
      setConfirmation({
        ...payload,
        requestFingerprint: `reassign-fp:${hash}`,
        idempotencyKey: `reassign:${orderId}:${hash}`,
      });
    } catch (error: unknown) {
      if (error instanceof Error)
        message.error(getErrorMessage(error, '请检查改配资料'));
    }
  };

  const executeConfirmed = async () => {
    if (!confirmation || submitting || disabled) return;
    try {
      await executionGuard.run(
        () => execution.mutateAsync(confirmation),
        (response) => {
          if (
            !isWriteResponse(response) ||
            !response.data?.reassignmentEventId ||
            response.data.orderId !== orderId
          ) {
            message.error('执行改配未返回有效结果，请核对后重试');
            return;
          }
          resetDirty();
          setConfirmation(null);
          message.success('整票改配成功！航程与母单已更新');
          void queryClient.invalidateQueries({ queryKey: ['orders'] });
          onSuccess();
          onClose();
        },
      );
    } catch (error: unknown) {
      message.error(getErrorMessage(error, '执行改配失败'));
    }
  };

  const diffColumns: ColumnsType<API.VoyageDifferenceItem> = [
    {
      title: '航程要素',
      dataIndex: 'label',
      width: 140,
    },
    {
      title: '当前原值',
      dataIndex: 'currentValue',
      render: (val: string) => <Text type="secondary">{val || '未填写'}</Text>,
    },
    {
      title: '改配目标值',
      dataIndex: 'targetValue',
      render: (val: string, record) => {
        if (record.isDifferent) {
          return (
            <Text strong style={{ color: '#cf1322' }}>
              {val || '未填写'}
            </Text>
          );
        }
        return <Text>{val || '未填写'}</Text>;
      },
    },
    {
      title: '变更状态',
      key: 'diffStatus',
      width: 90,
      render: (_, record) => {
        if (record.isDifferent) {
          return <Tag color="error">发生变动</Tag>;
        }
        return <Tag color="default">保持一致</Tag>;
      },
    },
  ];

  const columnSettings = useColumnSettings<
    ColumnsType<API.VoyageDifferenceItem>[number]
  >({
    tableKey: 'orders:reassignment-voyage-difference',
    columns: diffColumns,
  });

  return (
    <Modal
      title={`订单整票改配 - ${orderNo || orderId}`}
      open
      onCancel={onClose}
      width={900}
      confirmLoading={submitting}
      onOk={handleExecute}
      okButtonProps={{ disabled: !canExecute || Boolean(confirmation) }}
      cancelButtonProps={{ disabled: submitting }}
      closable={!submitting}
      mask={{ closable: !submitting }}
      okText="确认改配"
      cancelText="取消"
    >
      <Alert
        type="info"
        showIcon
        title="改配说明"
        description="改配操作将把当前订单完整切换到新航程或目标母单，同步更新船名航次、起运港/卸货港等航程要素。若目标母单已存在，将自动加入共享母单组。"
        style={{ marginBottom: 16 }}
      />

      {previewError && (
        <Alert
          type="error"
          showIcon
          title="改配校验未通过"
          description={previewError}
          style={{ marginBottom: 16 }}
        />
      )}

      <Form
        form={form}
        layout="vertical"
        initialValues={{
          responsibilityType: 'OWN_COMPANY',
          shippingLineId: initialShippingLineId,
        }}
        disabled={disabled || submitting || Boolean(confirmation)}
        onValuesChange={(changed: Partial<ReassignmentFormValues>) => {
          markDirty();
          if ('masterNo' in changed || 'shippingLineId' in changed) {
            setMatchLookup(null);
            setSelectedTeId(undefined);
          }
        }}
      >
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item label="母单提供方式">
              <Radio.Group
                value={targetType}
                onChange={(e) => {
                  markDirty();
                  setTargetType(e.target.value);
                  setMatchLookup(null);
                  setSelectedTeId(undefined);
                }}
              >
                <Radio.Button value="new">手工录入新母单</Radio.Button>
                <Radio.Button value="candidate">匹配已有共享母单</Radio.Button>
              </Radio.Group>
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              label="目标提单号 (MBL No)"
              required
              htmlFor="reassignment-master-no"
            >
              <Space.Compact style={{ width: '100%' }}>
                <Form.Item
                  name="masterNo"
                  noStyle
                  rules={[
                    { required: true, message: '请输入目标提单号' },
                    {
                      pattern: /^[A-Za-z0-9]+$/,
                      message:
                        '提单号只能包含英文字母和阿拉伯数字，不能包含空格或符号',
                    },
                  ]}
                >
                  <Input
                    id="reassignment-master-no"
                    placeholder="请输入目标提单号"
                  />
                </Form.Item>
                <Button
                  type="primary"
                  onClick={handleMatchCandidate}
                  loading={matching}
                >
                  检查/匹配
                </Button>
              </Space.Compact>
            </Form.Item>
          </Col>
        </Row>

        {candidateMatched && (
          <Space
            orientation="vertical"
            style={{ width: '100%', marginBottom: 16 }}
          >
            <Alert
              type="success"
              showIcon
              title={`已匹配现有共享母单：${candidateMatched.masterNo}`}
              description={`当前已有 ${candidateMatched.memberCount} 票成员订单。请明确选择本票要关联的实际航次。`}
            />
            <Select
              value={candidateTe?.id}
              placeholder="请选择目标实际航次"
              options={(candidateMatched.transportExecutions ?? []).map(
                (te) => ({
                  value: te.id,
                  label: `${te.vesselName || '-'} / ${te.voyageNo || '-'} / ${te.etd ? formatDate(te.etd, 'minute') : '无 ETD'}`,
                }),
              )}
              onChange={(id) => {
                markDirty();
                const te = (candidateMatched.transportExecutions ?? []).find(
                  (item) => item.id === id,
                );
                if (!te?.id || !te.version) return;
                setSelectedTeId(te.id);
              }}
            />
          </Space>
        )}

        <Row gutter={16}>
          <Col span={8}>
            <Form.Item
              name="shippingLineId"
              label="船公司"
              rules={[{ required: true, message: '请选择船公司' }]}
            >
              <Select
                placeholder="搜索选择船公司"
                showSearch={{
                  filterOption: false,
                  onSearch: async (k) => {
                    if (searchShippingLines) {
                      const opts = await searchShippingLines(k);
                      setCarrierOptions(opts);
                    }
                  },
                }}
                options={carrierOptions}
              />
            </Form.Item>
          </Col>
          <Col span={8}>
            <Form.Item name="vesselName" label="船名">
              <Input
                placeholder="船名"
                disabled={targetType === 'candidate' || undefined}
              />
            </Form.Item>
          </Col>
          <Col span={8}>
            <Form.Item name="voyageNo" label="航次">
              <Input
                placeholder="航次"
                disabled={targetType === 'candidate' || undefined}
              />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={16}>
          <Col span={8}>
            <Form.Item name="originLocationId" label="起运港 (POL)">
              <Select
                allowClear
                disabled={targetType === 'candidate' || undefined}
                placeholder="搜索起运港"
                showSearch={{
                  filterOption: false,
                  onSearch: async (k) => {
                    if (searchLocations) {
                      const opts = await searchLocations(k);
                      setOriginPortOptions(opts);
                    }
                  },
                }}
                options={originPortOptions}
              />
            </Form.Item>
          </Col>
          <Col span={8}>
            <Form.Item name="dischargeLocationId" label="卸货港 (POD)">
              <Select
                allowClear
                disabled={targetType === 'candidate' || undefined}
                placeholder="搜索卸货港"
                showSearch={{
                  filterOption: false,
                  onSearch: async (k) => {
                    if (searchLocations) {
                      const opts = await searchLocations(k);
                      setDischargePortOptions(opts);
                    }
                  },
                }}
                options={dischargePortOptions}
              />
            </Form.Item>
          </Col>
          <Col span={8}>
            <Form.Item name="transitLocationId" label="中转港">
              <Select
                allowClear
                disabled={targetType === 'candidate' || undefined}
                placeholder="搜索中转港"
                showSearch={{
                  filterOption: false,
                  onSearch: async (k) => {
                    if (searchLocations) {
                      const opts = await searchLocations(k);
                      setTransitPortOptions(opts);
                    }
                  },
                }}
                options={transitPortOptions}
              />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="etd" label="预计离港时间 (ETD)">
              <DatePicker
                showTime
                style={{ width: '100%' }}
                disabled={targetType === 'candidate' || undefined}
              />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="eta" label="预计到港时间 (ETA)">
              <DatePicker
                showTime
                style={{ width: '100%' }}
                disabled={targetType === 'candidate' || undefined}
              />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={16}>
          <Col span={8}>
            <Form.Item
              name="responsibilityType"
              label="责任归属类型"
              rules={[{ required: true, message: '请选择责任归属' }]}
            >
              <Select
                options={[
                  { label: '我司原因 (OWN_COMPANY)', value: 'OWN_COMPANY' },
                  { label: '客户原因 (CUSTOMER)', value: 'CUSTOMER' },
                  { label: '船东原因 (CARRIER)', value: 'CARRIER' },
                  { label: '海关原因 (CUSTOMS)', value: 'CUSTOMS' },
                  {
                    label: '不可抗力 (FORCE_MAJEURE)',
                    value: 'FORCE_MAJEURE',
                  },
                  { label: '其他原因 (OTHER)', value: 'OTHER' },
                ]}
              />
            </Form.Item>
          </Col>
          <Col span={16}>
            <Form.Item
              name="reason"
              label="改配原因说明"
              rules={[{ required: true, message: '请输入详细改配原因' }]}
            >
              <TextArea
                rows={2}
                placeholder="详细记录改配发生的背景与原因（将记录进不可变事件历史与审计日志）"
              />
            </Form.Item>
          </Col>
        </Row>

        <SeaExternalConfirmationFields orderId={orderId} />
      </Form>

      {!targetInput && (
        <Alert
          type="info"
          showIcon
          title={
            targetType === 'candidate'
              ? '请匹配已有母单，并明确选择目标实际航次后预览'
              : '填写目标提单号并选择船公司后，系统将校验改配条件'
          }
        />
      )}
      {matchLookup && matchQuery.data?.matched === false && (
        <Alert
          type="info"
          showIcon
          title="未找到已有母单，将按填写的航程新建母单"
        />
      )}
      <Button
        onClick={() => void previewQuery.refetch()}
        disabled={
          !previewEnabled || previewing || submitting || Boolean(confirmation)
        }
        style={{ marginTop: 12 }}
      >
        重新预览
      </Button>

      {/* 航程要素比对表格 */}
      <div style={{ marginTop: 12 }}>
        <Text
          strong
          style={{ fontSize: 14, marginBottom: 8, display: 'block' }}
        >
          航程要素变动实时比对：
        </Text>
        {previewing ? (
          <div style={{ textAlign: 'center', padding: 20 }}>
            <Spin description="计算比对差异..." />
          </div>
        ) : (
          <>
            <div
              style={{
                display: 'flex',
                justifyContent: 'flex-end',
                marginBottom: 8,
              }}
            >
              {columnSettings.entry}
            </div>
            <Table<API.VoyageDifferenceItem>
              columns={columnSettings.columns}
              dataSource={previewData?.differences || []}
              rowKey="fieldName"
              pagination={false}
              size="small"
              bordered
            />
          </>
        )}
      </div>
      <Modal
        title="确认提交整票改配？"
        open={Boolean(confirmation)}
        onCancel={() => setConfirmation(null)}
        onOk={executeConfirmed}
        confirmLoading={submitting}
        cancelButtonProps={{ disabled: submitting }}
        closable={!submitting}
        mask={{ closable: !submitting }}
        okText="确认改配"
        cancelText="返回修改"
      >
        <p>
          保留当前操作票号与流程状态，不会复制订单；当前航程将切换为目标 MBL
          权威航程。
        </p>
        <p>目标母单：{confirmation?.target.masterNo}</p>
        <p>请核对航程要素变动表。执行后将生成不可变改配历史记录。</p>
      </Modal>
    </Modal>
  );
}

export function SeaOrderReassignmentModal(
  props: SeaOrderReassignmentModalProps,
) {
  return props.open ? (
    <ReassignmentForm key={props.orderId} {...props} />
  ) : null;
}

export default SeaOrderReassignmentModal;
