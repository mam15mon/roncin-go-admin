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
import { formatDate, trimDecimal } from '@/utils/format';

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
  allowChargeable?: boolean;
  onConfirm: (quantity: string, source?: API.FeeQuantitySelection) => void;
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
  allowChargeable = true,
}: FeeQuantityAssistProps) {
  const [open, setOpen] = useState(false);
  const [source, setSource] = useState<'PLANNED' | 'ACTUAL'>('ACTUAL');
  const [basis, setBasis] = useState<
    'GROSS_WEIGHT' | 'CONFIRMED_CHARGEABLE_WEIGHT'
  >('GROSS_WEIGHT');
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
          setBasis('GROSS_WEIGHT');
          setOpen(true);
        }}
      >
        从订单带入
      </Button>
      {open && billingUnit?.id && (
        <QuantityPreview
          key={`${source}:${basis}`}
          orderId={orderId}
          billingUnitId={billingUnit.id}
          unitLabel={billingUnit.name || billingUnit.code || ''}
          source={source}
          basis={basis}
          showWeightBasis={allowChargeable && billingUnit.code === 'KG'}
          onBasisChange={setBasis}
          containerUnit={Boolean(containerUnit)}
          onSourceChange={setSource}
          onCancel={() => setOpen(false)}
          onConfirm={(quantity, selection) => {
            if (selection) onConfirm(quantity, selection);
            else onConfirm(quantity);
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
  basis,
  showWeightBasis,
  onBasisChange,
  containerUnit,
  onSourceChange,
  onCancel,
  onConfirm,
}: {
  orderId: string;
  billingUnitId: string;
  unitLabel: string;
  source: 'PLANNED' | 'ACTUAL';
  basis: 'GROSS_WEIGHT' | 'CONFIRMED_CHARGEABLE_WEIGHT';
  showWeightBasis: boolean;
  onBasisChange: (
    basis: 'GROSS_WEIGHT' | 'CONFIRMED_CHARGEABLE_WEIGHT',
  ) => void;
  containerUnit: boolean;
  onSourceChange: (source: 'PLANNED' | 'ACTUAL') => void;
  onCancel: () => void;
  onConfirm: (quantity: string, source?: API.FeeQuantitySelection) => void;
}) {
  const preview = useQuery({
    queryKey: [
      'orders',
      'fee-quantity-preview',
      orderId,
      billingUnitId,
      source,
      basis,
    ],
    queryFn: async ({ signal }) => {
      const response = await orderFeeServicePreviewFeeQuantity(
        {
          orderId,
          billingUnitId,
          source,
          ...(basis === 'CONFIRMED_CHARGEABLE_WEIGHT'
            ? { quantityBasis: basis }
            : {}),
        },
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
        if (data?.available && data.quantity)
          onConfirm(
            data.quantity,
            data.quantitySource
              ? { ...data.quantitySource, referenceQuantity: data.quantity }
              : undefined,
          );
      }}
      okText="确认带入"
      cancelText="取消"
      okButtonProps={{ disabled: !data?.available || !data.quantity }}
      width={560}
      destroyOnHidden
    >
      {showWeightBasis && (
        <Radio.Group
          aria-label="重量计费依据"
          value={basis}
          onChange={(event) => onBasisChange(event.target.value)}
          options={[
            { label: '本票实际毛重', value: 'GROSS_WEIGHT' },
            {
              label: '本票当前确认计费重',
              value: 'CONFIRMED_CHARGEABLE_WEIGHT',
            },
          ]}
          style={{ marginBottom: 16 }}
        />
      )}
      {(containerUnit || basis === 'CONFIRMED_CHARGEABLE_WEIGHT') && (
        <Radio.Group
          aria-label={containerUnit ? '箱量来源' : '确认计费重货量来源'}
          value={source}
          onChange={(event) => onSourceChange(event.target.value)}
          options={[
            {
              label: containerUnit ? '实际登记箱量' : '实际货物明细确认重量',
              value: 'ACTUAL',
            },
            {
              label: containerUnit ? '订舱计划箱量' : '订单委托货量确认重量',
              value: 'PLANNED',
            },
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
              ...(data.quantitySource?.confirmedAt
                ? [
                    {
                      key: 'confirmed-at',
                      label: '重量确认时间',
                      children: formatDate(
                        data.quantitySource.confirmedAt,
                        'datetime',
                      ),
                    },
                  ]
                : []),
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
  allowChargeable,
}: {
  orderId: string;
  billingUnits: BillingUnit[];
  disabled?: boolean;
  active: boolean;
  onValuesChange: () => void;
  allowChargeable?: boolean;
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
          allowChargeable={allowChargeable}
          billingUnit={billingUnits.find(
            (unit) => unit.id === form.getFieldValue('billingUnitId'),
          )}
          disabled={disabled}
          active={active}
          onConfirm={(quantity, selection) => {
            form.setFieldValue('quantity', quantity);
            form.setFieldValue(
              'quantitySource',
              selection || { mode: 'MANUAL' },
            );
            onValuesChange();
            void form.validateFields(['quantity']).catch(() => undefined);
          }}
        />
      )}
    </Form.Item>
  );
}
