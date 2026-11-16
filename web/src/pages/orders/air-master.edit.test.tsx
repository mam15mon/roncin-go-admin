import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { getAnyDirtyTabKey } from '@/components/layout/tabCloseGuard';
import {
  masterResponse,
  masterServices,
  renderMaster,
  setupMasterTests,
} from './air-master-test.fixtures';

setupMasterTests();

async function openEdit() {
  await waitFor(() =>
    expect(screen.getByRole('button', { name: '修改共同资料' })).toBeEnabled(),
  );
  fireEvent.click(screen.getByRole('button', { name: '修改共同资料' }));
  return screen.findByLabelText('测试主单正文');
}

describe('航空主单编辑快照', () => {
  it('预览在途继续编辑时，不采纳旧输入的迟到预览', async () => {
    renderMaster();
    const remarks = await openEdit();
    fireEvent.change(screen.getByLabelText('修改原因'), {
      target: { value: '修订正文' },
    });
    let finishPreview!: (
      value: API.PreviewAirMasterWaybillUpdateResponse,
    ) => void;
    masterServices.preview.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finishPreview = resolve;
        }),
    );
    fireEvent.click(screen.getByRole('button', { name: '预览影响' }));
    await waitFor(() => expect(masterServices.preview).toHaveBeenCalled());
    fireEvent.change(remarks, { target: { value: '预览期间的新正文' } });
    await act(async () =>
      finishPreview({ inputFingerprint: 'old-input', canExecute: true }),
    );
    expect(
      screen.queryByRole('button', { name: '确认影响并提交' }),
    ).not.toBeInTheDocument();
    expect(remarks).toHaveValue('预览期间的新正文');
    expect(masterServices.update).not.toHaveBeenCalled();
  });

  it('无修改时开始刷新，在途新增输入后仍保留原编辑基线', async () => {
    renderMaster();
    const remarks = await openEdit();
    let finishRefresh!: (value: API.GetAirMasterWaybillResponse) => void;
    masterServices.get.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finishRefresh = resolve;
        }),
    );
    fireEvent.click(screen.getByRole('button', { name: '刷 新' }));
    fireEvent.change(remarks, { target: { value: '刷新期间的新正文' } });
    await act(async () => finishRefresh(masterResponse('2')));
    expect(remarks).toHaveValue('刷新期间的新正文');
    expect(getAnyDirtyTabKey()).toBeDefined();
    fireEvent.change(screen.getByLabelText('修改原因'), {
      target: { value: '保留原基线' },
    });
    fireEvent.click(screen.getByRole('button', { name: '预览影响' }));
    await waitFor(() => expect(masterServices.preview).toHaveBeenCalled());
    expect(masterServices.preview.mock.calls[0][1].expectedVersion).toBe('1');
  });

  it('主查询变新不覆盖输入，预览和失败保存仍使用打开时版本及结构化正文', async () => {
    const view = renderMaster();
    const remarks = await openEdit();
    fireEvent.change(remarks, { target: { value: '本次主单正文' } });
    fireEvent.change(screen.getByLabelText('修改原因'), {
      target: { value: '修订印字内容' },
    });
    masterServices.get.mockResolvedValue(masterResponse('2'));
    await act(async () => {
      await view.queryClient.invalidateQueries({
        queryKey: ['air-export', 'master'],
      });
    });
    expect(remarks).toHaveValue('本次主单正文');
    expect(screen.getByLabelText('测试运输航班')).toHaveValue('运输航班-1');
    fireEvent.click(screen.getByRole('button', { name: '预览影响' }));
    await waitFor(() => expect(masterServices.preview).toHaveBeenCalled());
    expect(masterServices.preview).toHaveBeenLastCalledWith(
      { id: 'master' },
      expect.objectContaining({
        expectedVersion: '1',
        input: expect.objectContaining({
          flightNo: '运输航班-1',
          waybillContent: expect.objectContaining({ remarks: '本次主单正文' }),
        }),
      }),
    );
    fireEvent.click(
      await screen.findByRole('button', { name: '确认影响并提交' }),
    );
    await waitFor(() => expect(masterServices.update).toHaveBeenCalled());
    expect(masterServices.update.mock.calls[0][1].expectedVersion).toBe('1');
    expect(remarks).toHaveValue('本次主单正文');
    expect(getAnyDirtyTabKey()).toBeDefined();
  });

  it('刷新取消和失败保留输入、提醒与离开保护，成功后才采用新基线', async () => {
    renderMaster();
    const remarks = await openEdit();
    fireEvent.change(remarks, { target: { value: '待保留正文' } });
    masterServices.get.mockResolvedValue(masterResponse('2'));
    vi.mocked(document.hasFocus).mockReturnValue(true);
    fireEvent.focus(window);
    await waitFor(() =>
      expect(
        screen.getByText('这个主单已经发生变动，请重新刷新。'),
      ).toBeVisible(),
    );
    fireEvent.click(screen.getByRole('button', { name: '刷新主单资料' }));
    fireEvent.click(await screen.findByRole('button', { name: '保留编辑' }));
    await waitFor(() =>
      expect(
        screen.queryByRole('button', { name: '保留编辑' }),
      ).not.toBeInTheDocument(),
    );
    expect(remarks).toHaveValue('待保留正文');
    expect(getAnyDirtyTabKey()).toBeDefined();

    masterServices.get.mockResolvedValue(undefined);
    fireEvent.click(screen.getByRole('button', { name: '刷新主单资料' }));
    fireEvent.click(await screen.findByRole('button', { name: '确认刷新' }));
    await waitFor(() =>
      expect(
        screen.queryByRole('button', { name: '确认刷新' }),
      ).not.toBeInTheDocument(),
    );
    expect(remarks).toHaveValue('待保留正文');
    expect(
      screen.getByText('这个主单已经发生变动，请重新刷新。'),
    ).toBeVisible();
    expect(getAnyDirtyTabKey()).toBeDefined();

    masterServices.get.mockResolvedValue(masterResponse('2'));
    fireEvent.click(screen.getByRole('button', { name: '刷新主单资料' }));
    fireEvent.click(await screen.findByRole('button', { name: '确认刷新' }));
    await waitFor(() =>
      expect(screen.getByLabelText('测试主单正文')).toHaveValue('主单正文-2'),
    );
    expect(
      screen.queryByText('这个主单已经发生变动，请重新刷新。'),
    ).not.toBeInTheDocument();
    expect(getAnyDirtyTabKey()).toBeUndefined();
    fireEvent.change(screen.getByLabelText('修改原因'), {
      target: { value: '采用新基线' },
    });
    fireEvent.click(screen.getByRole('button', { name: '预览影响' }));
    await waitFor(() => expect(masterServices.preview).toHaveBeenCalled());
    expect(masterServices.preview.mock.calls[0][1].expectedVersion).toBe('2');
  });
});
