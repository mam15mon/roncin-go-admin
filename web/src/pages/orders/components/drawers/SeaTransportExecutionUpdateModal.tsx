import {
  App,
  Button,
  DatePicker,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { DefaultOptionType } from 'antd/es/select';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useEffect, useMemo, useState } from 'react';
import { useColumnSettings } from '@/components/ui/column-settings';
import {
  seaOrderChangeServiceExecuteSeaTransportExecutionUpdate,
  seaOrderChangeServicePreviewSeaTransportExecutionUpdate,
} from '@/services/roncin/seaOrderChangeService';
import { generateUUID } from '@/utils/uuid';
import SeaExternalConfirmationFields, {
  buildSeaExternalConfirmation,
  type SeaExternalConfirmationFormValues,
} from '../../templates/components/sea/SeaExternalConfirmationFields';

const { Text } = Typography;

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
    etd: values.etd ? dayjs(values.etd).toISOString() : undefined,
    eta: values.eta ? dayjs(values.eta).toISOString() : undefined,
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
  const [form] = Form.useForm<VoyageUpdateFormValues>();
  const { message } = App.useApp();
  const [previewing, setPreviewing] = useState(false);
  const [executing, setExecuting] = useState(false);
  const [preview, setPreview] =
    useState<API.SeaTransportExecutionUpdatePreviewData>();
  const [previewInput, setPreviewInput] =
    useState<API.SeaTransportExecutionUpdateInput>();
  const [idempotencyKey, setIdempotencyKey] = useState('');
  const [locationOptions, setLocationOptions] = useState<DefaultOptionType[]>(
    [],
  );

  const transportExecution = order.seaMasterBill;
  const expectedVersion = (
    transportExecution as
      | (API.SeaMasterBillSummary & { transportExecutionVersion?: string })
      | undefined
  )?.transportExecutionVersion;

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
    if (!open) return;
    form.setFieldsValue(initialValues);
    setPreview(undefined);
    setPreviewInput(undefined);
    setIdempotencyKey(`sea-transport-update-${generateUUID()}`);
  }, [form, initialValues, open]);

  const loadLocations = async (keyword?: string) => {
    if (!searchLocations) return;
    setLocationOptions(await searchLocations(keyword));
  };

  const handlePreview = async () => {
    if (!order.id || !expectedVersion) return;
    try {
      const values = await form.validateFields();
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
      setPreview(response.data);
      setPreviewInput(input);
    } catch (error: unknown) {
      if (!(typeof error === 'object' && error && 'errorFields' in error)) {
        message.error(
          error instanceof Error ? error.message : '共享航次预览失败',
        );
      }
    } finally {
      setPreviewing(false);
    }
  };

  const handleExecute = async () => {
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
      await onSuccess();
      onClose();
    } catch (error: unknown) {
      if (!(typeof error === 'object' && error && 'errorFields' in error)) {
        message.error(
          error instanceof Error ? error.message : '共享航次调整失败',
        );
      }
    } finally {
      setExecuting(false);
    }
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
          onSearch: (keyword) => void loadLocations(keyword),
        }}
        options={locationOptions}
        onFocus={() => void loadLocations()}
      />
    </Form.Item>
  );

  const differenceColumns: ColumnsType<API.VoyageDifferenceItem> = [
    { title: '字段', dataIndex: 'label' },
    { title: '调整前', dataIndex: 'currentValue' },
    { title: '调整后', dataIndex: 'targetValue' },
  ];

  const columnSettings =
    useColumnSettings<ColumnsType<API.VoyageDifferenceItem>[number]>({
      tableKey: 'orders:voyage-difference',
      columns: differenceColumns,
    });

  return (
    <Modal
      width={920}
      title={`共享航次调整 · ${order.orderNo || ''}`}
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
            <DatePicker showTime />
          </Form.Item>
          <Form.Item name="eta" label="ETA">
            <DatePicker showTime />
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
          <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}>
            {columnSettings.entry}
          </div>
          <Table<API.VoyageDifferenceItem>
            size="small"
            pagination={false}
            rowKey={(row) => row.fieldName || row.label || ''}
            dataSource={preview.differences ?? []}
            columns={columnSettings.columns}
          />
          {columnSettings.modal}
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
