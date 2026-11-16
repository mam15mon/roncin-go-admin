import { fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  masterResponse,
  masterServices,
  renderMaster,
  setupMasterTests,
} from './air-master-test.fixtures';

setupMasterTests();

async function createButton() {
  await waitFor(() => expect(masterServices.get).toHaveBeenCalled());
  await screen.findByRole('button', { name: '修改共同资料' });
  await waitFor(() =>
    expect(screen.getByRole('button', { name: '修改共同资料' })).toBeEnabled(),
  );
  return screen.getByRole('button', { name: '从本主单继续建票' });
}

describe('航空主单建票资格', () => {
  it.each([undefined, false])(
    'canAttach=%s 时即使成员为0也禁止继续建票',
    async (canAttach) => {
      masterServices.get.mockResolvedValue(
        masterResponse('1', {
          canAttach,
          memberCount: 0,
          attachBlockedReason: '该主单暂不允许新建操作票',
        }),
      );
      renderMaster();
      expect(await createButton()).toBeDisabled();
      expect(screen.getByText('该主单暂不允许新建操作票')).toBeVisible();
      expect(masterServices.push).not.toHaveBeenCalled();
    },
  );

  it.each(['DIRECT', 'HOUSE'])(
    '明确允许时携带服务端%s模式继续建票',
    async (documentMode) => {
      masterServices.get.mockResolvedValue(
        masterResponse('1', { documentMode, memberCount: 99 }),
      );
      renderMaster();
      fireEvent.click(await createButton());
      expect(masterServices.push).toHaveBeenCalledWith(
        `/orders/air-export/new?masterId=master&documentMode=${documentMode}`,
      );
    },
  );

  it('尚未固定模式的空主单要求先显式选择，不自动推断HOUSE', async () => {
    masterServices.get.mockResolvedValue(
      masterResponse('1', { documentMode: undefined }),
    );
    renderMaster();
    fireEvent.click(await createButton());
    expect(masterServices.push).not.toHaveBeenCalled();
    const proceed = await screen.findByRole('button', { name: '继续建票' });
    expect(proceed).toBeDisabled();
    fireEvent.click(screen.getByLabelText('直单（DIRECT，无 HAWB）'));
    fireEvent.click(proceed);
    expect(masterServices.push).toHaveBeenCalledWith(
      '/orders/air-export/new?masterId=master&documentMode=DIRECT',
    );
  });
});

describe('航空主单恢复关联', () => {
  it.each(['DIRECT', 'HOUSE'])(
    '%s恢复按模式处理真实HAWB',
    async (documentMode) => {
      masterServices.get.mockResolvedValue(
        masterResponse('1', { documentMode }),
      );
      renderMaster();
      await createButton();
      fireEvent.click(
        screen.getByRole('button', { name: '恢复关联已终止订单' }),
      );
      const order = await screen.findByLabelText('选择空运订单');
      fireEvent.mouseDown(order);
      fireEvent.click(await screen.findByText(/AE001/));
      fireEvent.change(screen.getByLabelText('关联原因'), {
        target: { value: '恢复操作' },
      });
      if (documentMode === 'DIRECT') {
        expect(
          screen.queryByLabelText('目标主单内分单号'),
        ).not.toBeInTheDocument();
      } else {
        const house = screen.getByLabelText('目标主单内分单号');
        expect(house).toHaveValue('H001');
        fireEvent.change(house, { target: { value: '' } });
        fireEvent.click(screen.getByRole('button', { name: '确 定' }));
        await waitFor(() =>
          expect(screen.getByText('请输入分单号')).toBeVisible(),
        );
        expect(masterServices.link).not.toHaveBeenCalled();
        fireEvent.change(house, { target: { value: 'H001' } });
      }
      fireEvent.click(screen.getByRole('button', { name: '确 定' }));
      await waitFor(() => expect(masterServices.link).toHaveBeenCalled());
      const payload = masterServices.link.mock.calls[0][1];
      expect(payload).toMatchObject({
        orderId: 'order',
        expectedOrderVersion: '3',
        expectedMasterVersion: '1',
      });
      if (documentMode === 'DIRECT')
        expect(payload).not.toHaveProperty('houseNo');
      else expect(payload.houseNo).toBe('H001');
    },
  );
});
