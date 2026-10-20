import { Alert, Button, Form, InputNumber, Space } from 'antd';
import Decimal from 'decimal.js';
import { useState } from 'react';
import { useUnsavedChangesGuard } from '@/components/layout/useUnsavedChangesGuard';

export function airDimensionVolumeCbm(
  length: string,
  width: string,
  height: string,
  count: number,
): string | undefined {
  try {
    const values = [length, width, height].map((value) => new Decimal(value));
    if (
      values.some((value) => !value.isFinite() || value.lte(0)) ||
      !Number.isSafeInteger(count) ||
      count < 1
    )
      return undefined;
    return values
      .reduce((volume, value) => volume.mul(value), new Decimal(count))
      .div(1000000)
      .toDecimalPlaces(6, Decimal.ROUND_HALF_UP)
      .toString();
  } catch {
    return undefined;
  }
}

/** 尺寸只辅助填写当前行体积，保存与重量计算始终只读取一次 CBM。 */
export default function AirCargoDimensionsAssist() {
  const form = Form.useFormInstance();
  const [length, setLength] = useState('');
  const [width, setWidth] = useState('');
  const [height, setHeight] = useState('');
  const count = Number(Form.useWatch('packageCount', form) || 1);
  const volume = airDimensionVolumeCbm(length, width, height, count);
  const { markDirty } = useUnsavedChangesGuard({ enabled: true });
  return (
    <Alert
      type="info"
      title="厘米尺寸辅助换算"
      style={{ marginBottom: 12 }}
      description={
        <Space orientation="vertical" size={8}>
          <Space wrap>
            <InputNumber
              aria-label="长度 cm"
              placeholder="长 cm"
              stringMode
              min="0.001"
              value={length}
              onChange={(value) => {
                setLength(value || '');
                markDirty();
              }}
            />
            <InputNumber
              aria-label="宽度 cm"
              placeholder="宽 cm"
              stringMode
              min="0.001"
              value={width}
              onChange={(value) => {
                setWidth(value || '');
                markDirty();
              }}
            />
            <InputNumber
              aria-label="高度 cm"
              placeholder="高 cm"
              stringMode
              min="0.001"
              value={height}
              onChange={(value) => {
                setHeight(value || '');
                markDirty();
              }}
            />
          </Space>
          <span>
            {count} 件 · 换算体积 {volume || '-'}{' '}
            m³。点击后填写当前体积，不与已有体积重复相加。
          </span>
          <Button
            size="small"
            disabled={!volume}
            onClick={() => {
              form.setFieldValue('volumeCbm', Number(volume));
              markDirty();
              void form.validateFields(['volumeCbm']).catch(() => undefined);
            }}
          >
            填入当前货物体积
          </Button>
        </Space>
      }
    />
  );
}
