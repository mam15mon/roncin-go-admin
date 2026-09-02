import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { Button, Form } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import {
  OrderShippingDocumentFields,
  SEA_HOUSE_RELEASE_TYPE_OPTIONS,
} from './order-plan-fields';

function renderFields({
  initialValues,
  transportMode = 'sea',
}: {
  initialValues?: Record<string, unknown>;
  transportMode?: 'sea' | 'air';
} = {}) {
  const onFinish = vi.fn();
  const onFinishFailed = vi.fn();

  render(
    <Form
      initialValues={initialValues}
      onFinish={onFinish}
      onFinishFailed={onFinishFailed}
    >
      <OrderShippingDocumentFields transportMode={transportMode} />
      <Button htmlType="submit">保存</Button>
    </Form>,
  );

  return { onFinish, onFinishFailed };
}

describe('分单 (HBL) 编辑', () => {
  it('提交时忽略纯空占位行并保留已有记录 ID', async () => {
    const { onFinish } = renderFields({
      initialValues: {
        shippingDocuments: [
          {
            id: 'doc-1',
            houseNo: 'HBL-001',
            releaseType: 'TELEX_RELEASE',
            status: 1,
          },
        ],
      },
    });

    fireEvent.click(screen.getByRole('button', { name: /添加分单 \(HBL\)/ }));
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));

    await waitFor(() => expect(onFinish).toHaveBeenCalledTimes(1));
    expect(onFinish.mock.calls[0][0].shippingDocuments).toEqual([
      {
        id: 'doc-1',
        houseNo: 'HBL-001',
        releaseType: 'TELEX_RELEASE',
        note: undefined,
      },
    ]);
  });

  it('按忽略大小写与首尾空格校验分单号重复', async () => {
    const { onFinish, onFinishFailed } = renderFields();
    fireEvent.click(screen.getByRole('button', { name: /添加分单 \(HBL\)/ }));

    const houseInputs = screen.getAllByPlaceholderText('分单号 (HBL)');
    fireEvent.change(houseInputs[0], { target: { value: 'HBL-001' } });
    fireEvent.change(houseInputs[1], { target: { value: '  hbl-001  ' } });
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));

    await waitFor(() => expect(onFinishFailed).toHaveBeenCalledTimes(1));
    expect(onFinish).not.toHaveBeenCalled();
    const errors = onFinishFailed.mock.calls[0][0].errorFields[0].errors;
    expect(errors[0]).toMatch(/分单号 .+ 重复/);
  });

  it('填写了备注但未填分单号时阻止提交', async () => {
    const { onFinish, onFinishFailed } = renderFields();
    const noteInputs = screen.getAllByPlaceholderText('备注');
    fireEvent.change(noteInputs[0], { target: { value: 'some note' } });
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));

    await waitFor(() => expect(onFinishFailed).toHaveBeenCalledTimes(1));
    expect(onFinish).not.toHaveBeenCalled();
    const errors = onFinishFailed.mock.calls[0][0].errorFields[0].errors;
    expect(errors[0]).toBe('请填写分单号 (HBL)');
  });

  it('已放货分单不可编辑或删除', () => {
    renderFields({
      initialValues: {
        shippingDocuments: [
          {
            id: 'released-doc',
            houseNo: 'HBL-001',
            status: 3,
          },
        ],
      },
    });

    expect(screen.getByDisplayValue('HBL-001')).toBeDisabled();
    expect(screen.queryByRole('button', { name: /delete/i })).toBeNull();
  });

  it('海运分单签放选项不混入放行和寄单状态', () => {
    expect(
      SEA_HOUSE_RELEASE_TYPE_OPTIONS.map((option) => option.value),
    ).toEqual(['TELEX_RELEASE', 'ORIGINAL', 'SEA_WAYBILL']);
  });
});
