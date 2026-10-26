import { useQueries, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  App,
  Button,
  DatePicker,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Tag,
  Typography,
} from 'antd';
import type { DefaultOptionType } from 'antd/es/select';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useEffect, useMemo, useRef, useState } from 'react';
import { useUnsavedChangesGuard } from '@/components/layout/useUnsavedChangesGuard';
import { ResponsiveTable as Table } from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import { orderServiceGetOrder } from '@/services/roncin/orderService';
import {
  seaOrderChangeServiceExecuteSeaTransportExecutionUpdate,
  seaOrderChangeServicePreviewSeaTransportExecutionUpdate,
} from '@/services/roncin/seaOrderChangeService';
import { isWriteResponse } from '@/utils/api';
import { formatDate } from '@/utils/format';
import { generateUUID } from '@/utils/uuid';
import SeaExternalConfirmationFields, {
  buildSeaExternalConfirmation,
  type SeaExternalConfirmationFormValues,
} from '../../templates/components/sea/SeaExternalConfirmationFields';

const { Text } = Typography;
const locationQueryKey = ['orders', 'shared-voyage-location'] as const;

type VoyageUpdateFormValues = SeaExternalConfirmationFormValues & {
  originLocationId?: string;
  dischargeLocationId?: string;
  transitLocationId?: string;
  vesselName?: string;
  voyageNo?: string;
  etd?: dayjs.Dayjs;
  eta?: dayjs.Dayjs;
  reason: string;
};

type SeaTransportExecutionUpdateModalProps = {
  open: boolean;
  order: API.Order;
  onClose: () => void;
  onSuccess: () => Promise<void> | void;
  searchLocations?: (keyword?: string) => Promise<DefaultOptionType[]>;
};

function toUpdateInput(
  values: VoyageUpdateFormValues,
): API.SeaTransportExecutionUpdateInput {
  return {
    originLocationId: values.originLocationId || undefined,
    dischargeLocationId: values.dischargeLocationId || undefined,
    transitLocationId: values.transitLocationId || undefined,
    vesselName: values.vesselName?.trim() || undefined,
    voyageNo: values.voyageNo?.trim() || undefined,
    etd: values.etd
      ? dayjs(values.etd).startOf('day').toISOString()
      : undefined,
    eta: values.eta
      ? dayjs(values.eta).startOf('day').toISOString()
      : undefined,
  };
}

/** 一次修改共享 TransportExecution，并明确展示受影响订单。 */
export default function SeaTransportExecutionUpdateModal({
  open,
  order,
  onClose,
  onSuccess,
  searchLocations,
}: SeaTransportExecutionUpdateModalProps) {
  const { markDirty } = useUnsavedChangesGuard({
    enabled: open,
    identity: order.id,
  });
  const [form] = Form.useForm<VoyageUpdateFormValues>();
  const { message } = App.useApp();
  const [previewing, setPreviewing] = useState(false);
  const [executing, setExecuting] = useState(false);
  const [preview, setPreview] =
    useState<API.SeaTransportExecutionUpdatePreviewData>();
  const [previewInput, setPreviewInput] =
    useState<API.SeaTransportExecutionUpdateInput>();
  const [idempotencyKey, setIdempotencyKey] = useState('');
  const previewRequestId = useRef(0);
  const queryClient = useQueryClient();
  const [locationKeyword, setLocationKeyword] = useState<string>();
  const locationQuery = useQuery({
    queryKey: [...locationQueryKey, locationKeyword],
    enabled: open && locationKeyword !== undefined && Boolean(searchLocations),
    queryFn: ({ queryKey }) => {
      if (!searchLocations) throw new Error('地点查询不可用');
      return searchLocations(queryKey[2] || undefined);
    },
    meta: { silent: true },
  });

  const transportExecution = order.seaMasterBill;
  const expectedVersion = transportExecution?.transportExecutionVersion;
  const memberOrderIds = open ? (preview?.memberOrderIds ?? []) : [];
  const memberOrders = useQueries({
    queries: memberOrderIds.map((id) => ({
      queryKey: ['orders', 'shared-voyage-member', id],
      queryFn: async () => {
        const response = await orderServiceGetOrder({ id });
        if (!response?.data?.orderNo) throw new Error('订单号读取失败');
        return response.data;
      },
      meta: { silent: true },
    })),
  });

  const initialLocationOptions = useMemo<DefaultOptionType[]>(
    () =>
      [
        [
          transportExecution?.originLocationId,
          transportExecution?.originLocationName,
        ],
        [
          transportExecution?.dischargeLocationId,
          transportExecution?.dischargeLocationName,
        ],
        [
          transportExecution?.transitLocationId,
          transportExecution?.transitLocationName,
        ],
      ]
        .filter(([id, name]) => id && name)
        .map(([value, label]) => ({ value, label })),
    [transportExecution],
  );
  // 已选地点从当前工作区查询缓存保活，搜索响应只属于各自关键词。
  const allLocationOptions = [
    ...initialLocationOptions,
    ...(locationQuery.data ?? []),
    ...queryClient
      .getQueriesData<DefaultOptionType[]>({ queryKey: locationQueryKey })
      .flatMap(([, options]) => options ?? []),
  ];

  const initialValues = useMemo<VoyageUpdateFormValues>(
    () => ({
      originLocationId: transportExecution?.originLocationId,
      dischargeLocationId: transportExecution?.dischargeLocationId,
      transitLocationId: transportExecution?.transitLocationId,
      vesselName: transportExecution?.vesselName,
      voyageNo: transportExecution?.voyageNo,
      etd: transportExecution?.etd ? dayjs(transportExecution.etd) : undefined,
      eta: transportExecution?.eta ? dayjs(transportExecution.eta) : undefined,
      reason: '',
      confirmedAt: dayjs(),
    }),
    [transportExecution],
  );

  useEffect(() => {
    previewRequestId.current += 1;
    if (!open) return;
    form.setFieldsValue(initialValues);
    setPreview(undefined);
    setPreviewInput(undefined);
    setPreviewing(false);
    setLocationKeyword(undefined);
    setIdempotencyKey(`sea-transport-update-${generateUUID()}`);
    return () => {
      previewRequestId.current += 1;
    };
  }, [form, initialValues, open]);

  const handlePreview = async () => {
    if (!order.id || !expectedVersion) return;
    const requestId = ++previewRequestId.current;
    try {
      const values = await form.validateFields();
      if (requestId !== previewRequestId.current) return;
      const input = toUpdateInput(values);
      setPreviewing(true);
      setPreview(undefined);
      const response =
        await seaOrderChangeServicePreviewSeaTransportExecutionUpdate(
          { orderId: order.id },
          {
            orderId: order.id,
            expectedTransportExecutionVersion: expectedVersion,
            input,
            reason: values.reason.trim(),
          },
        );
      if (requestId !== previewRequestId.current) return;
      setPreview(response?.data);
      setPreviewInput(input);
    } catch (error: unknown) {
      if (requestId !== previewRequestId.current) return;
      if (!(typeof error === 'object' && error && 'errorFields' in error)) {
        message.error(
          error instanceof Error ? error.message : '共享航程预览失败',
        );
      }
    } finally {
      if (requestId === previewRequestId.current) setPreviewing(false);
    }
  };

  const handleExecute = async (): Promise<false | undefined> => {
    if (
      !order.id ||
      !expectedVersion ||
      !preview?.executable ||
      !previewInput
    ) {
      return;
    }
    try {
      const values = await form.validateFields();
      setExecuting(true);
      const writeResponse1 =
        await seaOrderChangeServiceExecuteSeaTransportExecutionUpdate(
          { orderId: order.id },
          {
            orderId: order.id,
            expectedTransportExecutionVersion: expectedVersion,
            input: previewInput,
            reason: values.reason.trim(),
            confirmation: buildSeaExternalConfirmation(values),
            idempotencyKey,
          },
        );
      if (
        !isWriteResponse(writeResponse1) ||
        !writeResponse1.transportExecution?.id ||
        !writeResponse1.versionId
      )
        return false;
      await onSuccess();
      onClose();
    } catch (error: unknown) {
      if (!(typeof error === 'object' && error && 'errorFields' in error)) {
        message.error(
          error instanceof Error ? error.message : '共享航程修改失败',
        );
      }
    } finally {
      setExecuting(false);
    }

    return undefined;
  };

  const locationSelect = (
    name: 'originLocationId' | 'dischargeLocationId' | 'transitLocationId',
    label: string,
  ) => (
    <Form.Item name={name} label={label}>
      <Select
        allowClear
        showSearch={{
          filterOption: false,
          onSearch: (keyword) => setLocationKeyword(keyword.trim()),
        }}
        loading={locationQuery.isFetching}
        options={[
          ...allLocationOptions.filter(
            (option) => option.value === form.getFieldValue(name),
          ),
          ...(locationQuery.data ?? []),
        ].filter(
          (entry, index, options) =>
            options.findIndex((option) => option.value === entry.value) ===
            index,
        )}
        onFocus={() => setLocationKeyword('')}
      />
    </Form.Item>
  );

  const renderDifferenceValue = (
    value: string | undefined,
    row: API.VoyageDifferenceItem,
  ) => {
    if (!value) return '未填写';
    if (row.fieldName === 'etd' || row.fieldName === 'eta') {
      return formatDate(value, 'date');
    }
    if (row.fieldName?.endsWith('_location_id')) {
      return (
        allLocationOptions.find((option) => option.value === value)?.label ??
        `港口名称读取失败（${value}）`
      );
    }
    return value;
  };
  const differenceColumns: ColumnsType<API.VoyageDifferenceItem> = [
    { title: '字段', dataIndex: 'label' },
    {
      title: '调整前',
      dataIndex: 'currentValue',
      render: renderDifferenceValue,
    },
    {
      title: '调整后',
      dataIndex: 'targetValue',
      render: renderDifferenceValue,
    },
  ];

  const columnSettings = useColumnSettings<
    ColumnsType<API.VoyageDifferenceItem>[number]
  >({
    tableKey: 'orders:voyage-difference',
    columns: differenceColumns,
  });

  return (
    <Modal
      width={920}
      title={`修改共享航程 · ${order.orderNo || ''}`}
      open={open}
      onCancel={onClose}
      destroyOnHidden
      footer={[
        <Button key="cancel" onClick={onClose}>
          取消
        </Button>,
        <Button
          key="preview"
          disabled={previewing || executing || !expectedVersion}
          onClick={() => void handlePreview()}
        >
          {previewing ? '预览中…' : '预览影响'}
        </Button>,
        <Button
          type="primary"
          key="execute"
          disabled={!preview?.executable || previewing || executing}
          onClick={() => void handleExecute()}
        >
          {executing ? '执行中…' : '确认统一调整'}
        </Button>,
      ]}
    >
      {!expectedVersion ? (
        <Text type="danger">
          当前订单缺少有效的运输执行版本，请刷新后重试。
        </Text>
      ) : null}
      <Form
        form={form}
        layout="vertical"
        onValuesChange={() => {
          markDirty();
          previewRequestId.current += 1;
          setPreviewing(false);
          setPreview(undefined);
          setPreviewInput(undefined);
        }}
      >
        <Space align="start" size={16} style={{ width: '100%' }}>
          <Form.Item name="vesselName" label="船名">
            <Input maxLength={128} />
          </Form.Item>
          <Form.Item name="voyageNo" label="航次">
            <Input maxLength={128} />
          </Form.Item>
          <Form.Item name="etd" label="ETD">
            <DatePicker />
          </Form.Item>
          <Form.Item name="eta" label="ETA">
            <DatePicker />
          </Form.Item>
        </Space>
        <Space align="start" size={16} style={{ width: '100%' }}>
          {locationSelect('originLocationId', '起运港')}
          {locationSelect('dischargeLocationId', '卸货港')}
          {locationSelect('transitLocationId', '中转港')}
        </Space>
        <Form.Item
          name="reason"
          label="调整原因"
          rules={[
            { required: true, whitespace: true, message: '请输入调整原因' },
          ]}
        >
          <Input.TextArea rows={2} maxLength={500} showCount />
        </Form.Item>
        <SeaExternalConfirmationFields orderId={order.id || ''} />
      </Form>

      {preview ? (
        <Space orientation="vertical" size={12} style={{ width: '100%' }}>
          <Text strong>
            本次将影响 {preview.memberOrderIds?.length ?? 0} 张关联订单
          </Text>
          <Space wrap>
            {memberOrderIds.map((id, index) => {
              const result = memberOrders[index];
              return result?.data?.orderNo ? (
                <Typography.Link
                  key={id}
                  href={`/orders/sea-export/${id}`}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  {result.data.orderNo}
                </Typography.Link>
              ) : (
                <Text key={id} type={result?.isError ? 'danger' : 'secondary'}>
                  {result?.isError ? '订单读取失败' : '订单读取中'}（{id}）
                </Text>
              );
            })}
          </Space>
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
            size="small"
            pagination={false}
            rowKey={(row) => row.fieldName || row.label || ''}
            dataSource={preview.differences ?? []}
            columns={columnSettings.columns}
          />
          {(preview.impacts ?? []).map((impact) => (
            <Tag
              key={`${impact.factType}-${impact.referenceId}`}
              color="orange"
            >
              {impact.message}
            </Tag>
          ))}
        </Space>
      ) : null}
    </Modal>
  );
}
