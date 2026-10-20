import { Button, Form, Input, Tooltip, Typography } from 'antd';
import { formatDate, trimDecimal } from '@/utils/format';

export function FeeQuantityValue({ fee }: { fee: API.OrderFee }) {
  const source = fee.quantitySource;
  if (!source?.confirmationId) return <>{trimDecimal(fee.quantity)}</>;
  return (
    <Tooltip
      title={
        <>
          <div>
            {source.mode === 'DIRECT'
              ? '直接采用确认计费重'
              : '参考确认计费重后人工调整'}{' '}
            · v{source.confirmationVersion}
          </div>
          <div>
            参考 {trimDecimal(source.referenceQuantity)} {source.unit}，采用{' '}
            {trimDecimal(source.adoptedQuantity)} {source.unit}
          </div>
          <div>
            货量来源：
            {source.source === 'PLANNED' ? '订单委托货量' : '实际货物明细'} ·
            确认于 {formatDate(source.confirmedAt, 'datetime')}
          </div>
          {source.reason && <div>依据：{source.reason}</div>}
        </>
      }
    >
      <span>
        {trimDecimal(fee.quantity)}{' '}
        <Typography.Text type="secondary" style={{ fontSize: 11 }}>
          {source.mode === 'DIRECT' ? '确认重' : '调整'}
        </Typography.Text>
      </span>
    </Tooltip>
  );
}

/** 来源数值仅用于当前编辑态；权威快照仍由服务端锁内校验并保存。 */
export function quantitySelectionForSave(
  selection: API.FeeQuantitySelection | undefined,
  quantity: string | undefined,
): API.FeeQuantitySelection | undefined {
  if (!selection?.confirmationId) return undefined;
  const reference = selection.referenceQuantity || selection.adoptedQuantity;
  const adjusted =
    selection.mode === 'MANUAL_ADJUSTED' ||
    (reference !== undefined && Number(quantity) !== Number(reference));
  return {
    mode: adjusted ? 'MANUAL_ADJUSTED' : 'DIRECT',
    confirmationId: selection.confirmationId,
    confirmationVersion: selection.confirmationVersion,
    reason: selection.reason,
  };
}

export function FeeQuantitySourceFields({
  prefix = [],
  onSourceChange,
  disabled = false,
}: {
  prefix?: (string | number)[];
  onSourceChange: () => void;
  disabled?: boolean;
}) {
  return (
    <Form.Item noStyle shouldUpdate>
      {(form) => {
        const source = form.getFieldValue([...prefix, 'quantitySource']) as
          | API.FeeQuantitySelection
          | undefined;
        const adjusted =
          quantitySelectionForSave(
            source,
            form.getFieldValue([...prefix, 'quantity']),
          )?.mode === 'MANUAL_ADJUSTED';
        return (
          <>
            {[
              'mode',
              'confirmationId',
              'confirmationVersion',
              'adoptedQuantity',
              'referenceQuantity',
              'unit',
            ].map((key) => (
              <Form.Item
                key={key}
                name={[...prefix, 'quantitySource', key]}
                hidden
              >
                <Input />
              </Form.Item>
            ))}
            {source?.confirmationId && (
              <>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  确认计费重 v{source.confirmationVersion} · 参考{' '}
                  {source.referenceQuantity || source.adoptedQuantity} KG ·{' '}
                  {adjusted ? '人工调整' : '直接带入'}
                </Typography.Text>
                <Button
                  type="link"
                  size="small"
                  disabled={disabled}
                  onClick={() => {
                    form.setFieldValue([...prefix, 'quantitySource'], {
                      mode: 'MANUAL',
                    });
                    onSourceChange();
                  }}
                >
                  改为纯手工数量
                </Button>
                {adjusted && (
                  <Form.Item
                    name={[...prefix, 'quantitySource', 'reason']}
                    label="数量调整依据"
                    rules={[
                      {
                        required: true,
                        whitespace: true,
                        message: '请填写数量调整依据',
                      },
                    ]}
                  >
                    <Input maxLength={500} disabled={disabled} />
                  </Form.Item>
                )}
              </>
            )}
          </>
        );
      }}
    </Form.Item>
  );
}
