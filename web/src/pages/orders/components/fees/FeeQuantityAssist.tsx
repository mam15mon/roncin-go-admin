import { useQuery } from '@tanstack/react-query';
import {
  Alert,
  Button,
  Descriptions,
  Form,
  Modal,
  Radio,
  Spin,
  Typography,
} from 'antd';
import { useState } from 'react';
import { orderFeeServicePreviewFeeQuantity } from '@/services/roncin/orderFeeService';
import { trimDecimal } from '@/utils/format';

type BillingUnit = {
  id?: string;
  name?: string;
  code?: string;
  isContainerUnit?: boolean;
};

type FeeQuantityAssistProps = {
  orderId: string;
  billingUnit?: BillingUnit;
  disabled?: boolean;
  active?: boolean;
  onConfirm: (quantity: string) => void;
};

/** 显式请求并确认订单数量；来源、公式与精度均由服务端决定。 */
export default function FeeQuantityAssist(props: FeeQuantityAssistProps) {
  if (props.active === false || props.disabled) return null;
  return (
    <QuantityAssistSession
      key={`${props.orderId}:${props.billingUnit?.id ?? ''}`}
      {...props}
    />
  );
}

function QuantityAssistSession({
  orderId,
  billingUnit,
  onConfirm,
}: FeeQuantityAssistProps) {
  const [open, setOpen] = useState(false);
  const [source, setSource] = useState<'PLANNED' | 'ACTUAL'>('ACTUAL');
  const containerUnit =
    billingUnit?.isContainerUnit || billingUnit?.code === 'CONT';
  return (
    <>
      <Button
        type="link"
        size="small"
        style={{ padding: 0, fontSize: 12 }}
        disabled={!orderId || !billingUnit?.id}
        onClick={() => {
          setSource('ACTUAL');
          setOpen(true);
        }}
      >
        从订单带入
      </Button>
      {open && billingUnit?.id && (
        <QuantityPreview
          key={source}
          orderId={orderId}
          billingUnitId={billingUnit.id}
          unitLabel={billingUnit.name || billingUnit.code || ''}
          source={source}
          containerUnit={Boolean(containerUnit)}
          onSourceChange={setSource}
          onCancel={() => setOpen(false)}
          onConfirm={(quantity) => {
            onConfirm(quantity);
            setOpen(false);
          }}
        />
      )}
    </>
  );
}

function QuantityPreview({
  orderId,
  billingUnitId,
  unitLabel,
  source,
  containerUnit,
  onSourceChange,
  onCancel,
  onConfirm,
}: {
  orderId: string;
  billingUnitId: string;
  unitLabel: string;
  source: 'PLANNED' | 'ACTUAL';
  containerUnit: boolean;
  onSourceChange: (source: 'PLANNED' | 'ACTUAL') => void;
  onCancel: () => void;
  onConfirm: (quantity: string) => void;
}) {
  const preview = useQuery({
    queryKey: [
      'orders',
      'fee-quantity-preview',
      orderId,
      billingUnitId,
      source,
    ],
    queryFn: async ({ signal }) => {
      const response = await orderFeeServicePreviewFeeQuantity(
        { orderId, billingUnitId, source },
        { signal },
      );
      if (!response?.data) throw new Error('数量建议响应不完整，请重试');
      return response.data;
    },
    staleTime: 0,
    gcTime: 0,
    meta: { errorMessage: '获取订单数量建议失败' },
  });
  const data = preview.isFetching || preview.isError ? undefined : preview.data;
  return (
    <Modal
      title="从订单带入数量"
      open
      onCancel={onCancel}
      onOk={() => {
        if (data?.available && data.quantity) onConfirm(data.quantity);
      }}
      okText="确认带入"
      cancelText="取消"
      okButtonProps={{ disabled: !data?.available || !data.quantity }}
      width={560}
      destroyOnHidden
    >
      {containerUnit && (
        <Radio.Group
          aria-label="箱量来源"
          value={source}
          onChange={(event) => onSourceChange(event.target.value)}
          options={[
            { label: '实际登记箱量', value: 'ACTUAL' },
            { label: '订舱计划箱量', value: 'PLANNED' },
          ]}
          style={{ marginBottom: 16 }}
        />
      )}
      {preview.isFetching && (
        <Spin description="正在核对订单数量">
          <div style={{ minHeight: 100 }} />
        </Spin>
      )}
      {preview.isError && (
        <Alert
          type="error"
          showIcon
          title="获取数量建议失败"
          description="请重试，或取消后手工填写数量。"
          action={<Button onClick={() => void preview.refetch()}>重试</Button>}
        />
      )}
      {data && (
        <>
          <Descriptions
            size="small"
            column={1}
            items={[
              { key: 'unit', label: '计费单位', children: unitLabel },
              { key: 'source', label: '数据来源', children: data.sourceLabel },
              ...(data.inputs ?? []).map((input, index) => ({
                key: `input-${index}`,
                label: input.label,
                children: `${trimDecimal(input.value)} ${input.unit || ''}`,
              })),
              ...(data.formula
                ? [
                    {
                      key: 'formula',
                      label: '计算依据',
                      children: data.formula,
                    },
                  ]
                : []),
              ...(data.available
                ? [
                    {
                      key: 'quantity',
                      label: '建议数量',
                      children: (
                        <Typography.Text strong>
                          {trimDecimal(data.quantity)} {unitLabel}
                        </Typography.Text>
                      ),
                    },
                  ]
                : []),
            ]}
          />
          {!data.available && (
            <Alert
              type="warning"
              showIcon
              title="暂不能从订单带入"
              description={
                data.reason || '当前单位或订单数据不支持计算，请手工填写。'
              }
            />
          )}
          <Typography.Paragraph
            type="secondary"
            style={{ marginTop: 12, marginBottom: 0 }}
          >
            请核对来源与计算依据是否适用于本项费用。确认仅填入当前表单，仍可手工修改，保存后不会随订单变化自动重算。
          </Typography.Paragraph>
        </>
      )}
    </Modal>
  );
}

/** 监听真实表单单位字段，覆盖默认费项带入及用户主动切换。 */
export function FeeQuantityFormAssist({
  orderId,
  billingUnits,
  disabled,
  active,
  onValuesChange,
}: {
  orderId: string;
  billingUnits: BillingUnit[];
  disabled?: boolean;
  active: boolean;
  onValuesChange: () => void;
}) {
  return (
    <Form.Item
      noStyle
      shouldUpdate={(previous, current) =>
        previous.billingUnitId !== current.billingUnitId
      }
    >
      {(form) => (
        <FeeQuantityAssist
          orderId={orderId}
          billingUnit={billingUnits.find(
            (unit) => unit.id === form.getFieldValue('billingUnitId'),
          )}
          disabled={disabled}
          active={active}
          onConfirm={(quantity) => {
            form.setFieldValue('quantity', quantity);
            onValuesChange();
            void form.validateFields(['quantity']).catch(() => undefined);
          }}
        />
      )}
    </Form.Item>
  );
}
