import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { Button, Form } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import {
  OrderShippingDocumentFields,
  SEA_HOUSE_RELEASE_TYPE_OPTIONS,
  SEA_MASTER_DOCUMENT_TYPE_OPTIONS,
  SEA_MASTER_RELEASE_METHOD_OPTIONS,
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

describe('主分单分组编辑', () => {
  it('提交时忽略纯空占位行并保留已有记录 ID', async () => {
    const { onFinish } = renderFields({
      initialValues: {
        shippingDocuments: [
          {
            id: 'doc-1',
            masterNo: 'MBL-001',
            masterDocumentType: 'ORIGINAL_BL',
            masterReleaseMethod: 'TELEX_RELEASE',
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
        masterNo: 'MBL-001',
        masterDocumentType: 'ORIGINAL_BL',
        masterReleaseMethod: 'TELEX_RELEASE',
        houseNo: 'HBL-001',
        releaseType: 'TELEX_RELEASE',
        note: undefined,
      },
    ]);
  });

  it('只填写主单号时阻止提交', async () => {
    const { onFinish, onFinishFailed } = renderFields();
    fireEvent.change(screen.getByPlaceholderText('请输入主单号 (如 MBL-001)'), {
      target: { value: 'MBL-001' },
    });
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));

    await waitFor(() => expect(onFinishFailed).toHaveBeenCalledTimes(1));
    expect(onFinish).not.toHaveBeenCalled();
    expect(screen.getAllByText('请填写分单号').length).toBeGreaterThan(0);
  });

  it('提交时将共享主单属性复制到同组每条分单', async () => {
    const { onFinish } = renderFields({
      initialValues: {
        shippingDocuments: [
          {
            id: 'doc-1',
            masterNo: 'MBL-001',
            masterDocumentType: 'SEA_WAYBILL',
            masterReleaseMethod: 'EXPRESS_RELEASE',
            houseNo: 'HBL-001',
          },
        ],
      },
    });

    fireEvent.click(screen.getByRole('button', { name: /添加分单 \(HBL\)/ }));
    const houseInputs = screen.getAllByPlaceholderText(
      '请输入分单号 (如 HBL-001)',
    );
    fireEvent.change(houseInputs[1], { target: { value: 'HBL-002' } });
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));

    await waitFor(() => expect(onFinish).toHaveBeenCalledTimes(1));
    expect(onFinish.mock.calls[0][0].shippingDocuments).toEqual([
      expect.objectContaining({
        id: 'doc-1',
        masterDocumentType: 'SEA_WAYBILL',
        masterReleaseMethod: 'EXPRESS_RELEASE',
      }),
      expect.objectContaining({
        houseNo: 'HBL-002',
        masterDocumentType: 'SEA_WAYBILL',
        masterReleaseMethod: 'EXPRESS_RELEASE',
      }),
    ]);
  });

  it('跨主单组按忽略大小写与首尾空格校验分单号重复', async () => {
    const { onFinish, onFinishFailed } = renderFields();
    fireEvent.change(screen.getByPlaceholderText('请输入主单号 (如 MBL-001)'), {
      target: { value: 'MBL-001' },
    });
    fireEvent.change(screen.getByPlaceholderText('请输入分单号 (如 HBL-001)'), {
      target: { value: 'HBL-001' },
    });
    fireEvent.click(screen.getByRole('button', { name: /加拼主单 \(MBL\)/ }));

    const masterInputs = screen.getAllByPlaceholderText(
      '请输入主单号 (如 MBL-001)',
    );
    const houseInputs = screen.getAllByPlaceholderText(
      '请输入分单号 (如 HBL-001)',
    );
    fireEvent.change(masterInputs[1], { target: { value: 'MBL-002' } });
    fireEvent.change(houseInputs[1], { target: { value: '  hbl-001  ' } });
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));

    await waitFor(() => expect(onFinishFailed).toHaveBeenCalledTimes(1));
    expect(onFinish).not.toHaveBeenCalled();
    expect(screen.getAllByText('分单号重复')).toHaveLength(2);
  });

  it('当前操作票内不允许重复添加同一主单组', async () => {
    const { onFinish, onFinishFailed } = renderFields();
    fireEvent.change(screen.getByPlaceholderText('请输入主单号 (如 MBL-001)'), {
      target: { value: 'MBL-001' },
    });
    fireEvent.change(screen.getByPlaceholderText('请输入分单号 (如 HBL-001)'), {
      target: { value: 'HBL-001' },
    });
    fireEvent.click(screen.getByRole('button', { name: /加拼主单 \(MBL\)/ }));

    const masterInputs = screen.getAllByPlaceholderText(
      '请输入主单号 (如 MBL-001)',
    );
    const houseInputs = screen.getAllByPlaceholderText(
      '请输入分单号 (如 HBL-001)',
    );
    fireEvent.change(masterInputs[1], { target: { value: '  mbl-001  ' } });
    fireEvent.change(houseInputs[1], { target: { value: 'HBL-002' } });
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));

    await waitFor(() => expect(onFinishFailed).toHaveBeenCalledTimes(1));
    expect(onFinish).not.toHaveBeenCalled();
    expect(
      screen.getAllByText('该主单已在当前操作票中，请在原主单组下添加分单'),
    ).toHaveLength(2);
  });

  it('已放货分单及所属主单不可编辑或删除', () => {
    renderFields({
      initialValues: {
        shippingDocuments: [
          {
            id: 'released-doc',
            masterNo: 'MBL-001',
            masterDocumentType: 'ORIGINAL_BL',
            masterReleaseMethod: 'TELEX_RELEASE',
            houseNo: 'HBL-001',
            status: 3,
          },
        ],
      },
    });

    expect(screen.getByDisplayValue('MBL-001')).toBeDisabled();
    expect(screen.getByDisplayValue('HBL-001')).toBeDisabled();
    expect(screen.getAllByRole('combobox')[0]).toBeDisabled();
    expect(screen.getAllByRole('combobox')[1]).toBeDisabled();
    expect(
      screen.getByRole('button', { name: '删除分单 HBL-001' }),
    ).toBeDisabled();
  });

  it('空运使用 MAWB 和 HAWB 文案', () => {
    renderFields({ transportMode: 'air' });

    expect(screen.getByText('主单 (MAWB)')).toBeTruthy();
    expect(screen.getByText('分单号 (HAWB)')).toBeTruthy();
    expect(
      screen.getByPlaceholderText('请输入主单号 (如 MAWB-001)'),
    ).toBeTruthy();
    expect(
      screen.getByPlaceholderText('请输入分单号 (如 HAWB-001)'),
    ).toBeTruthy();
    expect(screen.queryByText('主单单证类型')).toBeNull();
    expect(screen.queryByText('主单签放方式')).toBeNull();
  });

  it('海运展示共享主单属性并说明跨操作票影响', () => {
    renderFields();

    expect(screen.getByText('主单单证类型')).toBeTruthy();
    expect(screen.getByText('主单签放方式')).toBeTruthy();
    expect(
      screen.getByText(
        '主单属性属于共享主单批次，修改后会影响其他引用同一主单的操作票。',
      ),
    ).toBeTruthy();
  });

  it('海运分单签放选项不混入放行和寄单状态', () => {
    expect(
      SEA_HOUSE_RELEASE_TYPE_OPTIONS.map((option) => option.value),
    ).toEqual(['TELEX_RELEASE', 'ORIGINAL', 'SEA_WAYBILL']);
  });

  it('海运主单单证与签放选项分别建模', () => {
    expect(
      SEA_MASTER_DOCUMENT_TYPE_OPTIONS.map((option) => option.value),
    ).toEqual(['ORIGINAL_BL', 'SEA_WAYBILL']);
    expect(
      SEA_MASTER_RELEASE_METHOD_OPTIONS.map((option) => option.value),
    ).toEqual(['ORIGINAL', 'TELEX_RELEASE', 'EXPRESS_RELEASE']);
  });
});
