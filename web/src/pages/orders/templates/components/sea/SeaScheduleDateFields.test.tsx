import { ProForm } from '@ant-design/pro-components';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import dayjs from 'dayjs';
import { describe, expect, it, vi } from 'vitest';
import { SeaScheduleDateFields } from './SeaTransportSection';

const etd = dayjs('2026-10-20T12:00:00');
const eta = dayjs('2026-10-25T12:00:00');
const etdRequired = '请选择 ETD（预计离港时间）';
const etaRequired = '请选择 ETA（预计到港时间）';

function setup(initialValues: { etd?: dayjs.Dayjs; eta?: dayjs.Dayjs }) {
  const onFinish = vi.fn().mockResolvedValue(true);
  render(
    <ProForm
      initialValues={initialValues}
      onFinish={onFinish}
      submitter={false}
    >
      <SeaScheduleDateFields />
      <button type="submit">保存草稿</button>
    </ProForm>,
  );
  return onFinish;
}

describe('海运订单草稿船期必填', () => {
  it('ETD、ETA 显示必填标识', () => {
    setup({});
    for (const label of ['ETD', 'ETA']) {
      expect(screen.getByText(label).closest('label')).toHaveClass(
        'ant-form-item-required',
      );
    }
  });

  it.each([
    {
      name: '两个日期均缺失',
      initialValues: {},
      errors: [etdRequired, etaRequired],
    },
    { name: '仅缺 ETD', initialValues: { eta }, errors: [etdRequired] },
    { name: '仅缺 ETA', initialValues: { etd }, errors: [etaRequired] },
  ])('$name 时阻止保存并明确提示', async ({ initialValues, errors }) => {
    const onFinish = setup(initialValues);
    fireEvent.click(screen.getByRole('button', { name: '保存草稿' }));

    for (const error of errors) {
      await waitFor(() => expect(screen.getByText(error)).toBeVisible());
    }
    for (const error of [etdRequired, etaRequired].filter(
      (message) => !errors.includes(message),
    )) {
      expect(screen.queryByText(error)).not.toBeInTheDocument();
    }
    expect(onFinish).not.toHaveBeenCalled();
  });

  it.each([
    { name: 'ETA 晚于 ETD', arrival: eta },
    {
      name: 'ETA 与 ETD 同日，即使带入的时分更早',
      arrival: etd.startOf('day'),
    },
  ])('$name 时允许保存', async ({ arrival }) => {
    const onFinish = setup({ etd, eta: arrival });
    expect(screen.getByLabelText('ETD')).toHaveValue('2026-10-20');
    expect(screen.getByLabelText('ETA')).toHaveValue(
      arrival.format('YYYY-MM-DD'),
    );
    fireEvent.click(screen.getByRole('button', { name: '保存草稿' }));

    await waitFor(() => expect(onFinish).toHaveBeenCalledOnce());
    expect(screen.queryByText(etdRequired)).not.toBeInTheDocument();
    expect(screen.queryByText(etaRequired)).not.toBeInTheDocument();
    expect(screen.queryByText('ETA 不能早于 ETD')).not.toBeInTheDocument();
  });

  it('ETA 早于 ETD 时仍阻止保存', async () => {
    const onFinish = setup({ etd, eta: etd.subtract(1, 'day') });
    fireEvent.click(screen.getByRole('button', { name: '保存草稿' }));

    await waitFor(() =>
      expect(screen.getByText('ETA 不能早于 ETD')).toBeVisible(),
    );
    expect(onFinish).not.toHaveBeenCalled();
  });

  it('把 ETD 改到已有 ETA 之后，清空 ETA 后必须重新填写才能保存', async () => {
    const onFinish = setup({ etd, eta });
    const departureInput = screen.getByLabelText('ETD');
    fireEvent.change(departureInput, {
      target: { value: '2026-10-26' },
    });
    fireEvent.keyDown(departureInput, { key: 'Enter' });

    await waitFor(() => expect(screen.getByLabelText('ETA')).toHaveValue(''));
    fireEvent.click(screen.getByRole('button', { name: '保存草稿' }));
    await waitFor(() => expect(screen.getByText(etaRequired)).toBeVisible());
    expect(onFinish).not.toHaveBeenCalled();
  });

  it('ETD 改到 ETA 同日时保留 ETA，不受带入时分影响', async () => {
    const onFinish = setup({ etd: etd.hour(23), eta: eta.hour(1) });
    const departureInput = screen.getByLabelText('ETD');
    fireEvent.change(departureInput, {
      target: { value: '2026-10-25' },
    });
    fireEvent.keyDown(departureInput, { key: 'Enter' });

    await waitFor(() => expect(departureInput).toHaveValue('2026-10-25'));
    expect(screen.getByLabelText('ETA')).toHaveValue('2026-10-25');
    fireEvent.click(screen.getByRole('button', { name: '保存草稿' }));
    await waitFor(() => expect(onFinish).toHaveBeenCalledOnce());
  });
});
