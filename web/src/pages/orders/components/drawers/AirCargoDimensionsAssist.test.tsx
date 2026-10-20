import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen } from '@testing-library/react';
import { Form, InputNumber } from 'antd';
import { describe, expect, it } from 'vitest';
import AirCargoDimensionsAssist, {
  airDimensionVolumeCbm,
} from './AirCargoDimensionsAssist';

describe('空运尺寸与 CBM 换算', () => {
  it('120 × 100 × 100 cm 单件换算为 1.2 m³，缺输入不可伪造体积', () => {
    expect(airDimensionVolumeCbm('120', '100', '100', 1)).toBe('1.2');
    expect(airDimensionVolumeCbm('120', '100', '100', 2)).toBe('2.4');
    expect(airDimensionVolumeCbm('', '100', '100', 1)).toBeUndefined();
  });
  it('明确填入覆盖当前体积，重复点击不累加', () => {
    renderWithApp(
      <Form initialValues={{ packageCount: 1, volumeCbm: 5 }}>
        <Form.Item name="volumeCbm" label="当前体积">
          <InputNumber />
        </Form.Item>
        <AirCargoDimensionsAssist />
      </Form>,
    );
    fireEvent.change(screen.getByRole('spinbutton', { name: '长度 cm' }), {
      target: { value: '120' },
    });
    fireEvent.change(screen.getByRole('spinbutton', { name: '宽度 cm' }), {
      target: { value: '100' },
    });
    fireEvent.change(screen.getByRole('spinbutton', { name: '高度 cm' }), {
      target: { value: '100' },
    });
    fireEvent.click(screen.getByRole('button', { name: '填入当前货物体积' }));
    expect(screen.getByLabelText('当前体积')).toHaveValue('1.2');
    fireEvent.click(screen.getByRole('button', { name: '填入当前货物体积' }));
    expect(screen.getByLabelText('当前体积')).toHaveValue('1.2');
  });
});
